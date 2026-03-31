# AI Guide: Implementing an Infinispan SQL Store Client

This guide provides all the essential information needed to implement a client application in any programming language for the Infinispan SQL Store demo.

## Overview

The client application creates an in-memory product catalogue using Infinispan's SQL Cache Stores. It exposes REST endpoints to query product data that is loaded from a PostgreSQL database through Infinispan caches.

### Architecture

```
Client Application (Your Implementation)
    ↓ REST API v3 (Digest Auth)
Infinispan Cache Servers
    ↓ JDBC
PostgreSQL Database
```

## Prerequisites

Before starting your implementation, ensure:

1. **Infrastructure is running**:
   ```bash
   docker compose up -d
   ```

2. **Protobuf schema is registered**:
   ```bash
   cd schema
   ./create-proto.sh
   ```

3. **Database schema is initialized** (retail-catalogue must run once):
   ```bash
   cd retail-catalogue
   mvn quarkus:dev -Dquarkus.devservices.enabled=false
   ```

## Infinispan Connection

### REST API Details

- **Base URL**: `http://localhost:11222/rest/v3`
- **Authentication**: HTTP Digest Authentication
  - Username: `admin`
  - Password: `secret`
  - **CRITICAL**: Algorithm MUST be `SHA-256` (not MD5 or other defaults)
- **OpenAPI Specification**: `http://localhost:11222/rest/v3/openapi`

### Testing Connection

```bash
# Should return: HEALTHY
curl http://localhost:11222/rest/v3/container/health/status
```

## Data Schema

The Protobuf schema is located at `schema/retail-schema.proto` and defines:

### RetailProductValue
Represents products in the catalogue (from `retailproduct` table):
- `code` (string): Product code
- `name` (string): Product name - **indexed for full-text search**
- `price` (double): Product price - **indexed for range queries**
- `stock` (int32): Stock quantity - **indexed for filtering**

### PurchasedProductValue
Represents sold products (joined from multiple tables):
- `name` (string): Product name - **indexed for full-text search**
- `country` (string): Customer country - **indexed for full-text search**

### PurchasedProductKey
Composite key for purchased products:
- `id` (int64): Command ID
- `products_id` (int64): Product ID

## Cache Configuration

Your implementation must create two caches on startup.

### Cache 1: catalogue-table-store

**Type**: Table-based SQL Store
**Configuration Template**: Copy from `inmemory-catalogue-quarkus/src/main/resources/tableStore.xml`

**Placeholders to replace**:
- `CONNECTION_URL` → `jdbc:postgresql://database:5432/retailstore`
- `USER_NAME` → `infinispan`
- `PASSWORD` → `secret`
- `DIALECT` → `POSTGRES`
- `DRIVER` → `org.postgresql.Driver`

**Endpoint**: `POST /rest/v3/caches/catalogue-table-store`
**Content-Type**: `application/xml`

### Cache 2: sold-products-query-store

**Type**: Query-based SQL Store
**Configuration Template**: Copy from `inmemory-catalogue-quarkus/src/main/resources/queryStore.xml`

**Placeholders to replace**: Same as above

**Endpoint**: `POST /rest/v3/caches/sold-products-query-store`
**Content-Type**: `application/xml`

### Cache Creation Logic

```
1. Check if cache exists: HEAD /rest/v3/caches/{cacheName}
   - 200 → Cache exists, skip creation
   - 404 → Cache doesn't exist, proceed to create

2. Create cache: POST /rest/v3/caches/{cacheName}
   - Body: XML configuration with placeholders replaced
   - 200 → Success
   - 409 → Already exists (race condition, safe to ignore)
```

## Critical Implementation Details

### 1. Digest Authentication with SHA-256

Most HTTP client libraries default to MD5 for digest auth. **You MUST specify SHA-256**:

Example configuration patterns:
```javascript
// Node.js with digest-fetch
new DigestClient(username, password, { algorithm: 'SHA-256' })
```

### 2. Ickle Query Syntax

Infinispan uses Ickle (Infinispan Query Language), similar to JPQL.

**Full-text search** (for @Text fields like `name` and `country`):
```
name: (+'searchTerm')
```

**NOT**:
- `name: '*searchTerm*'` ❌
- `name: (+searchTerm)` ❌ (missing quotes)
- `name LIKE '%searchTerm%'` ❌

**Exact match**:
```
code = 'c123'
```

**Range queries**:
```
stock >= 100
price : [20 to 100]
```

**Combining conditions**:
```
from retail.RetailProductValue where name: (+'Party') and stock >= 50
```

### 3. Query Execution

**Endpoint**: `GET /rest/v3/caches/{cacheName}/_search?query={ickleQuery}&max_results={limit}`

**CRITICAL**: `max_results` parameter:
- DO NOT use `Number.MAX_SAFE_INTEGER` or equivalent large numbers
- Infinispan cannot parse integers > ~2^31
- Use reasonable limit like **10000**

**Response format**:
```json
{
  "hit_count": 4,
  "hit_count_exact": true,
  "hits": [
    {
      "hit": {
        "_type": "retail.RetailProductValue",
        "code": "c123",
        "name": "Skirt Party",
        "price": 50.0,
        "stock": 20
      }
    }
  ]
}
```

Access results: `response.hits.map(item => item.hit)`

### 4. Cache Size

**Endpoint**: `GET /rest/v3/caches/{cacheName}/_size`

Returns plain text number (not JSON).

### 5. Reindexing

**Endpoint**: `POST /rest/v3/caches/{cacheName}/_reindex`

Trigger after cache creation or if search results seem incomplete.

## Required API Endpoints

Your client application should expose these REST endpoints (default port: 8180):

### GET /health

Returns service status and cache sizes.

