package services

import (
	"fmt"

	"clawreef/internal/models"
	"clawreef/internal/repository"
)

type AgentVariantTemplateService interface {
	ListPublic() ([]models.AgentVariantTemplate, error)
	GetByID(id int) (*models.AgentVariantTemplate, error)
	GetBySlug(slug string) (*models.AgentVariantTemplate, error)
	IncrementUsageCount(id int) error
}

type agentVariantTemplateService struct {
	repo repository.AgentVariantTemplateRepository
}

func NewAgentVariantTemplateService(repo repository.AgentVariantTemplateRepository) AgentVariantTemplateService {
	return &agentVariantTemplateService{repo: repo}
}

func (s *agentVariantTemplateService) ListPublic() ([]models.AgentVariantTemplate, error) {
	return s.repo.ListPublic()
}

func (s *agentVariantTemplateService) GetByID(id int) (*models.AgentVariantTemplate, error) {
	return s.repo.GetByID(id)
}

func (s *agentVariantTemplateService) GetBySlug(slug string) (*models.AgentVariantTemplate, error) {
	return s.repo.GetBySlug(slug)
}

func (s *agentVariantTemplateService) IncrementUsageCount(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid variant template ID")
	}
	return s.repo.IncrementUsageCount(id)
}
