# Architectural Overview & System Specification: AethelCore Engine

AethelCore adalah ekosistem miniatur Core Banking System (CBS) terdistribusi berbasis Golang yang mengimplementasikan standar akuntansi buku besar berpasangan (*double-entry ledger*), transaksi atomik berstandar ACID dengan jaminan determinisme konkurensi, kontrol operasional cabang (*Four-Eyes Principle / Maker-Checker*), jejak audit kriptografis *tamper-evident*, serta generator data transaksional otonom berbasis *cron-job*.

Seluruh arsitektur dikelola dalam satu Polyglot Monorepo menggunakan fitur native Go Workspaces (`go.work`) untuk mengintegrasikan mesin pemrosesan backend performa tinggi, gerbang API (*Backend-for-Frontend*), konsol operasional cabang, dan telemetri sistem.

---

## 1. Domain & Alur Bisnis Perbankan

Sistem ini memodelkan pilar fundamental perbankan komersial modern dengan isolasi tanggung jawab yang ketat:

```text
[ Nasabah / Mobile ]        [ Teller / Cabang ]
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
    Transactional Outbox (Kafka)           │ Database Write (PostgreSQL)
               ▼                           ▼
┌──────────────────────────────┐ ┌───────────────────────┐
│     EOD & Batch Worker       │ │   General Ledger &    │
│ - Daily Interest Accrual     │ │   Chart of Accounts   │
│ - Loan Amortization Schedule │ │ (Assets = Liab + Eq)  │
│ - Tax Calculation (PPh 20%)  │ └───────────────────────┘
└──────────────────────────────┘
```

### 1.1 Funding & Account Subsystem

- Mendukung pembukaan dan manajemen rekening simpanan nasabah: Tabungan Individu, Giro Komersial, dan Deposito Berjangka.
- **State Machine Rekening**: `ACTIVE`, `FROZEN`, `DORMANT`, dan `CLOSED`. Rekening berstatus non-aktif secara deterministik menolak mutasi debit maupun kredit baru.
- Pemisahan mutlak antara saldo efektif (*available balance*) dan saldo tertahan (*hold balance / escrow*) untuk memfasilitasi transaksi pre-otorisasi atau jaminan pinjaman.

### 1.2 Double-Entry Transaction Engine

- Prinsip fundamental akuntansi diterapkan secara absolut: tidak ada mutasi saldo tunggal (`UPDATE balance = balance - x`).
- Setiap pemindahan dana wajib menghasilkan minimal sepasang entri jurnal (*balanced entries*): satu akun didebit dan satu akun lainnya dikredit dengan nilai nominal yang persis sama.
- Penegakan konsistensi invarian akuntansi:

$$\sum \text{Debit} - \sum \text{Kredit} = 0$$

### 1.3 Branch Operations & Governance (Maker-Checker)

- Transaksi kas fisik teller cabang dengan nilai $\le \text{Rp50.000.000}$ diproses seketika (*straight-through processing*).
- Transaksi kas bernilai $> \text{Rp50.000.000}$ secara otomatis dialihkan ke status `PENDING_APPROVAL`, menuntut otorisasi digital terpisah dari Supervisor Cabang (*Checker*) sebelum mutasi buku besar dieksekusi (*Four-Eyes Principle*).

### 1.4 Lending & Amortization Subsystem

Simulasi pencairan pinjaman (*disbursement*), penyusunan jadwal pembayaran berbasis formula bunga anuitas, dan mekanisme penagihan cicilan berkala (*auto-debit*).

Pengelompokan status kolektibilitas kredit otomatis berdasarkan hari keterlambatan (*Days Past Due* / DPD):
- **KOL 1 (Lancar)**: Tunggakan 0 hari.
- **KOL 2 (Dalam Perhatian Khusus)**: Tunggakan 1–90 hari.
- **KOL 3 (Kurang Lancar)**: Tunggakan 91–120 hari.
- **KOL 4 (Diragukan)**: Tunggakan 121–180 hari.
- **KOL 5 (Macet)**: Tunggakan $> 180$ hari.

### 1.5 End-of-Day (EOD) Batch Engine

Modul otonom yang mengeksekusi penutupan buku harian perbankan (*day-end rollover*):
- Akrual bunga simpanan harian:

$$\text{Bunga Harian} = \frac{\text{Saldo Akhir} \times \text{Suku Bunga}}{365}$$

