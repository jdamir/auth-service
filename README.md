# Authentication & Session Microservice (Go)

High-performance, stateless authentication service utilizing **gRPC** for internal communication, **PostgreSQL** for immutable user records, and **Redis** for seamless session management.

## Tech Stack
* **Language:** Go (1.27+)
* **Transport:** gRPC / HTTP (via Nginx proxy)
* **Databases:** PostgreSQL 15, Redis 7
* **Token Architecture:** Short-lived JWT (Access) + UUID Session Clamping (Refresh)

## Architectural Flow
1. **User Registration:** Interacts with external Governmental SOAP Verification Platform to validate citizen identity parameters before standard registration layout.
2. **Seamless Session Lifecycles:** Dispatches JWT access credentials embedded with `group_id` constraints, reducing high-frequency database lookups across core functional modules (Agro-Tracker Core).

## Pre-requisites & Local Deployment

Before spinning up the containers, ensure you establish the shared external bridge network required for cross-module orchestration:

```bash
docker network create auth-shared-network
```

To run the full infrastructure suite (Database, Caching layers, Migrations, App Node, and Nginx reverse proxy):

```bash
docker compose up --build -d
```

## Environment Specifications
* `PORT`: Internal gRPC execution binding (`:50051`)
* `JWT_SECRET`: Crypto signature matrix configuration
* `DB_DSN`: PostgreSQL database resource mapping URL
