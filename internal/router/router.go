package router

import (
	"chat-session/internal/session"
	"github.com/go-chi/chi/v5"
)

func InitRouter(ssService session.Service) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/online/{username}", ssService.Online)
	return r
}
