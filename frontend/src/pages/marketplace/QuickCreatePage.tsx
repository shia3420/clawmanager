import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import UserLayout from '../../components/UserLayout';
import { useI18n } from '../../contexts/I18nContext';
import { agentVariantService } from '../../services/agentVariantService';
import { instanceService } from '../../services/instanceService';
import type { AgentVariantTemplate } from '../../types/agentVariant';
import { Cpu, HardDrive, MemoryStick, Loader2, AlertCircle, ArrowLeft, Package } from 'lucide-react';

const QuickCreatePage: React.FC = () => {
  const { slug } = useParams<{ slug: string }>();
  const navigate = useNavigate();
  const { t } = useI18n();
  const [variant, setVariant] = useState<AgentVariantTemplate | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  const [instanceName, setInstanceName] = useState('');
  const [cpu, setCpu] = useState(2);
  const [memory, setMemory] = useState(4);
  const [disk, setDisk] = useState(20);

  useEffect(() => {
    if (!slug) return;
    setLoading(true);
    agentVariantService.getBySlug(slug)
      .then((v) => {
        setVariant(v);
        setInstanceName(`${v.name} #${Date.now()}`);
        setCpu(v.recommended_cpu || 2);
        setMemory(v.recommended_memory || 4);
        setDisk(v.recommended_disk || 20);
      })
      .catch((err) => setError(err.response?.data?.error || t('marketplace.quickCreate.loadError')))
      .finally(() => setLoading(false));
  }, [slug, t]);

  const handleCreate = async () => {
    if (!variant || !instanceName.trim()) return;
    setCreating(true);
    setError(null);
    try {
      const instance = await instanceService.createInstance({
        name: instanceName.trim(),
        type: variant.runtime_type as any,
        skill_ids: variant.skill_ids?.length ? variant.skill_ids : undefined,
        openclaw_config_plan: variant.config_plan?.mode ? variant.config_plan as any : undefined,
        cpu_cores: cpu,
        memory_gb: memory,
        disk_gb: disk,
        os_type: 'ubuntu',
        os_version: '22.04',
      });
      if (slug) {
        agentVariantService.recordUsage(slug).catch(() => {});
      }
      navigate(`/instances/${instance.id}`);
    } catch (err: any) {
      setError(err.response?.data?.error || t('marketplace.quickCreate.createError'));
    } finally {
      setCreating(false);
    }
  };

  if (loading) {
    return (
      <UserLayout title={t('marketplace.quickCreate.title')}>
        <div className="flex items-center justify-center h-64">
          <Loader2 className="animate-spin w-6 h-6 text-red-600" />
          <span className="ml-3 text-slate-500">{t('marketplace.quickCreate.loading')}</span>
        </div>
      </UserLayout>
    );
  }

  if (error && !variant) {
    return (
      <UserLayout title={t('marketplace.quickCreate.title')}>
        <div className="max-w-lg mx-auto mt-12">
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg flex items-center gap-2">
            <AlertCircle size={18} />
            <span>{error}</span>
          </div>
          <button
            onClick={() => navigate('/marketplace')}
            className="mt-4 app-button-secondary flex items-center gap-2"
          >
            <ArrowLeft size={16} /> {t('marketplace.quickCreate.back')}
          </button>
        </div>
      </UserLayout>
    );
  }

  if (!variant) return null;

  return (
    <UserLayout title={t('marketplace.quickCreate.title')}>
      <div className="max-w-2xl mx-auto space-y-6">
        <button
          onClick={() => navigate('/marketplace')}
          className="inline-flex items-center gap-2 text-sm text-slate-500 hover:text-slate-900 transition-colors"
        >
          <ArrowLeft size={16} /> {t('marketplace.quickCreate.back')}
        </button>

        <div className="app-panel p-6">
          <div className="flex items-start gap-4">
            <div className="w-14 h-14 bg-slate-50 border border-slate-200 rounded-xl flex items-center justify-center text-red-600 shrink-0">
              <Cpu size={28} />
            </div>
            <div className="flex-1 min-w-0">
              <h2 className="text-xl font-bold text-slate-900">{variant.name}</h2>
              <p className="mt-1 text-sm text-slate-500 line-clamp-2">
                {variant.description || ''}
              </p>
              <div className="flex flex-wrap gap-2 mt-3">
                <span className="text-[10px] font-bold px-2 py-1 bg-slate-50 border border-slate-200 rounded-md text-slate-500 uppercase tracking-widest">
                  {t(`marketplace.categories.${variant.category}`) || variant.category}
                </span>
                <span className="text-[10px] text-red-700 font-mono bg-slate-50 px-2 py-1 rounded">
                  #{variant.runtime_type}
                </span>
              </div>
              {variant.image_registry && (
                <div className="mt-2 text-xs text-slate-400 font-mono">
                  {variant.image_registry}{variant.image_tag ? `:${variant.image_tag}` : ''}
                </div>
              )}
              {variant.skill_ids && variant.skill_ids.length > 0 && (
                <div className="mt-3 flex items-center gap-2">
                  <Package size={14} className="text-red-600" />
                  <span className="text-xs text-slate-600">
                    {t('marketplace.quickCreate.bundledSkills', { count: variant.skill_ids.length })}
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>

        <div className="app-panel p-6">
          <h3 className="text-lg font-semibold text-slate-900 mb-4">{t('marketplace.quickCreate.resourceConfig')}</h3>

          <div className="space-y-4">
            <div>
              <label className="flex items-center gap-2 text-sm font-medium text-slate-600 mb-1.5">
                <Cpu size={16} /> {t('marketplace.quickCreate.cpuCores')}
              </label>
              <div className="flex items-center gap-3">
                <input
                  type="range"
                  min="0.5"
                  max="16"
                  step="0.5"
                  value={cpu}
                  onChange={(e) => setCpu(parseFloat(e.target.value))}
                  className="flex-1"
                />
                <span className="text-sm font-semibold text-slate-900 w-12 text-right">{cpu}</span>
              </div>
            </div>

            <div>
              <label className="flex items-center gap-2 text-sm font-medium text-slate-600 mb-1.5">
                <MemoryStick size={16} /> {t('marketplace.quickCreate.memory')}
              </label>
              <div className="flex items-center gap-3">
                <input
                  type="range"
                  min="1"
                  max="64"
                  step="1"
                  value={memory}
                  onChange={(e) => setMemory(parseInt(e.target.value))}
                  className="flex-1"
                />
                <span className="text-sm font-semibold text-slate-900 w-12 text-right">{memory}GB</span>
              </div>
            </div>

            <div>
              <label className="flex items-center gap-2 text-sm font-medium text-slate-600 mb-1.5">
                <HardDrive size={16} /> {t('marketplace.quickCreate.disk')}
              </label>
              <div className="flex items-center gap-3">
                <input
                  type="range"
                  min="10"
                  max="500"
                  step="10"
                  value={disk}
                  onChange={(e) => setDisk(parseInt(e.target.value))}
                  className="flex-1"
                />
                <span className="text-sm font-semibold text-slate-900 w-12 text-right">{disk}GB</span>
              </div>
            </div>
          </div>
        </div>

        <div className="app-panel p-6">
          <h3 className="text-lg font-semibold text-slate-900 mb-4">{t('marketplace.quickCreate.instanceName')}</h3>
          <input
            type="text"
            value={instanceName}
            onChange={(e) => setInstanceName(e.target.value)}
            className="app-input w-full"
            placeholder={t('marketplace.quickCreate.namePlaceholder')}
          />
        </div>

        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-red-700 text-sm flex items-center gap-2">
            <AlertCircle size={16} />
            {error}
          </div>
        )}

        <button
          onClick={handleCreate}
          disabled={creating || !instanceName.trim()}
          className="w-full py-3 rounded-lg bg-red-600 text-white font-semibold shadow-md hover:bg-red-700 transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          {creating ? (
            <>
              <Loader2 className="animate-spin" size={18} />
              {t('marketplace.quickCreate.creating')}
            </>
          ) : (
            <>
              <Cpu size={18} />
              {t('marketplace.quickCreate.deploy', { name: instanceName.trim() || 'Instance' })}
            </>
          )}
        </button>
      </div>
    </UserLayout>
  );
};

export default QuickCreatePage;
