import React, { useState } from 'react';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Modal } from '@/components/ui/Modal';
import { formatRupiah } from '@/lib/formatters';

interface PendingTx {
  id: string;
  accountNumber: string;
  customerName: string;
  amount: number;
  makerName: string;
  createdAt: string;
  notes: string;
}

export const ApprovalQueuePage: React.FC = () => {
  const [pendingList, setPendingList] = useState<PendingTx[]>([
    {
      id: 'tx_9f81a7b2-3c10-4e56',
      accountNumber: '1001000012',
      customerName: 'PT Multi Graha Mandiri',
      amount: 75000000,
      makerName: 'teller_budi',
      createdAt: '2026-11-15 14:30:00',
      notes: 'Corporate over-the-counter cash withdrawal',
    },
  ]);

  const [selectedTx, setSelectedTx] = useState<PendingTx | null>(null);
  const [approvalNotes, setApprovalNotes] = useState('');
  const [actionFeedback, setActionFeedback] = useState<string | null>(null);

  const handleAction = (decision: 'APPROVE' | 'REJECT') => {
    if (!selectedTx) return;

    setActionFeedback(
      `Transaction ${selectedTx.id} ${decision === 'APPROVE' ? 'APPROVED' : 'REJECTED'}. Funds ${
        decision === 'APPROVE' ? 'debited and journal posted' : 'released back to available balance'
      }.`
    );

    setPendingList((prev) => prev.filter((item) => item.id !== selectedTx.id));
    setSelectedTx(null);
    setApprovalNotes('');
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900">Maker-Checker Authorization Queue</h1>
        <p className="text-xs text-slate-500">
          Enforcing Four-Eyes Principle for high-value branch operations (&gt; IDR 50,000,000)
        </p>
      </div>

      {actionFeedback && (
        <div className="p-4 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-lg text-sm font-semibold">
          {actionFeedback}
        </div>
      )}

      <Card>
        {pendingList.length === 0 ? (
          <div className="text-center py-8 text-slate-500 text-sm">
            Zero pending approval transactions. All branch operations reconciled.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 text-slate-600 uppercase border-b border-slate-200">
                <tr>
                  <th className="py-3 px-4">Tx ID</th>
                  <th className="py-3 px-4">Customer</th>
                  <th className="py-3 px-4">Account No</th>
                  <th className="py-3 px-4">Amount</th>
                  <th className="py-3 px-4">Maker (Teller)</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {pendingList.map((tx) => (
                  <tr key={tx.id} className="hover:bg-slate-50">
                    <td className="py-3 px-4 font-mono font-medium text-blue-600">{tx.id}</td>
                    <td className="py-3 px-4 font-medium text-slate-900">{tx.customerName}</td>
                    <td className="py-3 px-4 font-mono text-slate-600">{tx.accountNumber}</td>
                    <td className="py-3 px-4 font-mono font-bold text-slate-900">
                      {formatRupiah(tx.amount)}
                    </td>
                    <td className="py-3 px-4 font-mono text-slate-600">{tx.makerName}</td>
                    <td className="py-3 px-4">
                      <Badge variant="warning">PENDING APPROVAL</Badge>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <Button
                        variant="primary"
                        onClick={() => setSelectedTx(tx)}
                        className="text-xs px-3 py-1"
                      >
                        Review
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {/* Maker-Checker Authorization Modal */}
      <Modal
        isOpen={!!selectedTx}
        onClose={() => setSelectedTx(null)}
        title="Supervisor Dual Authorization"
      >
        {selectedTx && (
          <div className="space-y-4">
            <div className="p-3 bg-amber-50 border border-amber-200 rounded-md text-xs text-amber-800">
              <span className="font-bold">⚠️ Authorization Warning:</span> You are acting as the
              authorized Supervisor (Checker). Ensure physical customer identity verification has been
              performed.
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs bg-slate-50 p-3 rounded-lg border border-slate-200">
              <div>
                <span className="text-slate-500">Customer:</span>
                <p className="font-semibold text-slate-900 mt-0.5">{selectedTx.customerName}</p>
              </div>
              <div>
                <span className="text-slate-500">Account No:</span>
                <p className="font-mono font-semibold text-slate-900 mt-0.5">{selectedTx.accountNumber}</p>
              </div>
              <div>
                <span className="text-slate-500">Amount to Debit:</span>
                <p className="font-mono font-bold text-base text-rose-600 mt-0.5">
                  {formatRupiah(selectedTx.amount)}
                </p>
              </div>
              <div>
                <span className="text-slate-500">Initiating Maker:</span>
                <p className="font-mono font-semibold text-slate-900 mt-0.5">{selectedTx.makerName}</p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Checker Approval Notes
              </label>
              <textarea
                value={approvalNotes}
                onChange={(e) => setApprovalNotes(e.target.value)}
                placeholder="Physical ID and passbook verified against signature card..."
                rows={2}
                className="w-full px-3 py-2 border border-slate-300 rounded-md text-xs focus:ring-2 focus:ring-blue-500 focus:outline-none"
              />
            </div>

            <div className="flex gap-3 pt-2">
              <Button
                variant="danger"
                onClick={() => handleAction('REJECT')}
                className="flex-1"
              >
                Reject Transaction
              </Button>
              <Button
                variant="primary"
                onClick={() => handleAction('APPROVE')}
                className="flex-1 bg-emerald-600 hover:bg-emerald-700"
              >
                Approve & Execute Debit
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};
