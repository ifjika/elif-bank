# Application Flow & State Machines: AethelCore Engine

This document maps user flow diagrams, finite state machines (FSM) for transactions and accounts, and console navigational structures for banking operations.

---

## 1. User Flows

### 1.1 Peer-to-Peer Funds Transfer Flow (P2P Transfer)

```text
[ Customer / Mobile Client ]
        │
        │ 1. Send POST /api/v1/transfers (JWT + X-Idempotency-Key)
        ▼
[ API Gateway (Gin) ]
        │
        ├─► [ Validate JWT Token & Customer Claims ]
        ├─► [ Check Redis: Has Idempotency-Key already succeeded? ]
        │      └─► If YES: Return cached response (Complete).
        │      └─► If NO: Set "PROCESSING" state in Redis.
        │
        │ 2. Forward payload via gRPC: ExecuteTransfer()
        ▼
[ Core Ledger Engine (Go) ]
        │
        ├─► [ Begin Database Transaction (BEGIN) ]
        ├─► [ Sort Source & Destination Account UUIDs lexicographically ]
        ├─► [ Execute SELECT ... FOR UPDATE on both accounts ]
        ├─► [ Validate Account Status (Must be 'ACTIVE') ]
        ├─► [ Validate Balance: AvailableBalance >= Amount? ]
        │      └─► If Insufficient: ROLLBACK, return 422 Insufficient Funds.
        │
        ├─► [ Mutate Balances: Debit Source, Credit Destination ]
        ├─► [ Insert transactions table row (Header) ]
        ├─► [ Insert ledger_entries table rows (Balanced Debit & Credit + SHA-256 Chain) ]
        ├─► [ Insert outbox_events table row (Kafka Event) ]
        │
        ├─► [ COMMIT Database Transaction ]
        ▼
[ Success Response ]
        │
        ├─► Cache success response in Redis
        └─► Return HTTP 200 OK + Transaction Reference ID to Customer
```

---

### 1.2 Branch Cash Transactions & Maker-Checker Flow (> IDR 50,000,000)

```text
[ Teller (Maker) ]
        │
        │ 1. Submit Cash Withdrawal of IDR 75,000,000
        ▼
[ Core Ledger ]
        │
        ├─► Detect Amount: IDR 75,000,000 > IDR 50,000,000 (Approval threshold)
        ├─► Lock Effective Balance into 'hold_balance' (prevents double-spending)
        ├─► Save transaction with status: 'PENDING_APPROVAL'
        ▼
[ Branch Supervisor Portal (Checker) ]
        │
        │ 2. Receives notification: "Approval Required: Cash Withdrawal > 50M"
        │    Supervisor reviews physical ID & customer passbook
        │
        ├─► Option A: Supervisor clicks REJECT
        │      └─► Release hold_balance, update transaction status to 'REJECTED'
        │
        └─► Option B: Supervisor clicks APPROVE (Transmits Supervisor JWT)
               ├─► Verify: Checker ID != Maker ID
               ├─► Execute permanent balance debit on account
               ├─► Record double-entry journal to branch vault cash
               └─► Update transaction status to 'POSTED'
```

---

## 2. Finite State Machines (FSM)

### 2.1 Savings Account State Machine (Account FSM)

```text
    ┌────────────────┐
    │  DORMANT       │ (No activity for > 180 days)
    │                │
    └───────▲────────┘
            │ Inactive > 180 days
            │
            ▼ KYC Reverification
    ┌────────────────┐       Anomaly / Regulatory Freeze   ┌────────────────┐
    │  ACTIVE        ├────────────────────────────────────►│  FROZEN        │
    │  (Operational) │◄────────────────────────────────────┤ (Debits Blocked│
    └───────┬────────┘       Supervisor Unfreeze           └────────────────┘
            │
            │ Balance = IDR 0 & Customer closes account
            ▼
    ┌────────────────┐
    │  CLOSED        │ (Terminal State: Rejects all mutations permanently)
    └────────────────┘
```

### 2.2 Financial Transaction State Machine (Transaction FSM)

```text
                 [ Inbound Request ]
                         │
                         ▼
                  ┌──────────────┐
                  │  INITIATED   │
                  └──────┬───────┘
                         │
          Amount <= 50M? (or P2P Transfer)
                         ├──────────────────────────┐
                         │ Yes                      │ No (> 50M Branch Cash)
                         ▼                          ▼
                  ┌──────────────┐           ┌──────────────────┐
                  │  POSTED      │           │ PENDING_APPROVAL │
                  │ (Committed   │           └─────────┬────────┘
                  │  to Ledger)  │                     │
                  └──────────────┘           Supervisor│Decision
                                             ┌─────────┴─────────┐
                                             ▼                   ▼
                                      ┌──────────────┐    ┌──────────────┐
                                      │    POSTED    │    │   REJECTED   │
                                      └──────────────┘    └──────────────┘
```

---

## 3. Web Console Navigation Map

```text
[ Login Screen ]
   │
   ├─► Role = CUSTOMER ────► [ Customer Portal: Balance Check & Transfer History ]
   │
   ├─► Role = TELLER ──────► [ Teller Console: Branch Cash Deposit & Withdrawal ]
   │
   └─► Role = SUPERVISOR ──► [ Supervisor Portal: Approval Queue & Ledger Audit ]
```
