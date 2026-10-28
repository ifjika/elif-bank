-- ==============================================================================
-- Migration: 000001_init_bank_schema.up.sql (MySQL 8.0 & TiDB Compatible)
-- Dokumen Terkait: docs/TSD.md (Seksi 1) & docs/PRD.md
-- Engine: InnoDB / TiKV Distributed Storage
-- Karakter: utf8mb4 dengan presisi mikrodetik pada timestamp
-- ==============================================================================

-- 1. Tabel Kantor Cabang Bank
CREATE TABLE IF NOT EXISTS bank_branches (
    id VARCHAR(36) PRIMARY KEY,
    branch_code VARCHAR(10) NOT NULL UNIQUE,
    branch_name VARCHAR(100) NOT NULL,
    address TEXT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Tabel Pengguna Aplikasi & Hak Akses (RBAC)
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

-- 3. Tabel Rekening Nasabah & Akun Internal Buku Besar (Chart of Accounts)
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

-- 4. Tabel Header Transaksi Finansial
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    reference_id VARCHAR(64) NOT NULL UNIQUE, -- Kunci Idempotensi Klien
    transaction_type VARCHAR(32) NOT NULL,    -- TRANSFER, CASH_DEPOSIT, CASH_WITHDRAWAL, FEE, INTEREST
    amount BIGINT NOT NULL,
    description TEXT NOT NULL,
    maker_id VARCHAR(64) NOT NULL,
    checker_id VARCHAR(64) NULL,              -- Otorisasi Supervisor
    status ENUM('POSTED', 'PENDING_APPROVAL', 'REJECTED') NOT NULL DEFAULT 'POSTED',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_tx_created (created_at DESC),
    INDEX idx_tx_status (status),
    INDEX idx_tx_reference (reference_id),
    CONSTRAINT chk_tx_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. Tabel Jurnal Entri Buku Besar (Double-Entry Mutation Rows)
CREATE TABLE IF NOT EXISTS ledger_entries (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    transaction_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    entry_type ENUM('DR', 'CR') NOT NULL,     -- 'DR' (Debit) atau 'CR' (Kredit)
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,       -- SHA-256 chain dari baris akun sebelumnya
    current_hash VARCHAR(64) NOT NULL,        -- SHA-256(prev + account + amount + balanceAfter + timestamp)
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_entries_account_created (account_id, created_at DESC),
    INDEX idx_entries_tx (transaction_id),
    CONSTRAINT fk_entries_tx FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT,
    CONSTRAINT fk_entries_acc FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE RESTRICT,
    CONSTRAINT chk_entry_amount CHECK (amount > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. Tabel Transactional Outbox (Penerbitan Event Asinkron ke Kafka)
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
