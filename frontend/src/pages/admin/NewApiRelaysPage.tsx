import React, { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus, RefreshCw, Server, Trash2 } from 'lucide-react';
import AdminLayout from '../../components/AdminLayout';
import { useI18n } from '../../contexts/I18nContext';
import { newapiService, type NewApiRelay } from '../../services/newapiService';

function getErrorMessage(err: unknown, fallback: string) {
  const responseError = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
  if (responseError) {
    return responseError;
  }
  return err instanceof Error ? err.message : fallback;
}

const NewApiRelaysPage: React.FC = () => {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: '', base_url: '', relay_token: '', daily_limit: 100 });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const relaysQuery = useQuery({
    queryKey: ['newapi-relays'],
    queryFn: () => newapiService.listRelays(),
  });

  const relays = relaysQuery.data ?? [];

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      await newapiService.createRelay({
        name: form.name.trim(),
        base_url: form.base_url.trim(),
        relay_token: form.relay_token.trim(),
        daily_limit: Math.max(0, form.daily_limit),
      });
      setShowCreate(false);
      setForm({ name: '', base_url: '', relay_token: '', daily_limit: 100 });
      await queryClient.invalidateQueries({ queryKey: ['newapi-relays'] });
    } catch (err: unknown) {
      setFormError(getErrorMessage(err, t('newapi.createFailed')));
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (relay: NewApiRelay) => {
    if (!window.confirm(t('newapi.deleteConfirm', { name: relay.name }))) {
      return;
    }
    setDeletingId(relay.id);
    setActionError(null);
    try {
      await newapiService.deleteRelay(relay.id);
      await queryClient.invalidateQueries({ queryKey: ['newapi-relays'] });
    } catch (err: unknown) {
      setActionError(getErrorMessage(err, t('newapi.deleteFailed')));
    } finally {
      setDeletingId(null);
    }
  };

  const formatDate = (value: string) =>
    value ? new Date(value).toLocaleString() : '--';

  return (
    <AdminLayout title={t('newapi.adminTitle')}>
      {actionError && (
        <div className="mb-6 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-red-700">
          {actionError}
        </div>
      )}

      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[#b46c50]">
            {t('newapi.adminEyebrow')}
          </p>
          <h2 className="mt-2 text-xl font-semibold text-[#1d1713]">{t('newapi.adminDesc')}</h2>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => void queryClient.invalidateQueries({ queryKey: ['newapi-relays'] })}
            className="app-button-secondary"
          >
            <RefreshCw className="h-4 w-4" />
            {t('common.refresh')}
          </button>
          <button type="button" onClick={() => setShowCreate((v) => !v)} className="app-button-primary">
            <Plus className="h-4 w-4" />
            {t('newapi.createRelay')}
          </button>
        </div>
      </div>

      {showCreate && (
        <form
          onSubmit={handleCreate}
          className="mb-6 rounded-[24px] border border-[#ead8cf] bg-[linear-gradient(180deg,rgba(255,255,255,0.96)_0%,rgba(255,250,246,0.96)_100%)] p-6 shadow-[0_24px_60px_-46px_rgba(72,44,24,0.6)]"
        >
          {formError && (
            <div className="mb-4 rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-red-700">
              {formError}
            </div>
          )}
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-gray-700">{t('newapi.relayName')}</label>
              <input
                type="text"
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="app-input mt-1 block w-full"
                placeholder={t('newapi.relayNamePlaceholder')}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">{t('newapi.baseUrl')}</label>
              <input
                type="url"
                required
                value={form.base_url}
                onChange={(e) => setForm({ ...form, base_url: e.target.value })}
                className="app-input mt-1 block w-full"
                placeholder="https://relay.example.com"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">{t('newapi.relayToken')}</label>
              <input
                type="password"
                required
                value={form.relay_token}
                onChange={(e) => setForm({ ...form, relay_token: e.target.value })}
                className="app-input mt-1 block w-full"
                placeholder="sk-..."
                autoComplete="new-password"
              />
              <p className="mt-1 text-xs text-[#9d7d6e]">{t('newapi.relayTokenHint')}</p>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">{t('newapi.dailyLimit')}</label>
              <input
                type="number"
                min={0}
                required
                value={form.daily_limit}
                onChange={(e) => setForm({ ...form, daily_limit: Number(e.target.value) })}
                className="app-input mt-1 block w-full"
              />
              <p className="mt-1 text-xs text-[#9d7d6e]">{t('newapi.dailyLimitHint')}</p>
            </div>
          </div>
          <div className="mt-5 flex items-center gap-2">
            <button type="submit" disabled={submitting} className="app-button-primary disabled:cursor-not-allowed disabled:opacity-50">
              {submitting ? t('common.saving') : t('common.save')}
            </button>
            <button type="button" onClick={() => setShowCreate(false)} className="app-button-secondary">
              {t('common.cancel')}
            </button>
          </div>
        </form>
      )}

      <section className="overflow-hidden rounded-[28px] border border-[#ead8cf] bg-[linear-gradient(180deg,rgba(255,255,255,0.9)_0%,rgba(255,250,246,0.96)_100%)] shadow-[0_30px_80px_-52px_rgba(60,42,28,0.6)]">
        <div className="overflow-x-auto">
          <table className="min-w-full">
            <thead>
              <tr className="border-b border-[#f0e3db] bg-[linear-gradient(180deg,#fffaf6_0%,#fff6f1_100%)] text-left text-[11px] font-semibold uppercase tracking-[0.18em] text-[#9d7d6e]">
                <th className="px-6 py-4">{t('newapi.relayName')}</th>
                <th className="px-6 py-4">{t('newapi.baseUrl')}</th>
                <th className="px-6 py-4">{t('newapi.dailyLimit')}</th>
                <th className="px-6 py-4">{t('newapi.createdAt')}</th>
                <th className="px-6 py-4 text-right">{t('newapi.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {relays.map((relay) => (
                <tr key={relay.id} className="border-b border-[#f2e8e2] text-sm transition-colors hover:bg-[#fffaf6]">
                  <td className="px-6 py-5">
                    <div className="flex items-center gap-3">
                      <div className="rounded-xl border border-[#f1e2d9] bg-white p-2 shadow-[0_12px_30px_-24px_rgba(72,44,24,0.5)]">
                        <Server className="h-4 w-4 text-[#ef6b4a]" />
                      </div>
                      <div>
                        <p className="font-semibold text-[#1d1713]">{relay.name}</p>
                        <p className="mt-0.5 text-xs text-[#b09d93]">{relay.masked_token}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-5 font-mono text-xs text-[#7a6d66]">{relay.base_url}</td>
                  <td className="px-6 py-5 font-medium tabular-nums text-[#1d1713]">
                    {relay.daily_limit > 0 ? relay.daily_limit : '∞'}
                  </td>
                  <td className="px-6 py-5 text-[#7a6d66]">{formatDate(relay.created_at)}</td>
                  <td className="px-6 py-5 text-right">
                    <button
                      type="button"
                      disabled={deletingId === relay.id}
                      onClick={() => void handleDelete(relay)}
                      className="rounded-full border border-[#ead7cd] bg-white p-2 text-[#a05f46] transition-all hover:border-red-300 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
                      title={t('common.delete')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {!relaysQuery.isLoading && relays.length === 0 && (
          <div className="px-6 py-16 text-center text-sm text-[#8f8681]">
            {t('newapi.noRelays')}
          </div>
        )}

        {relaysQuery.isLoading && (
          <div className="px-6 py-16 text-center text-sm text-[#8f8681]">{t('common.loading')}</div>
        )}
      </section>
    </AdminLayout>
  );
};

export default NewApiRelaysPage;