- Pemotongan otomatis Pajak Penghasilan (PPh Final 20%) atas pendapatan bunga.
- Pemotongan biaya administrasi bulanan dan denda keterlambatan pinjaman.
- Konsolidasi neraca saldo penutupan (*trial balance*) untuk menjamin keseimbangan seluruh *Chart of Accounts* (COA).

### 1.6 Autonomous Traffic Generator

Pekerja otonom berbasis loop dinamis dan penjadwalan periodik yang memproduksi ribuan transaksi sintetis (transfer antar-rekening, setor/tarik kas, pembayaran merchant) secara berkelanjutan guna menguji konkurensi ekstrem, deteksi *race condition*, dan stabilitas latensi p99 database.

---

## 2. Struktur Direktori Monorepo (go.work)

```text
aethelcore/
├── .github/
│   └── workflows/
│       ├── test-core.yml        # CI linting & race-detector
│       └── build-images.yml     # Multi-stage Docker build
├── proto/                       # Sumber kebenaran kontrak antarmuka
│   ├── account/v1/account.proto
│   ├── ledger/v1/ledger.proto
│   ├── branch/v1/branch.proto
│   └── eod/v1/eod.proto
├── apps/
│   ├── api-gateway/             # Go: REST/JSON to gRPC translation, RBAC, Rate-limit
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   └── go.mod
│   ├── core-ledger/             # Go: Mesin transaksi double-entry utama
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── domain/          # Entities & repository interfaces
│   │   │   ├── usecase/         # Logika transfer, locking, audit chain
│   │   │   └── repository/      # Implementasi PostgreSQL (pgx)
│   │   └── go.mod
│   ├── worker-eod/              # Go: Batch engine penutupan hari (EOD)
│   │   ├── cmd/worker/main.go
│   │   └── go.mod
│   ├── simulator-cron/          # Go: Generator dummy transaksional otonom
│   │   ├── cmd/simulator/main.go
│   │   └── go.mod
│   └── web-console/             # Next.js 15: Portal Teller, Supervisor, & Nasabah
│       ├── src/
│       │   ├── app/
│       │   │   ├── branch/      # Portal Cabang (Layar Teller & Supervisor)
│       │   │   └── customer/    # Mock Mobile Banking Viewport
│       │   └── components/
│       └── package.json
├── pkg/                         # Modul bersama (Shared Go Packages)
│   ├── hashchain/               # Engine SHA-256 linked-list untuk audit record
│   ├── idempotency/             # Deduplikasi payload request berbasis hash
│   ├── money/                   # Penanganan integer presisi tetap (bebas float)
│   └── go.mod
├── deploy/                      # Lingkungan orkestrasi lokal
│   ├── docker-compose.yml       # PostgreSQL, Redis, Kafka, Prometheus, Grafana
│   ├── prometheus.yml
│   └── grafana/
│       └── dashboards/          # Visualisasi TPS, latency p99, ledger invariant
├── go.work                      # Orkestrasi workspace multi-modul Go
└── OVERVIEW.md
```

---

## 3. Skema Data & Database Design (PostgreSQL)

Seluruh nilai finansial disimpan dalam satuan integer terkecil (sen atau rupiah penuh) menggunakan tipe `BIGINT` untuk mencegah kesalahan pembulatan IEEE 754.

```sql
-- 1. Rekening Nasabah & Akun Internal Buku Besar
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Transaksi Finansial (Header)
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id VARCHAR(64) UNIQUE NOT NULL, -- Idempotency key
    transaction_type VARCHAR(32) NOT NULL,    -- TRANSFER, CASH_DEPOSIT, CASH_WITHDRAWAL, FEE, INTEREST
    amount BIGINT NOT NULL,
    description TEXT NOT NULL,
    maker_id VARCHAR(64) NOT NULL,
    checker_id VARCHAR(64) NULL,              -- Otorisasi Supervisor jika required
    status VARCHAR(16) NOT NULL DEFAULT 'POSTED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Jurnal Buku Besar (Double-Entry Mutation)
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
```

---

## 4. Analisis Vulnerability & Mitigasi Keamanan Siber

### 4.1 Race Condition & Deadlock Konkurensi

- **Vektor Serangan**: Dua transaksi transfer silang antara akun A dan akun B dieksekusi secara paralel. Penguncian baris database tanpa urutan deterministik akan memicu *cyclic deadlock*.
- **Mitigasi**: Mengurutkan identifier akun secara leksikografis sebelum query `FOR UPDATE` dieksekusi:

