export type TransactionType = 'TRANSFER' | 'CASH_DEPOSIT' | 'CASH_WITHDRAWAL' | 'FEE' | 'INTEREST';
export type TransactionStatus = 'POSTED' | 'PENDING_APPROVAL' | 'REJECTED';

export interface Transaction {
  id: string;
  referenceId: string;
  transactionType: TransactionType;
  amount: number;
  description: string;
  makerId: string;
  checkerId?: string;
  status: TransactionStatus;
  createdAt: string;
}

export interface LedgerEntry {
  id: number;
  transactionId: string;
  accountId: string;
  entryType: 'DR' | 'CR';
  amount: number;
  balanceAfter: number;
  previousHash: string;
  currentHash: string;
  createdAt: string;
}
