package router

import (
	"gophermart/internal/config"
	db "gophermart/internal/db/connections"
	"gophermart/internal/handler"
	"gophermart/internal/loger"
	"gophermart/internal/services"
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

	//config.go
	cfg, err := config.CreateConfig()
	if err != nil {
		return err
	}

	//db init
	db, err := db.InitDB(cfg.DataBaseURI)
	if err != nil {
		return err
	}

	service := services.CreateGophermartService(db.GetSqlDb())

	h := &handler.Handler{
		Cfg: cfg,
		Srv: service,
	}

	r := chi.NewRouter()

	srv := &http.Server{
		Addr:         "addr",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	r.Use(loger.RequestLogger)
	r.Post("/api/user/register", h.RegisterUser)

	return srv.ListenAndServe()
}
