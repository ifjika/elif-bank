import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
    <Card className="shadow-[0_0_40px_rgba(0,230,118,0.08)] border-emerald-500/25">
      <form onSubmit={handleSubmit} className="space-y-4">
        {errorMsg && (
          <div className="p-3 font-mono text-xs text-rose-400 rounded-md border bg-rose-500/10 border-rose-500/30">
            {errorMsg}
          </div>
        )}

        <div className="space-y-1.5">
          <Label htmlFor="username">Operator Username</Label>
          <Input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="e.g. teller_budi"
            className="font-mono"
            required
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="password">Password</Label>
          <Input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••••••"
            required
          />
        </div>

        <Button type="submit" variant="primary" isLoading={isLoading} className="mt-2 w-full font-semibold">
          Login
        </Button>

        <div className="mt-4 pt-3 border-t border-border/60 text-[11px] text-zinc-400 space-y-1">
          <div className="font-semibold text-emerald-400/90">
            Demo Accounts (Password: <code className="font-mono text-[#00E676]">Password123!</code>):
          </div>
          <div className="font-mono">• teller_budi (TELLER / Maker)</div>
          <div className="font-mono">• spv_hendra (SUPERVISOR / Checker)</div>
          <div className="font-mono">• admin_jika (ADMIN)</div>
        </div>
      </form>
    </Card>
  );
};
