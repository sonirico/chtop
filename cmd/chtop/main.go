package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/sonirico/chtop/internal/tui"
	"github.com/sonirico/chtop/pkg/ch"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "chtop:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := parseFlags()
	if err != nil {
		return err
	}

	client, err := ch.NewClient(ch.Config{
		Host:     cfg.host,
		Port:     cfg.port,
		User:     cfg.user,
		Password: cfg.password,
		Database: cfg.database,
		TLS:      cfg.tls,
	})
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	app, err := tui.NewApp(tui.AppConfig{
		Cluster: cfg.cluster,
		Host:    fmt.Sprintf("%s:%d", cfg.host, cfg.port),
		Client:  client,
	})
	if err != nil {
		return err
	}
	return app.Run(ctx)
}

type cliConfig struct {
	cluster  string
	host     string
	port     int
	user     string
	password string
	database string
	tls      bool
}

func parseFlags() (cliConfig, error) {
	c := cliConfig{
		cluster:  envOr("CHTOP_CLUSTER", ""),
		host:     envOr("CHTOP_HOST", "localhost"),
		user:     envOr("CHTOP_USER", "default"),
		password: envOr("CHTOP_PASSWORD", ""),
		database: envOr("CHTOP_DATABASE", "default"),
	}
	defaultPort := 9000
	if p := os.Getenv("CHTOP_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			defaultPort = n
		}
	}
	c.port = defaultPort

	flag.StringVar(&c.cluster, "cluster", c.cluster, "cluster label shown in the header")
	flag.StringVar(&c.host, "host", c.host, "ClickHouse host (env: CHTOP_HOST)")
	flag.IntVar(&c.port, "port", c.port, "ClickHouse native port (env: CHTOP_PORT)")
	flag.StringVar(&c.user, "user", c.user, "ClickHouse user (env: CHTOP_USER)")
	flag.StringVar(&c.password, "password", c.password, "ClickHouse password (env: CHTOP_PASSWORD)")
	flag.StringVar(
		&c.database,
		"database",
		c.database,
		"ClickHouse default database (env: CHTOP_DATABASE)",
	)
	flag.BoolVar(&c.tls, "tls", c.tls, "use TLS (default port becomes 9440)")
	flag.Parse()
	return c, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
