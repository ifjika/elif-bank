# Architectural Overview & System Specification: AethelCore Engine

AethelCore is a distributed miniature Core Banking System (CBS) ecosystem built in Golang. It implements double-entry general ledger accounting standards, ACID transactions with concurrency determinism guarantees, branch operational governance (*Four-Eyes Principle / Maker-Checker*), tamper-evident cryptographic audit trails, and an autonomous cron-driven transactional workload generator.

The entire architecture is organized as a Polyglot Monorepo utilizing native Go Workspaces (`go.work`) to coordinate high-performance backend transaction processing, an API Gateway (*Backend-for-Frontend*), branch operational consoles, and telemetry systems.

The database layer is unified on **MySQL 8.0 protocol compatibility** and native support for **TiDB Cloud Serverless (PingCAP)**.

---

## 1. Domain & Banking Business Flows

The system models the fundamental pillars of modern commercial banking with strict separation of concerns:

```text
[ Customer / Mobile ]       [ Teller / Branch ]
        │                            │
        ▼                            ▼
┌────────────────────────────────────────────────────────┐
│             API Gateway & Ingress Layer                │
│  - JWT Verification & Role-Based Access Control        │
│  - Token Bucket Rate Limiting (Redis)                  │
│  - Idempotency-Key Deduplication Engine                │
└──────────────────────────┬─────────────────────────────┘
                           │ gRPC / Protobuf
                           ▼
┌────────────────────────────────────────────────────────┐
│           AethelCore Transaction Engine                │
│  - Deterministic Lexicographical Row-Locking           │
│  - Double-Entry Balance Mutation                       │
│  - Cryptographic SHA-256 Hash-Chained Audit Trail      │
└──────────────┬───────────────────────────┬─────────────┘
               │                           │
    Transactional Outbox (Kafka)           │ Database Write (MySQL 8.0 / TiDB)
               ▼                           ▼
┌──────────────────────────────┐ ┌───────────────────────┐
│     EOD & Batch Worker       │ │   General Ledger &    │
│ - Daily Interest Accrual     │ │   Chart of Accounts   │
│ - Loan Amortization Schedule │ │ (Assets = Liab + Eq)  │
│ - Tax Calculation (PPh 20%)  │ └───────────────────────┘
└──────────────────────────────┘
```

### 1.1 Funding & Account Subsystem

- Manages customer deposit accounts: Individual Savings, Commercial Checking, and Time Deposits.
- **Account State Machine**: `ACTIVE`, `FROZEN`, `DORMANT`, and `CLOSED`. Inactive accounts deterministically reject all new debit or credit mutations.
- Strict separation between available balance (*available balance*) and reserved balance (*hold balance / escrow*) for loan collateral or authorization queues.

### 1.2 Double-Entry Transaction Engine

- Absolute enforcement of core accounting principles: single-sided balance updates (`UPDATE balance = balance - x`) are prohibited.
- Every fund movement produces at least one pair of balanced journal entries: one account debited and another credited with the exact same amount.
- Accounting invariant enforcement:

$$\sum \text{Debit} - \sum \text{Credit} = 0$$

### 1.3 Branch Operations & Governance (Maker-Checker)

- Over-the-counter physical cash transactions $\le \text{IDR 50,000,000}$ are processed straight-through.
- Cash transactions $> \text{IDR 50,000,000}$ automatically transition to `PENDING_APPROVAL`, requiring digital authorization from a branch Supervisor (*Checker*) before ledger mutation occurs (*Four-Eyes Principle*).

### 1.4 Lending & Amortization Subsystem

Simulates loan disbursement, annuity repayment schedule calculation, and automated periodic installment debits.

Automatic credit collectibility classification based on Days Past Due (DPD):
- **KOL 1 (Current / Performing)**: 0 days overdue.
- **KOL 2 (Special Mention)**: 1–90 days overdue.
- **KOL 3 (Substandard)**: 91–120 days overdue.
- **KOL 4 (Doubtful)**: 121–180 days overdue.
- **KOL 5 (Loss / Default)**: $> 180$ days overdue.

### 1.5 End-of-Day (EOD) Batch Engine

Autonomous module executing bank day-end rollover procedures:
- Daily savings interest accrual:

$$\text{Daily Interest} = \frac{\text{Closing Balance} \times \text{Interest Rate}}{365}$$

- Automated withholding tax (20% PPh Final) on interest income.
- Deduction of monthly maintenance fees and late payment penalties.
- Trial balance consolidation to ensure zero imbalance across all *Chart of Accounts* (COA).

