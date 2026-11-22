import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { Modal } from '@/components/ui/Modal';
import { ApprovalQueueSkeleton } from '@/components/ui/PageSkeletons';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { apiClient } from '@/lib/apiClient';
import { formatRupiah } from '@/lib/formatters';
import { AxiosError } from 'axios';
import React, { useCallback, useEffect, useState } from 'react';

interface PendingTx {
  id: string;
  account_number: string;
  customer_name: string;
  transaction_type: string;
  amount: number;
  maker_name: string;
  notes: string;
  created_at: string;
}

interface ApprovalResult {
  transaction_id: string;
  status: string;
  checker_id: string;
  balance_after: number;
  hold_balance_after: number;
  approved_at: string;
}

export const ApprovalQueuePage: React.FC = () => {
  const [pendingList, setPendingList] = useState<PendingTx[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedTx, setSelectedTx] = useState<PendingTx | null>(null);
  const [approvalNotes, setApprovalNotes] = useState('');
  const [actionFeedback, setActionFeedback] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchPending = useCallback(async () => {
    setIsLoading(true);
    try {
      const res = await apiClient.get<{ success: boolean; data: PendingTx[] }>('/branch/approvals/pending');
      setPendingList(res.data.data ?? []);
    } catch (err) {
      const axiosErr = err as AxiosError<{ error?: { message?: string } }>;
      setActionFeedback({
        type: 'error',
        text: axiosErr.response?.data?.error?.message || 'Failed to load pending approval queue.',
      });
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchPending();
  }, [fetchPending]);

  const handleAction = async (decision: 'APPROVE' | 'REJECT') => {
    if (!selectedTx) return;

    setIsSubmitting(true);
    setActionFeedback(null);

    try {
      const res = await apiClient.post<{ success: boolean; data: ApprovalResult }>(
        `/branch/approvals/${selectedTx.id}`,
        {
          action: decision,
          notes: approvalNotes.trim(),
        }
      );

      const result = res.data.data;
      setActionFeedback({
        type: 'success',
        text: `Transaction ${result.transaction_id} ${result.status}. Ledger Balance: ${formatRupiah(
          result.balance_after
        )} | Hold: ${formatRupiah(result.hold_balance_after)}.`,
      });

      setPendingList((prev) => prev.filter((item) => item.id !== selectedTx.id));
      setSelectedTx(null);
      setApprovalNotes('');
    } catch (err) {
      const axiosErr = err as AxiosError<{ error?: { message?: string } }>;
      setActionFeedback({
        type: 'error',
        text:
          axiosErr.response?.data?.error?.message ||
          `Failed to ${decision.toLowerCase()} transaction ${selectedTx.id}.`,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) {
    return <ApprovalQueueSkeleton />;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Maker-Checker Authorization Queue</h1>
          <p className="text-xs font-mono text-zinc-400">
            Enforcing Four-Eyes Principle for high-value branch operations (&gt; IDR 50,000,000)
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void fetchPending()} className="font-mono text-xs">
          Refresh Queue
        </Button>
      </div>

      {actionFeedback && (
        <div
          className={`p-4 rounded-lg border text-xs font-mono font-semibold ${
            actionFeedback.type === 'error'
              ? 'bg-rose-500/10 border-rose-500/30 text-rose-300'
              : 'bg-emerald-500/10 border-emerald-500/30 text-[#00E676]'
          }`}
        >
          {actionFeedback.text}
        </div>
      )}

      <Card>
        {pendingList.length === 0 ? (
          <div className="text-center py-8 text-zinc-400 font-mono text-xs">
            Zero pending approval transactions. All branch operations reconciled.
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tx ID</TableHead>
                <TableHead>Customer</TableHead>
                <TableHead>Account No</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Maker (Teller)</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pendingList.map((tx) => (
                <TableRow key={tx.id}>
                  <TableCell className="font-mono font-medium text-[#00E676] max-w-[140px] truncate" title={tx.id}>
                    {tx.id}
                  </TableCell>
                  <TableCell className="font-medium text-white">{tx.customer_name}</TableCell>
                  <TableCell className="font-mono text-zinc-300">{tx.account_number}</TableCell>
                  <TableCell className="font-mono text-xs text-zinc-300">{tx.transaction_type}</TableCell>
                  <TableCell className="font-mono tabular-nums font-bold text-white">
                    {formatRupiah(tx.amount)}
                  </TableCell>
                  <TableCell className="font-mono text-zinc-300">{tx.maker_name}</TableCell>
                  <TableCell>
                    <Badge variant="warning">PENDING APPROVAL</Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => setSelectedTx(tx)}
                      className="text-xs font-semibold"
                    >
                      Review
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Maker-Checker Authorization Modal */}
      <Modal
        isOpen={!!selectedTx}
        onClose={() => !isSubmitting && setSelectedTx(null)}
        title="Supervisor Dual Authorization"
      >
        {selectedTx && (
          <div className="space-y-4">
            <div className="p-3 bg-amber-500/10 border border-amber-500/30 rounded-md text-xs text-amber-300">
              <span className="font-bold">⚠️ Authorization Warning:</span> You are acting as the
              authorized Supervisor (Checker). Ensure physical customer identity verification has been
              performed.
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs bg-emerald-950/30 p-3 rounded-lg border border-border">
              <div>
                <span className="text-zinc-400">Customer:</span>
                <p className="font-semibold text-white mt-0.5">{selectedTx.customer_name}</p>
              </div>
              <div>
                <span className="text-zinc-400">Account No:</span>
                <p className="font-mono font-semibold text-white mt-0.5">{selectedTx.account_number}</p>
              </div>
              <div>
                <span className="text-zinc-400">Operation ({selectedTx.transaction_type}):</span>
                <p className="font-mono tabular-nums font-bold text-base text-rose-400 mt-0.5">
                  {formatRupiah(selectedTx.amount)}
                </p>
              </div>
              <div>
                <span className="text-zinc-400">Initiating Maker:</span>
                <p className="font-mono font-semibold text-[#00E676] mt-0.5">{selectedTx.maker_name}</p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-emerald-400/90 font-mono uppercase mb-1">
                Checker Approval Notes
              </label>
              <textarea
                value={approvalNotes}
                onChange={(e) => setApprovalNotes(e.target.value)}
                placeholder="Physical ID and passbook verified against signature card..."
                rows={2}
                className="w-full px-3 py-2 border border-input bg-transparent rounded-md text-xs text-zinc-100 focus:ring-2 focus:ring-ring focus:outline-none"
              />
            </div>

            <div className="flex gap-3 pt-2">
              <Button
                variant="danger"
                isLoading={isSubmitting}
                onClick={() => void handleAction('REJECT')}
                className="flex-1"
              >
                Reject Transaction
              </Button>
              <Button
                variant="primary"
                isLoading={isSubmitting}
                onClick={() => void handleAction('APPROVE')}
                className="flex-1 font-semibold"
              >
                Approve &amp; Execute
              </Button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};
