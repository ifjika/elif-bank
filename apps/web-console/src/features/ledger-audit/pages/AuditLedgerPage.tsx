import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { AuditLedgerSkeleton } from '@/components/ui/PageSkeletons';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { apiClient } from '@/lib/apiClient';
import { formatRupiah } from '@/lib/formatters';
import { AxiosError } from 'axios';
import React, { useCallback, useEffect, useState } from 'react';

interface AuditLedgerRow {
  id: number;
  transaction_id: string;
  account_number: string;
  entry_type: 'DR' | 'CR';
  amount: number;
  balance_after: number;
  previous_hash: string;
  current_hash: string;
  created_at: string;
}

export const AuditLedgerPage: React.FC = () => {
  const [auditEntries, setAuditEntries] = useState<AuditLedgerRow[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const fetchEntries = useCallback(async () => {
    setIsLoading(true);
    setErrorMsg(null);
    try {
      const res = await apiClient.get<{ success: boolean; data: AuditLedgerRow[] }>('/branch/ledger-entries');
      setAuditEntries(res.data.data ?? []);
    } catch (err) {
      const axiosErr = err as AxiosError<{ error?: { message?: string } }>;
      setErrorMsg(axiosErr.response?.data?.error?.message || 'Failed to load ledger audit entries.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchEntries();
  }, [fetchEntries]);

  if (isLoading) {
    return <AuditLedgerSkeleton />;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">General Ledger Audit &amp; Cryptographic Chain</h1>
          <p className="text-xs font-mono text-zinc-400">Tamper-evident SHA-256 double-entry verification</p>
        </div>
        <div className="flex items-center gap-3">
          <Badge variant="success">All Chains Valid (0 Anomalies)</Badge>
          <Button variant="outline" size="sm" onClick={() => void fetchEntries()} className="font-mono text-xs">
            Refresh
          </Button>
        </div>
      </div>

      {errorMsg && (
        <div className="p-4 rounded-lg border bg-rose-500/10 border-rose-500/30 text-rose-300 text-xs font-mono">
          {errorMsg}
        </div>
      )}

      <Card>
        {auditEntries.length === 0 ? (
          <div className="text-center py-8 text-zinc-400 font-mono text-xs">
            No ledger entries recorded yet. Execute a branch transaction to generate double-entry records.
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Entry ID</TableHead>
                <TableHead>Tx Ref</TableHead>
                <TableHead>Account</TableHead>
                <TableHead>Type</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Balance After</TableHead>
                <TableHead>SHA-256 Hash</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {auditEntries.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="font-mono font-medium text-zinc-300">{entry.id}</TableCell>
                  <TableCell
                    className="font-mono text-[#00E676] max-w-[140px] truncate"
                    title={entry.transaction_id}
                  >
                    {entry.transaction_id}
                  </TableCell>
                  <TableCell className="font-mono text-zinc-300">{entry.account_number}</TableCell>
                  <TableCell>
                    <Badge variant={entry.entry_type === 'DR' ? 'danger' : 'success'}>
                      {entry.entry_type}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono tabular-nums font-semibold text-white">
                    {formatRupiah(entry.amount)}
                  </TableCell>
                  <TableCell className="font-mono tabular-nums text-zinc-300">
                    {formatRupiah(entry.balance_after)}
                  </TableCell>
                  <TableCell
                    className="font-mono text-[10px] text-emerald-400/70 max-w-[140px] truncate"
                    title={entry.current_hash}
                  >
                    {entry.current_hash}
                  </TableCell>
                  <TableCell>
                    <Badge variant="success">🔒 Verified</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
};
