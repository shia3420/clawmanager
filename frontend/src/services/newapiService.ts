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

  exchange: async (relayName: string, email: string): Promise<NewApiExchangeResult> => {
    const response = await api.post('/integrations/newapi/sso/exchange', {
      relay_name: relayName,
      email,
    });
    return response.data.data;
  },
};
