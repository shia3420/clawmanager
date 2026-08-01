-- Add agent variant templates table (agent marketplace)
-- Ported from the marketplace fork (015/024/027), consolidated into a single
-- migration with 100-series naming so it never collides with upstream.
-- Scope: marketplace display + quick-create only (no review / versioning).

CREATE TABLE IF NOT EXISTS agent_variant_templates (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(100) NOT NULL UNIQUE,
  description TEXT,
  runtime_type VARCHAR(50) NOT NULL,
  image_registry VARCHAR(500) NULL,
  image_tag VARCHAR(255) NULL,
  skill_ids JSON DEFAULT ('[]'),
  config_plan JSON DEFAULT NULL,
  icon VARCHAR(50) DEFAULT 'bot',
  category VARCHAR(50) DEFAULT 'general',
  is_public BOOLEAN DEFAULT TRUE,
  created_by INT DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  recommended_cpu DECIMAL(5,2) NOT NULL DEFAULT 2.0,
  recommended_memory INT NOT NULL DEFAULT 4,
  recommended_disk INT NOT NULL DEFAULT 20,
  status VARCHAR(20) NOT NULL DEFAULT 'draft',
  readme_md TEXT,
  screenshot_urls TEXT,
  version INT NOT NULL DEFAULT 1,
  source_template_id INT DEFAULT NULL,
  usage_count INT NOT NULL DEFAULT 0,
  INDEX idx_runtime_type (runtime_type),
  INDEX idx_is_public (is_public),
  INDEX idx_category (category),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO agent_variant_templates (name, slug, description, runtime_type, image_registry, image_tag, skill_ids, config_plan, icon, category, is_public, recommended_cpu, recommended_memory, recommended_disk, status, version, readme_md)
VALUES
('Code Assistant', 'code-assistant', '你的个人代码审查员和结对编程伙伴。擅长调试复杂逻辑、建议重构方案、编写干净测试。角色设定：严谨的高级工程师，善于逐步解释推理过程。', 'openclaw', 'ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite', 'latest', '[]', '{"mode": "bundle"}', 'code', 'developer', TRUE, 2.0, 4, 20, 'published', 1, '## Code Assistant\n\n一个开箱即用的 OpenClaw 智能体，内置严谨高级工程师人设，擅长：\n\n- 复杂逻辑调试\n- 代码重构方案建议\n- 干净测试编写\n- 逐步解释推理过程'),
('Shell Expert', 'shell-expert', '命令行大师和系统管理员。快速编写 Shell 脚本、排查服务器问题、优化 DevOps 流程。角色设定：经验丰富的系统管理员，擅长一行命令解决问题。', 'openclaw', 'ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite', 'latest', '[]', '{"mode": "bundle"}', 'terminal', 'developer', TRUE, 2.0, 4, 20, 'published', 1, '## Shell Expert\n\n面向命令行与系统运维的 OpenClaw 智能体，擅长：\n\n- 快速编写 Shell 脚本\n- 排查服务器问题\n- 优化 DevOps 流程'),
('Document Writer', 'document-writer', '技术文档专家。将粗略笔记转化为精美的 API 文档、README 和架构指南。角色设定：耐心的技术写作专家，注重清晰而非术语堆砌。', 'openclaw', 'ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite', 'latest', '[]', '{"mode": "bundle"}', 'file-text', 'creative', TRUE, 2.0, 4, 20, 'published', 1, '## Document Writer\n\n技术写作专家智能体，擅长：\n\n- API 文档撰写\n- README 与架构指南\n- 将粗略笔记转化为清晰文档'),
('Research Analyst', 'research-analyst', '基于 Hermes 内核的深度推理引擎。分析研究论文、综合发现、构建逻辑论证。角色设定：严谨的学术研究者，具备跨领域专业知识。', 'hermes', 'ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite', 'latest', '[]', '{"mode": "bundle"}', 'brain', 'research', TRUE, 2.0, 4, 20, 'published', 1, '## Research Analyst\n\n基于 Hermes 内核的深度推理引擎，擅长：\n\n- 研究论文分析\n- 跨领域发现综合\n- 逻辑论证构建'),
('General Purpose', 'general-purpose', '一个空白的 OpenClaw 实例。没有预设技能或个性——完全按你的需求定制。适合希望完全掌控智能体配置的用户。', 'openclaw', 'ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite', 'latest', '[]', NULL, 'bot', 'general', TRUE, 2.0, 4, 20, 'published', 1, '## General Purpose\n\n一个空白 OpenClaw 实例。没有预设技能或个性，完全按你的需求定制。')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  runtime_type = VALUES(runtime_type),
  image_registry = VALUES(image_registry),
  image_tag = VALUES(image_tag),
  skill_ids = VALUES(skill_ids),
  config_plan = VALUES(config_plan),
  icon = VALUES(icon),
  category = VALUES(category),
  is_public = VALUES(is_public),
  recommended_cpu = VALUES(recommended_cpu),
  recommended_memory = VALUES(recommended_memory),
  recommended_disk = VALUES(recommended_disk),
  status = VALUES(status),
  version = VALUES(version),
  readme_md = VALUES(readme_md);
