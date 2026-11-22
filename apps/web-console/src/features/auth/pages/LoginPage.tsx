import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { useAuth } from '@/features/auth/context/AuthContext';
import { AxiosError } from 'axios';
import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const { login } = useAuth();

  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    setIsLoading(true);

    try {
      const user = await login(username, password);
      if (user.role === 'TELLER') {
        navigate('/branch/teller');
      } else if (user.role === 'SUPERVISOR') {
        navigate('/branch/supervisor');
      } else {
        navigate('/branch/audit');
      }
    } catch (err) {
      const axiosErr = err as AxiosError<{ error?: { message?: string } }>;
      setErrorMsg(
        axiosErr.response?.data?.error?.message ||
          'Authentication failed. Ensure API Gateway is running.'
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Card className="shadow-lg">
      <form onSubmit={handleSubmit} className="space-y-4">
        {errorMsg && (
          <div className="p-3 text-xs font-medium text-rose-700 bg-rose-50 rounded-md border border-rose-200">
            {errorMsg}
          </div>
        )}

        <div>
          <label className="block mb-1 text-xs font-semibold uppercase text-slate-600">
            Operator Username
          </label>
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="e.g. teller_budi"
            required
            className="px-3 py-2 w-full text-sm rounded-md border border-slate-300 focus:ring-2 focus:ring-blue-500 focus:outline-none"
          />
        </div>

        <div>
          <label className="block mb-1 text-xs font-semibold uppercase text-slate-600">
            Password
          </label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••••••"
            required
            className="px-3 py-2 w-full text-sm rounded-md border border-slate-300 focus:ring-2 focus:ring-blue-500 focus:outline-none"
          />
        </div>

        <Button type="submit" variant="primary" isLoading={isLoading} className="mt-2 w-full">
          Authenticate & Access Console
        </Button>

        <div className="mt-4 pt-3 border-t border-slate-100 text-[11px] text-slate-500 space-y-1">
          <div className="font-semibold text-slate-600">Demo Accounts (Password: <code className="font-mono">Password123!</code>):</div>
          <div className="font-mono">• teller_budi (TELLER / Maker)</div>
          <div className="font-mono">• spv_hendra (SUPERVISOR / Checker)</div>
          <div className="font-mono">• admin_jika (ADMIN)</div>
        </div>
      </form>
    </Card>
  );
};
