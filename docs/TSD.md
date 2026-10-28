# Technical Specification Document (TSD): AethelCore Engine

This technical specification details the database schema, low-level algorithmic designs, monetary math rules, and internal Golang domain interface contracts.

The database is built on **MySQL 8.0 protocol compatibility** and fully optimized for **TiDB Cloud Serverless (PingCAP)** distributed HTAP clusters.

---

## 1. Relational Database Schema (MySQL 8.0 & TiDB)

All banking tables (Accounts, Transactions, Ledger Entries, Outbox Events, Users, and Branches) reside within a single unified relational schema.

```sql
-- 1. Bank Branches Table
CREATE TABLE IF NOT EXISTS bank_branches (
    id VARCHAR(36) PRIMARY KEY,
    branch_code VARCHAR(10) NOT NULL UNIQUE,
    branch_name VARCHAR(100) NOT NULL,
    address TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Application Users Table (RBAC)
CREATE TABLE IF NOT EXISTS app_users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role ENUM('CUSTOMER', 'TELLER', 'SUPERVISOR', 'ADMIN') NOT NULL,
    branch_id VARCHAR(36) NULL,
    status ENUM('ACTIVE', 'SUSPENDED') NOT NULL DEFAULT 'ACTIVE',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    INDEX idx_users_role (role),
    INDEX idx_users_branch (branch_id),
    CONSTRAINT fk_users_branch FOREIGN KEY (branch_id) REFERENCES bank_branches(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Customer Accounts & General Ledger Chart of Accounts (COA)
CREATE TABLE IF NOT EXISTS accounts (
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

-- 4. Financial Transactions Header Table
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    reference_id VARCHAR(64) NOT NULL UNIQUE, -- Client idempotency key
    transaction_type VARCHAR(32) NOT NULL,    -- TRANSFER, CASH_DEPOSIT, CASH_WITHDRAWAL, FEE, INTEREST
    amount BIGINT NOT NULL,
    description TEXT NOT NULL,
    maker_id VARCHAR(64) NOT NULL,
    checker_id VARCHAR(64) NULL,              -- Authorizing Supervisor
    status ENUM('POSTED', 'PENDING_APPROVAL', 'REJECTED') NOT NULL DEFAULT 'POSTED',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_tx_created (created_at DESC),
    INDEX idx_tx_status (status),
    INDEX idx_tx_reference (reference_id),
    CONSTRAINT chk_tx_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. General Ledger Journal Entries (Double-Entry Mutation Rows)
CREATE TABLE IF NOT EXISTS ledger_entries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    transaction_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    entry_type ENUM('DR', 'CR') NOT NULL,     -- 'DR' (Debit) or 'CR' (Credit)
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,       -- SHA-256 chain from prior account row
    current_hash VARCHAR(64) NOT NULL,        -- SHA-256(prev + account + amount + balanceAfter + timestamp)
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_entries_account_created (account_id, created_at DESC),
    INDEX idx_entries_tx (transaction_id),
    CONSTRAINT fk_entries_tx FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT,
    CONSTRAINT fk_entries_acc FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT,
    CONSTRAINT chk_entry_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. Transactional Outbox Table (Asynchronous Event Publishing to Kafka)
CREATE TABLE IF NOT EXISTS outbox_events (
    id VARCHAR(36) PRIMARY KEY,
    aggregate_type VARCHAR(32) NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSON NOT NULL,
    status ENUM('PENDING', 'PUBLISHED', 'FAILED') NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    published_at DATETIME(6) NULL,
    INDEX idx_outbox_pending (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## 2. Low-Level Mechanics & Algorithms

### 2.1 Deadlock Prevention Algorithm: Lexicographical Row-Locking

**The Concurrency Challenge**:
When Customer A transfers to Customer B concurrently with Customer B transferring to Customer A:
* Transaction 1 locks Account A, then attempts to lock Account B.
* Transaction 2 locks Account B, then attempts to lock Account A.
* Result: **Cyclic Deadlock** (database transaction abort/timeout).

**Algorithmic Solution**:
Prior to executing `SELECT ... FOR UPDATE`, the application sorts account UUIDs lexicographically (alphabetical string ordering):

```text
Input: AccountID_1, AccountID_2
Step 1: If AccountID_1 > AccountID_2, swap order.
Step 2: Lock first AccountID row in SQL.
Step 3: Lock second AccountID row in SQL.
Result: No two concurrent transactions contest lock acquisition order. Deadlocks = 0%.
```

Go Implementation Snippet:
```go
func SortAccountIDs(idA, idB string) (first, second string) {
    if idA < idB {
        return idA, idB
    }
    return idB, idA
}
```

---

### 2.2 Cryptographic Audit Trail Algorithm (SHA-256 Hash Chain)

Each new ledger entry is bound to the preceding mutation entry of the same account, forming an immutable linked list.

**Formula**:

$$\text{CurrentHash} = \text{SHA256}(\text{PreviousHash} + \text{AccountID} + \text{Amount} + \text{BalanceAfter} + \text{CreatedAt})$$

If an attacker modifies a historical balance row directly in the database, the hash of that row and all subsequent rows breaks, triggering instant detection during automated EOD audit batches.

---

### 2.3 Monetary Math Rules (Fixed-Point Math)

1. **Strict Prohibition of `float32` / `float64`**:
   IEEE 754 floating-point numbers cannot accurately represent binary fractional numbers (e.g., `0.1 + 0.2 = 0.30000000000000004`). In banking, this causes penny rounding discrepancies.
2. **Smallest Integer Denomination (`int64` / `BIGINT`)**:
   All Indonesian Rupiah balances are stored as whole integer numbers (`Rp 100,000` is stored as `100000`). Sub-unit currencies like USD or EUR store amounts in cents (`$10.50` is stored as `1050`).

---

## 3. Internal Domain Interface Contracts (Go Ports)

In `internal/domain/`, pure contracts are defined free of external third-party dependencies:

```go
package domain

import "context"

type AccountRepository interface {
    GetByID(ctx context.Context, id string) (*Account, error)
    GetByAccountNumber(ctx context.Context, accNum string) (*Account, error)
    UpdateBalanceWithLock(ctx context.Context, tx TransactionContext, acc *Account) error
}

type LedgerRepository interface {
    CreateTransaction(ctx context.Context, tx TransactionContext, t *Transaction) error
    AppendEntries(ctx context.Context, tx TransactionContext, entries []LedgerEntry) error
    GetLastEntryHash(ctx context.Context, accountID string) (string, error)
}

type TransactionManager interface {
    ExecuteInTransaction(ctx context.Context, fn func(txCtx TransactionContext) error) error
}
```
This document forms the foundation for Use Case and Repository implementations.
