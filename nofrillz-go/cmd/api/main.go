package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"nofrillz/internal/api"
	"nofrillz/internal/app"
	"nofrillz/internal/config"
	"nofrillz/internal/logging"
)

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Admin-API-Key")
		w.Header().Set("Access-Control-Max-Age", "600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	args := os.Args[1:]
	if len(args) != 1 {
		panic(fmt.Errorf("no configuration file provided"))
	}

	config, err := config.NewConfig(args[0])
	if err != nil {
		panic(fmt.Errorf("error in config.NewConfig: %w", err))
	}

	logger, level, levelErr := logging.NewLogger("api", config.LogConfig().Level)
	if levelErr != nil {
		logger.Warn().Str("configured_level", config.LogConfig().Level).Msg("invalid log level configured, defaulting to info")
	}
	logger.Info().
		Str("addr", config.ApiConfig().Address).
		Str("level", level.String()).
		Msg("starting api")

	app, err := app.New(config, &logger)
	if err != nil {
		panic(err)
	}

	serveMux := http.NewServeMux()
	middleware := func(next http.HandlerFunc) http.HandlerFunc {
		return api.Middleware(app, next)
	}

	api.NewAIModelHandler(app).AddRoutes(serveMux, middleware)
	healthHandler := api.NewHealthHandler(app)
	healthHandler.AddRoutes(serveMux, middleware)

	topicsHandler := api.NewTopicsHandler(app)
	topicsHandler.AddRoutes(serveMux, middleware)

	bookmarksHandler := api.NewBookmarksHandler(app)
	bookmarksHandler.AddRoutes(serveMux, middleware)

	usersHandler := api.NewUsersHandler(app)
	usersHandler.AddRoutes(serveMux, middleware)

	sessionsHandler := api.NewSessionsHandler(app)
	sessionsHandler.AddRoutes(serveMux, middleware)

	feedHandler := api.NewFeedHandler(app)
	feedHandler.AddRoutes(serveMux, middleware)

	postsHandler := api.NewPostsHandler(app)
	postsHandler.AddRoutes(serveMux, middleware)

	adminHandler := api.NewAdminHandler(app)
	adminHandler.AddRoutes(serveMux, middleware)

	server := &http.Server{
		Addr:           config.ApiConfig().Address,
		Handler:        enableCORS(serveMux),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

// package main

// import (
// 	"context"
// 	"log"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"nofrillz/internal/app"
// 	"nofrillz/internal/config"
// 	"nofrillz/internal/httpapi"
// )

// func main() {
// 	cfg := config.MustLoad()
// 	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)

// 	a, err := app.New(cfg, logger)
// 	if err != nil {
// 		logger.Fatal(err)
// 	}
// 	defer a.Close()

// 	h := httpapi.NewRouter(a)

// 	srv := &http.Server{
// 		Addr:    cfg.HTTPAddr,
// 		Handler: h,
// 	}

// 	errCh := make(chan error, 1)
// 	go func() {
// 		errCh <- srv.ListenAndServe()
// 	}()

// 	stop := make(chan os.Signal, 1)
// 	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

// 	select {
// 	case <-stop:
// 	case err := <-errCh:
// 		logger.Println(err)
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	_ = srv.Shutdown(ctx)
// }
