# High-Level Architecture & System Design: AethelCore Engine

Dokumen ini menjelaskan pola arsitektur tingkat tinggi (*high-level architecture*), diagram topologi sistem, strategi komunikasi antar-layanan (*inter-service communication*), dan hirarki folder proyek AethelCore. Dokumen ini disesuaikan untuk kebutuhan implementasi monorepo berbasis Go Workspaces (`go.work`).

---

## 1. Filosofi Arsitektur

AethelCore dibangun dengan prinsip **Clean Architecture (Hexagonal / Ports & Adapters)** dan pola **Backend-for-Frontend (BFF)** terdistribusi:

1. **Separation of Concerns**: Memisahkan lapisan penanganan protokol (HTTP/REST via Gin, gRPC, GraphQL) dari logika bisnis murni (*domain logic* & akuntansi *double-entry*).
2. **Determinisme Konkurensi**: Lapisan transaksi buku besar (*ledger engine*) dirancang untuk mengeksekusi mutasi saldo secara berurutan dan terisolasi tanpa risiko *cyclic deadlock*.
3. **Auditability by Design**: Setiap perubahan saldo di database secara otomatis memicu pembuatan jejak audit kriptografis berantai (*SHA-256 hash chaining*).
4. **Polyglot Persistence Ringan**:
   * **PostgreSQL**: Bertindak sebagai *Primary Ledger of Record* (ACID, row locking, BIGINT).
   * **MariaDB / MySQL**: Menyimpan data identitas nasabah, profil cabang, dan data pengguna (*KYC & Auth Store*).
   * **Redis**: Menangani *Distributed Lock*, *Rate Limiting* (Token Bucket), dan *Idempotency Caching*.
   * **Kafka / Redpanda**: Broker event asinkron untuk mendistribusikan notifikasi mutasi dan *Transactional Outbox*.

---

## 2. Diagram Topologi Sistem (System Topology)

```text
       ┌──────────────────────┐           ┌──────────────────────┐
       │   Mobile Banking     │           │  Web Console Cabang  │
       │   (Kanal Nasabah)    │           │ (Teller & Supervisor)│
       └──────────┬───────────┘           └──────────┬───────────┘
                  │ HTTPS / REST (JSON)              │ HTTPS / GraphQL & REST
                  ▼                                  ▼
      ┌─────────────────────────────────────────────────────────────┐
      │               API GATEWAY & INGRESS LAYER                   │
      │                     (Go + Gin Framework)                    │
      │  - Routing & Reverse Proxy                                  │
      │  - Token Bucket Rate Limiting (Redis)                       │
      │  - Dual JWT Authentication (Maker & Checker)                │
      │  - Idempotency-Key Deduplication                            │
      └──────────────┬──────────────────────────────┬───────────────┘
                     │                              │
          gRPC/Protobuf (Internal)       SQL Connection (Pool)
                     ▼                              ▼
      ┌─────────────────────────────┐  ┌────────────────────────────┐
      │     CORE LEDGER ENGINE      │  │    CUSTOMER & AUTH DB      │
      │     (Microservice Go)       │  │    (MariaDB / MySQL)       │
      │ - Lexicographical Row Lock  │  │ - Data Nasabah (KYC)       │
      │ - Double-Entry Mutations    │  │ - Kredensial User & Role   │
      │ - Cryptographic Audit Trail │  └────────────────────────────┘
      └──────────────┬──────────────┘
                     │
         SQL Connection (pgx pool)
                     ▼
      ┌─────────────────────────────┐
      │   PRIMARY LEDGER DATABASE   │
      │        (PostgreSQL)         │
      │ - accounts (BIGINT Balance) │
      │ - transactions (Header)     │
      │ - ledger_entries (Chained)  │
      └──────────────┬──────────────┘
                     │
        CDC / Transactional Outbox
                     ▼
      ┌─────────────────────────────┐
      │    EVENT BROKER (KAFKA)     │
      └──────────────┬──────────────┘
                     │
         Consumer Event Subscription
                     ▼
      ┌─────────────────────────────┐  ┌────────────────────────────┐
      │     WORKER-EOD ENGINE       │  │    SIMULATOR-CRON          │
      │ - Daily Interest Accrual    │  │ - Autonomous Traffic       │
      │ - Loan Installment Billing  │  │ - Extreme Load Testing     │
      │ - PPh 20% Deductions        │  │ - Synthetic Transaction Gen│
      └─────────────────────────────┘  └────────────────────────────┘
```

