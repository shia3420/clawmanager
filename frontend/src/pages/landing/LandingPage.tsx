import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../contexts/AuthContext';
import { useI18n } from '../../contexts/I18nContext';
import { agentVariantService } from '../../services/agentVariantService';
import { instanceService } from '../../services/instanceService';
import type { AgentVariantTemplate } from '../../types/agentVariant';
import { Sparkles, Cpu, Puzzle, Loader2, Search, Zap } from 'lucide-react';

const CATEGORIES = [
  { id: 'All', labelKey: 'landing.categories.all' },
  { id: 'developer', labelKey: 'landing.categories.developer' },
  { id: 'creative', labelKey: 'landing.categories.creative' },
  { id: 'business', labelKey: 'landing.categories.business' },
  { id: 'research', labelKey: 'landing.categories.research' },
  { id: 'general', labelKey: 'landing.categories.general' },
];

const PENDING_TRIAL_KEY = 'clawmanager.pendingTrialVariant';

const LandingPage: React.FC = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const { isAuthenticated, user } = useAuth();
  const [variants, setVariants] = useState<AgentVariantTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [category, setCategory] = useState('All');
  const [searchQuery, setSearchQuery] = useState('');
  const [creatingId, setCreatingId] = useState<number | null>(null);
  const [trialError, setTrialError] = useState<string | null>(null);

  const skillCount = (v: AgentVariantTemplate) =>
    Array.isArray(v.skill_ids) ? v.skill_ids.length : 0;

  const createTrialInstance = useCallback(async (variant: AgentVariantTemplate) => {
    setCreatingId(variant.id);
    setTrialError(null);
    try {
      const instance = await instanceService.createInstance({
        type: variant.runtime_type as any,
        name: t('landing.trialInstanceName', { name: variant.name }),
        mode: 'lite',
        instance_mode: 'lite',
        skill_ids: variant.skill_ids?.length ? variant.skill_ids : undefined,
        openclaw_config_plan: variant.config_plan?.mode ? variant.config_plan as any : undefined,
        cpu_cores: 1,
        memory_gb: 1,
        disk_gb: 10,
        os_type: 'ubuntu',
        os_version: '22.04',
      });
      agentVariantService.recordUsage(variant.slug).catch(() => {});
      navigate(`/instances/${instance.id}`);
    } catch (err: any) {
      const msg = err.response?.data?.error || err.message || t('landing.createFailed');
      setTrialError(msg);
    } finally {
      setCreatingId(null);
    }
  }, [navigate, t]);

  useEffect(() => {
    agentVariantService
      .listPublic()
      .then(setVariants)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!isAuthenticated) return;
    const raw = sessionStorage.getItem(PENDING_TRIAL_KEY);
    if (!raw) return;
    sessionStorage.removeItem(PENDING_TRIAL_KEY);
    let pending: AgentVariantTemplate;
    try {
      pending = JSON.parse(raw);
    } catch {
      return;
    }
    createTrialInstance(pending);
  }, [isAuthenticated, createTrialInstance]);

  const handleTrial = (variant: AgentVariantTemplate) => {
    setTrialError(null);
    if (isAuthenticated) {
      createTrialInstance(variant);
    } else {
      sessionStorage.setItem(PENDING_TRIAL_KEY, JSON.stringify(variant));
      navigate('/login', { state: { from: '/' } });
    }
  };

  const filteredVariants = variants.filter((v) => {
    const matchesCategory = category === 'All' || v.category === category;
    const q = searchQuery.toLowerCase();
    const matchesSearch =
      !q ||
      v.name.toLowerCase().includes(q) ||
      (v.description || '').toLowerCase().includes(q);
    return matchesCategory && matchesSearch;
  });

  const countByCategory: Record<string, number> = { All: variants.length };
  variants.forEach((v) => {
    countByCategory[v.category] = (countByCategory[v.category] || 0) + 1;
  });

  return (
    <div className="w-full bg-slate-50 min-h-screen relative font-sans">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_0%,rgba(220,38,38,0.05),transparent_55%)] pointer-events-none" />

      {/* Hero */}
      <header className="max-w-7xl mx-auto px-6 py-10 relative z-20">
        <div className="flex items-center justify-between mb-16">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-red-600 rounded-xl flex items-center justify-center font-bold text-white text-xl shadow-lg shadow-red-600/20">
              Ω
            </div>
            <h1 className="text-xl font-bold tracking-tight text-slate-800">
              OpenClaw <span className="text-red-500">+ Hermes</span>
            </h1>
          </div>
          <div className="flex items-center gap-3">
            {isAuthenticated && (
              <button
                onClick={() => navigate(user?.role === 'admin' ? '/admin' : '/dashboard')}
                className="px-6 py-2 bg-red-600 hover:bg-red-500 text-white rounded-full text-sm font-bold transition-all cursor-pointer"
              >
                {user?.role === 'admin' ? t('landing.enterConsole') : t('landing.myInstances')}
              </button>
            )}
          </div>
        </div>

        <div className="text-center max-w-3xl mx-auto">
          <div className="inline-flex items-center gap-2 px-3 py-1 bg-red-50 border border-red-200 rounded-full text-xs font-medium text-red-600 mb-6">
            <Sparkles size={12} />
            {t('landing.heroBadge')}
          </div>
          <h2 className="text-4xl md:text-5xl font-bold tracking-tight mb-6 leading-[1.15] text-slate-900">
            {t('landing.heroTitle').split('\n').map((line, i) => (
              <React.Fragment key={i}>{i > 0 && <br />}{line}</React.Fragment>
            ))}
          </h2>
          <p className="text-slate-500 text-lg max-w-xl mx-auto">
            {t('landing.heroSubtitle')}
          </p>
        </div>
      </header>

      {/* Variant Cards */}
      <section className="max-w-7xl mx-auto px-6 pb-32 relative z-20">
        {/* Filters */}
        <div className="mb-10 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-2">
            {CATEGORIES.map((cat) => (
              <button
                key={cat.id}
                onClick={() => setCategory(cat.id)}
                className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                  category === cat.id
                    ? 'bg-red-600 text-white shadow-md'
                    : 'bg-white border border-slate-200 text-slate-600 hover:bg-slate-100'
                }`}
              >
                {t(cat.labelKey)}
                <span className="ml-1.5 opacity-50 text-xs">{countByCategory[cat.id] || 0}</span>
              </button>
            ))}
          </div>
          <div className="relative w-full sm:w-72">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={18} />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t('landing.searchPlaceholder')}
              className="w-full pl-10 pr-4 py-2.5 border border-slate-300 rounded-lg bg-white text-sm text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-red-100 focus:border-red-500"
            />
          </div>
        </div>

        {trialError && (
          <div className="mb-6 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-red-700 text-sm flex items-center justify-between">
            <span>{trialError}</span>
            <button onClick={() => setTrialError(null)} className="text-red-400 hover:text-red-600 ml-4">
              ✕
            </button>
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="animate-spin w-6 h-6 text-red-500" />
            <span className="ml-3 text-slate-500">{t('landing.loading')}</span>
          </div>
        ) : filteredVariants.length === 0 ? (
          <div className="text-center py-20">
            <Cpu className="mx-auto h-12 w-12 text-slate-300" />
            <h3 className="mt-4 text-lg font-semibold text-slate-700">{t('landing.noAgents')}</h3>
            <p className="mt-1 text-sm text-slate-400">{t('landing.noAgentsSubtitle')}</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
            {filteredVariants.map((variant) => (
              <div
                key={variant.id}
                className="app-panel flex flex-col hover:border-red-300 hover:shadow-lg transition-all duration-300"
              >
                <div className="p-6 flex-1">
                  <div className="flex items-start justify-between mb-4">
                    <div className="w-12 h-12 bg-red-50 border border-red-100 rounded-xl flex items-center justify-center text-red-500">
                      <Zap size={22} />
                    </div>
                    <span className="text-[10px] font-bold px-2 py-1 bg-slate-100 border border-slate-200 rounded-md text-slate-500 uppercase tracking-widest">
                      {t('landing.categories.' + variant.category) || variant.category}
                    </span>
                  </div>

                  <h3 className="text-lg font-bold mb-2 text-slate-900">{variant.name}</h3>
                  <p className="text-sm text-slate-500 mb-4 line-clamp-3 leading-relaxed">
                    {variant.description || t('landing.runtimeVariant', { type: variant.runtime_type })}
                  </p>

                  <div className="flex flex-wrap gap-2 mb-4">
                    <span className="text-[10px] text-red-600 font-mono bg-red-50 px-2 py-0.5 rounded">
                      #{variant.runtime_type}
                    </span>
                    <span className="text-[10px] text-slate-500 font-mono bg-slate-100 px-2 py-0.5 rounded flex items-center gap-1">
                      <Puzzle size={10} />
                      {t('landing.skillCount', { count: skillCount(variant) })}
                    </span>
                  </div>
                </div>

                <div className="border-t border-slate-100 px-6 py-4">
                  <button
                    onClick={() => handleTrial(variant)}
                    disabled={creatingId === variant.id}
                    className="w-full py-3 rounded-lg bg-red-600 hover:bg-red-500 text-white font-semibold text-sm shadow-md hover:translate-y-[-1px] transition-all disabled:opacity-50 flex items-center justify-center gap-2"
                  >
                    {creatingId === variant.id ? (
                      <Loader2 className="animate-spin" size={16} />
                    ) : (
                      <>
                        <Sparkles size={16} />
                        {t('landing.freeTrial')}
                      </>
                    )}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <footer className="py-12 border-t border-slate-200 px-6 text-center text-slate-400 text-[10px] uppercase font-bold tracking-[0.2em] relative z-20">
        {t('landing.footer')}
      </footer>
    </div>
  );
};

export default LandingPage;
