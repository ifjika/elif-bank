export type UserRole = 'CUSTOMER' | 'TELLER' | 'SUPERVISOR' | 'ADMIN';

export interface AuthUser {
  user_id: string;
  username: string;
  full_name: string;
  role: UserRole;
  branch_id?: string;
}

export interface AuthResponseData {
  access_token: string;
  user: AuthUser;
}
