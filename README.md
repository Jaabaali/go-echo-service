### Echo server wrapper for the [Echo](https://echo.labstack.com/)
- Integrates OTEL tracing, metrics and logging in a coherent way

## Usage
```go
package main

import (
	"context"
	"net/http"
	"os"
	"time"

	svc "github.com/aweis89/go-echo-service"
	"github.com/labstack/echo/v4/middleware"
	flag "github.com/spf13/pflag"
)

func main() {
	addr := flag.String("listen", ":8080", "listen address")
	otelHost := flag.String("otel-host", os.Getenv("DD_AGENT_HOST"), "otel collector host")
	otelPort := flag.String("otel-grpc-port", "4317", "otel collector grpc port")
	otelSampleRate := flag.Float64("otel-sample-rate", 0.2, "otel sample rate")
	flag.Parse()

	otelEndpoint := *otelHost + ":" + *otelPort
	service := svc.NewService("service-tts",
		svc.WithOtelEndpoint(otelEndpoint),
		svc.WithSampleRate(*otelSampleRate),
	)
	ctx := context.Background()

	// get router logr and shutdown function
	router, logr, shutdown := service.Setup(ctx)
	defer shutdown()

	// add middleware
	router.Use(middleware.CORS())

	logr.Info("starting server")
	// with graceful shutdown
	service.Start(ctx, &http.Server{
		Addr:              *addr,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	})
}
```
