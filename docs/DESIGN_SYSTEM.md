# Design System & UX Standards: AethelCore Console

This document defines the visual user interface (UI) and user experience (UX) standards for the AethelCore banking console (Teller Portal, Branch Supervisor, and Customer Simulation). It provides clear design guidance that reflects enterprise-grade financial precision and reliability.

---

## 1. Banking Design Principles

1. **Financial Figure Clarity**: Numerical currency values and percentages must never shift layout or truncate. Always use _tabular figures_ (equal-width numbers).
2. **Credibility & Security**: Palette dominated by deep navy and slate hues, conveying stability, security, and institutional trust.
3. **Multi-Step Confirmation**: Operations mutating balances or account states must include review steps and explicit confirmation prior to dispatch.
4. **Unambiguous State Indicators**: Transaction and account statuses must utilize high-contrast colors and distinct badges (Green for active/success, Red for frozen/failure, Amber for pending approvals).

---

## 2. Design Tokens

### 2.1 Color Palette

| Token                    | Hex Value             | Usage / Description                                                                    |
| :----------------------- | :-------------------- | :------------------------------------------------------------------------------------- |
| `--color-brand-primary`  | `#0F172A` (Slate 900) | Primary background for navigation bars, headers, and major headings.                   |
| `--color-brand-accent`   | `#2563EB` (Blue 600)  | Primary action buttons, interactive links, active focus rings.                         |
| `--color-surface-bg`     | `#F8FAFC` (Slate 50)  | Global application page background.                                                    |
| `--color-surface-card`   | `#FFFFFF` (White)     | Background for cards, data tables, and modal dialogs.                                  |
| `--color-border-subtle`  | `#E2E8F0` (Slate 200) | Table divider borders, card strokes, and form input outlines.                          |
| `--color-status-success` | `#16A34A` (Green 600) | Status badges for `ACTIVE`, credit mutations (incoming), successful transactions.      |
| `--color-status-warning` | `#D97706` (Amber 600) | Status badges for `PENDING_APPROVAL`, `DORMANT`, transaction threshold warnings.       |
| `--color-status-danger`  | `#DC2626` (Red 600)   | Status badges for `FROZEN`, `CLOSED`, debit mutations (outgoing), failed transactions. |

### 2.2 Typography

- **Primary Font Stack**: `Inter`, `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, sans-serif.
- **Monetary Font Utility**: Must apply CSS class `font-mono` or `font-variant-numeric: tabular-nums` across all components displaying balances, account numbers, and transfer sums.
- **Type Scale Hierarchy**:
  - `Heading 1` (Page Title): 24px / Semibold (600)
  - `Heading 2` (Card/Section Title): 18px / Semibold (600)
  - `Heading 3` (Subheading): 14px / Medium (500)
  - `Body Regular` (Explanatory Text): 14px / Regular (400)
  - `Caption / Meta`: 12px / Regular (400) - Slate 500
  - `Display Balance` (Prominent Balance Display): 32px / Bold (700) - Tabular Nums

### 2.3 Spacing Scale & Corner Radius

- **Spacing Grid**: Multiples of 4px (4px, 8px, 12px, 16px, 24px, 32px, 48px).
- **Border Radius**:
  - Buttons & Inputs: `6px` (Rounded Medium)
  - Cards & Containers: `10px` (Rounded Large)
  - Badges & Tags: `9999px` (Full Pill)

---

## 3. Standard UI Components

### 3.1 Account Balance Card

- Displays Account Number (with quick copy button), Customer Name, Account Type (Savings/Checking), Available Balance, and Hold Balance.
- Balance Display: Formatted with standard currency notation and thousand separators (e.g., `Rp 150,000,000.00` or `IDR 150,000,000`).

### 3.2 State Badges

Status indicators for accounts and transactions:

- `<Badge variant="success">ACTIVE</Badge>`
- `<Badge variant="warning">PENDING APPROVAL</Badge>`
- `<Badge variant="danger">FROZEN</Badge>`
- `<Badge variant="neutral">CLOSED</Badge>`

### 3.3 Maker-Checker Authorization Modal

- Modal dialog for Branch Supervisors (Checkers) reviewing cash transactions exceeding the authorization limit ($> \text{IDR 50,000,000}$).
- Required Elements:
  1. Transaction Summary: Customer Name, Account Number, Cash Amount, Teller Name (Maker), Creation Timestamp.
  2. Authorization limit warning banner.
  3. Approval/rejection reasoning input field (_approval notes_).
  4. Dual action controls: **Reject Transaction** (Red) and **Approve Transaction** (Green).

### 3.4 General Ledger Journal Table

- Displays paired double-entry journal records.
- Columns: Timestamp, Transaction Reference ID, Account Number, Type (`DR` / `CR`), Amount, Balance After Mutation, and SHA-256 Hash Verification status (green lock indicator when chain verified).

---

## 4. User Experience (UX) Rules

1. **Double-Click Protection (Debounce & State Disable)**:
   Financial submission buttons must immediately disable upon initial click and display an inline loading indicator to prevent duplicate submissions from the browser.
2. **Sensitive Data Masking**:
   National identity numbers (KTP/NIK) or private credentials must be masked by default with an explicit toggle to view.
3. **Real-Time Currency Formatting**:
   Monetary inputs must automatically format thousands separators as the user types (e.g., typing `50000000` formats in real time to `Rp 50,000,000`).
4. **Auditable Toast Notifications**:
   Success and error feedback toasts must include the transaction `Reference ID` to streamline operational tracebacks.
