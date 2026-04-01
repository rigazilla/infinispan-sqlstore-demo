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

## Infinispan Connection

### REST API Details

- **Base URL**: `http://localhost:11222`
- **Authentication**: HTTP Digest Authentication (DIGEST auth is REQUIRED, BASIC authentication is NOT supported)
  - Username: `admin`
  - Password: `secret`
  - **CRITICAL**: Algorithm MUST be `SHA-256` (not MD5 or other defaults)
- **OpenAPI Specification**: `http://localhost:11222/rest/v3/openapi`

**IMPORTANT**: Use `http://localhost:11222` as the base URL (without `/rest/v3`). The OpenAPI schema returns full paths like `/rest/v3/caches/{cacheName}`, so append these directly to the base URL to avoid path duplication.

### IMPORTANT: Use OpenAPI Schema

All the needed Infinispan operations are describe by the openAPI schema. Have a look at https://swagger.io/specification/
for more info.

**You MUST fetch and use the OpenAPI schema to discover all endpoints you need in the code.**
OpenAPI schema is avalable as a json file infinispan-api.json
```

1. Use the OpenAPI schema to get the endpoint for the operation you need
2. The operationId filed in the json schema is the name of the operation you're looking for
2. Build the correct endpoint paths from the schema
3. Action are not specified via a query param, i.e. '?action=', they are part of the endpoint prepend by an '_' instead, i.e. '{cacheName}/_search'
4. You must use these endpoint also for curl dev and testing commands

**Operation IDs you'll need**:
- `postCache` or `putCache` - Create/update cache
- `getCacheSize` - Get number of entries in cache
- `reindex` - Rebuild search indexes
- `postQueryCache` - Execute Ickle queries (RECOMMENDED - use POST method)
- `queryCache` - Execute Ickle queries (alternative GET method)
- `getContainerHealthStatus - Get the container health status
- `cacheExists` - Determines if a cache exists
- `getCacheSize` - Retrieves the number of entries in the cache

### Testing Connection

```bash
# Should return: HEALTHY
curl {getContainerHealthStatus endpoint}
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

If caches doesn't not exists, your implementation MUST create two caches on startup.
You MUST use cache configurations available in inmemory-catalogue-quarkus/src/main/resources

### Cache 1: catalogue-table-store

**Type**: Table-based SQL Store
**Configuration Template**: Copy from `inmemory-catalogue-quarkus/src/main/resources/tableStore.xml`

**Placeholders to replace**:
- `CONNECTION_URL` → `jdbc:postgresql://database:5432/retailstore`
- `USER_NAME` → `infinispan`
- `PASSWORD` → `secret`
- `DIALECT` → `POSTGRES`
- `DRIVER` → `org.postgresql.Driver`

**API Call**: Use `postCache` or `putCache` operation from OpenAPI schema
**Content-Type**: `application/xml`

### Cache 2: sold-products-query-store

**Type**: Query-based SQL Store
**Configuration Template**: Copy from `inmemory-catalogue-quarkus/src/main/resources/queryStore.xml`

**Placeholders to replace**: Same as above

**API Call**: Use `postCache` or `putCache` operation from OpenAPI schema
**Content-Type**: `application/xml`

### Cache Creation Logic

Consult the OpenAPI schema for exact endpoints. The typical flow:

```
1. Check if cache exists (HEAD request to cache endpoint)
   - 200 or 204 → Cache exists, skip creation
   - 404 → Cache doesn't exist, proceed to create

2. Create cache (POST request with XML configuration)
   - Body: XML configuration with placeholders replaced
   - 200 → Success
   - 409 or 400 with "already exists" → Race condition, safe to ignore

3. CRITICAL: Trigger reindex after cache creation
   - POST to {reindex endpoint} (see section 5 below)
   - SQL Store caches don't auto-populate indexes
   - Without this step, all queries will return empty results!
```

## Critical Implementation Details

### 1. Digest Authentication with SHA-256

Most HTTP client libraries default to MD5 for digest auth. **You MUST specify SHA-256**:

**IMPORTANT: Use well-tested digest authentication libraries instead of implementing it yourself.**

Recommended libraries:
- **Node.js**: `digest-fetch` - `new DigestClient('admin', 'secret', { algorithm: 'SHA-256' })`
- **Python**: `requests` with `HTTPDigestAuth` - `auth=HTTPDigestAuth('admin', 'secret')`
- **Go**: `github.com/icholy/digest`
- **Java**: Apache HttpClient with DigestScheme
- **curl**: `curl --digest -u admin:secret`

Custom implementations often fail with errors like `400 - COM00501: Expected padding`. If curl works but your code doesn't, your digest auth implementation is likely the issue.

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

**RECOMMENDED: Use POST method for queries to avoid URL encoding issues with special characters.**

**Use the OpenAPI schema to find the `postQueryCache` operation.**

**Method**: POST
**Endpoint**: {queryCache endpoint}
**Content-Type**: `application/json`

Request body (JSON):
```json
{
  "query": "from retail.RetailProductValue where name: (+'Party')",
  "max_results": 10000,
  "start_offset": 0,
  "hit_count_accuracy": 0
}
```

