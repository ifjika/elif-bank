import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { UserRole } from '@/types/auth';

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [role, setRole] = useState<UserRole>('TELLER');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    // Mock authentication for development scaffolding
    setTimeout(() => {
      localStorage.setItem('access_token', 'mock_jwt_token_sample');
      localStorage.setItem(
        'user_info',
        JSON.stringify({
          userId: 'usr_sample_01',
          username: username || (role === 'TELLER' ? 'teller_jkt01' : 'spv_jkt01'),
          role: role,
          branchId: 'branch-001',
        })
      );
      setIsLoading(false);

      if (role === 'TELLER') {
        navigate('/branch/teller');
      } else if (role === 'SUPERVISOR') {
        navigate('/branch/supervisor');
      } else {
        navigate('/branch/audit');
      }
    }, 400);
  };

  return (
    <Card className="shadow-lg">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-xs font-semibold uppercase text-slate-600 mb-1">
            Operator Username
          </label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="e.g. teller_jkt01"
            className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"
          />
        </div>

        <div>
          <label className="block text-xs font-semibold uppercase text-slate-600 mb-1">
            Role Assignment (RBAC)
          </label>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value as UserRole)}
            className="w-full px-3 py-2 border border-slate-300 rounded-md text-sm focus:ring-2 focus:ring-blue-500 focus:outline-none"
          >
            <option value="TELLER">Teller (Maker)</option>
            <option value="SUPERVISOR">Supervisor (Checker)</option>
            <option value="ADMIN">System Admin</option>
          </select>
        </div>

        <Button type="submit" variant="primary" isLoading={isLoading} className="w-full mt-2">
          Authenticate & Access Console
        </Button>
      </form>
    </Card>
  );
};
