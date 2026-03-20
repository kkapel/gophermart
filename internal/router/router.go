package router

import (
	"gophermart/internal/loger"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func Run() error {
	// логгер
	if err := loger.Initialize("INFO"); err != nil {
		return err
	}
	defer loger.Log.Sync()
	loger.Log.Info("Router module starts")
	// Добавить config.go

	r := chi.NewRouter()

	srv := &http.Server{
		Addr:         "addr",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	r.Use(loger.RequestLogger)
	//r.Post("/api/user/register")

	return srv.ListenAndServe()
}
