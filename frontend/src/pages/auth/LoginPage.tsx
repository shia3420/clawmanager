import React, { useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../../contexts/AuthContext';
import { useI18n } from '../../contexts/I18nContext';
import LanguageSwitcher from '../../components/LanguageSwitcher';

const LoginPage: React.FC = () => {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [relayName, setRelayName] = useState('');
  const [email, setEmail] = useState('');
  const [ssoLoading, setSsoLoading] = useState(false);
  const { login, exchangeNewApi, isLoading, error, clearError } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();
  const location = useLocation();
  const formRef = useRef<HTMLFormElement | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    clearError();
    
    try {
      await login(username, password);
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from || '/dashboard', { replace: true });
    } catch (err) {
      // Error is handled by auth context
    }
  };

  const handleNewApiExchange = async (e: React.FormEvent) => {
    e.preventDefault();
    clearError();
    setSsoLoading(true);

    try {
      await exchangeNewApi(relayName, email);
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from || '/dashboard', { replace: true });
    } catch {
      // Error is handled by auth context
    } finally {
      setSsoLoading(false);
    }
  };

  const handlePasswordKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key !== 'Enter' || e.nativeEvent.isComposing) {
      return;
    }

    e.preventDefault();
    formRef.current?.requestSubmit();
  };

  const handleFormKeyDownCapture = (e: React.KeyboardEvent<HTMLFormElement>) => {
    if (e.key !== 'Enter' || e.nativeEvent.isComposing) {
      return;
    }

    const target = e.target as HTMLElement | null;
    if (!target) {
      return;
    }

    const tagName = target.tagName.toLowerCase();
    if (tagName === 'textarea') {
      return;
    }

    e.preventDefault();
    formRef.current?.requestSubmit();
  };

  return (
    <div className="app-shell flex min-h-screen items-center justify-center px-4">
      <div className="app-panel-warm relative w-full max-w-md space-y-8 p-8">
        <div className="flex justify-end">
          <LanguageSwitcher />
        </div>
        <div>
          <h2 className="text-center text-3xl font-bold text-gray-900">
            {t('auth.signInTitle')}
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600">
            {t('auth.subtitle')}
          </p>
        </div>
        
        {error && (
          <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-red-700">
            {error}
          </div>
        )}
        
        <form
          ref={formRef}
          className="mt-8 space-y-6"
          onSubmit={handleSubmit}
          onKeyDownCapture={handleFormKeyDownCapture}
        >
          <div className="space-y-4">
            <div>
              <label htmlFor="username" className="block text-sm font-medium text-gray-700">
                {t('auth.username')}
              </label>
              <input
                id="username"
                name="username"
                type="text"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                className="app-input mt-1 block w-full"
                placeholder={t('auth.usernamePlaceholder')}
              />
            </div>
            
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700">
                {t('auth.password')}
              </label>
              <input
                id="password"
                name="password"
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                onKeyDown={handlePasswordKeyDown}
                autoComplete="current-password"
                className="app-input mt-1 block w-full"
                placeholder={t('auth.passwordPlaceholder')}
              />
            </div>
          </div>

          <div>
            <button
              type="submit"
              disabled={isLoading}
              className="app-button-primary flex w-full disabled:cursor-not-allowed disabled:opacity-50"
            >
              {isLoading ? t('auth.signingIn') : t('auth.signIn')}
            </button>
          </div>
        </form>

        <div className="flex items-center gap-3 pt-2">
          <div className="h-px flex-1 bg-gray-200" />
          <span className="text-xs font-medium uppercase tracking-[0.18em] text-gray-400">
            {t('newapi.or')}
          </span>
          <div className="h-px flex-1 bg-gray-200" />
        </div>

        <form className="space-y-6" onSubmit={handleNewApiExchange}>
          <div className="space-y-4">
            <div>
              <label htmlFor="relayName" className="block text-sm font-medium text-gray-700">
                {t('newapi.relayName')}
              </label>
              <input
                id="relayName"
                name="relayName"
                type="text"
                required
                value={relayName}
                onChange={(e) => setRelayName(e.target.value)}
                className="app-input mt-1 block w-full"
                placeholder={t('newapi.relayNamePlaceholder')}
              />
            </div>

            <div>
              <label htmlFor="ssoEmail" className="block text-sm font-medium text-gray-700">
                {t('newapi.email')}
              </label>
              <input
                id="ssoEmail"
                name="ssoEmail"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="app-input mt-1 block w-full"
                placeholder={t('newapi.emailPlaceholder')}
              />
            </div>
          </div>

          <div>
            <button
              type="submit"
              disabled={ssoLoading}
              className="app-button-secondary flex w-full disabled:cursor-not-allowed disabled:opacity-50"
            >
              {ssoLoading ? t('newapi.exchanging') : t('newapi.exchange')}
            </button>
            <p className="mt-3 text-center text-xs leading-5 text-gray-500">
              {t('newapi.exchangeHint')}
            </p>
          </div>
        </form>
      </div>
    </div>
  );
};

export default LoginPage;
