package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/trace"
)

var ReqIDLoggerKey = "id"

func (svc *service) Start(ctx context.Context, server *http.Server) {
	// Start server
	go func() {
		server.Handler = svc.router
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			svc.logr.Error(err, "shutting down server")
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := svc.router.Shutdown(ctx); err != nil {
		svc.logr.Error(err, "shutting down server")
	}
}

func (svc *service) setup(ctx context.Context) (*echo.Echo, logr.Logger, func()) {
	shutdown, err := svc.SetupOtel(ctx)
	if err != nil {
		svc.logr.Error(err, "failed to setup otel")
		os.Exit(1)
	}

	router := echo.New()
	svc.router = router
	slog := slog.New(slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			// TODO make level configurable
			Level: slog.LevelInfo,
		},
	))
	svc.logr = logr.FromSlogHandler(slog.Handler())
	router.Use(slogecho.NewWithConfig(slog, slogecho.Config{
		WithRequestID: true,
		Filters:       []slogecho.Filter{svc.telemetryRun},
	}))
	router.Use(middleware.RequestID())
	router.Use(middleware.Recover())
	router.Use(echoprometheus.NewMiddleware("http"))
	router.Use(otelecho.Middleware(svc.name,
		otelecho.WithSkipper(svc.tracerSkip)))

	router.HTTPErrorHandler = func(err error, c echo.Context) {
		ctx := c.Request().Context()
		trace.SpanFromContext(ctx).RecordError(err)
		router.DefaultHTTPErrorHandler(err, c)
	}

	// Add logr to context with request id
	router.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqLogr := svc.logr.WithValues(ReqIDLoggerKey, c.Response().Header().Get(echo.HeaderXRequestID))
			context := logr.NewContext(c.Request().Context(), reqLogr)
			c.SetRequest(c.Request().WithContext(context))
			return next(c)
		}
	})

	router.GET(svc.metricsPath, echoprometheus.NewHandler())
	router.GET(svc.healthPath, func(c echo.Context) error {
		return c.String(http.StatusOK, "OK") //nolint:wrapcheck
	})

	svc.router = router
	return svc.router, svc.logr, shutdown
}

func (svc *service) requestLoggingConfig() slogecho.Config {
	return slogecho.Config{
		WithRequestID: true,
		Filters:       []slogecho.Filter{svc.telemetryRun},
	}
}

func (svc *service) tracerSkip(c echo.Context) bool {
	return !svc.telemetryRun(c)
}

func (svc *service) telemetryRun(c echo.Context) bool {
	skipReqLoggingPaths := []string{svc.healthPath, svc.metricsPath}
	for _, path := range skipReqLoggingPaths {
		if strings.HasPrefix(c.Path(), path) {
			return false
		}
	}
	return true
}
