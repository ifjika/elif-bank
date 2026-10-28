# Application Flow & State Machines: AethelCore Engine

Dokumen ini memetakan diagram alur pengguna (*user flow*), *finite state machine* transaksi dan rekening, serta navigasi antarmuka untuk portal operasional perbankan.

---

## 1. Diagram Alur Pengguna (User Flows)

### 1.1 Alur Transfer Dana Antar-Rekening (P2P Transfer)

```text
[ Nasabah / Klien ]
        │
        │ 1. Kirim POST /api/v1/transfers (JWT + X-Idempotency-Key)
        ▼
[ API Gateway (Gin) ]
        │
        ├─► [ Validasi Token JWT & Klaim Nasabah ]
        ├─► [ Cek Redis: Apakah Idempotency-Key sudah pernah sukses? ]
        │      └─► Jika YA: Kembalikan cached response (Selesai).
        │      └─► Jika TIDAK: Set state "PROCESSING" di Redis.
        │
        │ 2. Forward payload via gRPC: ExecuteTransfer()
        ▼
[ Core Ledger Engine (Go) ]
        │
        ├─► [ Buka Transaksi Database (BEGIN) ]
        ├─► [ Urutkan UUID Akun Asal & Tujuan secara leksikografis ]
        ├─► [ Eksekusi SELECT ... FOR UPDATE pada kedua akun ]
        ├─► [ Validasi Status Rekening (Wajib 'ACTIVE') ]
        ├─► [ Validasi Saldo: AvailableBalance >= Amount? ]
        │      └─► Jika Saldo Kurang: ROLLBACK, kembalikan 422 Insufficient Funds.
        │
        ├─► [ Mutasi Saldo: Kurangi Akun Asal, Tambah Akun Tujuan ]
        ├─► [ Insert Tabel transactions (Header) ]
        ├─► [ Insert Tabel ledger_entries (Debit & Kredit berpasangan + SHA256 Chain) ]
        ├─► [ Insert Tabel outbox_events (Event Kafka) ]
        │
        ├─► [ COMMIT Transaksi Database ]
        ▼
[ Respon Sukses ]
        │
        ├─► Simpan hasil sukses di Redis cache
        └─► Kembalikan HTTP 200 OK + Transaction Reference ID ke Nasabah
```

---

### 1.2 Alur Transaksi Kas Cabang & Maker-Checker (> Rp 50 Juta)

```text
[ Teller (Maker) ]
        │
        │ 1. Input Transaksi Penarikan Kas Rp 75.000.000
        ▼
[ Core Ledger ]
        │
        ├─► Deteksi Nominal: Rp 75.000.000 > Rp 50.000.000 (Threshold limit)
        ├─► Kunci Saldo Efektif sebagai 'hold_balance' (agar tidak ditarik ganda)
        ├─► Simpan transaksi dengan status: 'PENDING_APPROVAL'
        ▼
[ Portal Supervisor (Checker) ]
        │
        │ 2. Muncul notifikasi "Approval Required: Cash Withdrawal > 50 Jt"
        │    Supervisor meninjau dokumen fisik & KTP nasabah
        │
        ├─► Opsi A: Supervisor Klik REJECT
        │      └─► Lepaskan hold_balance, ubah status transaksi menjadi 'REJECTED'
        │
        └─► Opsi B: Supervisor Klik APPROVE (Kirim JWT Supervisor)
               ├─► Verifikasi: Checker ID != Maker ID
               ├─► Eksekusi debit permanen pada rekening
               ├─► Catat jurnal double-entry ke kas fisik cabang
               └─► Ubah status transaksi menjadi 'POSTED'
```

---

## 2. Finite State Machines (FSM)

### 2.1 State Machine Rekening Simpanan (Account FSM)

```text
    ┌────────────────┐
    │  DORMANT       │ (Tidak ada transaksi > 180 hari)
    │                │
    └───────▲────────┘
            │ Tidak aktif > 180 hari
            │
            ▼ Verifikasi ulang KTP
    ┌────────────────┐       Blokir Anomali / Hukum       ┌────────────────┐
    │  ACTIVE        ├───────────────────────────────────►│  FROZEN        │
    │  (Siap Pakai)  │◄───────────────────────────────────┤ (Debit Diblokir│
    └───────┬────────┘       Buka Blokir Supervisor       └────────────────┘
            │
            │ Saldo = Rp 0 & Nasabah menutup rekening
            ▼
    ┌────────────────┐
    │  CLOSED        │ (Status Final: Tidak dapat menerima mutasi apapun)
    └────────────────┘
```

### 2.2 State Machine Transaksi Finansial (Transaction FSM)

```text
                [ Request Masuk ]
                        │
                        ▼
                 ┌──────────────┐
                 │  INITIATED   │
                 └──────┬───────┘
                        │
         Nominal <= 50 Jt? (atau Transfer P2P)
                        ├──────────────────────────┐
                        │ Ya                       │ Tidak (> 50 Jt Cabang)
                        ▼                          ▼
                 ┌──────────────┐           ┌──────────────────┐
                 │  POSTED      │           │ PENDING_APPROVAL │
                 │ (Buku Besar  │           └─────────┬────────┘
                 │  Tercatat)   │                     │
                 └──────────────┘           Supervisor│Decision
                                            ┌─────────┴─────────┐
                                            ▼                   ▼
                                     ┌──────────────┐    ┌──────────────┐
                                     │    POSTED    │    │   REJECTED   │
                                     └──────────────┘    └──────────────┘
```

---

## 3. Pemetaan Navigasi Konsol Web (Screen Navigation Map)

```text
[ Login Screen ]
   │
   ├─► Role = CUSTOMER ────► [ Customer Portal: Cek Saldo & Histori Transfer ]
   │
   ├─► Role = TELLER ──────► [ Teller Console: Setor Tunai / Tarik Tunai Kas ]
   │
   └─► Role = SUPERVISOR ──► [ Supervisor Portal: Approval Queue & Ledger Audit ]
```
