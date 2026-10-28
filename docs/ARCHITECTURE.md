# High-Level Architecture & System Design: AethelCore Engine

This document details the high-level architectural patterns, system topology diagram, inter-service communication strategies, and project directory hierarchy of AethelCore, tailored for a Go Workspaces (`go.work`) monorepo setup.

---

## 1. Architectural Philosophy

AethelCore is designed following **Clean Architecture (Hexagonal / Ports & Adapters)** and a distributed **Backend-for-Frontend (BFF)** pattern:

1. **Separation of Concerns**: Decouples protocol ingress (HTTP/REST via Gin, gRPC, GraphQL) from pure domain logic and double-entry accounting rules.
2. **Concurrency Determinism**: The core ledger transaction layer executes balance mutations sequentially and in isolation without risks of cyclic deadlocks.
3. **Auditability by Design**: Every balance mutation triggers cryptographic audit hash chaining (SHA-256) within the database transaction.
4. **Unified Database of Record & Caching**:
   * **MySQL 8.0 / TiDB (PingCAP)**: Serves as the *Primary Ledger & Identity Store of Record* (ACID transactions, pessimistic row locking, BIGINT precision, and distributed scalability).
   * **Redis**: Handles *Distributed Locks*, *Token Bucket Rate Limiting*, and *Idempotency Caching*.
   * **Kafka / Redpanda**: Asynchronous event broker distributing mutation events via the *Transactional Outbox Pattern*.

---

## 2. System Topology Diagram

```text
       ┌──────────────────────┐           ┌──────────────────────┐
       │   Mobile Banking     │           │  Branch Web Console  │
       │  (Customer Channel)  │           │ (Teller & Supervisor)│
       └──────────┬───────────┘           └──────────┬───────────┘
                  │ HTTPS / REST (JSON)              │ HTTPS / GraphQL & REST
                  ▼                                  ▼
      ┌─────────────────────────────────────────────────────────────┐
      │               API GATEWAY & INGRESS LAYER                   │
      │                     (Go + Gin Framework)                    │
      │  - Routing & Reverse Proxy                                  │
      │  - Token Bucket Rate Limiting (Redis)                       │
      │  - Dual JWT Authentication (Maker & Checker)                │
      │  - Idempotency-Key Deduplication Engine                     │
      └──────────────┬──────────────────────────────┬───────────────┘
                     │                              │
          gRPC/Protobuf (Internal)         Distributed Cache / Lock
                     ▼                              ▼
      ┌─────────────────────────────┐  ┌────────────────────────────┐
      │     CORE LEDGER ENGINE      │  │        REDIS 7 CACHE       │
      │     (Microservice Go)       │  │ - Token Bucket Rate Limit  │
      │ - Lexicographical Row Lock  │  │ - Idempotency SETNX Locks  │
      │ - Double-Entry Mutations    │  │ - Session Replay Shield    │
      │ - Cryptographic Audit Trail │  └────────────────────────────┘
      └──────────────┬──────────────┘
                     │
          SQL Connection Pool (MySQL Driver / go-sql-driver)
                     ▼
      ┌─────────────────────────────────────────────────────────────┐
      │         UNIFIED DATABASE OF RECORD: MySQL 8.0 / TiDB        │
      │                  (Local MySQL or TiDB Cloud)                │
      │  - accounts (BIGINT Balance, Available vs Hold)             │
      │  - transactions (Header, Maker-Checker State)               │
      │  - ledger_entries (Chained Cryptographic Audit Trail)       │
      │  - app_users & bank_branches (RBAC & KYC Store)             │
      │  - outbox_events (Transactional Outbox Pattern)             │
      └──────────────┬──────────────────────────────────────────────┘
                     │
        CDC / Transactional Outbox
                     ▼
      ┌─────────────────────────────┐
      │    EVENT BROKER (KAFKA)     │
      └──────────────┬──────────────┘
                     │
         Consumer Event Subscription
                     ▼
      ┌─────────────────────────────┐  ┌────────────────────────────┐
      │     WORKER-EOD ENGINE       │  │    SIMULATOR-CRON          │
      │ - Daily Interest Accrual    │  │ - Autonomous Traffic       │
      │ - Loan Installment Billing  │  │ - Extreme Load Testing     │
      │ - PPh 20% Deductions        │  │ - Synthetic Transaction Gen│
      └─────────────────────────────┘  └────────────────────────────┘
```

