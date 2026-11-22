import React, { createContext, useContext, useEffect, useState } from 'react';
import { apiClient } from '@/lib/apiClient';
import { setAccessToken } from '@/lib/tokenStore';
import { AuthResponseData, AuthUser } from '@/types/auth';

interface AuthContextValue {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isInitializing: boolean;
  login: (username: string, password: string) => Promise<AuthUser>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isInitializing, setIsInitializing] = useState(true);

  // Silent refresh on app mount using HttpOnly refresh_token cookie
  useEffect(() => {
    let isMounted = true;

    const restoreSession = async () => {
      try {
        const res = await apiClient.post<{ success: boolean; data: AuthResponseData }>('/auth/refresh');
        if (isMounted && res.data?.success && res.data.data) {
          setAccessToken(res.data.data.access_token);
          setUser(res.data.data.user);
        }
      } catch {
        if (isMounted) {
          setAccessToken(null);
          setUser(null);
        }
      } finally {
        if (isMounted) {
          setIsInitializing(false);
        }
      }
    };

    restoreSession();
    return () => {
      isMounted = false;
    };
  }, []);

  const login = async (username: string, password: string): Promise<AuthUser> => {
    const res = await apiClient.post<{ success: boolean; data: AuthResponseData }>('/auth/login', {
      username,
      password,
    });
    const { access_token, user: loggedInUser } = res.data.data;
    setAccessToken(access_token);
    setUser(loggedInUser);
    return loggedInUser;
  };

  const logout = async (): Promise<void> => {
    try {
      await apiClient.post('/auth/logout');
    } finally {
      setAccessToken(null);
      setUser(null);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isInitializing,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
};

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return ctx;
}
