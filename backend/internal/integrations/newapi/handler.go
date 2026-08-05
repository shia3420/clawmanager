package newapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"clawreef/internal/middleware"
	"clawreef/internal/repository"
	"clawreef/internal/utils"

	"github.com/gin-gonic/gin"
)

// writeServiceError maps module sentinel errors to proper status codes without
// touching core response handling.
func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrRelayNotFound), errors.Is(err, ErrIdentityLinkNotFound):
		utils.Error(c, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrRelayInvalid):
		utils.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUpstreamRejected):
		utils.Error(c, http.StatusBadGateway, err.Error())
	default:
		utils.HandleError(c, err)
	}
}

// Handler exposes the New API relay / SSO module over its own route prefix.
type Handler struct {
	svc       Service
	jwtSecret string
	jwtExpiry time.Duration
}

// NewHandler creates the module HTTP handler.
func NewHandler(svc Service, jwtSecret string, jwtExpiry time.Duration) *Handler {
	return &Handler{svc: svc, jwtSecret: jwtSecret, jwtExpiry: jwtExpiry}
}

// RegisterRoutes mounts the module routes under api.Group("/api/v1/integrations/newapi").
func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userRepo repository.UserRepository) {
	group := api.Group("/integrations/newapi")

	admin := group.Group("/admin")
	admin.Use(middleware.Auth())
	admin.Use(middleware.SetUserInfo(userRepo))
	admin.Use(middleware.NewAdminAuth(userRepo))
	{
		admin.POST("/relays", h.createRelay)
		admin.GET("/relays", h.listRelays)
		admin.DELETE("/relays/:id", h.deleteRelay)
		admin.GET("/identity-links", h.listIdentityLinks)
		admin.POST("/identity-links/:id/unlink", h.unlinkIdentityLink)
	}

	sso := group.Group("/sso")
	{
		sso.POST("/exchange", h.exchange)
	}

	me := group.Group("/me")
	me.Use(middleware.Auth())
	me.Use(middleware.SetUserInfo(userRepo))
	{
		me.GET("", h.me)
	}
}

type createRelayRequest struct {
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	RelayToken string `json:"relay_token"`
	DailyLimit int64  `json:"daily_limit"`
}

func (h *Handler) createRelay(c *gin.Context) {
	var req createRelayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	userID := 0
	if v, ok := c.Get("userID"); ok {
		userID = v.(int)
	}
	if err := h.svc.CreateRelayKey(c.Request.Context(), req.Name, req.BaseURL, req.RelayToken, req.DailyLimit, userID); err != nil {
		writeServiceError(c, err)
		return
	}
	utils.Success(c, http.StatusCreated, "newapi relay registered", gin.H{"name": req.Name})
}

func (h *Handler) listRelays(c *gin.Context) {
	views, err := h.svc.ListRelayKeys()
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "ok", views)
}

func (h *Handler) deleteRelay(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.Error(c, http.StatusBadRequest, "relay id is invalid")
		return
	}
	if err := h.svc.DeleteRelayKey(id); err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "newapi relay deleted", nil)
}

func (h *Handler) listIdentityLinks(c *gin.Context) {
	links, err := h.svc.ListIdentityLinks()
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "identity links retrieved", gin.H{
		"items": links,
	})
}

func (h *Handler) unlinkIdentityLink(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		utils.Error(c, http.StatusBadRequest, "identity link id is invalid")
		return
	}
	if err := h.svc.UnlinkIdentityLink(id); err != nil {
		writeServiceError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "identity link unlinked", nil)
}

type exchangeRequest struct {
	RelayName      string `json:"relay_name"`
	DashboardToken string `json:"dashboard_token"`
}

type exchangeResponse struct {
	SessionToken string `json:"session_token"`
	UserID       int    `json:"user_id"`
	RelayName    string `json:"relay_name"`
	RelayBaseURL string `json:"relay_base_url"`
	CreatedUser  bool   `json:"created_user"`
}

func (h *Handler) exchange(c *gin.Context) {
	var req exchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ValidationError(c, err)
		return
	}
	result, err := h.svc.ExchangeSSO(c.Request.Context(), req.RelayName, req.DashboardToken)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	sessionToken, err := utils.GenerateToken(utils.TokenClaims{
		UserID:    result.UserID,
		TokenType: "access",
	}, h.jwtSecret, h.jwtExpiry)
	if err != nil {
		utils.HandleError(c, err)
		return
	}
	utils.Success(c, http.StatusOK, "sso exchange succeeded", exchangeResponse{
		SessionToken: sessionToken,
		UserID:       result.UserID,
		RelayName:    result.RelayName,
		RelayBaseURL: result.RelayBaseURL,
		CreatedUser:  result.CreatedUser,
	})
}

func (h *Handler) me(c *gin.Context) {
	userID := 0
	if v, ok := c.Get("userID"); ok {
		userID = v.(int)
	}
	utils.Success(c, http.StatusOK, "ok", gin.H{
		"user_id":      userID,
		"relay_module": "enabled",
	})
}
