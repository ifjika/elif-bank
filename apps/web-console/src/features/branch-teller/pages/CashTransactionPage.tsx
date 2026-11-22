import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { CurrencyInput } from '@/components/ui/currency-input';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { apiClient } from '@/lib/apiClient';
import { formatRupiah } from '@/lib/formatters';
import { AxiosError } from 'axios';
import React, { useState } from 'react';

interface CashTxResult {
  transaction_id: string;
  reference_id: string;
  transaction_type: string;
  amount: number;
  balance_after: number;
  hold_balance_after: number;
  status: string;
  created_at: string;
}

interface StatusBanner {
  type: 'success' | 'warning' | 'error';
  text: string;
  refId?: string;
  balanceAfter?: number;
  holdBalanceAfter?: number;
}

export const CashTransactionPage: React.FC = () => {
  const [accountNumber, setAccountNumber] = useState('1001000012');
  const [txType, setTxType] = useState<'CASH_DEPOSIT' | 'CASH_WITHDRAWAL'>('CASH_WITHDRAWAL');
  const [amount, setAmount] = useState<number>(75000000);
  const [notes, setNotes] = useState('Corporate cash withdrawal');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [statusMessage, setStatusMessage] = useState<StatusBanner | null>(null);

  const isHighValue = amount > 50000000;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (amount <= 0 || !accountNumber.trim()) return;

    setIsSubmitting(true);
    setStatusMessage(null);

    try {
      const idempKey = typeof crypto !== 'undefined' && crypto.randomUUID
        ? crypto.randomUUID()
        : `branch-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

      const res = await apiClient.post<{ success: boolean; data: CashTxResult }>(
        '/branch/cash-transactions',
        {
          account_number: accountNumber.trim(),
          transaction_type: txType,
          amount,
          notes: notes.trim(),
        },
        {
          headers: { 'X-Idempotency-Key': idempKey },
        }
      );

      const data = res.data.data;
      if (data.status === 'PENDING_APPROVAL') {
        setStatusMessage({
          type: 'warning',
          text: `Transaction exceeds threshold limit (> ${formatRupiah(50000000)}). Status: PENDING_APPROVAL. Awaiting Supervisor authorization.`,
          refId: data.transaction_id,
          balanceAfter: data.balance_after,
          holdBalanceAfter: data.hold_balance_after,
        });
      } else {
        setStatusMessage({
          type: 'success',
          text: `Transaction executed via Straight-Through Processing (STP). Status: ${data.status}.`,
          refId: data.transaction_id,
          balanceAfter: data.balance_after,
          holdBalanceAfter: data.hold_balance_after,
        });
      }
    } catch (err) {
      const axiosErr = err as AxiosError<{ error?: { message?: string } }>;
      const msg =
        axiosErr.response?.data?.error?.message ||
        'Failed to submit transaction to Core Ledger. Please verify account number and balance.';
      setStatusMessage({
        type: 'error',
        text: msg,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-white tracking-tight">Over-the-Counter Cash Operations</h1>
        <p className="text-xs font-mono text-zinc-400">Deposit and withdrawal counter service</p>
      </div>

      {statusMessage && (
        <div
          className={`p-4 rounded-lg border text-xs font-mono ${
            statusMessage.type === 'error'
              ? 'bg-rose-500/10 border-rose-500/30 text-rose-300'
              : statusMessage.type === 'warning'
              ? 'bg-amber-500/10 border-amber-500/30 text-amber-300'
              : 'bg-emerald-500/10 border-emerald-500/30 text-[#00E676]'
          }`}
        >
          <div className="font-semibold">{statusMessage.text}</div>
          {statusMessage.refId && (
            <div className="mt-1.5 opacity-90 space-y-0.5">
              <div>Transaction ID: {statusMessage.refId}</div>
              {statusMessage.balanceAfter !== undefined && (
                <div>
                  Ledger Balance: {formatRupiah(statusMessage.balanceAfter)} • Hold Balance:{' '}
                  {formatRupiah(statusMessage.holdBalanceAfter ?? 0)}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <Card>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label>Transaction Operation</Label>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setTxType('CASH_WITHDRAWAL')}
                className={`py-2 px-3 text-xs font-mono font-semibold rounded-md border text-center transition-colors ${
                  txType === 'CASH_WITHDRAWAL'
                    ? 'bg-[#00E676] text-[#050807] border-[#00E676]'
                    : 'bg-transparent text-zinc-300 border-border hover:bg-emerald-950/40'
                }`}
              >
                Cash Withdrawal (Tarik Kas)
              </button>
              <button
                type="button"
                onClick={() => setTxType('CASH_DEPOSIT')}
                className={`py-2 px-3 text-xs font-mono font-semibold rounded-md border text-center transition-colors ${
                  txType === 'CASH_DEPOSIT'
                    ? 'bg-[#00E676] text-[#050807] border-[#00E676]'
                    : 'bg-transparent text-zinc-300 border-border hover:bg-emerald-950/40'
                }`}
              >
                Cash Deposit (Setor Kas)
              </button>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="accountNumber">Customer Account Number</Label>
            <Input
              id="accountNumber"
              type="text"
              value={accountNumber}
              onChange={(e) => setAccountNumber(e.target.value)}
              placeholder="e.g. 1001000012"
              className="font-mono"
              required
            />
            <p className="text-[11px] font-mono text-zinc-400">
              Demo accounts: <span className="text-[#00E676]">1001000012</span> (Rp 100M) •{' '}
              <span className="text-[#00E676]">1001000045</span> (Rp 50M)
            </p>
          </div>

          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <Label htmlFor="amount">Amount (IDR)</Label>
              {isHighValue && (
                <Badge variant="warning">Maker-Checker Triggered (&gt; 50M)</Badge>
              )}
            </div>
            <CurrencyInput
              id="amount"
              value={amount}
              onValueChange={setAmount}
              required
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="notes">Operational Notes</Label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
              className="w-full px-3 py-2 border border-input bg-transparent rounded-md text-sm text-zinc-100 focus:ring-2 focus:ring-ring focus:outline-none"
            />
          </div>

          <Button
            type="submit"
            variant="primary"
            isLoading={isSubmitting}
            disabled={amount <= 0 || !accountNumber.trim()}
            className="w-full font-semibold"
          >
            Submit Transaction to Core Ledger
          </Button>
        </form>
      </Card>
    </div>
  );
};
