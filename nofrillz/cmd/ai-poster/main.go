package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/aiposter"
	"nofrillz/internal/aitools"
	"nofrillz/internal/config"
	"nofrillz/internal/db"
	"nofrillz/internal/logging"
	"nofrillz/internal/posts"
	"nofrillz/internal/snowid"
	"nofrillz/internal/users"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		panic(fmt.Errorf("no configuration file provided"))
	}

	cfg, err := config.NewConfig(args[0])
	if err != nil {
		panic(fmt.Errorf("error in config.NewConfig: %w", err))
	}

	logger, level, levelErr := logging.NewLogger("ai-poster", cfg.LogConfig().Level)
	if levelErr != nil {
		logger.Warn().Str("configured_level", cfg.LogConfig().Level).Msg("invalid log level configured, defaulting to info")
	}

	mysql, err := db.NewMySQL(cfg.MySQLConfig())
	if err != nil {
		panic(err)
	}
	defer mysql.Close()

	idGenerator, err := snowid.NewDefault(cfg.IDGeneratorConfig())
	if err != nil {
		panic(err)
	}

	aiAccountsRepository := aiaccounts.NewRepository(mysql.DB)
	aiAccountsService := aiaccounts.NewService(aiAccountsRepository)
	postsRepository := posts.NewRepository(mysql.DB)
	postsService := posts.NewService(postsRepository, idGenerator)
	aiTools, err := aitools.NewFromConfig(cfg.AIToolsConfig())
	if err != nil {
		panic(err)
	}
	generator := aigenerator.NewAIToolsGenerator(aiTools, users.NewService(users.NewRepository(mysql.DB)), postsService)

	aiPosterConfig := cfg.AIPosterConfig()
	runner := aiposter.NewRunner(
		&logger,
		aiposter.NewSQLTransactionManager(mysql.DB),
		aiAccountsService,
		postsService,
		generator,
		idGenerator,
		time.Duration(aiPosterConfig.PollIntervalSeconds)*time.Second,
		aiPosterConfig.BatchSize,
		time.Duration(aiPosterConfig.StaleRunningAfterMinutes)*time.Minute,
	)

	runner.SetSchedule(aiPosterConfig.Schedule())
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info().
		Str("level", level.String()).
		Int("poll_interval_seconds", aiPosterConfig.PollIntervalSeconds).
		Int("batch_size", aiPosterConfig.BatchSize).
		Int("stale_running_after_minutes", aiPosterConfig.StaleRunningAfterMinutes).
		Msg("starting ai-poster")
	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
	logger.Info().Msg("stopped ai-poster")
}
