-- ==============================================================================
-- Migration: 000001_init_ledger_schema.up.sql
-- Dokumen Terkait: docs/TSD.md (Seksi 1.1) & docs/PRD.md (Seksi 3.1 - 3.3)
-- Fungsi: Inisialisasi skema inti Double-Entry Ledger, Transaksi, dan Outbox
-- ==============================================================================

-- 1. Pastikan ekstensi UUID aktif di PostgreSQL
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 2. Tabel Rekening Nasabah & Chart of Accounts (COA)
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_number VARCHAR(32) UNIQUE NOT NULL,
    customer_id UUID NULL,
    account_type VARCHAR(16) NOT NULL, -- ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    balance BIGINT NOT NULL DEFAULT 0,
    hold_balance BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, FROZEN, DORMANT, CLOSED
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_positive_hold CHECK (hold_balance >= 0)
);

-- Indeks B-Tree untuk pencarian cepat nomor rekening dan status
CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_accounts_customer_id ON accounts(customer_id);

-- 3. Tabel Header Transaksi Finansial
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id VARCHAR(64) UNIQUE NOT NULL, -- Kunci Idempotensi dari Klien
    transaction_type VARCHAR(32) NOT NULL,    -- TRANSFER, CASH_DEPOSIT, CASH_WITHDRAWAL, FEE, INTEREST
    amount BIGINT NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL,
    maker_id VARCHAR(64) NOT NULL,
    checker_id VARCHAR(64) NULL,              -- Terisi saat Supervisor meng-approve
    status VARCHAR(16) NOT NULL DEFAULT 'POSTED', -- POSTED, PENDING_APPROVAL, REJECTED
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status);

-- 4. Tabel Jurnal Entri Buku Besar (Double-Entry Mutation Rows)
CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    entry_type VARCHAR(2) NOT NULL,           -- 'DR' (Debit) atau 'CR' (Kredit)
    amount BIGINT NOT NULL CHECK (amount > 0),
    balance_after BIGINT NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,       -- SHA-256 chain dari baris mutasi akun sebelumnya
    current_hash VARCHAR(64) NOT NULL,        -- SHA-256(prev_hash + account_id + amount + balance_after)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indeks komposit B-Tree untuk query rekening koran berurutan
CREATE INDEX idx_entries_account_created ON ledger_entries(account_id, created_at DESC);
CREATE INDEX idx_entries_tx_id ON ledger_entries(transaction_id);

-- 5. Tabel Transactional Outbox (Penerbitan Event Asinkron ke Kafka)
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(32) NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING, PUBLISHED, FAILED
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_outbox_pending ON outbox_events(status, created_at) WHERE status = 'PENDING';
