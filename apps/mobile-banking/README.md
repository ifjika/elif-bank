# AethelCore Mobile Banking (`apps/mobile-banking`)

This application serves as the **Customer Digital Banking Channel (Mobile Banking / M-Banking)** for the AethelCore banking ecosystem.

---

## 📱 Application Overview & Persona

* **Target User**: Nasabah Perorangan (*Retail Banking Customer*).
* **Primary Scope**:
  - Secure Customer Login & Biometric Authentication (FaceID / Fingerprint).
  - Real-time Account Balance View (Separation of Available Balance vs Hold Balance).
  - Peer-to-Peer (P2P) Real-Time Funds Transfer with mandatory `X-Idempotency-Key`.
  - Transaction Mutation History & e-Statements.
  - QRIS / Merchant Payment Simulation.
  - Push Notifications for account mutations (powered by Kafka Outbox consumer).

---

## 🏛️ Architecture & Recommended Tech Stack

* **Framework**: React Native with Expo (TypeScript).
* **Navigation**: React Navigation (Bottom Tabs + Native Stack) / Expo Router.
* **State Management**: TanStack Query (React Query) + Zustand.
* **Styling**: NativeWind (Tailwind CSS for React Native) adhering to [`docs/DESIGN_SYSTEM.md`](../../docs/DESIGN_SYSTEM.md).
* **Security & Hardening**:
  - Encrypted Storage (Expo SecureStore / React Native Keychain) for JWT Refresh Tokens.
  - SSL/TLS Pinning for Ingress API Gateway communication.
  - Anti-Replay Idempotency Key Generation (UUIDv4) on every funds transfer.

---

## 🗂️ Directory Hierarchy

```text
apps/mobile-banking/
├── src/
│   ├── app/                      # Root entry & providers
│   ├── navigation/               # Route stacks & Bottom Tab navigators
│   │   ├── RootNavigator.tsx
│   │   ├── AuthStack.tsx
│   │   └── AppTabs.tsx
│   ├── screens/                  # Customer Screen Viewports
│   │   ├── auth/                 # Login, PIN verification, Biometrics
│   │   ├── home/                 # Account balance card & quick action shortcuts
│   │   ├── transfer/             # P2P Transfer form, beneficiary list, receipt
│   │   ├── history/              # Mutation statements & transaction search
│   │   └── profile/              # Account settings, limits, customer KYC view
│   ├── components/               # Mobile UI Components (Keypad, BottomSheet, Cards)
│   ├── hooks/                    # Custom hooks (useAuth, useAccounts, useTransfer)
│   ├── services/                 # API Gateway HTTP Client (Axios + SecureStore)
│   └── types/                    # Domain models & API Contracts
├── app.json                      # Expo application manifest
├── package.json
└── tsconfig.json
```

---

## 🔗 Related Documentation
* [**PRD.md**](../../docs/PRD.md) — Section 2 & 3.1 (Customer Persona & Funding Accounts).
* [**APP_FLOW.md**](../../docs/APP_FLOW.md) — Section 1.1 (Peer-to-Peer Transfer Flow).
* [**DESIGN_SYSTEM.md**](../../docs/DESIGN_SYSTEM.md) — Monetary Formatting & Tabular Numbers.
