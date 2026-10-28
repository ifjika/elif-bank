# Security Guide & Threat Modeling: AethelCore Engine

This document outlines cybersecurity policies, the role-based access control (RBAC) matrix, threat modeling mitigations (OWASP Top 10 API Security), input sanitization rules, and secrets management standards.

---

## 1. Role-Based Access Control Matrix (RBAC Matrix)

Access permissions are enforced based on the caller's verified JWT claims:

| Module / Endpoint                           |      Customer      | Teller (Maker) | Supervisor (Checker) | System Admin |
| :------------------------------------------ | :----------------: | :------------: | :------------------: | :----------: |
| **Check Own Account Balance**               |         ✅         |       ❌       |          ❌          |      ❌      |
| **P2P Funds Transfer**                      | ✅ (IDR 25M Limit) |       ❌       |          ❌          |      ❌      |
| **View Customer Account Details**           |         ❌         |       ✅       |          ✅          |      ✅      |
| **Initiate Branch Cash Deposit/Withdrawal** |         ❌         |   ✅ (Maker)   |          ❌          |      ❌      |
| **Approve Cash Transaction > 50M**          |         ❌         |       ❌       |     ✅ (Checker)     |      ❌      |
| **Trigger Manual EOD Batch Execution**      |         ❌         |       ❌       |          ❌          |      ✅      |
| **Access GraphQL Audit & System Logs**      |         ❌         |       ❌       |          ✅          |      ✅      |

### 1.1 Cryptographic Maker-Checker Rules (Anti-Self-Approval)

1. **Dual Identity Verification**: The backend strictly validates:
   $$\text{maker\_id} \neq \text{checker\_id}$$
2. **Distinct Keypairs/Claims**: JWT tokens for Makers and Checkers are signed with distinctive keys or specific role claims independently verified by the API Gateway.

---

## 2. Threat Modeling Mitigations (OWASP API Security)

### 2.1 Broken Object Level Authorization (BOLA / IDOR)

- **Attack Vector**: A malicious customer alters the `account_id` route parameter to view balances or initiate unauthorized transfers from another customer's account.
- **Mitigation**:
  - The API Gateway cross-verifies that the target `account_id` in the database belongs to the `customer_id` cryptographically signed inside the caller's JWT token.
  - Customers cannot submit arbitrary `source_account_id` values without ownership validation.

### 2.2 Replay Attacks & Network Retries

- **Attack Vector**: Cellular network drops cause client applications to retransmit transfer requests, triggering multiple balance debits.
- **Mitigation**:
  - Mandatory **`X-Idempotency-Key`** header (unique client-generated UUIDv4).
  - Gateway performs atomic key acquisition in Redis:
    ```go
    // Initial status stored in Redis with 24-hour TTL
    SETNX idempotency:{key} "PROCESSING" EX 86400
    ```
  - If the key already exists with status `SUCCESS`, the Gateway immediately replays the cached response without touching the core transaction database.

### 2.3 Race Conditions & Negative Balances (Concurrency Attacks)

- **Attack Vector**: A customer initiates 10 simultaneous IDR 1,000,000 withdrawals from an account with only IDR 1,000,000 balance using distributed automated bots.
- **Mitigation**:
  - Strict pessimistic row-level locking: `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`.
  - Database-level check constraint: `CONSTRAINT chk_positive_balance CHECK (balance >= 0)`.
  - If available funds are insufficient, the transaction fails and rolls back instantly.

### 2.4 SQL Injection & Mass Assignment

- **Attack Vector**: Injecting malicious SQL syntax into transaction description or name search fields.
- **Mitigation**:
  - Strict prohibition of dynamic SQL string concatenation. All SQL queries use parameterized placeholders via `pgx` and `sqlx`.
  - Input payload sanitization and schema validation using `go-playground/validator`.

---

## 3. Secrets Management & Credential Safety

1. **Zero Hardcoded Secrets Policy**:
   - Database credentials, JWT private keys, and Redis authentication tokens must never be committed to source code repositories.
   - All configurations are injected via Environment Variables or `.env` files explicitly listed in `.gitignore`.
2. **Credential Password Hashing**:
   - All user passwords in `app_users` are salted and hashed using **Argon2id** or **Bcrypt** with a work factor $\ge 12$.
3. **Audit Trails Without Sensitive PII**:
   - Application telemetry logs (Stdout, Grafana Loki) must redact raw PINs, card CVVs, and complete authorization tokens.
