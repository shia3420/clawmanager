import api from './api';
import type { AgentVariantTemplate } from '../types/agentVariant';

export const agentVariantService = {
  listPublic: async (): Promise<AgentVariantTemplate[]> => {
    const response = await api.get('/agent-variants');
    return response.data.data.variants;
  },

  getBySlug: async (slug: string): Promise<AgentVariantTemplate> => {
    const response = await api.get(`/agent-variants/${slug}`);
    return response.data.data;
  },

  recordUsage: async (slug: string): Promise<void> => {
    await api.post(`/agent-variants/${slug}/usage`);
  },
};
