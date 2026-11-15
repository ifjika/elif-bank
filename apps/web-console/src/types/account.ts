export type AccountStatus = 'ACTIVE' | 'FROZEN' | 'DORMANT' | 'CLOSED';
export type AccountType = 'ASSET' | 'LIABILITY' | 'EQUITY' | 'REVENUE' | 'EXPENSE';

export interface Account {
  id: string;
  accountNumber: string;
  customerId?: string;
  accountType: AccountType;
  currency: string;
  balance: number;
  availableBalance: number;
  holdBalance: number;
  status: AccountStatus;
}
