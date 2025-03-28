package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"qumran.jesarx.com/internal/data"
	"qumran.jesarx.com/internal/mailer"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

const version = "1.0.0"

type config struct {
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	env     string
	port    int
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		tustedOrigins []string
	}
}

type application struct {
	logger *slog.Logger
	config config
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	// Config file configuration
	viper.SetConfigFile("../../config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("yaml config file not found: %s ", err))
	}

	var cfg config

	// MAIN CONFIG
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Enviroment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", viper.GetString("database.dsn"), "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-iddle-conns", 25, "PostgreSQL max iddle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-iddle-time", 1*time.Minute, "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 8, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 16, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "20372de296e321", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "121fa461714fa2", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Jesarx <jesarx@riseup.net>", "SMTP sender")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.tustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	logger.Info("database connection pool stablished")

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))

	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	// Max number of open connections in pool
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Max number of idle connection in pool
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Max idle timeout in pool
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// 5 second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
