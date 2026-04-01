[![Infinispan](https://design.jboss.org/infinispan/logo/final/PNG/infinispan_logo_rgb_lightbluewhite_darkblue_600px.png)](https://infinispan.org/)

# Infinispan SQL Store Demo - OpenAPI Edition

This demo illustrates how to build a high-performance product catalogue by integrating a traditional SQL-based backend with a distributed in-memory cache using Infinispan.

**This branch (`openapi-demo`) is a derived work that demonstrates how easy it is to implement applications using the new Infinispan 16.1 OpenAPI interface, with assistance from an AI coding agent.**

## AI Assistant and OpenAPI for polyglot development

This demo serves as a practical example of how OpenAPI could be a good choice for vibe-coding polyglot applications. Key benefits include:

- **Rapid Prototyping**: AI agents can quickly understand and implement OpenAPI specifications
- **Code Generation**: Automatic generation of client code from OpenAPI definitions
- **Multi-language Support**: Seamless implementation across different technology stacks

### OpenAPI's Role in AI-Assisted Development

Infinispan 16.1's OpenAPI interface provides detailed API descriptions that help AI agents:

- Parse specifications to understand operations
- Generate precise client code
- Handle data types and responses accurately
- Work autonomously with clear contracts

This demo shows productivity gains from AI-assisted development using OpenAPI.

### Polyglot Architecture

OpenAPI's language-agnostic nature enables applications to be written in any programming language:

- **Language Independence**: Implement clients in Java, Python, Go, JavaScript, or any language with HTTP support
- **Consistent Interfaces**: Same API contract across all implementations
- **Technology Flexibility**: Choose the best language for each component

## Vibe Coding

Tell your coding assistant to read the instructions in the [AI_CODING_INMEMORY_CATALOG.md](AI_CODING_INMEMORY_CATALOG.md) and ask them to vibe-code an inmemory catalog application in your preferred programming language using the already available Spring Boot and Quarkus implementations as an additional reference.

## Testing

Before testing the application you need to start the rest of the system.

### Start Infinispan and PostgreSQL
Start Infinispan and PostgreSQL using Docker Compose:

```shell
docker compose up -d
```

This starts:
- Infinispan servers (LON on port 11222, NYC on port 31222)
- PostgreSQL database on port 5432

### Setup Schemas

The system setup is out of the scope of the demo, so these manual steps are needed.

Register the Protobuf schema with Infinispan:

```shell
cd schema
./create-proto.sh
```

The retail-catalogue service creates the database schema and populates sample data:

```shell
cd retail-catalogue
mvn quarkus:dev -Dquarkus.devservices.enabled=false
```

Once everything is running ask the assistant to run the application and test it.