---

## 3. Inter-Service Communication Protocols

1. **External to API Gateway (Inbound)**:
   * **RESTful HTTP/JSON** via **Gin**: Used for customer and teller transactional operations (e.g., Funds Transfer, Cash Deposit, Cash Withdrawal).
   * **GraphQL** (`gqlgen`): Used for backoffice management, fetching complex ledger relations, and audit reporting without over-fetching.
2. **Gateway to Core Ledger (Internal IPC)**:
   * **gRPC / Protocol Buffers (HTTP/2)**: Provides ultra-low latency, compact binary serialization, and strictly typed interface contracts.
3. **Core Ledger to Background Workers (Asynchronous)**:
   * **Transactional Outbox Pattern via Kafka / Redpanda**: Guarantees that every committed database transaction publishes an event without data loss from network partitions.

---

## 4. Code Architecture Pattern: Clean Architecture

Each microservice under `apps/` adopts a consistent layered architecture:

```text
HTTP / gRPC Request
       │
       ▼
[ Delivery / Controller Layer ]  --> (Gin Handlers, gRPC Servers)
       │ (DTO validation)
       ▼
[ Usecase / Service Layer ]      --> (Transfer logic, Maker-Checker, Business Rules)
       │ (Entity orchestration)
       ▼
[ Domain Layer ]                 --> (Pure entities, accounting invariants, errors)
       │ (Via Interfaces)
       ▼
[ Repository / Adapter Layer ]   --> (MySQL / TiDB via sqlx / database/sql, Redis)
```

**Benefits**:
* Business logic remains decoupled from SQL queries or web frameworks.
* Replacing databases or adding new adapters requires zero changes to the Use Case layer (*Dependency Inversion Principle*).

---

## 5. Monorepo Directory Hierarchy (`go.work`)

```text
elif-bank/
├── .github/
│   └── workflows/
│       └── ci-pipeline.yml          # GitHub Actions: test, race-detector, lint
├── proto/                           # Source of truth for gRPC/Protobuf contracts
│   ├── account/v1/account.proto
│   ├── ledger/v1/ledger.proto
│   ├── branch/v1/branch.proto
│   └── eod/v1/eod.proto
├── apps/                            # Executable microservices
│   ├── api-gateway/                 # Go + Gin: Ingress REST, GraphQL, RBAC, JWT
│   │   ├── cmd/server/main.go
│   │   ├── internal/delivery/http/  # Gin handlers & middleware
│   │   ├── internal/delivery/graph/ # GraphQL resolvers & schema
│   │   ├── internal/client/         # gRPC client to Core Ledger
│   │   └── go.mod
│   ├── core-ledger/                 # Go: Core double-entry transaction engine
│   │   ├── cmd/server/main.go
│   │   ├── internal/domain/         # Entity, value objects, domain interfaces
│   │   ├── internal/usecase/        # Transfer engine, locking logic, audit chain
│   │   ├── internal/repository/     # MySQL / TiDB sqlx adapter
│   │   └── go.mod
│   ├── worker-eod/                  # Go: Day-end rollover batch engine (EOD)
│   │   ├── cmd/worker/main.go
│   │   └── go.mod
│   └── simulator-cron/              # Go: Autonomous transactional generator
│       ├── cmd/simulator/main.go
│       └── go.mod
├── pkg/                             # Shared utility packages
│   ├── hashchain/                   # SHA-256 hash chaining utilities
│   ├── idempotency/                 # Redis idempotency middleware & logic
│   ├── money/                       # Fixed-point precision math (BIGINT)
│   └── go.mod
├── deploy/                          # Local container configurations
│   ├── docker-compose.yml           # MySQL 8.0 / TiDB-compatible, Redis 7
│   ├── migrations/mysql_tidb/       # Unified Banking DDL Schema
│   └── config/
├── docs/                            # Architectural documentation & system guides
├── go.work                          # Go multi-module workspace orchestrator
└── README.md
```
