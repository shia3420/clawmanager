package repository

import (
	"clawreef/internal/models"

	"github.com/upper/db/v4"
)

type AgentVariantTemplateRepository interface {
	ListPublic() ([]models.AgentVariantTemplate, error)
	GetByID(id int) (*models.AgentVariantTemplate, error)
	GetBySlug(slug string) (*models.AgentVariantTemplate, error)
	IncrementUsageCount(id int) error
}

type agentVariantTemplateRepository struct {
	sess db.Session
}

func NewAgentVariantTemplateRepository(sess db.Session) AgentVariantTemplateRepository {
	return &agentVariantTemplateRepository{sess: sess}
}

func (r *agentVariantTemplateRepository) collection() db.Collection {
	return r.sess.Collection("agent_variant_templates")
}

func (r *agentVariantTemplateRepository) ListPublic() ([]models.AgentVariantTemplate, error) {
	var templates []models.AgentVariantTemplate
	err := r.collection().Find(db.Cond{"status": "published"}).OrderBy("-usage_count", "id").All(&templates)
	return templates, err
}

func (r *agentVariantTemplateRepository) GetByID(id int) (*models.AgentVariantTemplate, error) {
	var template models.AgentVariantTemplate
	err := r.collection().Find(db.Cond{"id": id}).One(&template)
	if err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

func (r *agentVariantTemplateRepository) GetBySlug(slug string) (*models.AgentVariantTemplate, error) {
	var template models.AgentVariantTemplate
	err := r.collection().Find(db.Cond{"slug": slug}).One(&template)
	if err != nil {
		if err == db.ErrNoMoreRows {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

func (r *agentVariantTemplateRepository) IncrementUsageCount(id int) error {
	return r.collection().Find(db.Cond{"id": id}).Update(map[string]interface{}{
		"usage_count": db.Raw("usage_count + 1"),
	})
}
