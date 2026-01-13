// Package main provides a production-ready single binary deployment example
// for Beluga AI applications. This example demonstrates:
// - Proper application lifecycle management
// - Health check endpoints (liveness, readiness, startup)
// - Graceful shutdown with request draining
// - OpenTelemetry integration for observability
// - Configuration via environment variables
// - Prometheus metrics exposure
//
// Build and run:
//
//	go build -o ai-service ./main.go
//	OPENAI_API_KEY=sk-... ./ai-service
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	// Import LLM providers to register them
	_ "github.com/lookatitude/beluga-ai/pkg/llms/providers/openai"
)

// =============================================================================
// Build Information (set via ldflags)
// =============================================================================

var (
	version   = "dev"
	buildTime = "unknown"
)

// =============================================================================
// Configuration
// =============================================================================

// Config holds all application configuration.
type Config struct {
	// Server settings
	Port            string
	MetricsPort     string
	ShutdownTimeout time.Duration

	// Observability
	OTELEndpoint string
	ServiceName  string
	LogLevel     string
	LogFormat    string

	// LLM settings
	LLMProvider string
	LLMModel    string
	LLMTimeout  time.Duration
	LLMAPIKey   string
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		MetricsPort:     getEnv("METRICS_PORT", "9090"),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),

		OTELEndpoint: getEnv("OTEL_ENDPOINT", ""),
		ServiceName:  getEnv("OTEL_SERVICE_NAME", "ai-service"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		LogFormat:    getEnv("LOG_FORMAT", "json"),

		LLMProvider: getEnv("LLM_PROVIDER", "openai"),
		LLMModel:    getEnv("LLM_MODEL", "gpt-4"),
		LLMTimeout:  getDurationEnv("LLM_TIMEOUT", 30*time.Second),
		LLMAPIKey:   os.Getenv("OPENAI_API_KEY"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// =============================================================================
// Application
// =============================================================================

// App represents the main application.
type App struct {
	config         *Config
	logger         *slog.Logger
	tracer         trace.Tracer
	meter          metric.Meter
	llmProvider    iface.ChatModel
	httpServer     *http.Server
	metricsServer  *http.Server
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider

	// Health state
	ready   atomic.Bool
	healthy atomic.Bool

	// Request tracking
	activeRequests atomic.Int64

	// Metrics
	requestCounter metric.Int64Counter
	latencyHist    metric.Float64Histogram
}

// NewApp creates a new application instance.
func NewApp(config *Config) (*App, error) {
	app := &App{
		config: config,
	}

	// Initialize logger
	app.initLogger()

	// Initialize observability
	if err := app.initObservability(); err != nil {
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}

	// Initialize LLM provider
	if err := app.initLLM(); err != nil {
		return nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// Initialize HTTP servers
	app.initHTTPServers()

	// Mark as healthy (but not ready until fully started)
	app.healthy.Store(true)

	return app, nil
}

func (a *App) initLogger() {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLogLevel(a.config.LogLevel),
	}

	if a.config.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	a.logger = slog.New(handler)
	slog.SetDefault(a.logger)
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (a *App) initObservability() error {
	ctx := context.Background()

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(a.config.ServiceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return err
	}

	// Initialize tracing if endpoint is configured
	if a.config.OTELEndpoint != "" {
		exporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(a.config.OTELEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return err
		}

		a.tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(a.tracerProvider)
	}

	a.tracer = otel.Tracer(a.config.ServiceName)

	// Initialize metrics
	promExporter, err := prometheus.New()
	if err != nil {
		return err
	}

	a.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(a.meterProvider)
	a.meter = a.meterProvider.Meter(a.config.ServiceName)

	// Create metrics
	a.requestCounter, err = a.meter.Int64Counter(
		"ai_requests_total",
		metric.WithDescription("Total number of requests"),
	)
	if err != nil {
		return err
	}

	a.latencyHist, err = a.meter.Float64Histogram(
		"ai_request_duration_seconds",
		metric.WithDescription("Request duration in seconds"),
	)
	if err != nil {
		return err
	}

	// Initialize Beluga LLM metrics
	llms.InitMetrics(a.meter, a.tracer)

	return nil
}

func (a *App) initLLM() error {
	if a.config.LLMAPIKey == "" {
		a.logger.Warn("No LLM API key configured, using mock responses")
		return nil
	}

	config := llms.NewConfig(
		llms.WithProvider(a.config.LLMProvider),
		llms.WithModelName(a.config.LLMModel),
		llms.WithAPIKey(a.config.LLMAPIKey),
		llms.WithTimeout(a.config.LLMTimeout),
	)

	provider, err := llms.NewProvider(context.Background(), a.config.LLMProvider, config)
	if err != nil {
		return err
	}

	a.llmProvider = provider
	return nil
}

func (a *App) initHTTPServers() {
	// Main HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleRoot)
	mux.HandleFunc("/chat", a.handleChat)
	mux.HandleFunc("/health/live", a.handleLiveness)
	mux.HandleFunc("/health/ready", a.handleReadiness)
	mux.HandleFunc("/health/startup", a.handleStartup)

	a.httpServer = &http.Server{
		Addr:         ":" + a.config.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", a.getMetricsHandler())

	a.metricsServer = &http.Server{
		Addr:    ":" + a.config.MetricsPort,
		Handler: metricsMux,
	}
}

func (a *App) getMetricsHandler() http.Handler {
	// The prometheus exporter registers itself with the default promhttp handler
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Collect metrics
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("# Metrics endpoint\n"))
		// In production, use promhttp.Handler() from prometheus/promhttp
	})
}

// =============================================================================
// HTTP Handlers
// =============================================================================

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := map[string]string{
		"service": a.config.ServiceName,
		"version": version,
		"status":  "running",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *App) handleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Track active requests
	a.activeRequests.Add(1)
	defer a.activeRequests.Add(-1)

	// Start tracing span
	var span trace.Span
	if a.tracer != nil {
		ctx, span = a.tracer.Start(ctx, "handleChat",
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.Path),
			),
		)
		defer span.End()
	}

	// Check readiness
	if !a.ready.Load() {
		a.recordRequest(ctx, "error", "not_ready", start)
		http.Error(w, "Service not ready", http.StatusServiceUnavailable)
		return
	}

	// Get query
	query := r.URL.Query().Get("q")
	if query == "" {
		a.recordRequest(ctx, "error", "bad_request", start)
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	if span != nil {
		span.SetAttributes(attribute.String("query", query))
	}

	// Check for LLM provider
	if a.llmProvider == nil {
		// Mock response when no provider
		response := map[string]string{
			"response": fmt.Sprintf("Echo: %s (no LLM configured)", query),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		a.recordRequest(ctx, "success", "mock", start)
		return
	}

	// Generate response
	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant. Be concise."),
		schema.NewHumanMessage(query),
	}

	result, err := a.llmProvider.Generate(ctx, messages)
	if err != nil {
		a.logger.Error("LLM generation failed", "error", err)
		a.recordRequest(ctx, "error", "llm_error", start)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"response": result.GetContent(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	a.recordRequest(ctx, "success", "llm", start)
}

func (a *App) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if a.healthy.Load() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT OK"))
	}
}

