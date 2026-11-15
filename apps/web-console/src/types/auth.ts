export type UserRole = 'CUSTOMER' | 'TELLER' | 'SUPERVISOR' | 'ADMIN';

export interface AuthUser {
  userId: string;
  username: string;
  role: UserRole;
  branchId?: string;
}

export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
}
