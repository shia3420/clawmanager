import api from './api';

export interface NewApiRelay {
  id: number;
  name: string;
  base_url: string;
  daily_limit: number;
  masked_token: string;
  created_by: number;
  created_at: string;
}

export interface NewApiExchangeResult {
  session_token: string;
  user_id: number;
  relay_name: string;
  relay_base_url: string;
  created_user: boolean;
}

export interface NewApiIdentityLink {
  id: number;
  user_id: number;
  username: string;
  email: string;
  role: string;
  relay_key_id: number;
  relay_name: string;
  relay_base_url: string;
  external_id: string;
  upstream_user_id: string;
  token_name: string;
  has_credential: boolean;
  today_used: number;
  today_limit: number;
  created_at: string;
  last_used_at?: string;
}

export const newapiService = {
  listRelays: async (): Promise<NewApiRelay[]> => {
    const response = await api.get('/integrations/newapi/admin/relays');
    return response.data.data ?? [];
  },

  createRelay: async (request: {
    name: string;
    base_url: string;
    relay_token: string;
    daily_limit: number;
  }): Promise<void> => {
    await api.post('/integrations/newapi/admin/relays', request);
  },

  deleteRelay: async (id: number): Promise<void> => {
    await api.delete(`/integrations/newapi/admin/relays/${id}`);
  },

  exchange: async (relayName: string, dashboardToken: string): Promise<NewApiExchangeResult> => {
    const response = await api.post('/integrations/newapi/sso/exchange', {
      relay_name: relayName,
      dashboard_token: dashboardToken,
    });
    return response.data.data;
  },

  listIdentityLinks: async (): Promise<NewApiIdentityLink[]> => {
    const response = await api.get('/integrations/newapi/admin/identity-links');
    return response.data.data?.items ?? [];
  },

  unlinkIdentityLink: async (id: number): Promise<void> => {
    await api.post(`/integrations/newapi/admin/identity-links/${id}/unlink`);
  },
};