```go
firstID, secondID := fromID, toID
if firstID > secondID {
    firstID, secondID = toID, fromID
}
tx.Exec(ctx, "SELECT id, balance FROM accounts WHERE id IN ($1, $2) ORDER BY id FOR UPDATE", firstID, secondID)
```

### 4.2 Network Partition & Replay Attack

- **Vektor Serangan**: Kegagalan jaringan di tengah proses menyebabkan kanal pengguna mengirim ulang request transfer (*retry storm*), berisiko mendebit saldo ganda.
- **Mitigasi**: Penerapan kunci idempotensi wajib (`X-Idempotency-Key`). Lapisan Gateway memvalidasi state kunci pada Redis dengan status atomik `ACQUIRING` sebelum membuka transaksi database. Request duplikat akan menerima respons ter-*cache* tanpa eksekusi ulang.

### 4.3 Bypass Otorisasi Maker-Checker

- **Vektor Serangan**: Manipulasi parameter *client-side* untuk memalsukan status persetujuan transaksi di atas limit kas teller.
- **Mitigasi**: Penegakan verifikasi kriptografis berbasis *dual JWT claims*. Token Maker dan Checker divalidasi terhadap *public key* yang berbeda. Backend secara eksplisit menolak mutasi jika identitas `maker_id` identik dengan `checker_id`.

### 4.4 Silent Ledger Tampering (Insider Threat)

- **Vektor Serangan**: Manipulasi langsung baris database oleh akun administratif internal.
- **Mitigasi**: Setiap entri buku besar dihubungkan menggunakan rantai hash SHA-256:

$$\text{Hash}_n = \text{SHA256}(\text{Hash}_{n-1} \parallel \text{AccountID} \parallel \text{Amount} \parallel \text{BalanceAfter})$$

Batch audit EOD memverifikasi keutuhan rantai hash secara berulang; modifikasi ilegal pada satu entri akan memutus rantai dan memicu alarm anomali.

---

## 5. Spesifikasi Generator Data Otomatis (Cron Simulator)

Modul `apps/simulator-cron` bertindak sebagai generator lalu lintas transaksi otonom:

### 5.1 Cold-Start Provisioning

- Menghasilkan 5 entitas kantor cabang fiktif.
- Menghasilkan 10 akun Teller dan 5 akun Supervisor dengan kredensial terisolasi.
- Menghasilkan 1.000 akun rekening nasabah lengkap dengan saldo awal acak realistis.

### 5.2 High-Frequency Transaction Loop

- **Interval eksekusi**: $10\text{--}50\text{ ms}$ (Target: 50–200 TPS stabil).
- **Komposisi beban kerja**:
  - **70%**: Transfer peer-to-peer antar-rekening nasabah.
  - **15%**: Transaksi pembayaran pihak ketiga / merchant.
  - **10%**: Setor dan tarik tunai teller cabang.
  - **5%**: Transaksi gagal terencana (saldo tidak cukup, rekening beku) untuk menguji konsistensi rollback database.

### 5.3 Scheduled EOD Engine Loop

- **Frekuensi eksekusi**: Berjalan setiap interval 5 menit (mengemulasikan siklus penutupan buku 1 hari kalender perbankan).
- **Aktivitas batch**: Menjalankan kalkulasi akrual bunga harian simpanan, kapitalisasi bunga periodik, dan evaluasi keterlambatan cicilan pinjaman secara batch.

---

## 6. Metrik Kinerja & Standar Kuantitatif Target

| Metrik Evaluasi | Target Standar Produksi | Metrik Validasi |
| :--- | :--- | :--- |
| **Throughput Mutasi ACID** | $\ge 2.500 \text{ TPS}$ | Uji beban gRPC menggunakan tool `ghz` |
| **Latensi Transaksi (p99)** | $\le 20 \text{ ms}$ | Terukur pada panel metrik Prometheus |
| **Integritas Akuntansi** | $0 \text{ Anomali}$ | $\sum \text{Debit} - \sum \text{Kredit} = 0$ secara absolut |
| **Ketahanan Konkurensi** | $0{,}00\% \text{ Race Condition}$ | 500 goroutine paralel mendebit 1 rekening |
| **Footprint Memori Backend** | $\le 40 \text{ MB per pod}$ | Profiling alokasi heap melalui runtime `pprof` |