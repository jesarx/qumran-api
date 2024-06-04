package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"qumran.jesarx.com/internal/data"

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
	env  string
	port int
}

type application struct {
	logger *slog.Logger
	config config
	models data.Models
}

func main() {
	viper.SetConfigFile("../../config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("yaml config file not found: %s ", err))
	}
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Enviroment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", viper.GetString("database.dsn"), "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-iddle-conns", 25, "PostgreSQL max iddle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-iddle-time", 1*time.Minute, "PostgreSQL max connection idle time")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	logger.Info("database connection pool stablished")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting server", "addr", srv.Addr, "env", cfg.env)

	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
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
