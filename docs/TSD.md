# Technical Specification Document (TSD): AethelCore Engine

Dokumen spesifikasi teknis ini menguraikan skema database mendalam, desain algoritma tingkat rendah (*low-level algorithms*), aturan penanganan matematika finansial, dan kontrak *interface* internal Golang.

---

## 1. Skema Database Relasional Mendalam

### 1.1 Database Primer: PostgreSQL (Core Ledger)

Data finansial sensitif disimpan pada PostgreSQL dengan tipe data presisi dan index optimal.

```sql
-- Ekstensi UUID native
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Rekening Nasabah & Akun Internal Buku Besar (COA)
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_number VARCHAR(32) UNIQUE NOT NULL,
    customer_id UUID NULL,
    account_type VARCHAR(16) NOT NULL, -- ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    balance BIGINT NOT NULL DEFAULT 0,
    hold_balance BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_positive_hold CHECK (hold_balance >= 0)
);

CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_accounts_customer_id ON accounts(customer_id);

-- 2. Transaksi Finansial (Tabel Header Transaksi)
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id VARCHAR(64) UNIQUE NOT NULL, -- Kunci idempotensi
    transaction_type VARCHAR(32) NOT NULL,    -- TRANSFER, CASH_DEPOSIT, CASH_WITHDRAWAL, FEE, INTEREST
    amount BIGINT NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL,
    maker_id VARCHAR(64) NOT NULL,
    checker_id VARCHAR(64) NULL,              -- Otorisasi Supervisor jika required
    status VARCHAR(16) NOT NULL DEFAULT 'POSTED', -- POSTED, PENDING_APPROVAL, REJECTED
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX idx_transactions_status ON transactions(status);

-- 3. Jurnal Buku Besar (Double-Entry Mutation Rows)
CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    account_id UUID NOT NULL REFERENCES accounts(id),
    entry_type VARCHAR(2) NOT NULL,           -- 'DR' (Debit) atau 'CR' (Kredit)
    amount BIGINT NOT NULL CHECK (amount > 0),
    balance_after BIGINT NOT NULL,
    previous_hash VARCHAR(64) NOT NULL,       -- SHA-256 chain dari baris akun sebelumnya
    current_hash VARCHAR(64) NOT NULL,        -- SHA-256(prev_hash + account_id + amount + balance_after)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entries_account_created ON ledger_entries(account_id, created_at DESC);
CREATE INDEX idx_entries_tx_id ON ledger_entries(transaction_id);

-- 4. Transactional Outbox (Penerbitan Event Asinkron)
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
```

### 1.2 Database Pendukung: MariaDB / MySQL (KYC & Auth Store)

Untuk memisahkan data otentikasi identitas dan profil nasabah dari database mutasi buku besar:

```sql
-- Tabel Profil Pengguna & Cabang Bank di MariaDB
CREATE TABLE bank_branches (
    id VARCHAR(36) PRIMARY KEY,
    branch_code VARCHAR(10) UNIQUE NOT NULL,
    branch_name VARCHAR(100) NOT NULL,
    address TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE app_users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role ENUM('CUSTOMER', 'TELLER', 'SUPERVISOR', 'ADMIN') NOT NULL,
    branch_id VARCHAR(36) NULL,
    status ENUM('ACTIVE', 'SUSPENDED') DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (branch_id) REFERENCES bank_branches(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 2. Desain Algoritma Tingkat Rendah (Low-Level Mechanics)

### 2.1 Algoritma Pencegahan Deadlock: Lexicographical Row-Locking

**Masalah Klasik Konkurensi**:
Bila nasabah A mentransfer ke nasabah B secara bersamaan dengan nasabah B mentransfer ke nasabah A:
* Transaksi 1 mengunci akun A, lalu mencoba mengunci akun B.
* Transaksi 2 mengunci akun B, lalu mencoba mengunci akun A.
* Hasil: **Cyclic Deadlock** (database crash/timeout).

**Solusi Algoritmik**:
Sebelum query `SELECT ... FOR UPDATE` dijalankan, kode aplikasi wajib mengurutkan UUID akun secara leksikografis (berdasarkan urutan abjad string):

```text
Input: AccountID_1, AccountID_2
Langkah 1: Jika AccountID_1 > AccountID_2, tukar posisi.
Langkah 2: Lock baris AccountID_pertama terlebih dahulu di SQL.
Langkah 3: Lock baris AccountID_kedua berikutnya di SQL.
Hasil: Tidak ada dua transaksi yang pernah berebut urutan kunci. Deadlock = 0%.
```

Cuplikan Logika Go:
```go
func SortAccountIDs(idA, idB string) (first, second string) {
    if idA < idB {
        return idA, idB
    }
    return idB, idA
}
```

---

### 2.2 Algoritma Audit Trail Kriptografis (SHA-256 Hash Chain)

Setiap entri buku besar (*ledger entry*) baru diikatkan dengan entri mutasi sebelumnya pada akun yang sama, membentuk rantai blok (*blockchain-like linked list*).

**Formula**:
$$\text{CurrentHash} = \text{SHA256}(\text{PreviousHash} + \text{AccountID} + \text{Amount} + \text{BalanceAfter} + \text{CreatedAt})$$

Jika seseorang mencoba memodifikasi nominal saldo pada baris historis di database, nilai hash baris tersebut dan seluruh baris berikutnya akan rusak, memicu alarm anomali pada audit batch harian.

---

### 2.3 Aturan Matematika Moneter (Fixed-Point Math)

1. **Larangan Penggunaan `float32` / `float64`**:
   Tipe data float IEEE 754 memiliki representasi biner pecahan yang tidak presisi (misal: `0.1 + 0.2 = 0.30000000000000004`). Dalam perbankan, ini memicu kebocoran sen (*penny rounding fraud*).
2. **Penggunaan Satuan Integer Terkecil (`int64` / `BIGINT`)**:
   Seluruh nilai Rupiah disimpan dalam angka bulat penuh. Nilai $100.000$ disimpan sebagai angka `100000`. Jika menangani mata uang dengan subunit sen (seperti USD/EUR), $10.50$ disimpan sebagai $1050$ sen.

---

## 3. Definisi Interface Domain Internal (Go Ports)

Di lapisan `internal/domain/`, kita mendefinisikan kontrak interface yang murni tanpa ketergantungan library eksternal:

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
Dokumen ini menjadi fondasi implementasi kode di lapisan Use Case dan Repository.
