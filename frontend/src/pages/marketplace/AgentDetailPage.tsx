import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import UserLayout from '../../components/UserLayout';
import { useI18n } from '../../contexts/I18nContext';
import { agentVariantService } from '../../services/agentVariantService';
import type { AgentVariantTemplate } from '../../types/agentVariant';
import {
  Cpu, HardDrive, MemoryStick, Loader2, AlertCircle, ArrowLeft,
  Puzzle, Eye, Tag, Clock, Zap
} from 'lucide-react';

const AgentDetailPage: React.FC = () => {
  const { t } = useI18n();
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const [variant, setVariant] = useState<AgentVariantTemplate | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!slug) return;
    setLoading(true);
    agentVariantService.getBySlug(slug)
      .then(setVariant)
      .catch((err) => setError(err.response?.data?.error || t('marketplace.detail.notFoundMsg')))
      .finally(() => setLoading(false));
  }, [slug, t]);

  if (loading) {
    return (
      <UserLayout title={t('marketplace.detail.title')}>
        <div className="flex items-center justify-center h-64">
          <Loader2 className="animate-spin w-6 h-6 text-red-600" />
          <span className="ml-3 text-slate-500">{t('marketplace.detail.loading')}</span>
        </div>
      </UserLayout>
    );
  }

  if (error || !variant) {
    return (
      <UserLayout title={t('marketplace.detail.notFound')}>
        <div className="max-w-lg mx-auto mt-12">
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg flex items-center gap-2">
            <AlertCircle size={18} />
            <span>{error || t('marketplace.detail.notFoundMsg')}</span>
          </div>
          <button
            onClick={() => navigate('/marketplace')}
            className="mt-4 app-button-secondary flex items-center gap-2"
          >
            <ArrowLeft size={16} /> {t('marketplace.detail.back')}
          </button>
        </div>
      </UserLayout>
    );
  }

  const skillCount = Array.isArray(variant.skill_ids) ? variant.skill_ids.length : 0;

  return (
    <UserLayout title={variant.name}>
      <div className="max-w-4xl mx-auto space-y-6">
        <button
          onClick={() => navigate('/marketplace')}
          className="inline-flex items-center gap-2 text-sm text-slate-500 hover:text-slate-900 transition-colors"
        >
          <ArrowLeft size={16} /> {t('marketplace.detail.back')}
        </button>

        <div className="app-panel p-6 lg:p-8">
          <div className="flex flex-col lg:flex-row lg:items-start gap-6">
            <div className="w-16 h-16 bg-slate-50 border border-slate-200 rounded-xl flex items-center justify-center text-red-600 shrink-0">
              <Cpu size={32} />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex flex-wrap items-start justify-between gap-4">
                <div>
                  <h1 className="text-2xl font-bold text-slate-900">{variant.name}</h1>
                  <p className="mt-2 text-slate-500 leading-relaxed max-w-2xl">
                    {variant.description || ''}
                  </p>
                </div>
                <button
                  onClick={() => navigate(`/marketplace/${variant.slug}/quick-create`)}
                  className="shrink-0 px-6 py-3 rounded-lg bg-red-600 text-white font-semibold shadow-md hover:bg-red-700 transition-all flex items-center gap-2"
                >
                  <Zap size={18} />
                  {t('marketplace.detail.deployNow')}
                </button>
              </div>

              <div className="flex flex-wrap gap-2 mt-4">
                <span className="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium bg-red-50 text-red-700">
                  <Tag size={12} />
                  {t(`marketplace.categories.${variant.category}`) || variant.category}
                </span>
                <span className="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium bg-slate-100 text-slate-600">
                  <Cpu size={12} />
                  {variant.runtime_type}
                </span>
                {skillCount > 0 && (
                  <span className="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium bg-slate-100 text-slate-600">
                    <Puzzle size={12} />
                    {t('marketplace.skillCount', { count: skillCount })}
                  </span>
                )}
                <span className="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium bg-slate-100 text-slate-600">
                  <Eye size={12} />
                  v{variant.version}
                </span>
                <span className="inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium bg-slate-100 text-slate-600">
                  <Clock size={12} />
                  {t('marketplace.deploymentCount', { count: variant.usage_count || 0 })}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 app-panel p-6 lg:p-8">
            <h2 className="text-lg font-semibold text-slate-900 mb-4">{t('marketplace.detail.about')}</h2>
            {variant.readme_md ? (
              <div className="text-sm text-slate-600 leading-relaxed whitespace-pre-wrap">
                {variant.readme_md}
              </div>
            ) : (
              <p className="text-slate-400 italic">{t('marketplace.detail.noDocs')}</p>
            )}
          </div>

          <div className="space-y-6">
            <div className="app-panel p-5">
              <h3 className="text-sm font-semibold text-slate-900 mb-3">{t('marketplace.detail.recommendedResources')}</h3>
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm text-slate-600">
                    <Cpu size={16} className="text-red-600" />
                    CPU
                  </div>
                  <span className="font-semibold text-slate-900">{variant.recommended_cpu} {t('marketplace.detail.cores')}</span>
                </div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm text-slate-600">
                    <MemoryStick size={16} className="text-red-600" />
                    {t('marketplace.detail.memory')}
                  </div>
                  <span className="font-semibold text-slate-900">{variant.recommended_memory} GB</span>
                </div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-sm text-slate-600">
                    <HardDrive size={16} className="text-red-600" />
                    {t('marketplace.detail.disk')}
                  </div>
                  <span className="font-semibold text-slate-900">{variant.recommended_disk} GB</span>
                </div>
              </div>
            </div>

            {skillCount > 0 && (
              <div className="app-panel p-5">
                <h3 className="text-sm font-semibold text-slate-900 mb-3">
                  {t('marketplace.detail.skillsCount', { count: skillCount })}
                </h3>
                <div className="flex flex-wrap gap-2">
                  {variant.skill_ids.map((id) => (
                    <span
                      key={id}
                      className="inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-xs font-medium bg-slate-50 border border-slate-200 text-slate-600"
                    >
                      <Puzzle size={10} />
                      #{id}
                    </span>
                  ))}
                </div>
              </div>
            )}

            <div className="app-panel p-5">
              <button
                onClick={() => navigate(`/marketplace/${variant.slug}/quick-create`)}
                className="w-full py-2.5 rounded-lg bg-red-600 text-white text-sm font-semibold shadow-md hover:bg-red-700 transition-all flex items-center justify-center gap-2"
              >
                <Zap size={16} />
                {t('marketplace.detail.deployNow')}
              </button>
              <button
                onClick={() => navigate(`/marketplace/${variant.slug}/quick-create`)}
                className="w-full mt-2 py-2.5 rounded-lg border border-slate-300 bg-white text-sm font-semibold text-slate-600 hover:bg-slate-50 hover:border-red-400 transition-all"
              >
                {t('marketplace.detail.customizeDeploy')}
              </button>
            </div>
          </div>
        </div>
      </div>
    </UserLayout>
  );
};

export default AgentDetailPage;
