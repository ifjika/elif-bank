import React from 'react';
import { Card } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { formatRupiah } from '@/lib/formatters';
import { Link } from 'react-router-dom';

export const TellerDashboardPage: React.FC = () => {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Teller Terminal</h1>
          <p className="text-xs text-slate-500">Branch Cash Counter & Direct Operations</p>
        </div>
        <Link
          to="/branch/teller/cash"
          className="bg-brand-accent text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          + New Cash Transaction
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <Card title="Branch Vault Available" subtitle="Physical cash reserve">
          <div className="text-2xl font-bold font-mono text-slate-900 mt-2">
            {formatRupiah(1250000000)}
          </div>
          <p className="text-xs text-slate-400 mt-1">Updated in real-time</p>
        </Card>

        <Card title="Maker-Checker Limit" subtitle="Authorization threshold">
          <div className="text-2xl font-bold font-mono text-amber-600 mt-2">
            {formatRupiah(50000000)}
          </div>
          <p className="text-xs text-slate-400 mt-1">&gt; 50M requires Supervisor approval</p>
        </Card>

        <Card title="Today's Handled Volume" subtitle="Branch transactions">
          <div className="text-2xl font-bold font-mono text-slate-900 mt-2">42 Txs</div>
          <Badge variant="success" className="mt-2">
            Normal Activity
          </Badge>
        </Card>
      </div>
    </div>
  );
};
