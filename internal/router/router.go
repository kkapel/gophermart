package router

import (
	"gophermart/internal/accrual"
	"gophermart/internal/auth"
	"gophermart/internal/config"
	db "gophermart/internal/db/connections"
	"gophermart/internal/handler"
	"gophermart/internal/loger"
	"gophermart/internal/services"
	"log/slog"
	"net/http"
	"time"
)

func Run() error {
	// логгер
	if err := loger.Initialize("INFO"); err != nil {
		return err
	}
	loger.Log.Info("Router module starts")

	//config.go
	cfg, err := config.CreateConfig()

	loger.Log.Info("Starting server",
		slog.String("addr", cfg.RunAddress),
		slog.String("db", cfg.DataBaseURI))

	if err != nil {
		return err
	}

	//db init
	db, err := db.InitDB(cfg.DataBaseURI, "file://migrations")
	if err != nil {
		return err
	}

	accrual := accrual.NewAccrual(cfg.AccrualSystemAddress)
	service := services.CreateGophermartService(db.GetSqlDb(), accrual)

	h := &handler.Handler{
		Cfg: cfg,
		Srv: service,
	}

	//r := chi.NewRouter()
	mux := http.NewServeMux()

	// Роутер
	// Эндпойнты без аутентификации
	mux.HandleFunc("POST /api/user/register", h.RegisterUser)
	mux.HandleFunc("POST /api/user/login", h.UserAuth)

	// Эндпойнты с аутентификацией
	mux.Handle("POST /api/user/orders", authMiddleware(h.SaveOrder))
	mux.Handle("POST /api/user/balance/withdraw", authMiddleware(h.Withdraw))
	mux.Handle("GET /api/user/orders", authMiddleware(h.GetOrders))
	mux.Handle("GET /api/user/balance", authMiddleware(h.GetUserBalance))
	mux.Handle("GET /api/user/withdrawals", authMiddleware(h.GetWithdrawals))

	logerMux := loger.RequestLogger(mux)

	srv := &http.Server{
		Addr:         cfg.RunAddress,
		Handler:      logerMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	/*
		r.Use(loger.RequestLogger)

		r.Group(func(r chi.Router) {
			r.Post("/api/user/register", h.RegisterUser)
			r.Post("/api/user/login", h.UserAuth)
		})

		// С аутентификацией

			r.Group(func(r chi.Router) {
				r.Use(auth.AuthMiddleware)
				r.Post("/api/user/orders", h.SaveOrder)
				r.Post("/api/user/balance/withdraw", h.Withdraw)
				r.Get("/api/user/orders", h.GetOrders)
				r.Get("/api/user/balance", h.GetUserBalance)
				r.Get("/api/user/withdrawals", h.GetWithdrawals)
			})

	*/

	go service.GetOrdersForAccrual() // Фоновое задание на запросы в accrual

	return srv.ListenAndServe()
}

func authMiddleware(f func(http.ResponseWriter, *http.Request)) http.Handler {

	return auth.AuthMiddleware(http.HandlerFunc(f))
}
