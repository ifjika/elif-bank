# Product Requirements Document (PRD)

## Project: AethelCore - Mini Core Banking System (CBS)

This document details the product requirements, banking business logic, system boundaries, and target metrics for AethelCore. It is designed to be clear and accessible for engineers learning banking architectures and high-performance Golang backends.

---

## 1. Introduction: What is a Core Banking System (CBS)?

In the commercial banking world, a **Core Banking System (CBS)** acts as the central engine of bank operations. It is responsible for:

1. Opening and managing customer deposit and loan accounts.
2. Processing balance mutations (deposits, withdrawals, transfers).
3. Maintaining a perpetually balanced accounting General Ledger (Double-Entry).
4. Executing day-end reconciliation and rollover (_End of Day_ / EOD).
5. Preventing internal fraud through dual-authorization controls (_Four-Eyes Principle / Maker-Checker_).

The objective of this project is to build an industry-standard, miniature distributed CBS replica using modern engineering practices, demonstrating robust architecture and concurrency resilience.

---

## 2. User Personas

| Persona                              | Role                                        | Primary Responsibilities                                                                                        |
| :----------------------------------- | :------------------------------------------ | :-------------------------------------------------------------------------------------------------------------- |
| **Customer**                         | Digital channel end user (_Mobile Banking_) | Performs funds transfers, checks balances, reviews statement histories, and earns savings interest.             |
| **Teller (Maker)**                   | Branch operational staff                    | Processes physical cash deposits and over-the-counter withdrawals at the branch.                                |
| **Supervisor (Checker)**             | Branch operations officer                   | Reviews and approves high-risk transactions (e.g., large over-the-counter cash withdrawals).                    |
| **Automated System (Worker/Daemon)** | Background worker                           | Executes daily interest calculations (EOD), continuous high-load traffic simulation, and data integrity audits. |

---

## 3. Scope & Key Feature Specifications

### 3.1 Funding & Account Management

- **Account Types**:
  - _Individual Savings_: Liquid customer savings account with daily accrued interest.
  - _Commercial Checking_: Business/merchant operational account with high-volume transaction activity.
  - _Time Deposit (Certificate of Deposit)_: Term-locked savings account with early withdrawal penalties.
- **Account State Machine**:
  - `ACTIVE`: Fully operational account; accepts debit and credit mutations.
  - `FROZEN`: Temporarily restricted account (e.g., regulatory hold or anomaly detection). Rejects all outgoing debit transactions.
  - `DORMANT`: Inactive account with zero customer-initiated transactions for > 180 days. Requires KYC reverification to reactivate.
  - `CLOSED`: Permanently terminated account with zero balance. Rejects all mutations unconditionally.
- **Effective Balance & Hold Balance Separation**:
  - $\text{Total Balance} = \text{Available Balance} + \text{Hold Balance}$
  - `Hold Balance` is reserved for collateral, pre-authorizations, or pending approval queues without prematurely mutating ledger journals.

---

### 3.2 Double-Entry General Ledger Engine

- **Core Rule**: Single-sided balance mutations (`UPDATE accounts SET balance = balance - 1000`) are strictly prohibited.
- **Balanced Journaling**: Every fund movement must generate at least two corresponding ledger entry rows:
  - Origin / Asset account decreases $\to$ recorded on the appropriate ledger side.
  - Destination / Liability account increases $\to$ recorded on the opposing ledger side.
- **Accounting Invariant**:
  $$\sum \text{Debit} - \sum \text{Credit} = 0$$
  If in any transaction total debits do not equal total credits, the database transaction must `ROLLBACK` immediately.

---

### 3.3 Branch Governance: Maker-Checker (Four-Eyes Principle)

- **Threshold**: Over-the-counter branch cash transactions $\le \text{IDR 50,000,000}$ are processed immediately (_Straight-Through Processing_).
- **Dual Authorization**: Branch cash transactions $> \text{IDR 50,000,000}$ automatically transition to `PENDING_APPROVAL` status.
- **Separation of Duties**:
  - The transaction originator (`maker_id`) is strictly prohibited from approving their own transaction.
  - A branch supervisor (`checker_id`) must authenticate with distinct credentials/tokens to either `APPROVE` or `REJECT` the transaction.

---

### 3.4 Automated Day-End Rollover (End-of-Day / EOD Engine)

- Scheduled background worker emulating bank calendar date advancement:
  1. **Interest Accrual**: Computes daily accrued interest on the closing balance of every active account:
     $$\text{Daily Interest} = \frac{\text{Closing Balance} \times \text{Interest Rate}}{365}$$
  2. **Withholding Tax (PPh Final 20%)**: Automatically withholds income tax on interest yields per Indonesian financial regulations.
  3. **Account Maintenance Fees**: Debits monthly account administration fees.
  4. **Loan Collectibility Classification**: Categorizes loan accounts into regulatory performance buckets (KOL 1–5) based on Days Past Due (DPD).
  5. **Trial Balance Reconciliation**: Sums all assets, liabilities, and equity to verify the fundamental accounting equation.

---

### 3.5 Autonomous Traffic Simulator (Cron Simulator)

- Continuously generates synthetic transaction loads to:
  - Benchmark system performance under extreme concurrency.
  - Validate race-condition prevention and deadlock-free row locking.
  - Stream real-time telemetry metrics to Prometheus/Grafana dashboards.

---

## 4. Non-Functional Requirements (NFR)

1. **ACID Transaction Guarantees**: Atomicity, Consistency, Isolation, and Durability strictly enforced by the MySQL / TiDB database engine.
2. **Concurrency Determinism**: Simultaneous cross-transfers must never produce deadlocks or unauthorized negative balances.
3. **Auditability (Tamper-Evident)**: Every ledger mutation row is linked cryptographically via SHA-256 hash chaining to detect any direct database tampering immediately.
4. **API Idempotency**: Ingress endpoints must reject or safely return cached responses when clients retransmit identical requests due to network timeouts.
5. **Performance & Latency Targets**:
   - Transaction throughput target: $\ge 2,500 \text{ TPS}$.
   - 99th percentile latency (p99): $\le 20 \text{ ms}$.
