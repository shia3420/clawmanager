-- Seed the "FinProc Tender Monitor" agent variant template (business/finance vertical).
-- Ported from the marketplace fork (docs/finproc-tender-monitor-skill.md).
-- Prerequisite: the finproc-tender-monitor skill must be imported via the
-- skill-hub upload flow (skill_key 'finproc-tender-monitor') before this runs;
-- the template's skill_ids are wired by key lookup so no hardcoded id is used.
-- Also makes the skill publicly attachable so any user can quick-create from the template.

UPDATE skills
SET visibility = 'public'
WHERE skill_key = 'finproc-tender-monitor'
  AND source_type = 'uploaded'
  AND status = 'active';

INSERT INTO agent_variant_templates (
  name, slug, description, runtime_type, image_registry, image_tag, skill_ids,
  config_plan, icon, category, is_public, recommended_cpu, recommended_memory,
  recommended_disk, status, version, readme_md
)
SELECT
  '金融采购招标监控助手',
  'finproc-tender-monitor',
  '金融采购招标监控助手（FinProc Tender Monitor）。按日定时检索中国金融采购领域招标信息（金采网 cfcpn.com、中银智采 boc.cn），按关注关键词自动筛选标讯并推送结构化摘要通知。',
  'openclaw',
  'ghcr.io/yuan-lab-llm/agentsruntime/openclaw-lite',
  'latest',
  COALESCE(JSON_ARRAY(s.id), JSON_ARRAY()),
  '{"mode": "bundle"}',
  'search',
  'business',
  TRUE,
  2.0, 4, 20,
  'published', 1,
  '## 金融采购招标监控助手（FinProc Tender Monitor）\n\n基于 OpenClaw 的金融采购招标信息监控智能体。无硬编码爬虫，技能核心为 Prompt，Agent 用 LLM + 浏览器自主执行。\n\n### 监控网站\n\n| 网站 | URL | 说明 |\n|---|---|---|\n| 金采网 | http://www.cfcpn.com/ | 中国金融集中采购网，金融机构采购信息聚合，免登录浏览 |\n| 中银智采 | https://ctpch.fmscop.bankofchina.com/ | 中国银行采购平台（备用入口 boc.cn/aboutboc/bi6/）|\n\n### 能力\n\n- 按日定时检索两平台的采购公告、征集公告、结果公告、更正公告\n- 按关注关键词（必含 / 或含 / 排除）自动筛选匹配标讯\n- 结构化摘要通知：标题、日期、来源、链接、摘要、匹配关键词\n- 记忆驱动进化：自动记录网站结构与更新频率，网站改版时自动重新学习\n\n### 使用方式\n\n1. 以此模板创建实例（自动附加 finproc-tender-monitor 技能）\n2. 告诉 agent 关注的关键词，例如「关注网络安全和数据治理相关的招标」\n3. Agent 会按调度定期监控并推送通知；可随时对话调整关键词或监控范围'
FROM (SELECT 1) AS dummy
LEFT JOIN (SELECT id FROM skills WHERE skill_key = 'finproc-tender-monitor' AND status = 'active' ORDER BY id LIMIT 1) AS s ON 1 = 1
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  skill_ids = VALUES(skill_ids),
  config_plan = VALUES(config_plan),
  icon = VALUES(icon),
  category = VALUES(category),
  is_public = VALUES(is_public),
  status = VALUES(status),
  readme_md = VALUES(readme_md);
