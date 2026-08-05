import { createContext, useContext, useEffect, useRef, type ReactNode } from 'react';
import { useAuthStore } from '../stores/authStore';

interface AuthContextType {
  isAuthenticated: boolean;
  isLoading: boolean;
  user: any;
  error: string | null;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  exchangeNewApi: (relayName: string, dashboardToken: string) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const store = useAuthStore();
  const ssoProcessedRef = useRef(false);

  useEffect(() => {
    // New API single sign-on: the upstream dashboard opens the agent platform
    // as {entryPath}?relay=<relayName>&newapi_token=<dashboardToken>. When the
    // URL carries both params, exchange the dashboard token for a ClawManager
    // session instead of resuming any stale session left by a previous user.
    const params = new URLSearchParams(window.location.search);
    const relayName = params.get('relay');
    const dashboardToken = params.get('newapi_token');

    if (relayName && dashboardToken) {
      if (ssoProcessedRef.current) return;
      ssoProcessedRef.current = true;

      // Clear any existing session first so the previous user cannot win the
      // auto-login race while the exchange is in flight.
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
      store.setUser(null);
      store.setAuthenticated(false);

      // Drop the token from the address bar so it does not linger in history.
      const cleanUrl = window.location.pathname + window.location.hash;
      window.history.replaceState(null, '', cleanUrl);

      void store.exchangeNewApi(relayName, dashboardToken);
      return;
    }

    // Check if user is already logged in
    void store.fetchCurrentUser();
  }, []);

  const value: AuthContextType = {
    isAuthenticated: store.isAuthenticated,
    isLoading: store.isLoading,
    user: store.user,
    error: store.error,
    login: store.login,
    register: store.register,
    exchangeNewApi: store.exchangeNewApi,
    logout: store.logout,
    clearError: store.clearError,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
