[![Infinispan](https://design.jboss.org/infinispan/logo/final/PNG/infinispan_logo_rgb_lightbluewhite_darkblue_600px.png)](https://infinispan.org/)

# Infinispan SQL Store Demo

This demo illustrates how to build a high-performance product catalogue by integrating a traditional SQL-based backend with a distributed in-memory cache using Infinispan.

## Quick Start

### 1. Start Infrastructure

Start Infinispan and PostgreSQL using Docker Compose:

```shell
docker compose up -d
```

This starts:
- Infinispan servers (LON on port 11222, NYC on port 31222)
- PostgreSQL database on port 5432

### 2. Register Protobuf Schema

Register the Protobuf schema with Infinispan:

```shell
cd schema
./create-proto.sh
```

This uploads the retail schema to Infinispan's protobuf metadata cache.

### 3. Start Retail Catalogue

The retail-catalogue service creates the database schema and populates sample data:

```shell
cd retail-catalogue
mvn quarkus:dev -Dquarkus.devservices.enabled=false
```

The service runs on **http://localhost:8080**

You can verify it's working:
```shell
curl http://localhost:8080/commands
```

## Available Implementations

This demo provides multiple implementations of the in-memory catalogue service:

- **inmemory-catalogue-quarkus** - Quarkus implementation (HotRod client)
- **inmemory-catalogue-spring-boot** - Spring Boot 3 implementation (HotRod client)
- **inmemory-catalogue-nodejs** - Node.js/Express implementation (REST API v3) ⭐ NEW

Refer to each directory's README for specific setup instructions.

### Quick Start - Node.js Implementation

```shell
cd inmemory-catalogue-nodejs
npm install
npm start
```

The service runs on **http://localhost:8180**
