# AethelCore Engine (elif-bank)

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Architecture](https://img.shields.io/badge/Architecture-Hexagonal%20%2F%20Clean-orange?style=flat)]()
[![Monorepo](https://img.shields.io/badge/Monorepo-Go%20Workspaces-blue?style=flat)]()
[![Database](https://img.shields.io/badge/Database-MySQL%208.0%20%7C%20TiDB%20%7C%20Redis-blueviolet?style=flat)]()
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

AethelCore (**elif-bank**) is a high-performance, distributed **Mini Core Banking System (CBS)** engineered in **Golang**. It replicates the mission-critical financial backend operations of modern enterprise banks, implementing double-entry accounting standards, ACID transaction isolation with strict concurrency determinism, branch governance (_Maker-Checker / Four-Eyes Principle_), tamper-evident cryptographic audit trails, and resilient API Gateway ingress.

The system is designed for **MySQL 8.0 protocol compatibility** and native support for **TiDB Cloud Serverless (PingCAP)**—delivering distributed HTAP scalability with standard MySQL tooling.

---

## 👨‍💻 About the Author

Hi, I'm **Jika**!

- 🐙 **GitHub**: [@ifjika](https://github.com/ifjika)
- 🌐 **Portfolio Website**: [ifjika-dev.vercel.app](https://ifjika-dev.vercel.app)

I engineered this project as an end-to-end deep dive into **Core Banking System (CBS) architecture**. Commercial core banking systems serve as the critical heartbeat of modern financial infrastructure—where software bugs translate to direct monetary loss or regulatory failures. My objective with **AethelCore** is to demystify and master the foundational mechanics of high-concurrency ledger design, zero-deadlock row locking, cryptographic data integrity, and enterprise-grade distributed transaction patterns in Go on a modern distributed SQL database (MySQL / TiDB).

---

## 🏛️ Key Architectural Features

### 1. Double-Entry General Ledger Engine

- **Mathematical Invariant**: Every monetary movement creates at least two balanced journal entries:
  $$\sum \text{Debit} - \sum \text{Credit} = 0$$
- **Zero Direct Updates**: Raw balance mutations (`UPDATE accounts SET balance = balance - x`) are strictly prohibited; all state mutations are recorded via immutable debit and credit journals.

### 2. Concurrency Determinism & Deadlock Prevention

- **Lexicographical Row-Locking**: Eliminates cyclic deadlocks during simultaneous cross-transfers (e.g., Account A $\to$ Account B while Account B $\to$ Account A) by sorting account UUIDs alphabetically before acquiring pessimistic SQL row locks (`SELECT ... FOR UPDATE`).
- **Fixed-Point Math**: Replaces IEEE 754 floating-point types (`float32`/`float64`) with exact whole integer arithmetic (`BIGINT` / `int64`) to eliminate penny-rounding fraud.

### 3. Tamper-Evident Cryptographic Audit Trail

- Every ledger row is bound to the preceding mutation entry of the same account via a linked **SHA-256 Hash Chain**:
  $$\text{CurrentHash} = \text{SHA256}(\text{PreviousHash} \parallel \text{AccountID} \parallel \text{Amount} \parallel \text{BalanceAfter} \parallel \text{Timestamp})$$
- Any direct manipulation of historical database records breaks the cryptographic chain and triggers immediate anomaly alarms during automated batch audits.

### 4. Branch Governance: Four-Eyes Principle (Maker-Checker)

- **Straight-Through Processing (STP)**: Over-the-counter cash transactions $\le \text{IDR 50,000,000}$ are posted immediately by Tellers (Makers).
- **Dual Authorization**: Transactions $> \text{IDR 50,000,000}$ automatically transition to `PENDING_APPROVAL`, locking funds into `hold_balance` until authorized by a Supervisor (Checker) with distinct cryptographic JWT claims (`maker_id != checker_id`).

### 5. Defensive API Gateway & Resilience Layer

- **Atomic Idempotency Shield**: Uses Redis atomic `SETNX` locks with `X-Idempotency-Key` headers to protect against network timeouts, retry storms, and replay attacks.
- **Token Bucket Rate Limiting**: Distributed rate-limiting middleware safeguarding against brute force and denial-of-service attempts.
- **RFC 7807 Problem Details**: Consistent, standardized error responses across all REST endpoints.

---

## 📐 System Topology

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

## 🗂️ Monorepo Structure (`go.work`)

```text
elif-bank/
├── apps/
│   ├── api-gateway/                 # Ingress REST API, JWT RBAC, Rate Limiting, Idempotency
│   │   ├── internal/auth/           # HMAC-SHA256 Token Manager & RFC 7519 Claims
│   │   ├── internal/delivery/http/  # Gin REST Handlers & DTOs
│   │   └── internal/middleware/     # Auth, Idempotency & Rate Limit Middlewares
│   ├── core-ledger/                 # Primary Double-Entry Transaction Processing Engine
│   │   ├── internal/domain/         # Pure Entities (Account, Transaction, LedgerEntry) & Ports
│   │   └── internal/usecase/        # Transfer Engine, Locking Algorithms & Hash Chaining
│   ├── worker-eod/                  # Scheduled End-of-Day (EOD) Batch Engine (Interest Accrual, Taxes)
│   └── simulator-cron/              # Autonomous High-Frequency Synthetic Transaction Generator
├── pkg/                             # Shared Reusable Go Libraries
│   ├── hashchain/                   # SHA-256 Linked-List Cryptographic Audit Hasher
│   └── money/                       # Overflow-protected Fixed-Point Integer Arithmetic
├── deploy/                          # Container Orchestration & Migrations
│   ├── docker-compose.yml           # Local cluster (MySQL 8.0 / TiDB-compatible, Redis 7)
│   └── migrations/mysql_tidb/       # Unified Banking DDL Schema
├── docs/                            # Comprehensive Enterprise Architecture & System Documentation
├── go.work                          # Go Multi-Module Workspace Configuration
└── README.md
```

---

## 🛠️ Tech Stack & Engineering Standards

| Domain                       | Technology / Standard                           | Usage                                                                       |
| :--------------------------- | :---------------------------------------------- | :-------------------------------------------------------------------------- |
| **Language**                 | **Go (Golang 1.23+)**                           | Ultra-low latency, concurrent goroutine worker execution                    |
| **Architecture**             | **Hexagonal / Clean Architecture**              | Decoupled domain business rules from database and transport protocols       |
| **Workspace**                | **Go Workspaces (`go.work`)**                   | Monorepo multi-module coordination                                          |
| **Database of Record**       | **MySQL 8.0 / TiDB (PingCAP)**                  | Unified ACID ledger, row-level locking, accounts, and KYC/Auth schema       |
| **Cache & Distributed Lock** | **Redis 7**                                     | Distributed locking, Token Bucket rate limiting, atomic `SETNX` idempotency |
| **Event Streaming**          | **Kafka / Redpanda**                            | Asynchronous events via Transactional Outbox Pattern                        |
| **API Transport**            | **Gin (REST / JSON) & gRPC / Protobuf**         | External customer ingress and low-latency internal microservice IPC         |
| **Testing**                  | **Go `testing`, AAA Pattern, `-race` detector** | Comprehensive unit & concurrency race-condition testing                     |

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.23+** installed on your workstation.
- **Docker** & **Docker Compose** installed and running.

### 1. Clone & Setup Environment

```bash
git clone https://github.com/ifjika/elif-bank.git
cd elif-bank

# Copy environment configuration template
cp .env.example .env
```

### 2. Start Infrastructure via Docker Compose

```bash
docker compose -f deploy/docker-compose.yml up -d
```

- **MySQL 8.0 (TiDB Compatible)**: `localhost:3306` (Database: `aethel_bank_db`, User: `aethel_user`)
- **Redis**: `localhost:6379`

### 3. Run Test Suite Across All Modules

Execute unit tests with Go's built-in data race detector:

```bash
# Test shared packages
go test -v ./pkg/...

# Test Core Ledger engine
go test -v ./apps/core-ledger/...

# Test API Gateway & Middlewares
go test -v ./apps/api-gateway/...
```

---

## 📚 Architectural Documentation

The project includes in-depth technical documentation in the [`docs/`](docs/) directory:

- 📄 [**OVERVIEW.md**](docs/OVERVIEW.md) — System Overview, Domain Models & Performance Targets.
- 📄 [**PRD.md**](docs/PRD.md) — Product Requirements Document & Banking Business Logic.
- 📄 [**ARCHITECTURE.md**](docs/ARCHITECTURE.md) — Clean Architecture, Hexagonal Boundaries & System Topology.
- 📄 [**TSD.md**](docs/TSD.md) — Technical Specification: Database Schema, Lexicographical Locking & Math Invariants.
- 📄 [**APP_FLOW.md**](docs/APP_FLOW.md) — User Flows, Transaction FSM & Account State Machines.
- 📄 [**SECURITY_GUIDE.md**](docs/SECURITY_GUIDE.md) — RBAC Matrix, Threat Modeling (OWASP API) & Secrets Management.
- 📄 [**INTEGRATION.md**](docs/INTEGRATION.md) — RFC 7807 Error Specs, REST Endpoints & Webhook Payloads.
- 📄 [**TESTING_GUIDE.md**](docs/TESTING_GUIDE.md) — Testing Standards, AAA Pattern & Concurrency Simulation.
- 📄 [**DESIGN_SYSTEM.md**](docs/DESIGN_SYSTEM.md) — UI/UX Standards, Monetary Formatting & Design Tokens.
- 📄 [**IMPLEMENTATION_PLAN.md**](docs/IMPLEMENTATION_PLAN.md) — Phased Engineering Roadmap & Definition of Done.

---

## 📄 License

This project is open-source software licensed under the [MIT License](LICENSE).