---

## 3. Komunikasi Antar-Layanan (Communication Protocols)

1. **Eksternal ke API Gateway (Inbound)**:
   * **RESTful HTTP/JSON** menggunakan framework **Gin**: Digunakan untuk operasi transaksional nasabah dan teller (misal: Transfer Dana, Setor Kas, Tarik Kas).
   * **GraphQL** (`gqlgen`): Digunakan untuk konsol manajerial, penarikan riwayat mutasi dengan relasi kompleks, dan pelaporan audit tanpa *over-fetching*.
2. **Gateway ke Core Ledger (Internal IPC)**:
   * **gRPC / Protocol Buffers (HTTP/2)**: Menghasilkan komunikasi dengan latensi sangat rendah, payload biner ringkas, dan kontrak antarmuka yang sangat ketat (*type-safe*).
3. **Core Ledger ke Worker Latar Belakang (Asinkron)**:
   * **Transactional Outbox Pattern via Kafka / Redpanda**: Menjamin setiap transaksi yang sukses ditulis di database akan memicu event tanpa resiko kehilangan data akibat kegagalan jaringan.

---

## 4. Pola Arsitektur Kode: Clean Architecture

Setiap microservice di dalam folder `apps/` mengadopsi struktur berlapis yang konsisten:

```text
HTTP / gRPC Request
       │
       ▼
[ Delivery / Controller Layer ]  --> (Gin Handlers, gRPC Servers)
       │ (Validasi payload DTO)
       ▼
[ Usecase / Service Layer ]      --> (Logika Transfer, Maker-Checker, Aturan Bisnis)
       │ (Orkestrasi entitas)
       ▼
[ Domain Layer ]                 --> (Model murni, aturan akuntansi, errors)
       │ (Melalui Interfaces)
       ▼
[ Repository / Adapter Layer ]   --> (PostgreSQL via pgx, MariaDB via sqlx, Redis)
```

**Keuntungan bagi Pemula**:
* Logika bisnis tidak tercampur aduk dengan kode query SQL atau framework web.
* Jika database diganti atau ditambah driver baru, kode di lapisan Use Case tidak perlu diubah sama sekali (*Dependency Inversion Principle*).

---

## 5. Hirarki Folder Proyek Monorepo (`go.work`)

```text
elif-bank/
├── .github/
│   └── workflows/
│       └── ci-pipeline.yml          # GitHub Actions: test, race-detector, lint
├── proto/                           # Sumber kebenaran kontrak gRPC/Protobuf
│   ├── account/v1/account.proto
│   ├── ledger/v1/ledger.proto
│   ├── branch/v1/branch.proto
│   └── eod/v1/eod.proto
├── apps/                            # Aplikasi yang dapat dieksekusi (Microservices)
│   ├── api-gateway/                 # Go + Gin: Ingress REST, GraphQL, RBAC, JWT
│   │   ├── cmd/server/main.go
│   │   ├── internal/delivery/http/  # Gin handlers & middleware
│   │   ├── internal/delivery/graph/ # GraphQL resolvers & schema
│   │   ├── internal/client/         # gRPC client ke Core Ledger
│   │   └── go.mod
│   ├── core-ledger/                 # Go: Mesin transaksi double-entry utama
│   │   ├── cmd/server/main.go
│   │   ├── internal/domain/         # Entity, value objects, domain interfaces
│   │   ├── internal/usecase/        # Transfer engine, locking logic, audit chain
│   │   ├── internal/repository/     # PostgreSQL pgx adapter & MariaDB adapter
│   │   └── go.mod
│   ├── worker-eod/                  # Go: Batch engine penutupan hari (EOD)
│   │   ├── cmd/worker/main.go
│   │   └── go.mod
│   └── simulator-cron/              # Go: Generator dummy transaksional otonom
│       ├── cmd/simulator/main.go
│       └── go.mod
├── pkg/                             # Pustaka utilitas bersama (Shared Packages)
│   ├── hashchain/                   # Utilitas SHA-256 hash chaining
│   ├── idempotency/                 # Middleware & logic idempotensi Redis
│   ├── money/                       # Fixed-point precision math (BIGINT)
│   └── go.mod
├── deploy/                          # Konfigurasi container lokal
│   ├── docker-compose.yml           # PostgreSQL, MariaDB, Redis, Redpanda
│   └── config/
├── docs/                            # Dokumentasi arsitektur & panduan sistem
├── go.work                          # Orkestrator multi-module Go
└── README.md
```