### 1.6 Autonomous Traffic Generator

Background worker driving dynamic request loops and periodic schedules to generate thousands of synthetic transactions (transfers, deposits, withdrawals, merchant settlements) to benchmark extreme concurrency, race conditions, and database p99 latencies.

---

## 2. Monorepo Directory Structure (go.work)

```text
elif-bank/
├── .github/
│   └── workflows/
│       ├── test-core.yml        # CI linting & race-detector
│       └── build-images.yml     # Multi-stage Docker build
├── proto/                       # Single source of truth for interface contracts
│   ├── account/v1/account.proto
│   ├── ledger/v1/ledger.proto
│   ├── branch/v1/branch.proto
│   └── eod/v1/eod.proto
├── apps/
│   ├── api-gateway/             # Go: REST/JSON to gRPC translation, RBAC, Rate-limit
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   └── go.mod
│   ├── core-ledger/             # Go: Core double-entry transaction engine
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/          # Entities & repository interfaces
│   │   │   ├── usecase/         # Transfer logic, locking, audit chain
│   │   │   └── repository/      # MySQL 8.0 / TiDB implementation
│   │   └── go.mod
│   ├── worker-eod/              # Go: Day-end rollover batch engine (EOD)
│   │   ├── cmd/worker/main.go
│   │   └── go.mod
│   ├── simulator-cron/          # Go: Autonomous synthetic traffic generator
│   │   ├── cmd/simulator/main.go
│   │   └── go.mod
│   └── web-console/             # Next.js 15: Teller, Supervisor & Customer portal
│       ├── src/
│       │   ├── app/
│       │   │   ├── branch/      # Branch portal (Teller & Supervisor screens)
│       │   │   └── customer/    # Mock Mobile Banking Viewport
│       │   └── components/
│       └── package.json
├── pkg/                         # Shared Go Packages
│   ├── hashchain/               # SHA-256 linked-list engine for audit records
│   ├── idempotency/             # Hash-based request payload deduplication
│   ├── money/                   # Fixed-point precision integer math (no floats)
│   └── go.mod
├── deploy/                      # Local orchestration environment
│   ├── docker-compose.yml       # MySQL 8.0 / TiDB-compatible, Redis 7
│   ├── migrations/mysql_tidb/   # Unified schema migrations
│   ├── prometheus.yml
│   └── grafana/
│       └── dashboards/          # Visualization for TPS, p99 latency, ledger invariants
├── docs/                        # Architecture documentation & system guides
├── go.work                      # Multi-module Go workspace orchestration
└── README.md
```

---

## 3. Data Schema & Database Design (MySQL 8.0 / TiDB)

All monetary amounts are stored in the smallest integer denomination (cents or whole rupiah) using `BIGINT` to prevent IEEE 754 floating-point rounding errors.

