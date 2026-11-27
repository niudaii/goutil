// Package serverx provides standardized HTTP server lifecycle management with
// signal.NotifyContext graceful shutdown, /healthz probes, panic recovery,
// and security/cache response header middleware.
package serverx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 30 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultMaxHeaderBytes    = 1 << 20
)

// ServerConfig configures Run for graceful HTTP server execution.
type ServerConfig struct {
	Addr              string
	Listener          net.Listener
	Handler           http.Handler
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	ShutdownTimeout   time.Duration
	OnListen          func(net.Addr)
	OnShutdown        func()
}

// SignalContext returns a context canceled on SIGINT or SIGTERM.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// Run starts an HTTP server with safe timeouts and gracefully shuts it down when ctx is canceled.
func Run(ctx context.Context, cfg ServerConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Handler == nil {
		return errors.New("serverx: Handler is required")
	}

	ln := cfg.Listener
	if ln == nil {
		addr := cfg.Addr
		if addr == "" {
			addr = ":8080"
		}
		var err error
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return err
		}
	}
	defer ln.Close()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           cfg.Handler,
		ReadHeaderTimeout: durationOr(cfg.ReadHeaderTimeout, defaultReadHeaderTimeout),
		ReadTimeout:       durationOr(cfg.ReadTimeout, defaultReadTimeout),
		WriteTimeout:      durationOr(cfg.WriteTimeout, defaultWriteTimeout),
		IdleTimeout:       durationOr(cfg.IdleTimeout, defaultIdleTimeout),
		MaxHeaderBytes:    intOr(cfg.MaxHeaderBytes, defaultMaxHeaderBytes),
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()

	if cfg.OnListen != nil {
		cfg.OnListen(ln.Addr())
	}

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownTimeout := durationOr(cfg.ShutdownTimeout, defaultShutdownTimeout)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		if cfg.OnShutdown != nil {
			cfg.OnShutdown()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func durationOr(v, fallback time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return fallback
}

func intOr(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
