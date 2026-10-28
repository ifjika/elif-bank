# Integration & API Specification: AethelCore Engine

This document specifies external API interface contracts (RESTful & GraphQL), industry-standard error payloads, event/webhook delivery formats, and third-party integration mechanisms.

---

## 1. Response & Error Standards (RFC 7807)

All REST API endpoints in the API Gateway (Gin) return unified response payloads.

### 1.1 Success Response Format
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

### 1.2 Standard Error Response Format (RFC 7807 Problem Details)
```json
{
  "success": false,
  "error": {
    "code": "INSUFFICIENT_FUNDS",
    "message": "Available balance in source account is insufficient for this transaction amount.",
    "status_code": 422,
    "details": [
      {
        "field": "amount",
        "issue": "Requested 5,000,000 IDR exceeds available balance 1,250,000 IDR"
      }
    ],
    "reference_id": "req_a1b2c3d4-e5f6-7890"
  }
}
```

### 1.3 Error Codes Dictionary

| Error Code | HTTP Status | Description |
| :--- | :--- | :--- |
| `UNAUTHORIZED` | 401 | JWT token is missing, invalid, or expired. |
| `FORBIDDEN_MAKER_CHECKER`| 403 | Originating Maker is forbidden from approving their own transaction. |
| `ACCOUNT_NOT_ACTIVE` | 422 | Source or destination account status is `FROZEN`, `DORMANT`, or `CLOSED`. |
| `INSUFFICIENT_FUNDS` | 422 | Available balance is less than required debit amount. |
| `DUPLICATE_IDEMPOTENCY_KEY`| 409 / 200 | Idempotency key is currently being processed or already succeeded. |
| `RATE_LIMIT_EXCEEDED` | 429 | Client exceeded request rate limit (*Token Bucket* exhausted). |
| `INTERNAL_LEDGER_ERROR` | 500 | Core ledger anomaly or broken accounting balance invariant. |

---

## 2. RESTful API Contracts (Ingress Gateway)

### 2.1 User Authentication & Sessions (OAuth2 / JWT)
* **`POST /api/v1/auth/login`**
  * **Header**: `Content-Type: application/json`
  * **Request Body**:
    ```json
    {
      "username": "teller_jkt01",
      "password": "SuperSecretPassword123!",
      "role": "TELLER"
    }
    ```
  * **Response**: Returns `access_token` (15-minute TTL) and `refresh_token` (7-day TTL).

### 2.2 Peer-to-Peer Transfer (P2P Transfer)
* **`POST /api/v1/transfers`**
  * **Mandatory Headers**:
    * `Authorization: Bearer <jwt_token>`
    * `X-Idempotency-Key: <unique_uuid_v4>`
  * **Request Body**:
    ```json
    {
      "source_account_number": "1001000012",
      "destination_account_number": "1001000045",
      "amount": 2500000,
      "description": "Invoice Payment #8821"
    }
    ```
  * **Note**: `amount` is specified in whole integer units (Rupiah without decimals).

### 2.3 Branch Cash Transactions (Cash Deposit & Withdrawal)
* **`POST /api/v1/branch/cash-transactions`**
  * **Mandatory Headers**: `Authorization: Bearer <teller_jwt>`, `X-Idempotency-Key: <uuid>`
  * **Request Body**:
    ```json
    {
      "account_number": "1001000012",
      "transaction_type": "CASH_WITHDRAWAL",
      "amount": 75000000,
      "notes": "Corporate customer cash withdrawal"
    }
    ```
  * **Response Logic**:
    * If amount $\le \text{IDR 50,000,000} \to$ Immediate execution (`status: "POSTED"`).
    * If amount $> \text{IDR 50,000,000} \to$ Awaiting supervisor approval (`status: "PENDING_APPROVAL"`).

### 2.4 Supervisor Authorization (Checker)
* **`POST /api/v1/branch/approvals/:transaction_id`**
  * **Mandatory Header**: `Authorization: Bearer <supervisor_jwt>`
  * **Request Body**:
    ```json
    {
      "action": "APPROVE", 
      "notes": "Physical customer ID and passbook verification confirmed"
    }
    ```

---

## 3. GraphQL Contracts (Backoffice & Audit Console)

The `/graphql` endpoint is used by management and audit consoles to fetch nested data graphs efficiently.

### 3.1 GraphQL Schema Example
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
  entryType: String! # DR or CR
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

## 4. Webhook & Event Streaming Format (Kafka / Redpanda)

Every transaction successfully committed into the database emits an event to Kafka via the **Transactional Outbox Pattern**.

* **Topic**: `core.ledger.transaction-posted.v1`
* **Key**: `account_id` (guarantees FIFO partition ordering for each account)
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
