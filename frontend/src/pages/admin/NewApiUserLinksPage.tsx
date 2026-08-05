import React, { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { KeyRound, RefreshCw, ShieldOff, UserRound } from 'lucide-react';
import AdminLayout from '../../components/AdminLayout';
import { useI18n } from '../../contexts/I18nContext';
import { newapiService, type NewApiIdentityLink } from '../../services/newapiService';

function getErrorMessage(err: unknown, fallback: string) {
  const responseError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
  if (responseError) {
    return responseError;
  }
  return err instanceof Error ? err.message : fallback;
}

const NewApiUserLinksPage: React.FC = () => {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [actionError, setActionError] = useState<string | null>(null);
  const [unlinkingId, setUnlinkingId] = useState<number | null>(null);

  const linksQuery = useQuery({
    queryKey: ['newapi-identity-links'],
    queryFn: () => newapiService.listIdentityLinks(),
  });

  const links = linksQuery.data ?? [];

  const handleUnlink = async (link: NewApiIdentityLink) => {
    if (!window.confirm(t('newapi.unlinkConfirm'))) {
      return;
    }
    setUnlinkingId(link.id);
    setActionError(null);
    try {
      await newapiService.unlinkIdentityLink(link.id);
      await queryClient.invalidateQueries({ queryKey: ['newapi-identity-links'] });
    } catch (err: unknown) {
      setActionError(getErrorMessage(err, t('newapi.unlinkFailed')));
    } finally {
      setUnlinkingId(null);
    }
  };

  const formatDate = (value?: string) =>
    value ? new Date(value).toLocaleString() : t('newapi.never');

  return (
    <AdminLayout title={t('newapi.userLinksAdminTitle')}>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[#b46c50]">
            {t('newapi.userLinksEyebrow')}
          </p>
          <h2 className="mt-2 text-xl font-semibold text-[#1d1713]">{t('newapi.userLinksDesc')}</h2>
        </div>
        <button
          type="button"
          onClick={() => void queryClient.invalidateQueries({ queryKey: ['newapi-identity-links'] })}
          className="app-button-secondary"
        >
          <RefreshCw className="h-4 w-4" />
          {t('common.refresh')}
        </button>
      </div>

      <div className="mb-6 rounded-2xl border border-[#ead8cf] bg-[#fffaf6] px-4 py-3 text-sm text-[#7a6d66]">
        {t('newapi.linksHint')}
      </div>

      {actionError && (
        <div className="mb-6 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-red-700">
          {actionError}
        </div>
      )}

      <section className="overflow-hidden rounded-[28px] border border-[#ead8cf] bg-[linear-gradient(180deg,rgba(255,255,255,0.9)_0%,rgba(255,250,246,0.96)_100%)] shadow-[0_30px_80px_-52px_rgba(60,42,28,0.6)]">
        <div className="overflow-x-auto">
          <table className="min-w-full">
            <thead>
              <tr className="border-b border-[#f0e3db] bg-[linear-gradient(180deg,#fffaf6_0%,#fff6f1_100%)] text-left text-[11px] font-semibold uppercase tracking-[0.18em] text-[#9d7d6e]">
                <th className="px-6 py-4">{t('newapi.usernameLabel')}</th>
                <th className="px-6 py-4">{t('newapi.relayLabel')}</th>
                <th className="px-6 py-4">{t('newapi.tokenName')}</th>
                <th className="px-6 py-4">{t('newapi.credential')}</th>
                <th className="px-6 py-4">{t('newapi.todayUsed')}</th>
                <th className="px-6 py-4">{t('newapi.lastUsed')}</th>
                <th className="px-6 py-4 text-right">{t('newapi.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {links.map((link) => (
                <tr key={link.id} className="border-b border-[#f2e8e2] text-sm transition-colors hover:bg-[#fffaf6]">
                  <td className="px-6 py-5">
                    <div className="flex items-center gap-3">
                      <div className="rounded-xl border border-[#f1e2d9] bg-white p-2 shadow-[0_12px_30px_-24px_rgba(72,44,24,0.5)]">
                        <UserRound className="h-4 w-4 text-[#ef6b4a]" />
                      </div>
                      <div>
                        <p className="font-semibold text-[#1d1713]">
                          {link.username || `#${link.user_id}`}
                          <span className="ml-2 rounded-full bg-[#f1e2d9] px-2 py-0.5 text-[10px] font-medium text-[#a05f46]">
                            {link.role || 'user'}
                          </span>
                        </p>
                        <p className="mt-0.5 text-xs text-[#b09d93]">{link.email || '--'}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-5">
                    <p className="font-medium text-[#1d1713]">{link.relay_name || '--'}</p>
                    <p className="mt-0.5 font-mono text-xs text-[#b09d93]">{link.relay_base_url || '--'}</p>
                    {link.external_id && (
                      <p className="mt-0.5 text-xs text-[#b09d93]">
                        {link.upstream_user_id ? `${link.external_id} · ${link.upstream_user_id}` : link.external_id}
                      </p>
                    )}
                  </td>
                  <td className="px-6 py-5">
                    <span className="inline-flex items-center gap-1.5 rounded-full bg-[#f1e2d9] px-2.5 py-1 font-mono text-xs text-[#8a4f38]">
                      <KeyRound className="h-3 w-3" />
                      {link.token_name || '--'}
                    </span>
                  </td>
                  <td className="px-6 py-5">
                    <span
                      className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium ${
                        link.has_credential ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'
                      }`}
                    >
                      {link.has_credential ? t('newapi.credentialYes') : t('newapi.credentialNo')}
                    </span>
                  </td>
                  <td className="px-6 py-5">
                    <p className="font-semibold tabular-nums text-[#1d1713]">{link.today_used}</p>
                    <p className="text-xs text-[#b09d93]">
                      {t('newapi.todayLimit')}: {link.today_limit > 0 ? link.today_limit : '∞'}
                    </p>
                  </td>
                  <td className="px-6 py-5 text-[#7a6d66]">{formatDate(link.last_used_at)}</td>
                  <td className="px-6 py-5 text-right">
                    <button
                      type="button"
                      disabled={unlinkingId === link.id}
                      onClick={() => void handleUnlink(link)}
                      className="rounded-full border border-[#ead7cd] bg-white p-2 text-[#a05f46] transition-all hover:border-red-300 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
                      title={t('newapi.unlink')}
                    >
                      <ShieldOff className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {!linksQuery.isLoading && links.length === 0 && (
          <div className="px-6 py-16 text-center text-sm text-[#8f8681]">{t('newapi.noLinks')}</div>
        )}

        {linksQuery.isLoading && (
          <div className="px-6 py-16 text-center text-sm text-[#8f8681]">{t('common.loading')}</div>
        )}
      </section>
    </AdminLayout>
  );
};

export default NewApiUserLinksPage;
