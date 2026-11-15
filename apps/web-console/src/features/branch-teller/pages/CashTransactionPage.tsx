import React, { useState } from 'react';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { formatRupiah } from '@/lib/formatters';

export const CashTransactionPage: React.FC = () => {
  const [accountNumber, setAccountNumber] = useState('');
  const [txType, setTxType] = useState<'CASH_DEPOSIT' | 'CASH_WITHDRAWAL'>('CASH_WITHDRAWAL');
  const [rawAmount, setRawAmount] = useState('75000000');
  const [notes, setNotes] = useState('Corporate cash withdrawal');
  const [statusMessage, setStatusMessage] = useState<{
    type: 'success' | 'warning' | 'error';
    text: string;
    refId?: string;
  } | null>(null);

  const parsedAmount = parseInt(rawAmount, 10) || 0;
  const isHighValue = parsedAmount > 50000000;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (isHighValue) {
      setStatusMessage({
        type: 'warning',
        text: `Transaction exceeds threshold limit (> ${formatRupiah(50000000)}). Funds locked to hold_balance. Status: PENDING_APPROVAL. Awaiting Supervisor authorization.`,
        refId: 'tx_' + Math.random().toString(36).substring(2, 10),
      });
    } else {
      setStatusMessage({
        type: 'success',
        text: `Transaction successfully executed via Straight-Through Processing (STP). Status: POSTED.`,
        refId: 'tx_' + Math.random().toString(36).substring(2, 10),
      });
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Over-the-Counter Cash Operations</h1>
        <p className="text-xs text-slate-500">Deposit and withdrawal counter service</p>
      </div>

      {statusMessage && (
        <div
          className={`p-4 rounded-lg border text-sm ${
            statusMessage.type === 'warning'
              ? 'bg-amber-50 border-amber-200 text-amber-800'
              : 'bg-emerald-50 border-emerald-200 text-emerald-800'
          }`}
        >
          <div className="font-semibold">{statusMessage.text}</div>
          {statusMessage.refId && (
            <div className="text-xs mt-1 font-mono">Reference ID: {statusMessage.refId}</div>
          )}
        </div>
      )}

      <Card>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold uppercase text-slate-600 mb-1">
              Transaction Operation
            </label>
            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setTxType('CASH_WITHDRAWAL')}
                className={`py-2 px-3 text-xs font-semibold rounded-md border text-center transition-colors ${
                  txType === 'CASH_WITHDRAWAL'
                    ? 'bg-slate-900 text-white border-slate-900'
                    : 'bg-white text-slate-700 border-slate-300 hover:bg-slate-50'
                }`}
              >
                Cash Withdrawal (Tarik Kas)
              </button>
              <button
                type="button"
                onClick={() => setTxType('CASH_DEPOSIT')}
                className={`py-2 px-3 text-xs font-semibold rounded-md border text-center transition-colors ${
                  txType === 'CASH_DEPOSIT'
                    ? 'bg-slate-900 text-white border-slate-900'
                    : 'bg-white text-slate-700 border-slate-300 hover:bg-slate-50'
                }`}
              >
                Cash Deposit (Setor Kas)
              </button>
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase text-slate-600 mb-1">
              Customer Account Number
            </label>
            <input
              type="text"
              value={accountNumber}
              onChange={(e) => setAccountNumber(e.target.value)}
              placeholder="e.g. 1001000012"
              className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm font-mono focus:ring-2 focus:ring-blue-500 focus:outline-none"
              required
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="block text-xs font-semibold uppercase text-slate-600">
                Amount (IDR Whole Integer)
              </label>
              {isHighValue && (
                <Badge variant="warning">Maker-Checker Triggered (&gt; 50M)</Badge>
              )}
            </div>
            <input
              type="number"
              value={rawAmount}
              onChange={(e) => setRawAmount(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm font-mono focus:ring-2 focus:ring-blue-500 focus:outline-none"
              required
            />
            <p className="text-xs text-slate-500 mt-1 font-mono">
              Formatted: {formatRupiah(parsedAmount)}
            </p>
          </div>

          <div>
            <label className="block text-xs font-semibold uppercase text-slate-600 mb-1">
              Operational Notes
            </label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={2}
              className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"
            />
          </div>

          <Button type="submit" variant="primary" className="w-full">
            Submit Transaction to Core Ledger
          </Button>
        </form>
      </Card>
    </div>
  );
};
