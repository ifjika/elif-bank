# Implementation Plan & Roadmap: Elif Core Bank Engine (Elif Bank)

This document defines the roadmap, phased feature implementation matrix, and the active task checklist for the feature currently being implemented.

---

## 1. Feature Implementation Matrix

|  Phase  | Module & Feature                                      | Associated Tech Stack                                        |        Status        | Definition of Done                                                                                                                     |
| :-----: | :---------------------------------------------------- | :----------------------------------------------------------- | :------------------: | :------------------------------------------------------------------------------------------------------------------------------------- |
|  **1**  | **Environment Setup & Multi-Module Workspace**        | Go 1.23+, `go.work`, Docker Compose, MySQL 8.0 / TiDB, Redis |   🟢 **Completed**   | All database and cache configurations ready; Go monorepo & frontend directory structures cleanly initialized.                          |
|  **2**  | **Database Schema Design & SQL Migrations**           | MySQL 8.0, TiDB Cloud Serverless, SQL DDL                    |   🟢 **Completed**   | Unified DDL migration schema (`000001_init_bank_schema.up.sql`) created with constraints and accounts/users tables.                    |
|  **3**  | **Core Ledger Engine: Double-Entry & Row Locking**    | Go, Lexicographical Locking, SHA-256 Hashchain, Money Math   |   🟢 **Completed**   | Atomic Double-Entry logic, lexicographical sorting, SHA-256 chain, and AAA unit tests passing 100%.                                    |
|  **4**  | **API Gateway: Gin REST, JWT, & Idempotency**         | Go, Gin Framework, JWT, Redis Token Bucket, RESTful API      |   🟢 **Completed**   | Gateway REST API, JWT token manager, Redis idempotency & rate limiting passing 100% unit tests.                                        |
| **4.1** | **End-to-End JWT Authentication & RBAC Login System** | Go, `bcrypt`, MySQL/TiDB, React, Axios                       | 🟡 **Active / Next** | Full authentication flow: database user lookup, password hash verify, JWT issue, Gin handler, and React Web Console login integration. |
|  **5**  | **Branch Operations: Maker-Checker System**           | Go, Gin, RBAC, Dual JWT Claims, React UI                     |      ⚪ Planned      | Cash transactions > 50M IDR enter `PENDING_APPROVAL` status and can only be approved/rejected by a Checker.                            |
|  **6**  | **GraphQL Endpoint for Backoffice / Audit**           | Go, `gqlgen`, GraphQL                                        |      ⚪ Planned      | GraphQL query can display account relations and validate transaction data integrity hash chains.                                       |
|  **7**  | **Batch Engine: End-of-Day (EOD) Worker**             | Go, Cron/Tickers, Transactional Outbox, Kafka / Redpanda     |      ⚪ Planned      | Worker computes daily interest accrual, 20% withholding tax (PPh), and monthly admin fees via automated batch.                         |
|  **8**  | **Autonomous Traffic Simulator**                      | Go Concurrency (Goroutines, Channels), Stress Testing        |      ⚪ Planned      | Generator produces 50–200 TPS continuous synthetic transactions to benchmark latency and stability.                                    |
|  **9**  | **CI/CD Pipeline & Test Automation**                  | GitHub Actions, `go test -race`, `golangci-lint`             |      ⚪ Planned      | Automated CI pipeline executes linting, race-detector, and build tests on every git commit.                                            |

---

## 2. Active Task Checklist: Phase 4.1 - End-to-End JWT Authentication & Login System

> [!IMPORTANT]
> **Checklist Usage Rules**:
> This section **ONLY** contains the micro-steps for the feature **CURRENTLY** in progress. Once all steps below are confirmed and verified, this checklist is **cleared**, and replaced with the micro-steps of the next feature according to the implementation table above.

### Active Phase: **Phase 4.1 - End-to-End JWT Authentication & RBAC Login System**

- [ ] **4.1.1 Password Hashing & Verification Engine**
  - Implement secure password hashing and constant-time hash comparison using `golang.org/x/crypto/bcrypt` (Cost $\ge 12$).
  - Provide unit tests verifying hash uniqueness and correct password verification.

- [ ] **4.1.2 Database User Repository (MySQL / TiDB)**
  - Define `UserRepository` port in `apps/api-gateway/internal/domain/user.go` or shared package:
    - `GetByUsername(ctx context.Context, username string) (*User, error)`
  - Implement repository queries matching `app_users` table in `deploy/migrations/mysql_tidb/000001_init_bank_schema.up.sql`.

- [ ] **4.1.3 Authentication Usecase & Business Logic**
  - Implement `AuthUsecase.Login(ctx context.Context, username, password string) (*LoginResult, error)`:
    - Input validation: reject empty credentials.
    - Query user by username. Return `ErrInvalidCredentials` if not found (prevent account enumeration).
    - Status check: reject if user is `SUSPENDED` (return `ErrUserSuspended`).
    - Verify password hash via bcrypt.
    - Generate Access Token (15-min TTL) and Refresh Token (7-day TTL) with role claims (`CUSTOMER`, `TELLER`, `SUPERVISOR`, `ADMIN`).

- [ ] **4.1.4 Gin REST Delivery Handler & Routing**
  - Implement `HandleLogin(c *gin.Context)` under `POST /api/v1/auth/login`.
  - Enforce payload binding (`username`, `password`).
  - Map errors to RFC 7807 responses:
    - 400 Bad Request: Malformed JSON or missing fields.
    - 401 Unauthorized: Invalid username or wrong password.
    - 403 Forbidden: Account suspended.
    - 200 OK: Return JWT access token, refresh token, and user profile.

- [ ] **4.1.5 Frontend Web Console Integration (`apps/web-console`)**
  - Replace mock login in `apps/web-console/src/features/auth/pages/LoginPage.tsx` with live API call to `/api/v1/auth/login`.
  - Store JWT access token & user profile in `localStorage`.
  - Validate Axios interceptors in `src/lib/apiClient.ts` to attach `Authorization: Bearer <token>` on all outbound requests.
  - Implement automatic role-based redirect upon login:
    - `TELLER` $\to$ `/branch/teller`
    - `SUPERVISOR` $\to$ `/branch/supervisor`
    - `ADMIN` $\to$ `/branch/audit`

- [ ] **4.1.6 Comprehensive AAA Unit & Integration Tests**
  - Write unit tests in `apps/api-gateway/internal/auth/login_test.go` covering:
    - Happy path: Valid credentials return valid JWT.
    - Negative: Wrong password returns 401.
    - Negative: Unknown username returns 401.
    - Negative: Suspended user returns 403.
    - Negative: Empty fields return 400.
