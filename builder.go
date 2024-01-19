package service

import (
	"github.com/go-logr/logr"
	"github.com/labstack/echo/v4"
)

type service struct {
	name           string
	otelEndpoint   string
	otelSampleRate float64
	healthPath     string
	metricsPath    string

	router *echo.Echo
	logr   logr.Logger
}

var (
	DefaultHealthPath  = "/health"
	DefaultMetricsPath = "/metrics"
	DefaultSampleRate  = 0.2
)

func NewService(serviceName string, opts ...func(*service)) *service {
	svc := &service{
		name: serviceName,
	}
	for _, opt := range opts {
		opt(svc)
	}
	if svc.healthPath == "" {
		svc.healthPath = DefaultHealthPath
	}
	if svc.metricsPath == "" {
		svc.metricsPath = DefaultMetricsPath
	}
	if svc.otelSampleRate == 0 {
		svc.otelSampleRate = DefaultSampleRate
	}
	return svc
}

func WithHealthPath(path string) func(*service) {
	return func(s *service) {
		s.healthPath = path
	}
}

func WithMetricsPath(path string) func(*service) {
	return func(s *service) {
		s.metricsPath = path
	}
}

func WithSampleRate(rate float64) func(*service) {
	return func(s *service) {
		s.otelSampleRate = rate
	}
}

func WithOtelEndpoint(endpoint string) func(*service) {
	return func(s *service) {
		s.otelEndpoint = endpoint
	}
}
