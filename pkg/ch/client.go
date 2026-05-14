// Package ch wraps the official ClickHouse Go driver with focused helpers
// for the chtop TUI: clusters, databases, tables, parts, processes, query log
// and the kill action. Callers do not need to know about clickhouse-go.
package ch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// QueryIDPrefix tags every query chtop issues so the processes / query_log
// views can hide their own traffic. Public so the SQL queries here and any
// later view can reference the same constant.
const QueryIDPrefix = "chtop-"

// ErrConfig is returned when a required Config field is missing.
var ErrConfig = errors.New("missing required config")

// Config holds connection settings for one ClickHouse cluster entrypoint.
// The native (port 9000 / 9440 TLS) protocol is used.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	TLS      bool

	// mTLS files. TLSCAFile sets a custom Root CA; TLSCertFile + TLSKeyFile
	// together enable client certificate authentication.
	TLSCertFile string
	TLSKeyFile  string
	TLSCAFile   string

	DialTimeout time.Duration
	ReadTimeout time.Duration
}

// Client is the chtop-facing ClickHouse client. Safe for concurrent use.
type Client struct {
	conn driver.Conn
}

// NewClient validates cfg and opens a connection. The handshake happens
// lazily on the first query; use Ping to surface auth/TLS errors early.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("host: %w", ErrConfig)
	}
	if cfg.Port == 0 {
		cfg.Port = 9000
		if cfg.TLS {
			cfg.Port = 9440
		}
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 30 * time.Second
	}
	opts := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Username: cfg.User,
			Password: cfg.Password,
			Database: cfg.Database,
		},
		DialTimeout: cfg.DialTimeout,
		ReadTimeout: cfg.ReadTimeout,
		Settings: clickhouse.Settings{
			"max_execution_time": 30,
		},
	}
	tlsCfg, err := buildTLSConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("tls: %w", err)
	}
	if tlsCfg != nil {
		opts.TLS = tlsCfg
	}
	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Ping issues a server round-trip and returns auth / network errors.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.conn.Ping(ctx); err != nil {
		return fmt.Errorf("clickhouse ping: %w", err)
	}
	return nil
}

// Close releases the underlying connection pool.
func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// tagged returns ctx wrapped with a per-call query_id prefixed by
// QueryIDPrefix. Used by every query method on Client so we can filter our
// own traffic out of system.processes and system.query_log.
func (c *Client) tagged(ctx context.Context) context.Context {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return clickhouse.Context(ctx, clickhouse.WithQueryID(QueryIDPrefix+hex.EncodeToString(b[:])))
}
