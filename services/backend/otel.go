package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)


type MultiHandler struct {
	handlers []slog.Handler
}

func InitTelemetry(cfg *Config) (cleanup func(),logger *slog.Logger,tracer trace.Tracer,err error) {
	ctx := context.Background()

	// OTEL resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracing
	traceCleanup, tracer, err := initTracing(ctx, res, cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize logging
	logCleanup, logger, err := initLogging(ctx, res, cfg)
	if err != nil {
		traceCleanup()
		return nil, nil, nil, fmt.Errorf("failed to initialize logging: %w", err)
	}

	return func() {
		logCleanup()
		traceCleanup()
	}, logger, tracer, nil
}

func initTracing(ctx context.Context, res *resource.Resource, cfg *Config) (func(), trace.Tracer, error) {
	// OTLP GRPC trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpointGRPC),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create trace exportor: %w", err)
	}

	// Trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Tracer
	tracer := otel.Tracer(cfg.ServiceName)

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error shutting down tracer provider", "error", err)
		}
	}, tracer, nil
}

func initLogging(ctx context.Context, res *resource.Resource, cfg *Config) (func(), *slog.Logger, error) {
	// OTLP GRPC log exporter
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.OtelExporterOtlpEndpointGRPC),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	// Log processor
	logProcessor := sdklog.NewBatchProcessor(logExporter)

	// Logger Provider
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(logProcessor),
		sdklog.WithResource(res),
	)

	// Set global logger provider
	global.SetLoggerProvider(loggerProvider)	

	// Handlers
	var handlers []slog.Handler
	var logHandler slog.Handler

	// OpenTelemetry slog handler
	otelHander := otelslog.NewHandler(cfg.ServiceName)
	handlers = append(handlers, otelHander)

	// check if console log is enable
	if cfg.EnableConsoleLog {
		consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		handlers = append(handlers, consoleHandler)
	}

	if len(handlers) == 1 {
		logHandler = handlers[0]
	}	else {
		logHandler = NewMultiHandler(handlers...)
	}

	// Set global logger
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := loggerProvider.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error shutting down logger provider", "error", err)
		}
	}, logger, nil
}

// Implements slog.Handler interface methods
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}

  return &MultiHandler{handlers: newHandlers}
}

func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}

var excludedPaths = map[string]bool {
	"/api/health": true,
}

func TracingMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if excludedPaths[c.FullPath()] {
			c.Next()
			return
		}
		otelgin.Middleware(serviceName)
	}
}

// Custom logging middleware that works with OpenTelemetry
func LoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		if excludedPaths[c.FullPath()] {
			c.Next()
			return
		}
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Add request context to logger
		ctx := c.Request.Context()
		
		// Get trace information if available
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		logger.InfoContext(ctx, "Request started",
			slog.String("method", method),
			slog.String("path", path),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
		)

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		logLevel := slog.LevelInfo
		if status >= 400 {
			logLevel = slog.LevelError
		} else if status >= 300 {
			logLevel = slog.LevelWarn
		}

		logger.Log(ctx, logLevel, "Request completed",
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
		)
	}
}
