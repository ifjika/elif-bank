# Implementation Plan & Roadmap: Elif Core Bank Engine (Elif Bank)

> **Document Version:** `v1.4.0` | **Last Updated:** `2026-11-23` | **Status:** `Active`

This document defines the roadmap, phased feature implementation matrix, and the active task checklist for the feature currently being implemented.

---

## 1. Feature Implementation Matrix

|  Phase  | Module & Feature                                      | Associated Tech Stack                                        |        Status        | Definition of Done                                                                                                                              |
| :-----: | :---------------------------------------------------- | :----------------------------------------------------------- | :------------------: | :---------------------------------------------------------------------------------------------------------------------------------------------- |
|  **1**  | **Environment Setup & Multi-Module Workspace**        | Go 1.23+, `go.work`, Docker Compose, MySQL 8.0 / TiDB, Redis |   🟢 **Completed**   | All database and cache configurations ready; Go monorepo & frontend directory structures cleanly initialized.                                   |
|  **2**  | **Database Schema Design & SQL Migrations**           | MySQL 8.0, TiDB Cloud Serverless, SQL DDL                    |   🟢 **Completed**   | Unified 8-table banking-standard DDL (`000001_init_bank_schema.up.sql`) migrated & verified on TiDB Cloud Serverless.                           |
|  **3**  | **Core Ledger Engine: Double-Entry & Row Locking**    | Go, Lexicographical Locking, SHA-256 Hashchain, Money Math   |   🟢 **Completed**   | Atomic Double-Entry logic, lexicographical sorting, SHA-256 chain, and AAA unit tests passing 100%.                                             |
|  **4**  | **API Gateway: Gin REST, JWT, & Idempotency**         | Go, Gin Framework, JWT, Redis Token Bucket, RESTful API      |   🟢 **Completed**   | Gateway REST API, JWT token manager, Redis idempotency & rate limiting passing 100% unit tests.                                                 |
| **4.1** | **End-to-End JWT Authentication & RBAC Login System** | Go, `bcrypt`, TiDB Cloud, `HttpOnly` Cookie, React In-Memory |   🟢 **Completed**   | Live TiDB Cloud `app_users` lookup & auto-seeding, Bcrypt cost-12, dual JWT (`HttpOnly` cookie + RAM), silent refresh & AAA tests passing 100%. |
|  **5**  | **Branch Operations: Maker-Checker System**           | Go, Gin, RBAC, Dual JWT Claims, React UI                     |   🟢 **Completed**   | Cash transactions > 50M IDR enter `PENDING_APPROVAL` status and can only be approved/rejected by a Checker.                                     |
|  **6**  | **GraphQL Endpoint for Backoffice / Audit**           | Go, `gqlgen`, GraphQL                                        | 🟡 **Active / Next** | GraphQL query can display account relations and validate transaction data integrity hash chains.                                                |
|  **7**  | **Batch Engine: End-of-Day (EOD) Worker**             | Go, Cron/Tickers, Transactional Outbox, Kafka / Redpanda     |      ⚪ Planned      | Worker computes daily interest accrual, 20% withholding tax (PPh), and monthly admin fees via automated batch.                                  |
|  **8**  | **Autonomous Traffic Simulator**                      | Go Concurrency (Goroutines, Channels), Stress Testing        |      ⚪ Planned      | Generator produces 50–200 TPS continuous synthetic transactions to benchmark latency and stability.                                             |
|  **9**  | **CI/CD Pipeline & Test Automation**                  | GitHub Actions, `go test -race`, `golangci-lint`             |      ⚪ Planned      | Automated CI pipeline executes linting, race-detector, and build tests on every git commit.                                                     |

---

## 2. Active Task Checklist: Phase 6 - GraphQL Endpoint for Backoffice / Audit

> [!IMPORTANT]
> **Checklist Usage Rules**:
> This section **ONLY** contains the micro-steps for the feature **CURRENTLY** in progress. Once all steps below are confirmed and verified, this checklist is **cleared**, and replaced with the micro-steps of the next feature according to the implementation table above.

### Active Phase: **Phase 6 - GraphQL Endpoint for Backoffice / Audit**

- [ ] **6.1** Initialize `gqlgen` GraphQL schema and resolvers in `apps/api-gateway`:
  - Define types for `Account`, `Transaction`, `LedgerEntry`, and `IntegrityVerificationResult`.
- [ ] **6.2** Implement GraphQL queries for account relations and SHA-256 hash chain verification:
  - Query account balances (`balance`, `hold_balance`, `available_balance`) and transaction history.
  - Validate sequential `previous_hash` -> `current_hash` links across `ledger_entries`.
- [ ] **6.3** Protect `/graphql` with JWT + RBAC (`SUPERVISOR`, `ADMIN`) and write AAA unit tests.

---

## 3. Verified Checkpoints

| Checkpoint | Date | Scope Completed |
| :--- | :--- | :--- |
| **Phase 4.1** | `2026-11-23` | End-to-End JWT Auth (`bcrypt` cost-12, `HttpOnly` Refresh Cookie, In-Memory Access Token), TiDB `app_users` seeding, `last_login_at` tracking, React Matrix UI & `CurrencyInput`. |
| **Phase 5** | `2026-11-23` | Branch Operations Maker-Checker System (`branch_cash_usecase`, `branch_approval_usecase`, `/api/v1/branch/*` Gin handlers, TiDB `000002` + `ensureBranchTransactionColumns`, live React Web Console integration, `.air.toml` Go hot-reload). |