Fields:
- `query` (required): The Ickle query string
- `max_results` (optional): Maximum number of results (use 10000 or less, NOT MAX_INT)
- `start_offset` (optional): Result offset for pagination
- `hit_count_accuracy` (optional): Hit count accuracy

**Why POST instead of GET?**
- Avoids URL encoding issues with special characters in Ickle queries (parentheses, quotes, plus signs)
- More reliable across different HTTP client libraries
- Handles complex queries without digest auth complications
- JSON body is easier to construct than URL query parameters

**Alternative: GET method**
If you prefer GET, use the `queryCache` operation. Note: Some HTTP client libraries have issues with digest authentication and URL-encoded special characters. Test with curl first.

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

**Use the OpenAPI schema to find the `getCacheSize` operation.**

Returns: Integer (JSON format) representing the number of entries.

### 5. Reindexing

**CRITICAL: SQL Store caches require manual reindexing after creation.**

SQL Store caches with indexing enabled do not automatically populate search indexes from the database. After creating a cache, you **must** trigger a reindex via openAPI endpoint, or all queries will return empty results (except exact key lookups).

**Example**:
```bash
curl --digest -u admin:secret -X POST {reindex endpoint}
  
```

**When to reindex**:
- REQUIRED: Immediately after creating a new SQL Store cache
- Optional: If search results seem incomplete or stale
- Optional: After database schema changes

**Common mistake**: Using the wrong endpoint like `/search/indexes?action=reindex` (returns "Resource not found"). The correct endpoint from the OpenAPI schema is `/_reindex`.

## Required API Endpoints

Your inmemory application should expose these REST endpoints (default port: 8180):

### GET /health

Returns service status and cache sizes.

**Example Response**:
```
Service is up! catalogue[18] sold_products[62]
```

**Implementation**:
```
1. Call getCacheSize operation for catalogue-table-store
2. Call getCacheSize operation for sold-products-query-store
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
1. Call reindex operation for catalogue-table-store
2. Call reindex operation for sold-products-query-store
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

Call queryCache operation on catalogue-table-store
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

Call queryCache operation on sold-products-query-store
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

### 1. Fetch the OpenAPI Schema First

```bash
curl -s http://localhost:11222/rest/v3/openapi > infinispan-api.json
```

Parse this schema to find the correct endpoints for each operation.

### 2. Verify Caches are Created

Use the `getCacheList` operation from the OpenAPI schema to list all caches.

Should include: `catalogue-table-store` and `sold-products-query-store`

### 3. Check Cache Sizes

Use the `getCacheSize` operation from the OpenAPI schema.

Expected result: `18` (or similar number) for catalogue-table-store

### 4. Test Your Application Endpoints

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

1. **Not Using OpenAPI Schema**: Hardcoding endpoints will break. Always fetch and parse the OpenAPI schema to discover correct endpoints.

2. **Using GET instead of POST for queries**: Use POST method (`postQueryCache` operation) to avoid URL encoding issues with special characters in Ickle queries.

3. **Digest Auth without SHA-256**: Most libraries default to MD5. Must explicitly specify SHA-256.

4. **Custom Digest Auth Implementation**: Use proven libraries (digest-fetch, HTTPDigestAuth, etc.) instead of implementing yourself. Custom implementations often fail with errors like `400 - COM00501: Expected padding`.

5. **Wrong Query Syntax**: Ickle is NOT SQL. Full-text search requires `name: (+'term')` format.

6. **Large max_results**: Use 10000 or less, not MAX_INT.

7. **Missing Quotes in Queries**: `name: (+Party)` fails, must be `name: (+'Party')`

8. **Not Checking Cache Existence**: Always check if cache exists (HEAD returns 200 or 204) before creating to avoid errors.

9. **Forgetting to Reindex**: SQL Store caches REQUIRE manual reindexing after creation. Without this, queries return empty results. Use the correct endpoint: `POST /_reindex` (not `/search/indexes?action=reindex`).

10. **Wrong Reindex Endpoint**: The correct endpoint is `/_reindex`, not `/search/indexes?action=reindex`. Check the OpenAPI schema for the exact path.

11. **Wrong Content-Type**: Cache creation requires `Content-Type: application/xml`, query POST requires `Content-Type: application/json`

## Reference Implementations

Study these for language-specific examples:

- **Java/Quarkus**: `inmemory-catalogue-quarkus/`
  - Uses Infinispan HotRod client (not REST)
  - Good reference for query syntax and cache configuration

- **Java/Spring Boot**: `inmemory-catalogue-spring-boot/`
  - Similar to Quarkus implementation
  - Shows Spring integration patterns

## Additional Resources

- [Infinispan REST API v3 Documentation](https://infinispan.org/docs/stable/titles/rest/rest.html)
- [Ickle Query Language Guide](https://infinispan.org/docs/stable/titles/query/query.html#ickle-query-language)
- [Infinispan SQL Cache Store](https://infinispan.org/docs/stable/titles/configuring/configuring.html#sql-cache-store_persistence)
