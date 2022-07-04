package v1

import (
	"github.com/go-chi/chi/v5"
	"gophermart/internal/auth"
	"gophermart/internal/service"
)

type Handler struct {
	services     *service.Services
	tokenManager auth.TokenManager
}

func NewHandler(services *service.Services, tokenManager auth.TokenManager) *Handler {
	return &Handler{
		services:     services,
		tokenManager: tokenManager,
	}
}

func (h *Handler) Init(r chi.Router) {
	h.initUserRoutes(r)
}