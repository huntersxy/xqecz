package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
)

func New(deps app.Deps, mediaRegistrar func(*gin.Engine), registrar func(*gin.RouterGroup)) *gin.Engine {
	if deps.Cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), requestLogger(), CORS(deps.Cfg.CORSOrigins))

	if mediaRegistrar != nil {
		mediaRegistrar(r)
	}

	r.NoRoute(func(c *gin.Context) { Fail(c, 404, "接口不存在") })

	if registrar != nil {
		registrar(r.Group("/api"))
	}
	return r
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		cost := time.Since(start)
		if c.Writer.Status() >= 400 {
			slog.Warn("http",
				"req", c.Request.Method+" "+c.Request.URL.Path,
				"status", c.Writer.Status(),
				"cost", cost,
			)
		} else if cost >= 500*time.Millisecond {
			slog.Info("http slow",
				"req", c.Request.Method+" "+c.Request.URL.Path,
				"status", c.Writer.Status(),
				"cost", cost,
			)
		}
	}
}

func Run(deps app.Deps, build func(app.Deps) *gin.Engine) error {
	ln, port, err := listenAdaptive(deps.Cfg.Port)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           build(deps),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", fmt.Sprintf("http://localhost:%d", port), "env", deps.Cfg.Env)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signalContext()
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func listenAdaptive(preferred int) (net.Listener, int, error) {
	for p := preferred; p < preferred+50; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			return ln, p, nil
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			return nil, 0, err
		}
		slog.Warn("port in use, trying next", "port", p)
	}
	return nil, 0, fmt.Errorf("no free port in range %d-%d", preferred, preferred+49)
}

func Check(deps app.Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if deps.Redis != nil {
		if err := deps.Redis.Ping(ctx); err != nil {
			slog.Warn("redis unreachable", "err", err)
		} else {
			slog.Info("redis connected")
		}
	}
	if deps.DB != nil {
		if sqlDB, err := deps.DB.DB(); err == nil {
			if err := sqlDB.PingContext(ctx); err != nil {
				slog.Warn("mysql unreachable", "err", err)
			} else {
				slog.Info("mysql connected", "database", deps.Cfg.MySQL.Database)
			}
		}
	}
}
