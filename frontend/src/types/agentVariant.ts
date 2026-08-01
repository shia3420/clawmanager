export interface AgentVariantTemplate {
  id: number;
  name: string;
  slug: string;
  description?: string;
  runtime_type: string;
  image_registry?: string;
  image_tag?: string;
  skill_ids: number[];
  config_plan?: {
    mode?: string;
    bundle_id?: number;
    resource_ids?: number[];
  };
  icon: string;
  category: string;
  is_public: boolean;
  created_by?: number;
  created_at: string;
  updated_at: string;

  recommended_cpu: number;
  recommended_memory: number;
  recommended_disk: number;
  status: 'draft' | 'published' | 'deprecated' | 'archived';
  readme_md?: string;
  screenshot_urls?: string;
  version: number;
  source_template_id?: number;
  usage_count: number;
}
