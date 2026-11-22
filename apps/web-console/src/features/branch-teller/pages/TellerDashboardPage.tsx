import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowUpRight, ArrowDownLeft, ShieldAlert, Vault } from 'lucide-react';
import {
  ShadcnCard,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
} from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { formatRupiah } from '@/lib/formatters';

export const TellerDashboardPage: React.FC = () => {
  const recentCounterTxs = [
    {
      ref: 'tx_8821a4c1',
      account: '1001000012',
      type: 'CASH_DEPOSIT',
      amount: 15000000,
      status: 'POSTED' as const,
      time: '14:22:10',
    },
    {
      ref: 'tx_9f81a7b2',
      account: '1001000045',
      type: 'CASH_WITHDRAWAL',
      amount: 75000000,
      status: 'PENDING_APPROVAL' as const,
      time: '14:30:00',
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-white tracking-tight">Teller Terminal</h1>
          <p className="text-xs font-mono text-zinc-400">
            Branch Cash Counter &amp; Straight-Through Operations
          </p>
        </div>
        <Button asChild variant="primary" className="font-semibold">
          <Link to="/branch/teller/cash">+ New Cash Transaction</Link>
        </Button>
      </div>

      {/* shadcn/ui KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <ShadcnCard className="border-emerald-500/20">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-xs font-mono uppercase text-zinc-400">
              Branch Vault Available
            </CardTitle>
            <Vault className="w-4 h-4 text-[#00E676]" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono tabular-nums text-[#00E676]">
              {formatRupiah(1250000000)}
            </div>
            <CardDescription className="mt-1 font-mono">Physical cash reserve (IDR)</CardDescription>
          </CardContent>
        </ShadcnCard>

        <ShadcnCard className="border-amber-500/20">
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-xs font-mono uppercase text-zinc-400">
              Maker-Checker Limit
            </CardTitle>
            <ShieldAlert className="w-4 h-4 text-amber-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono tabular-nums text-amber-400">
              {formatRupiah(50000000)}
            </div>
            <CardDescription className="mt-1 font-mono">
              &gt; 50M triggers Supervisor approval
            </CardDescription>
          </CardContent>
        </ShadcnCard>

        <ShadcnCard>
          <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
            <CardTitle className="text-xs font-mono uppercase text-zinc-400">
              Today&apos;s Counter Volume
            </CardTitle>
            <ArrowUpRight className="w-4 h-4 text-emerald-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono tabular-nums text-white">42 Txs</div>
            <Badge variant="success" className="mt-2">
              Normal Activity
            </Badge>
          </CardContent>
        </ShadcnCard>
      </div>

      {/* Recent Counter Activity Table */}
      <ShadcnCard>
        <CardHeader className="border-b border-border/60">
          <CardTitle className="text-base text-white">Recent Counter Mutations</CardTitle>
          <CardDescription className="font-mono">
            Live branch teller session activity
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Reference ID</TableHead>
                <TableHead>Account</TableHead>
                <TableHead>Operation</TableHead>
                <TableHead>Amount</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Time</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {recentCounterTxs.map((tx) => (
                <TableRow key={tx.ref}>
                  <TableCell className="font-mono text-[#00E676]">{tx.ref}</TableCell>
                  <TableCell className="font-mono text-zinc-300">{tx.account}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1.5 font-mono text-xs">
                      {tx.type === 'CASH_DEPOSIT' ? (
                        <ArrowDownLeft className="w-3.5 h-3.5 text-[#00E676]" />
                      ) : (
                        <ArrowUpRight className="w-3.5 h-3.5 text-rose-400" />
                      )}
                      {tx.type}
                    </span>
                  </TableCell>
                  <TableCell className="font-mono tabular-nums font-semibold text-white">
                    {formatRupiah(tx.amount)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={tx.status === 'POSTED' ? 'success' : 'warning'}>
                      {tx.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right font-mono text-zinc-400">{tx.time}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </ShadcnCard>
    </div>
  );
};