**Example Response**:
```
Service is up! catalogue[18] sold_products[62]
```

**Implementation**:
```
1. Get size of catalogue-table-store
2. Get size of sold-products-query-store
3. Return formatted string
```

### GET /reindex

Triggers reindexing of both caches.

**Example Response**:
```
Reindex launched
```

**Implementation**:
```
1. POST /rest/v3/caches/catalogue-table-store/_reindex
2. POST /rest/v3/caches/sold-products-query-store/_reindex
3. Return success message
```

### GET /catalogue

Lists all products with optional filters.

**Query Parameters**:
- `name` (optional): Product name to search (full-text)
- `stock` (optional): Minimum stock level
- `price-min` (optional): Minimum price
- `price-max` (optional): Maximum price

**Example Requests**:
```bash
GET /catalogue
GET /catalogue?name=Party
GET /catalogue?stock=100
GET /catalogue?price-min=20&price-max=100
GET /catalogue?name=Skirt&stock=10
```

**Implementation**:
```
Build Ickle query:
- If name: add "name: (+'<name>')"
- If stock: add "stock >= <stock>"
- If price-min and price-max: add "price : [<min> to <max>]"

Query: "from retail.RetailProductValue where <conditions>"
If no conditions: "from retail.RetailProductValue"

Execute query on catalogue-table-store
Return: hits.map(item => item.hit)
```

**Example Response**:
```json
[
  {
    "_type": "retail.RetailProductValue",
    "code": "c123",
    "name": "Skirt Party",
    "price": 50.0,
    "stock": 20
  }
]
```

### GET /catalogue/:code

Returns a single product by code.

**Example Request**:
```bash
GET /catalogue/c123
```

**Implementation**:
```
Query: "from retail.RetailProductValue where code = '<code>'"
max_results: 1

If hits.length === 0: return 404
Else: return hits[0].hit
```

**Example Response**:
```json
{
  "_type": "retail.RetailProductValue",
  "code": "c123",
  "name": "Skirt Party",
  "price": 50.0,
  "stock": 20
}
```

### GET /sales

Lists sold products with optional filters.

**Query Parameters**:
- `name` (optional): Product name to search (full-text)
- `country` (optional): Customer country to search (full-text)

**Example Requests**:
```bash
GET /sales
GET /sales?name=Skirt
GET /sales?country=Spain
GET /sales?name=Party&country=France
```

**Implementation**:
```
Build Ickle query:
- If name: add "name: (+'<name>')"
- If country: add "country: (+'<country>')"

Query: "from retail.PurchasedProductValue where <conditions>"
If no conditions: "from retail.PurchasedProductValue"

Execute query on sold-products-query-store
Return: hits.map(item => item.hit)
```

**Example Response**:
```json
[
  {
    "_type": "retail.PurchasedProductValue",
    "name": "Skirt Party",
    "country": "Spain"
  }
]
```

## Testing Your Implementation

### 1. Verify Caches are Created

```bash
curl --digest -u admin:secret http://localhost:11222/rest/v3/caches
```

Should include: `catalogue-table-store` and `sold-products-query-store`

### 2. Check Cache Sizes

```bash
curl --digest -u admin:secret http://localhost:11222/rest/v3/caches/catalogue-table-store/_size
```

Should return: `18` (or similar number)

### 3. Test Direct Query

```bash
curl --digest -u admin:secret -G \
  "http://localhost:11222/rest/v3/caches/catalogue-table-store/_search" \
  --data-urlencode "query=from retail.RetailProductValue where name: (+'Party')"
```

Should return products with "Party" in the name.

### 4. Test Your Endpoints

```bash
# Health check
curl http://localhost:8180/health

# Get all products
curl http://localhost:8180/catalogue

# Get product by code
curl http://localhost:8180/catalogue/c123

# Search by name
curl "http://localhost:8180/catalogue?name=Party"

# Filter by stock
curl "http://localhost:8180/catalogue?stock=100"

# Search sales
curl "http://localhost:8180/sales?name=Skirt"

# Filter by country
curl "http://localhost:8180/sales?country=Spain"
```

## Common Pitfalls

1. **Digest Auth without SHA-256**: Most libraries default to MD5. Must explicitly specify SHA-256.

2. **Wrong Query Syntax**: Ickle is NOT SQL. Full-text search requires `name: (+'term')` format.

3. **Large max_results**: Use 10000 or less, not MAX_INT.

4. **Missing Quotes in Queries**: `name: (+Party)` fails, must be `name: (+'Party')`

5. **Not Checking Cache Existence**: Always check if cache exists before creating to avoid errors.

6. **Forgetting to Reindex**: After cache creation, trigger reindex to populate search indexes.

7. **Wrong Content-Type**: Cache creation requires `Content-Type: application/xml`

## Reference Implementations

Study these for language-specific examples:

- **Java/Quarkus**: `inmemory-catalogue-quarkus/`
  - Uses Infinispan HotRod client (not REST)
  - Good reference for query syntax and cache configuration

- **Java/Spring Boot**: `inmemory-catalogue-spring-boot/`
  - Similar to Quarkus implementation
  - Shows Spring integration patterns

- **Node.js/Express**: `inmemory-catalogue-nodejs/`
  - Uses REST API v3 (closest to what you'll implement)
  - Shows digest-fetch library with SHA-256
  - Reference for REST endpoint patterns

## Additional Resources

- [Infinispan REST API v3 Documentation](https://infinispan.org/docs/stable/titles/rest/rest.html)
- [Ickle Query Language Guide](https://infinispan.org/docs/stable/titles/query/query.html#ickle-query-language)
- [Infinispan SQL Cache Store](https://infinispan.org/docs/stable/titles/configuring/configuring.html#sql-cache-store_persistence)
