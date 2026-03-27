package router

import (
	"gophermart/internal/auth"
	"gophermart/internal/config"
	db "gophermart/internal/db/connections"
	"gophermart/internal/handler"
	"gophermart/internal/loger"
	"gophermart/internal/services"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
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

	loger.Log.Info("Starting server",
		zap.String("addr", cfg.RunAddress),
		zap.String("db", cfg.DataBaseURI))

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
		Addr:         cfg.RunAddress,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	r.Use(loger.RequestLogger)

	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", h.RegisterUser)
		r.Post("/api/user/login", h.UserAuth)
	})

	// С аутентификацией
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Post("/api/user/orders", h.SaveOrder)
	})

	return srv.ListenAndServe()
}
