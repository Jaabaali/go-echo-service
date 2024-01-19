package service

import (
	"context"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	ddotel "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentelemetry"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func (svc *service) SetupOtel(ctx context.Context) (func(), error) {
	tp := ddotel.NewTracerProvider(
		tracer.WithSampler(tracer.NewRateSampler(svc.otelSampleRate)),
		tracer.WithLogger(dataDogLogger{svc.logr}),
		tracer.WithService(svc.name),
	)
	// Register the trace provider as global
	otel.SetTracerProvider(tp)

	// Set global propagator to tracecontext
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		// Shutdown the exporter to ensure traces are delivered
		_ = tp.Shutdown()
	}, nil
}

type dataDogLogger struct {
	logr logr.Logger
}

func (dl dataDogLogger) Log(msg string) {
	dl.logr.Error(nil, msg, "tracer", "datadog")
}
