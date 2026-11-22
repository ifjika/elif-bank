-- ==============================================================================
-- Migration: 000001_init_bank_schema.up.sql (MySQL 8.0 & TiDB Cloud Compatible)
-- Related Documents: docs/TSD.md (Section 1) & docs/PRD.md
-- Engine: InnoDB / TiKV Distributed Storage
-- Standard: Enterprise Core Banking (CIF, COA, Double-Entry, Maker-Checker, Hash Chain)
-- ==============================================================================

-- 1. Bank Branches Table (Kantor Cabang)
CREATE TABLE IF NOT EXISTS bank_branches (
    id CHAR(36) PRIMARY KEY,
    branch_code VARCHAR(10) NOT NULL UNIQUE,
    branch_name VARCHAR(100) NOT NULL,
    address TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Application Users Table (Internal Operators & Portal RBAC)
CREATE TABLE IF NOT EXISTS app_users (
    id CHAR(36) PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role ENUM('CUSTOMER', 'TELLER', 'SUPERVISOR', 'ADMIN') NOT NULL,
    branch_id CHAR(36) NULL,
    status ENUM('ACTIVE', 'SUSPENDED') NOT NULL DEFAULT 'ACTIVE',
    last_login_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    INDEX idx_users_role (role),
    INDEX idx_users_branch (branch_id),
    CONSTRAINT fk_users_branch FOREIGN KEY (branch_id) REFERENCES bank_branches(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. User Sessions Table (Stateful Refresh Token & Security Revocation Audit)
CREATE TABLE IF NOT EXISTS user_sessions (
    id CHAR(36) PRIMARY KEY,                   -- Matches JWT JTI claim
    user_id CHAR(36) NOT NULL,
    token_hash CHAR(64) NOT NULL UNIQUE,       -- SHA-256 hash of refresh token
    ip_address VARCHAR(45) NULL,
    user_agent VARCHAR(255) NULL,
    expires_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_sessions_user_exp (user_id, expires_at),
    CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES app_users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. Customers Table (CIF - Customer Information File & KYC)
CREATE TABLE IF NOT EXISTS customers (
    id CHAR(36) PRIMARY KEY,
    cif_number VARCHAR(20) NOT NULL UNIQUE,    -- Bank-wide unique Customer ID (e.g., CIF-000001)
    full_name VARCHAR(120) NOT NULL,
    identity_type ENUM('KTP', 'PASSPORT', 'NPWP', 'CORPORATE_NIB') NOT NULL,
    identity_number_hash CHAR(64) NOT NULL UNIQUE, -- SHA-256 of ID number (zero raw PII index exposure)
    kyc_status ENUM('PENDING', 'VERIFIED', 'REJECTED') NOT NULL DEFAULT 'PENDING',
    risk_rating ENUM('LOW', 'MEDIUM', 'HIGH') NOT NULL DEFAULT 'LOW',
    home_branch_id CHAR(36) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    INDEX idx_customers_kyc (kyc_status),
    CONSTRAINT fk_customers_branch FOREIGN KEY (home_branch_id) REFERENCES bank_branches(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. Customer Accounts & General Ledger Chart of Accounts (COA)
CREATE TABLE IF NOT EXISTS accounts (
    id CHAR(36) PRIMARY KEY,
    account_number VARCHAR(32) NOT NULL UNIQUE,
    customer_id CHAR(36) NULL,                 -- NULL for internal bank GL accounts
    branch_id CHAR(36) NULL,
    account_type ENUM('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE') NOT NULL,
    product_code VARCHAR(32) NOT NULL DEFAULT 'SAVINGS', -- SAVINGS, CHECKING, TIME_DEPOSIT, GL_VAULT_CASH, GL_TAX_PAYABLE
    is_internal_gl BOOLEAN NOT NULL DEFAULT FALSE,
    currency CHAR(3) NOT NULL DEFAULT 'IDR',   -- ISO 4217
    balance BIGINT NOT NULL DEFAULT 0,         -- Whole smallest unit (IDR)
    hold_balance BIGINT NOT NULL DEFAULT 0,    -- Escrowed / pending-approval funds
    status ENUM('ACTIVE', 'FROZEN', 'DORMANT', 'CLOSED') NOT NULL DEFAULT 'ACTIVE',
    version BIGINT NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    INDEX idx_accounts_status (status),
    INDEX idx_accounts_customer (customer_id),
    INDEX idx_accounts_product (product_code),
    CONSTRAINT fk_accounts_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE RESTRICT,
    CONSTRAINT fk_accounts_branch FOREIGN KEY (branch_id) REFERENCES bank_branches(id) ON DELETE SET NULL,
    CONSTRAINT chk_positive_hold CHECK (hold_balance >= 0),
    CONSTRAINT chk_balance_cover_hold CHECK (balance >= hold_balance)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. Financial Transactions Header Table (Maker-Checker & Business Value Date)
CREATE TABLE IF NOT EXISTS transactions (
    id CHAR(36) PRIMARY KEY,
    reference_id VARCHAR(64) NOT NULL UNIQUE, -- Client idempotency key
    transaction_type ENUM('TRANSFER', 'CASH_DEPOSIT', 'CASH_WITHDRAWAL', 'FEE', 'INTEREST', 'TAX') NOT NULL,
    amount BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    business_date DATE NOT NULL,               -- Banking ledger/value date
    description TEXT NOT NULL,
    maker_id CHAR(36) NOT NULL,
    checker_id CHAR(36) NULL,                  -- Authorizing Supervisor (must differ from maker_id)
    checker_notes TEXT NULL,
    status ENUM('POSTED', 'PENDING_APPROVAL', 'REJECTED') NOT NULL DEFAULT 'POSTED',
    approved_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_tx_created (created_at DESC),
    INDEX idx_tx_business_date (business_date),
    INDEX idx_tx_status (status),
    INDEX idx_tx_reference (reference_id),
    CONSTRAINT chk_tx_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. General Ledger Journal Entries (Double-Entry Mutation Rows + SHA-256 Chain)
CREATE TABLE IF NOT EXISTS ledger_entries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    transaction_id CHAR(36) NOT NULL,
    account_id CHAR(36) NOT NULL,
    entry_type ENUM('DR', 'CR') NOT NULL,     -- 'DR' (Debit) or 'CR' (Credit)
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    previous_hash CHAR(64) NOT NULL,           -- SHA-256 chain from prior account row
    current_hash CHAR(64) NOT NULL,            -- SHA-256(prev + account + amount + balanceAfter + timestamp)
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_entries_account_created (account_id, created_at DESC),
    INDEX idx_entries_tx (transaction_id),
    CONSTRAINT fk_entries_tx FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT,
    CONSTRAINT fk_entries_acc FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT,
    CONSTRAINT chk_entry_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 8. Transactional Outbox Table (Asynchronous Event Publishing to Kafka)
CREATE TABLE IF NOT EXISTS outbox_events (
    id CHAR(36) PRIMARY KEY,
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
