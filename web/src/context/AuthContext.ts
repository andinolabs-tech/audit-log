import { createContext, useContext } from 'react';

export interface AuthContextValue {
  token: string | null;
}

export const AuthContext = createContext<AuthContextValue>({ token: null });

export function useAuth(): AuthContextValue {
  return useContext(AuthContext);
}
