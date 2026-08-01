package handlers

import (
	"net/http"

	"clawreef/internal/services"
	"clawreef/internal/utils"

	"github.com/gin-gonic/gin"
)

type VariantTemplateHandler struct {
	service services.AgentVariantTemplateService
}

func NewVariantTemplateHandler(service services.AgentVariantTemplateService) *VariantTemplateHandler {
	return &VariantTemplateHandler{service: service}
}

func (h *VariantTemplateHandler) ListPublic(c *gin.Context) {
	templates, err := h.service.ListPublic()
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Variant templates retrieved successfully", gin.H{"variants": templates})
}

func (h *VariantTemplateHandler) GetBySlug(c *gin.Context) {
	template, err := h.service.GetBySlug(c.Param("slug"))
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	if template == nil {
		utils.Error(c, http.StatusNotFound, "Variant template not found")
		return
	}
	utils.Success(c, http.StatusOK, "Variant template retrieved successfully", template)
}

func (h *VariantTemplateHandler) RecordUsage(c *gin.Context) {
	template, err := h.service.GetBySlug(c.Param("slug"))
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	if template == nil {
		utils.Error(c, http.StatusNotFound, "Variant template not found")
		return
	}
	if err := h.service.IncrementUsageCount(template.ID); err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "Variant template usage recorded", nil)
}
