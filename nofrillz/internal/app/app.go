package app

import (
	"github.com/rs/zerolog"

	"nofrillz/internal/admin"
	"nofrillz/internal/aiaccounts"
	"nofrillz/internal/aigenerator"
	"nofrillz/internal/aitools"
	"nofrillz/internal/apns"
	"nofrillz/internal/bookmarks"
	"nofrillz/internal/cache"
	"nofrillz/internal/config"
	"nofrillz/internal/db"
	"nofrillz/internal/feed"
	"nofrillz/internal/follows"
	"nofrillz/internal/likes"
	"nofrillz/internal/limiters"
	"nofrillz/internal/notifications"
	"nofrillz/internal/posts"
	"nofrillz/internal/sessions"
	"nofrillz/internal/snowid"
	"nofrillz/internal/users"
)

type App struct {
	Config        *config.Config
	Logger        *zerolog.Logger
	MySQL         *db.MySQL
	Redis         *cache.Redis
	IDGenerator   *snowid.Generator
	Limiters      *limiters.Limiters
	Users         *users.Service
	Posts         *posts.Service
	AIAccounts    *aiaccounts.Service
	AITools       aitools.Tools
	Admin         *admin.Service
	Feed          *feed.Service
	Follows       *follows.Service
	Likes         *likes.Service
	Bookmarks     *bookmarks.Service
	Sessions      *sessions.Service
	APNS          *apns.Service
	Notifications *notifications.Service
}

func New(config *config.Config, logger *zerolog.Logger) (*App, error) {
	mysql, err := db.NewMySQL(config.MySQLConfig())
	if err != nil {
		return nil, err
	}

	redis, err := cache.NewRedis(config.RedisConfig())
	if err != nil {
		_ = mysql.Close()
		return nil, err
	}

	idGenerator, err := snowid.NewDefault(config.IDGeneratorConfig())
	if err != nil {
		panic(err)
	}

	limiters := limiters.NewLimiters(config.RateLimitsConfig())

	usersRepository := users.NewRepository(mysql.DB)
	usersService := users.NewService(usersRepository)
	postsRepository := posts.NewRepository(mysql.DB)
	postsService := posts.NewService(postsRepository, idGenerator)
	apnsRepository := apns.NewRepository(mysql.DB)
	apnsProvider, err := apns.NewProviderFromConfig(config.APNSConfig(), logger)
	if err != nil {
		_ = redis.Close()
		_ = mysql.Close()
		return nil, err
	}
	apnsService := apns.NewService(logger, apnsRepository, apnsProvider)
	notificationsRepository := notifications.NewRepository(mysql.DB)
	notificationsService := notifications.NewService(logger, notificationsRepository, apnsService)
	aiAccountsRepository := aiaccounts.NewRepository(mysql.DB)
	aiAccountsService := aiaccounts.NewService(aiAccountsRepository)
	aiTools, err := aitools.NewFromConfig(config.AIToolsConfig())
	if err != nil {
		_ = redis.Close()
		_ = mysql.Close()
		return nil, err
	}
	aiGenerator := aigenerator.NewAIToolsGenerator(aiTools, usersService, postsService)
	adminRepository := admin.NewRepository(mysql.DB)
	adminService := admin.NewService(admin.NewSQLTransactionManager(mysql.DB), aiAccountsService, usersService, postsService, aiTools, aiGenerator, idGenerator, adminRepository)
	adminService.SetSchedule(config.AIPosterConfig().Schedule())
	feedRepository := feed.NewRepository(mysql.DB)
	feedService := feed.NewService(feedRepository)
	followsRepository := follows.NewRepository(mysql.DB)
	followsService := follows.NewService(followsRepository)
	likesRepository := likes.NewRepository(mysql.DB)
	likesService := likes.NewService(likesRepository)
	bookmarksRepository := bookmarks.NewRepository(mysql.DB)
	bookmarksService := bookmarks.NewService(bookmarksRepository, postsService)
	sessionsRepository := sessions.NewRepository(mysql.DB)
	sessionsService := sessions.NewService(sessionsRepository)

	instance := &App{
		IDGenerator:   idGenerator,
		Config:        config,
		Logger:        logger,
		MySQL:         mysql,
		Redis:         redis,
		Limiters:      limiters,
		Users:         usersService,
		Posts:         postsService,
		AIAccounts:    aiAccountsService,
		AITools:       aiTools,
		Admin:         adminService,
		Feed:          feedService,
		Follows:       followsService,
		Likes:         likesService,
		Bookmarks:     bookmarksService,
		Sessions:      sessionsService,
		APNS:          apnsService,
		Notifications: notificationsService,
	}

	if logger != nil {
		logger.Debug().Msg("application services initialized")
	}

	return instance, nil
}

func (a *App) Close() {
	if a == nil {
		return
	}
	if a.Logger != nil {
		a.Logger.Debug().Msg("closing application resources")
	}
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	if a.MySQL != nil {
		_ = a.MySQL.Close()
	}
}
