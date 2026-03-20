package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func Run() error {
	// Добавить логгер
	// Добавить config.go

	r := chi.NewRouter()

	srv := &http.Server{
		Addr:         "addr",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	//r.Post("/api/user/register")

	return srv.ListenAndServe()
}
