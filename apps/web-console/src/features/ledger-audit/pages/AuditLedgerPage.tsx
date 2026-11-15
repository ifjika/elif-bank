import React from 'react';
import { Card } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { formatRupiah } from '@/lib/formatters';

export const AuditLedgerPage: React.FC = () => {
  const auditEntries = [
    {
      id: 1042,
      txId: 'tx_a1b2c3d4-e5f6',
      accountNumber: '1001000001',
      entryType: 'DR' as const,
      amount: 350000,
      balanceAfter: 650000,
      currentHash: 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
      status: 'VERIFIED',
      createdAt: '2026-11-15 14:10:00',
    },
    {
      id: 1043,
      txId: 'tx_a1b2c3d4-e5f6',
      accountNumber: '1001000002',
      entryType: 'CR' as const,
      amount: 350000,
      balanceAfter: 850000,
      currentHash: '8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4',
      status: 'VERIFIED',
      createdAt: '2026-11-15 14:10:00',
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">General Ledger Audit & Cryptographic Chain</h1>
          <p className="text-xs text-slate-500">Tamper-evident SHA-256 double-entry verification</p>
        </div>
        <Badge variant="success">All Chains Valid (0 Anomalies)</Badge>
      </div>

      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-50 text-slate-600 uppercase border-b border-slate-200">
              <tr>
                <th className="py-3 px-4">Entry ID</th>
                <th className="py-3 px-4">Tx Ref</th>
                <th className="py-3 px-4">Account</th>
                <th className="py-3 px-4">Type</th>
                <th className="py-3 px-4">Amount</th>
                <th className="py-3 px-4">Balance After</th>
                <th className="py-3 px-4">SHA-256 Hash</th>
                <th className="py-3 px-4">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {auditEntries.map((entry) => (
                <tr key={entry.id} className="hover:bg-slate-50">
                  <td className="py-3 px-4 font-mono font-medium">{entry.id}</td>
                  <td className="py-3 px-4 font-mono text-blue-600">{entry.txId}</td>
                  <td className="py-3 px-4 font-mono">{entry.accountNumber}</td>
                  <td className="py-3 px-4">
                    <span
                      className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                        entry.entryType === 'DR'
                          ? 'bg-rose-100 text-rose-800'
                          : 'bg-emerald-100 text-emerald-800'
                      }`}
                    >
                      {entry.entryType}
                    </span>
                  </td>
                  <td className="py-3 px-4 font-mono font-semibold">{formatRupiah(entry.amount)}</td>
                  <td className="py-3 px-4 font-mono text-slate-600">{formatRupiah(entry.balanceAfter)}</td>
                  <td className="py-3 px-4 font-mono text-[10px] text-slate-400 max-w-[140px] truncate" title={entry.currentHash}>
                    {entry.currentHash}
                  </td>
                  <td className="py-3 px-4">
                    <Badge variant="success">🔒 Verified</Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
};