```sql
-- 1. Customer Accounts & General Ledger Internal Accounts
CREATE TABLE accounts (
    id VARCHAR(36) PRIMARY KEY,
    account_number VARCHAR(32) NOT NULL UNIQUE,
    customer_id VARCHAR(36) NULL,
    account_type ENUM('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE') NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    balance BIGINT NOT NULL DEFAULT 0,
    hold_balance BIGINT NOT NULL DEFAULT 0,
    status ENUM('ACTIVE', 'FROZEN', 'DORMANT', 'CLOSED') NOT NULL DEFAULT 'ACTIVE',
    version BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    INDEX idx_accounts_status (status),
    INDEX idx_accounts_customer (customer_id),
    CONSTRAINT chk_positive_hold CHECK (hold_balance >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Financial Transactions (Header)
CREATE TABLE transactions (
    id VARCHAR(36) PRIMARY KEY,
    reference_id VARCHAR(64) NOT NULL UNIQUE, -- Idempotency key
    transaction_type VARCHAR(32) NOT NULL,    -- TRANSFER, CASH_DEPOSIT, CASH_WITHDRAWAL, FEE, INTEREST
    amount BIGINT NOT NULL,
    description TEXT NOT NULL,
    maker_id VARCHAR(64) NOT NULL,
    checker_id VARCHAR(64) NULL,              -- Supervisor authorization if required
    status ENUM('POSTED', 'PENDING_APPROVAL', 'REJECTED') NOT NULL DEFAULT 'POSTED',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_tx_created (created_at DESC),
    INDEX idx_tx_status (status),
    INDEX idx_tx_reference (reference_id),
    CONSTRAINT chk_tx_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. General Ledger Journal Entries (Double-Entry Mutation)
CREATE TABLE ledger_entries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    transaction_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    entry_type ENUM('DR', 'CR') NOT NULL,     -- 'DR' (Debit) or 'CR' (Credit)
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,       -- SHA-256 chain from previous account entry
    current_hash VARCHAR(64) NOT NULL,        -- SHA-256(prev_hash + account_id + amount + balance_after + timestamp)
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_entries_account_created (account_id, created_at DESC),
    INDEX idx_entries_tx (transaction_id),
    CONSTRAINT fk_entries_tx FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT,
    CONSTRAINT fk_entries_acc FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT,
    CONSTRAINT chk_entry_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## 4. Vulnerability Analysis & Cyber Security Mitigations

### 4.1 Race Conditions & Concurrency Deadlocks

- **Attack Vector**: Two cross-transfer transactions between Account A and Account B executed in parallel. Locking rows without deterministic order triggers cyclic deadlocks.
- **Mitigation**: Sort account identifiers lexicographically prior to executing `SELECT ... FOR UPDATE`:

```go
firstID, secondID := fromID, toID
if firstID > secondID {
    firstID, secondID = toID, fromID
}
tx.Exec(ctx, "SELECT id, balance FROM accounts WHERE id IN (?, ?) ORDER BY id FOR UPDATE", firstID, secondID)
```

### 4.2 Network Partitions & Replay Attacks

- **Attack Vector**: Network failure midway through request handling causes the client to retry transfer requests, risking double-debiting balances.
- **Mitigation**: Mandatory idempotency keys (`X-Idempotency-Key`). Gateway performs atomic key acquisition in Redis with `ACQUIRING` status before opening a database transaction. Duplicate requests receive cached responses without re-execution.

### 4.3 Maker-Checker Authorization Bypass

- **Attack Vector**: Manipulating client-side parameters to fake approval status on cash transactions exceeding teller limits.
- **Mitigation**: Dual JWT claims cryptographic verification. Maker and Checker tokens are validated against separate public keys or roles. The backend explicitly rejects mutations if `maker_id` is identical to `checker_id`.

### 4.4 Silent Ledger Tampering (Insider Threat)

- **Attack Vector**: Direct row alteration in the database by internal privileged administrative accounts.
- **Mitigation**: Every ledger entry is linked via a SHA-256 hash chain:

$$\text{Hash}_n = \text{SHA256}(\text{Hash}_{n-1} \parallel \text{AccountID} \parallel \text{Amount} \parallel \text{BalanceAfter})$$

EOD audit batches recursively verify chain continuity; illicit row modification breaks the chain and triggers immediate security alerts.

---

## 5. Automated Data Generator Specifications (Cron Simulator)

The `apps/simulator-cron` module acts as an autonomous transactional traffic generator:

### 5.1 Cold-Start Provisioning

- Provisions 5 fictitious bank branch entities.
- Provisions 10 Teller accounts and 5 Supervisor accounts with isolated credentials.
- Provisions 1,000 customer accounts populated with realistic initial balances.

### 5.2 High-Frequency Transaction Loop

- **Execution Interval**: $10\text{--}50\text{ ms}$ (Target: 50–200 sustained TPS).
- **Workload Composition**:
  - **70%**: Peer-to-peer customer fund transfers.
  - **15%**: Third-party merchant / bill payments.
  - **10%**: Branch teller cash deposits and withdrawals.
  - **5%**: Controlled failure transactions (insufficient funds, frozen accounts) to test database rollback consistency.

### 5.3 Scheduled EOD Engine Loop

- **Execution Frequency**: Runs every 5-minute interval (emulating a 1-day banking calendar rollover cycle).
- **Batch Tasks**: Runs savings interest accruals, periodic interest capitalizations, and loan delinquency assessments in batch.

---

## 6. Performance Metrics & Quantitative Target Standards

| Evaluation Metric | Production Target Standard | Validation Method |
| :--- | :--- | :--- |
| **ACID Mutation Throughput** | $\ge 2,500 \text{ TPS}$ | gRPC load testing using `ghz` |
| **Transaction Latency (p99)** | $\le 20 \text{ ms}$ | Measured via Prometheus metric panels |
| **Accounting Invariant Integrity** | $0 \text{ Anomalies}$ | Absolute $\sum \text{Debit} - \sum \text{Credit} = 0$ |
| **Concurrency Resilience** | $0.00\% \text{ Race Conditions}$ | 500 parallel goroutines debiting 1 account |
| **Backend Memory Footprint** | $\le 40 \text{ MB per pod}$ | Heap allocation profiling via `pprof` |
