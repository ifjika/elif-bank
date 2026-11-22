# Implementation Plan & Roadmap: Elif Core Bank Engine (Elif Bank)

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
|  **5**  | **Branch Operations: Maker-Checker System**           | Go, Gin, RBAC, Dual JWT Claims, React UI                     | 🟡 **Active / Next** | Cash transactions > 50M IDR enter `PENDING_APPROVAL` status and can only be approved/rejected by a Checker.                                     |
|  **6**  | **GraphQL Endpoint for Backoffice / Audit**           | Go, `gqlgen`, GraphQL                                        |      ⚪ Planned      | GraphQL query can display account relations and validate transaction data integrity hash chains.                                                |
|  **7**  | **Batch Engine: End-of-Day (EOD) Worker**             | Go, Cron/Tickers, Transactional Outbox, Kafka / Redpanda     |      ⚪ Planned      | Worker computes daily interest accrual, 20% withholding tax (PPh), and monthly admin fees via automated batch.                                  |
|  **8**  | **Autonomous Traffic Simulator**                      | Go Concurrency (Goroutines, Channels), Stress Testing        |      ⚪ Planned      | Generator produces 50–200 TPS continuous synthetic transactions to benchmark latency and stability.                                             |
|  **9**  | **CI/CD Pipeline & Test Automation**                  | GitHub Actions, `go test -race`, `golangci-lint`             |      ⚪ Planned      | Automated CI pipeline executes linting, race-detector, and build tests on every git commit.                                                     |

---

## 2. Active Task Checklist: Phase 5 - Branch Operations: Maker-Checker System

> [!IMPORTANT]
> **Checklist Usage Rules**:
> This section **ONLY** contains the micro-steps for the feature **CURRENTLY** in progress. Once all steps below are confirmed and verified, this checklist is **cleared**, and replaced with the micro-steps of the next feature according to the implementation table above.

### Active Phase: **Phase 5 - Branch Operations: Maker-Checker System**

- [ ] **5.1** Implement branch cash withdrawal/deposit usecase in `apps/core-ledger/internal/usecase/branch_cash_usecase.go`:
  - Transactions $\le \text{IDR 50,000,000} \to$ Straight-Through Processing (`status: POSTED`).
  - Transactions $> \text{IDR 50,000,000} \to$ Lock funds to `hold_balance`, `status: PENDING_APPROVAL`.
- [ ] **5.2** Implement Supervisor approval usecase in `apps/core-ledger/internal/usecase/branch_approval_usecase.go`:
  - Enforce _Four-Eyes Principle_ (`maker_id != checker_id`; return `403 Forbidden` if equal).
  - Branch `APPROVE`: permanent balance mutation, release `hold_balance`, record double-entry journal, status `POSTED`.
  - Branch `REJECT`: release `hold_balance` back to effective balance, status `REJECTED`.
- [ ] **5.3** Create Gin HTTP Handlers for branch endpoints in `apps/api-gateway/internal/delivery/http/branch_handler.go`:
  - `POST /api/v1/branch/cash-transactions` (Role: TELLER / Maker).
  - `POST /api/v1/branch/approvals/:transaction_id` (Role: SUPERVISOR / Checker).
- [ ] **5.4** Integrate live endpoints into React `apps/web-console` (`CashTransactionPage.tsx` & `ApprovalQueuePage.tsx`) and write comprehensive AAA-standard unit tests.
