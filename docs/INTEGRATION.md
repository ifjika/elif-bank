# Integration & API Specification: AethelCore Engine

Dokumen ini mendokumentasikan spesifikasi kontrak antarmuka API eksternal (RESTful & GraphQL), payload error berstandar industri, penanganan *webhook/event*, serta mekanisme integrasi layanan pihak ketiga.

---

## 1. Standar Format Respons & Error (RFC 7807)

Seluruh endpoint REST API di lapisan API Gateway (Gin) mengembalikan struktur respons terpadu.

### 1.1 Format Respons Sukses
```json
{
  "success": true,
  "data": {
    "reference_id": "tx_9f81a7b2-3c10-4e56-8a9d-b8d234a1c512",
    "status": "POSTED",
    "amount": 5000000,
    "currency": "IDR",
    "created_at": "2026-10-28T14:30:00Z"
  },
  "meta": {
    "server_time": "2026-10-28T14:30:00Z"
  }
}
```

### 1.2 Format Respons Error Standar (RFC 7807 Problem Details)
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "Saldo efektif rekening asal tidak mencukupi untuk nominal transaksi ini.",
    "status_code": 422,
    "details": [
      {
        "field": "amount",
        "issue": "Requested 5.000.000 IDR exceeds available balance 1.250.000 IDR"
      }
    ],
    "reference_id": "req_a1b2c3d4-e5f6-7890"
  }
}
```

### 1.3 Kamus Kode Error (Error Codes Dictionary)

| Kode Error | HTTP Status | Deskripsi |
| :--- | :--- | :--- |
| `UNAUTHORIZED` | 401 | Token JWT tidak valid, kedaluwarsa, atau tidak disertakan. |
| `FORBIDDEN_MAKER_CHECKER`| 403 | Maker dilarang menyetujui transaksi buatannya sendiri. |
| `ACCOUNT_NOT_ACTIVE` | 422 | Rekening sumber atau tujuan berstatus `FROZEN`, `DORMANT`, atau `CLOSED`. |
| `INSUFFICIENT_FUNDS` | 422 | Saldo efektif (*available balance*) kurang dari nominal debit. |
| `DUPLICATE_IDEMPOTENCY_KEY`| 409 / 200 | Kunci idempotensi sedang diproses atau sudah berhasil dieksekusi sebelumnya. |
| `RATE_LIMIT_EXCEEDED` | 429 | Klien melampaui kuota panggilan per detik (*Token Bucket* penuh). |
| `INTERNAL_LEDGER_ERROR` | 500 | Anomali sistem buku besar atau ketidakseimbangan invarian akuntansi. |

---

## 2. Kontrak RESTful API (Ingress Gateway)

### 2.1 Autentikasi Pengguna & Sesi (OAuth2 / JWT)
* **`POST /api/v1/auth/login`**
  * **Header**: `Content-Type: application/json`
  * **Body Request**:
    ```json
    {
      "username": "teller_jkt01",
      "password": "SuperSecretPassword123!",
      "role": "TELLER"
    }
    ```
  * **Respons**: Mengembalikan `access_token` (JWT berumur 15 menit) dan `refresh_token` (berumur 7 hari).

### 2.2 Transfer Antar-Rekening (P2P Transfer)
* **`POST /api/v1/transfers`**
  * **Header Wajib**:
    * `Authorization: Bearer <jwt_token>`
    * `X-Idempotency-Key: <unique_uuid_v4>`
  * **Body Request**:
    ```json
    {
      "source_account_number": "1001000012",
      "destination_account_number": "1001000045",
      "amount": 2500000,
      "description": "Pembayaran Invoice #8821"
    }
    ```
  * **Catatan**: Satuan `amount` adalah Rupiah penuh (integer tanpa desimal).

### 2.3 Transaksi Setor & Tarik Tunai Cabang (Branch Cash)
* **`POST /api/v1/branch/cash-transactions`**
  * **Header Wajib**: `Authorization: Bearer <teller_jwt>`, `X-Idempotency-Key: <uuid>`
  * **Body Request**:
    ```json
    {
      "account_number": "1001000012",
      "transaction_type": "CASH_WITHDRAWAL",
      "amount": 75000000,
      "notes": "Penarikan kas nasabah korporat"
    }
    ```
  * **Logika Respons**:
    * Jika nominal $\le \text{Rp50.000.000} \to$ Langsung sukses (`status: "POSTED"`).
    * Jika nominal $> \text{Rp50.000.000} \to$ Menunggu otorisasi (`status: "PENDING_APPROVAL"`).

### 2.4 Otorisasi Transaksi oleh Supervisor (Checker)
* **`POST /api/v1/branch/approvals/:transaction_id`**
  * **Header Wajib**: `Authorization: Bearer <supervisor_jwt>`
  * **Body Request**:
    ```json
    {
      "action": "APPROVE", 
      "notes": "Dokumen verifikasi fisik KTP & buku tabungan terverifikasi"
    }
    ```

---

## 3. Kontrak GraphQL (Backoffice & Audit Console)

Endpoint GraphQL (`/graphql`) digunakan oleh konsol manajemen untuk menarik relasi data yang kompleks secara efisien.

### 3.1 Skema Query GraphQL (Contoh)
```graphql
type Account {
  id: ID!
  accountNumber: String!
  accountType: String!
  balance: Float!
  holdBalance: Float!
  status: String!
  recentEntries(limit: Int): [LedgerEntry!]!
}

type LedgerEntry {
  id: ID!
  entryType: String! # DR atau CR
  amount: Float!
  balanceAfter: Float!
  previousHash: String!
  currentHash: String!
  createdAt: String!
}

type Query {
  getAccountLedger(accountNumber: String!): Account
  verifyLedgerHashIntegrity(accountNumber: String!): Boolean!
}
```

---

## 4. Format Webhook & Event Streaming (Kafka / Redpanda)

Setiap transaksi yang berhasil dibukukan (*committed*) ke dalam database PostgreSQL akan menerbitkan event ke topik Kafka menggunakan pola **Transactional Outbox**.

* **Topic**: `core.ledger.transaction-posted.v1`
* **Key**: `account_id` (menjamin pesan untuk rekening yang sama diproses secara urut/FIFO)
* **Payload**:
```json
{
  "event_id": "evt_77a1c92b-81d3-456a-b89e-9081234a56b7",
  "event_type": "TRANSACTION_POSTED",
  "timestamp": "2026-10-28T14:30:00Z",
  "payload": {
    "transaction_id": "tx_9f81a7b2-3c10-4e56-8a9d-b8d234a1c512",
    "transaction_type": "TRANSFER",
    "amount": 2500000,
    "source_account": "1001000012",
    "destination_account": "1001000045",
    "status": "POSTED"
  }
}
```
