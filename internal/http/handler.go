package http

import (
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"gophermart/internal/auth"
	v1 "gophermart/internal/http/v1"
	"gophermart/internal/service"
	"net/http"
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

//TODO config?
func (h *Handler) Init(/*cfg *config.Config*/) *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		middleware.Compress(5),
	)

	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		//c.String(http.StatusOK, "pong") // TODO Ping
	})

	h.initAPI(router)

	return router
}

func (h *Handler) initAPI(router chi.Router) {
	handlerV1 := v1.NewHandler(h.services, h.tokenManager)
	router.Route("/api", func(r chi.Router) {
		handlerV1.Init(r)
	})
}