func (a *App) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if a.ready.Load() {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("NOT READY"))
	}
}

func (a *App) handleStartup(w http.ResponseWriter, r *http.Request) {
	// Same as readiness for this example
	a.handleReadiness(w, r)
}

func (a *App) recordRequest(ctx context.Context, status, source string, start time.Time) {
	duration := time.Since(start)

	if a.requestCounter != nil {
		a.requestCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("status", status),
				attribute.String("source", source),
			),
		)
	}

	if a.latencyHist != nil {
		a.latencyHist.Record(ctx, duration.Seconds(),
			metric.WithAttributes(
				attribute.String("status", status),
			),
		)
	}
}

// =============================================================================
// Lifecycle Management
// =============================================================================

// Start starts the application.
func (a *App) Start(ctx context.Context) error {
	a.logger.Info("Starting application",
		"version", version,
		"build_time", buildTime,
		"port", a.config.Port,
		"metrics_port", a.config.MetricsPort,
	)

	// Start metrics server
	go func() {
		a.logger.Info("Starting metrics server", "port", a.config.MetricsPort)
		if err := a.metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("Metrics server error", "error", err)
		}
	}()

	// Start main server
	go func() {
		a.logger.Info("Starting HTTP server", "port", a.config.Port)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			a.logger.Error("HTTP server error", "error", err)
		}
	}()

	// Mark as ready
	a.ready.Store(true)
	a.logger.Info("Application ready")

	return nil
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("Shutting down application")

	// Mark as not ready (stop accepting new requests)
	a.ready.Store(false)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, a.config.ShutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup

	// Shutdown HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.logger.Info("Stopping HTTP server")
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("HTTP server shutdown error", "error", err)
		}
	}()

	// Shutdown metrics server
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.logger.Info("Stopping metrics server")
		if err := a.metricsServer.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("Metrics server shutdown error", "error", err)
		}
	}()

	// Wait for servers to stop
	wg.Wait()

	// Log active requests
	if active := a.activeRequests.Load(); active > 0 {
		a.logger.Info("Waiting for in-flight requests", "count", active)
	}

	// Flush telemetry
	a.logger.Info("Flushing telemetry")
	if a.tracerProvider != nil {
		if err := a.tracerProvider.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("Tracer shutdown error", "error", err)
		}
	}
	if a.meterProvider != nil {
		if err := a.meterProvider.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("Meter shutdown error", "error", err)
		}
	}

	a.logger.Info("Shutdown complete")
	return nil
}

// =============================================================================
// Main
// =============================================================================

func main() {
	// Load configuration
	config := LoadConfig()

	// Create application
	app, err := NewApp(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create application: %v\n", err)
		os.Exit(1)
	}

	// Create context that listens for signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		app.logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	}()

	// Start application
	if err := app.Start(ctx); err != nil {
		app.logger.Error("Failed to start application", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	<-ctx.Done()

	// Shutdown
	if err := app.Shutdown(context.Background()); err != nil {
		app.logger.Error("Shutdown error", "error", err)
		os.Exit(1)
	}
}
