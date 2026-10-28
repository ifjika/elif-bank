# Implementation Plan & Roadmap: AethelCore Engine

Dokumen ini mendefinisikan peta jalan (*roadmap*), tabel fitur yang akan dibuat secara bertahap, dan checklist langkah kerja mikro (*active task checklist*) untuk fitur yang saat ini sedang aktif dikerjakan.

---

## 1. Tabel Rencana Fitur (Feature Implementation Matrix)

| Fase | Modul & Fitur | Kebutuhan Tech Stack Terkait | Status | Kriteria Selesai (Definition of Done) |
| :---: | :--- | :--- | :---: | :--- |
| **1** | **Setup Lingkungan & Multi-Module Workspace** | Go 1.23+, `go.work`, Docker Compose, PostgreSQL, MariaDB, Redis | 🟢 **Selesai** | Seluruh kontainer database & cache terkonfigurasi; struktur folder monorepo Go terinisialisasi bersih. |
| **2** | **Desain Skema Database & Migrasi SQL** | PostgreSQL, MariaDB, SQL DDL, `golang-migrate` / raw scripts | 🟢 **Selesai** | Berkas migrasi DDL PostgreSQL & MariaDB terbuat lengkap dengan constraint dan volume mount Docker. |
| **3** | **Core Ledger Engine: Double-Entry & Row Locking** | Go, `pgx/v5`, Lexicographical Locking, SHA-256 Hashchain | 🟢 **Selesai** | Logika Double-Entry atomik, sorting leksikografis, SHA-256 chain, dan unit test AAA lulus 100%. |
| **4** | **API Gateway: Gin REST, JWT, & Idempotensi** | Go, Gin Framework, JWT, Redis Token Bucket, RESTful API | 🟢 **Selesai** | Gateway REST API, JWT auth, RBAC, Redis idempotency & rate limiting lulus unit test 100%. |
| **5** | **Branch Operations: Maker-Checker System** | Go, Gin, RBAC, Dual JWT Claims | 🟡 **Sedang Dikerjakan** | Transaksi kas > 50 Jt masuk status `PENDING_APPROVAL` dan dapat di-approve/reject hanya oleh Checker. |
| **6** | **GraphQL Endpoint untuk Backoffice / Audit** | Go, `gqlgen`, GraphQL | ⚪ Belum Dimulai | Query GraphQL dapat menampilkan relasi akun dan validasi rantai hash integritas data transaksi. |
| **7** | **Batch Engine: End-of-Day (EOD) Worker** | Go, Cron/Tickers, Transactional Outbox, Kafka / Redpanda | ⚪ Belum Dimulai | Worker menghitung akrual bunga harian, PPh 20%, dan biaya administrasi secara batch otomatis. |
| **8** | **Autonomous Traffic Simulator** | Go Concurrency (Goroutines, Channels), Stress Testing | ⚪ Belum Dimulai | Generator memproduksi 50–200 TPS transaksi sintetis berkelanjutan untuk menguji latensi dan stabilitas. |
| **9** | **CI/CD Pipeline & Otomasi Pengujian** | GitHub Actions, `go test -race`, `golangci-lint` | ⚪ Belum Dimulai | Pipeline CI otomatis mengeksekusi linting, race-detector, dan build test pada setiap commit git. |

---

## 2. Active Task Checklist (Fitur yang Sedang Dikerjakan)

> [!IMPORTANT]
> **Aturan Penggunaan Checklist**:
> Seksi ini **HANYA** memuat langkah-langkah mikro dari fitur yang **SEDANG** dikerjakan saat ini. Begitu seluruh langkah di bawah ini selesai dikonfirmasi dan diverifikasi, checklist ini akan **dihapus kembali** (*cleared*), lalu diisi dengan rincian langkah mikro dari fitur berikutnya sesuai tabel implementasi di atas.

### Fase Aktif Saat Ini: **Fase 5 - Branch Operations: Maker-Checker System**

- [ ] **5.1** Buat usecase penarikan/penyetoran kas cabang di `apps/core-ledger/internal/usecase/branch_cash_usecase.go`:
  - Transaksi $\le \text{Rp50.000.000} \to$ Straight-Through Processing (`status: POSTED`).
  - Transaksi $> \text{Rp50.000.000} \to$ Kunci saldo ke `hold_balance`, `status: PENDING_APPROVAL`.
- [ ] **5.2** Buat usecase persetujuan Supervisor di `apps/core-ledger/internal/usecase/branch_approval_usecase.go`:
  - Penegakan *Four-Eyes Principle* (`maker_id != checker_id`, jika sama kembalikan error `403 Forbidden`).
  - Cabang `APPROVE`: mutasi saldo permanen, lepas `hold_balance`, catat jurnal double-entry, status `POSTED`.
  - Cabang `REJECT`: lepaskan `hold_balance` ke saldo efektif, status `REJECTED`.
- [ ] **5.3** Buat HTTP Handler Gin untuk endpoint cabang di `apps/api-gateway/internal/delivery/http/branch_handler.go`:
  - `POST /api/v1/branch/cash-transactions` (Role: TELLER / Maker).
  - `POST /api/v1/branch/approvals/:transaction_id` (Role: SUPERVISOR / Checker).
- [ ] **5.4** Tulis unit test komprehensif berstandar AAA di `apps/core-ledger/internal/usecase/branch_cash_usecase_test.go` dan `apps/api-gateway/internal/delivery/http/branch_handler_test.go`.
