// Package mock provides mock implementations for testing
package mock

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/monitoring/iface"
)

// MockMonitor provides a mock implementation of the Monitor interface for testing.
type MockMonitor struct {
	LoggerValue               iface.Logger
	TracerValue               iface.Tracer
	MetricsValue              iface.MetricsCollector
	HealthCheckerValue        iface.HealthChecker
	SafetyCheckerValue        iface.SafetyChecker
	EthicalCheckerValue       iface.EthicalChecker
	BestPracticesCheckerValue iface.BestPracticesChecker
	StartCalled               bool
	StopCalled                bool
	IsHealthyValue            bool
}

// NewMockMonitor creates a new mock monitor with default mock implementations.
func NewMockMonitor() *MockMonitor {
	return &MockMonitor{
		LoggerValue:               &MockLogger{},
		TracerValue:               &MockTracer{},
		MetricsValue:              &MockMetricsCollector{},
		HealthCheckerValue:        &MockHealthChecker{},
		SafetyCheckerValue:        &MockSafetyChecker{},
		EthicalCheckerValue:       &MockEthicalChecker{},
		BestPracticesCheckerValue: &MockBestPracticesChecker{},
		IsHealthyValue:            true,
	}
}

// Core interface implementations.
func (m *MockMonitor) Logger() iface.Logger                 { return m.LoggerValue }
func (m *MockMonitor) Tracer() iface.Tracer                 { return m.TracerValue }
func (m *MockMonitor) Metrics() iface.MetricsCollector      { return m.MetricsValue }
func (m *MockMonitor) HealthChecker() iface.HealthChecker   { return m.HealthCheckerValue }
func (m *MockMonitor) SafetyChecker() iface.SafetyChecker   { return m.SafetyCheckerValue }
func (m *MockMonitor) EthicalChecker() iface.EthicalChecker { return m.EthicalCheckerValue }
func (m *MockMonitor) BestPracticesChecker() iface.BestPracticesChecker {
	return m.BestPracticesCheckerValue
}

// Lifecycle management.
func (m *MockMonitor) Start(ctx context.Context) error {
	m.StartCalled = true
	return nil
}

func (m *MockMonitor) Stop(ctx context.Context) error {
	m.StopCalled = true
	return nil
}

func (m *MockMonitor) IsHealthy(ctx context.Context) bool {
	return m.IsHealthyValue
}

// MockLogger provides a mock implementation of the Logger interface.
type MockLogger struct {
	DebugCalls      []DebugCall
	InfoCalls       []InfoCall
	WarningCalls    []WarningCall
	ErrorCalls      []ErrorCall
	FatalCalls      []FatalCall
	WithFieldsCalls []WithFieldsCall
}

type DebugCall struct {
	Ctx     context.Context
	Fields  map[string]any
	Message string
}

type InfoCall struct {
	Ctx     context.Context
	Fields  map[string]any
	Message string
}

type WarningCall struct {
	Ctx     context.Context
	Fields  map[string]any
	Message string
}

type ErrorCall struct {
	Ctx     context.Context
	Fields  map[string]any
	Message string
}

type FatalCall struct {
	Ctx     context.Context
	Fields  map[string]any
	Message string
}

type WithFieldsCall struct {
	Fields map[string]any
	Return iface.ContextLogger
}

func (m *MockLogger) Debug(ctx context.Context, message string, fields ...map[string]any) {
	var fieldMap map[string]any
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	m.DebugCalls = append(m.DebugCalls, DebugCall{
		Ctx: ctx, Message: message, Fields: fieldMap,
	})
}

func (m *MockLogger) Info(ctx context.Context, message string, fields ...map[string]any) {
	var fieldMap map[string]any
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	m.InfoCalls = append(m.InfoCalls, InfoCall{
		Ctx: ctx, Message: message, Fields: fieldMap,
	})
}

func (m *MockLogger) Warning(ctx context.Context, message string, fields ...map[string]any) {
	var fieldMap map[string]any
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	m.WarningCalls = append(m.WarningCalls, WarningCall{
		Ctx: ctx, Message: message, Fields: fieldMap,
	})
}

func (m *MockLogger) Error(ctx context.Context, message string, fields ...map[string]any) {
	var fieldMap map[string]any
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	m.ErrorCalls = append(m.ErrorCalls, ErrorCall{
		Ctx: ctx, Message: message, Fields: fieldMap,
	})
}

func (m *MockLogger) Fatal(ctx context.Context, message string, fields ...map[string]any) {
	var fieldMap map[string]any
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	m.FatalCalls = append(m.FatalCalls, FatalCall{
		Ctx: ctx, Message: message, Fields: fieldMap,
	})
}

func (m *MockLogger) WithFields(fields map[string]any) iface.ContextLogger {
	return &MockContextLogger{Logger: m, Fields: fields}
}

// MockContextLogger provides a mock implementation of the ContextLogger interface.
type MockContextLogger struct {
	Logger *MockLogger
	Fields map[string]any
}

func (m *MockContextLogger) Debug(ctx context.Context, message string, fields ...map[string]any) {
	m.Logger.Debug(ctx, message, fields...)
}

func (m *MockContextLogger) Info(ctx context.Context, message string, fields ...map[string]any) {
	m.Logger.Info(ctx, message, fields...)
}

func (m *MockContextLogger) Error(ctx context.Context, message string, fields ...map[string]any) {
	m.Logger.Error(ctx, message, fields...)
}

// MockTracer provides a mock implementation of the Tracer interface.
type MockTracer struct {
	StartSpanCalls     []StartSpanCall
	FinishSpanCalls    []iface.Span
	GetSpanCalls       []string
	GetTraceSpansCalls []string
}

type StartSpanCall struct {
	Ctx     context.Context
	Return  iface.Span
	Name    string
	Options []iface.SpanOption
}

func (m *MockTracer) StartSpan(ctx context.Context, name string, opts ...iface.SpanOption) (context.Context, iface.Span) {
	call := StartSpanCall{
		Ctx: ctx, Name: name, Options: opts, Return: &MockSpan{},
	}
	m.StartSpanCalls = append(m.StartSpanCalls, call)
	return ctx, call.Return
}

func (m *MockTracer) FinishSpan(span iface.Span) {
	m.FinishSpanCalls = append(m.FinishSpanCalls, span)
}

func (m *MockTracer) GetSpan(spanID string) (iface.Span, bool) {
	m.GetSpanCalls = append(m.GetSpanCalls, spanID)
	return &MockSpan{}, true
}

func (m *MockTracer) GetTraceSpans(traceID string) []iface.Span {
	m.GetTraceSpansCalls = append(m.GetTraceSpansCalls, traceID)
	return []iface.Span{&MockSpan{}}
}

// MockSpan provides a mock implementation of the Span interface.
type MockSpan struct {
	LogCalls         []LogCall
	SetErrorCalls    []error
	SetStatusCalls   []string
	SetTagCalls      []SetTagCall
	GetDurationValue time.Duration
	IsFinishedValue  bool
}

type LogCall struct {
	Fields  map[string]any
	Message string
}

type SetTagCall struct {
	Value any
	Key   string
}

func (m *MockSpan) Log(message string, fields ...map[string]any) {
	var fieldMap map[string]any
	if len(fields) > 0 {
		fieldMap = fields[0]
	}
	m.LogCalls = append(m.LogCalls, LogCall{Message: message, Fields: fieldMap})
}

func (m *MockSpan) SetError(err error) {
	m.SetErrorCalls = append(m.SetErrorCalls, err)
}

func (m *MockSpan) SetStatus(status string) {
	m.SetStatusCalls = append(m.SetStatusCalls, status)
}

func (m *MockSpan) GetDuration() time.Duration {
	return m.GetDurationValue
}

func (m *MockSpan) IsFinished() bool {
	return m.IsFinishedValue
}

func (m *MockSpan) SetTag(key string, value any) {
	m.SetTagCalls = append(m.SetTagCalls, SetTagCall{Key: key, Value: value})
}

// MockMetricsCollector provides a mock implementation of the MetricsCollector interface.
type MockMetricsCollector struct {
	CounterCalls    []CounterCall
	GaugeCalls      []GaugeCall
	HistogramCalls  []HistogramCall
	TimingCalls     []TimingCall
	IncrementCalls  []IncrementCall
	StartTimerCalls []StartTimerCall
}

type CounterCall struct {
	Ctx         context.Context
	Labels      map[string]string
	Name        string
	Description string
	Value       float64
}

type GaugeCall struct {
	Ctx         context.Context
	Labels      map[string]string
	Name        string
	Description string
	Value       float64
}

type HistogramCall struct {
	Ctx         context.Context
	Labels      map[string]string
	Name        string
	Description string
	Value       float64
}

type TimingCall struct {
	Ctx         context.Context
	Labels      map[string]string
	Name        string
	Description string
	Duration    time.Duration
}

type IncrementCall struct {
	Ctx         context.Context
	Labels      map[string]string
	Name        string
	Description string
}

type StartTimerCall struct {
	Ctx    context.Context
	Return iface.Timer
	Labels map[string]string
	Name   string
}

func (m *MockMetricsCollector) Counter(ctx context.Context, name, description string, value float64, labels map[string]string) {
	m.CounterCalls = append(m.CounterCalls, CounterCall{
		Ctx: ctx, Name: name, Description: description, Value: value, Labels: labels,
	})
}

func (m *MockMetricsCollector) Gauge(ctx context.Context, name, description string, value float64, labels map[string]string) {
	m.GaugeCalls = append(m.GaugeCalls, GaugeCall{
		Ctx: ctx, Name: name, Description: description, Value: value, Labels: labels,
	})
}

func (m *MockMetricsCollector) Histogram(ctx context.Context, name, description string, value float64, labels map[string]string) {
	m.HistogramCalls = append(m.HistogramCalls, HistogramCall{
		Ctx: ctx, Name: name, Description: description, Value: value, Labels: labels,
	})
}

func (m *MockMetricsCollector) Timing(ctx context.Context, name, description string, duration time.Duration, labels map[string]string) {
	m.TimingCalls = append(m.TimingCalls, TimingCall{
		Ctx: ctx, Name: name, Description: description, Duration: duration, Labels: labels,
	})
}

func (m *MockMetricsCollector) Increment(ctx context.Context, name, description string, labels map[string]string) {
	m.IncrementCalls = append(m.IncrementCalls, IncrementCall{
		Ctx: ctx, Name: name, Description: description, Labels: labels,
	})
}

func (m *MockMetricsCollector) StartTimer(ctx context.Context, name string, labels map[string]string) iface.Timer {
	call := StartTimerCall{
		Ctx: ctx, Name: name, Labels: labels, Return: &MockTimer{},
	}
	m.StartTimerCalls = append(m.StartTimerCalls, call)
	return call.Return
}

// MockTimer provides a mock implementation of the Timer interface.
type MockTimer struct {
	StopCalls []StopCall
}

type StopCall struct {
	Ctx         context.Context
	Description string
}

func (m *MockTimer) Stop(ctx context.Context, description string) {
	m.StopCalls = append(m.StopCalls, StopCall{Ctx: ctx, Description: description})
}

// MockHealthChecker provides a mock implementation of the HealthChecker interface.
type MockHealthChecker struct {
	RegisterCheckCalls []RegisterCheckCall
	RunChecksCalls     []RunChecksCall
	IsHealthyCalls     []IsHealthyCall
}

type RegisterCheckCall struct {
	Return error
	Check  iface.HealthCheckFunc
	Name   string
}

type RunChecksCall struct {
	Ctx    context.Context
	Return map[string]iface.HealthCheckResult
}

type IsHealthyCall struct {
	Ctx    context.Context
	Return bool
}

func (m *MockHealthChecker) RegisterCheck(name string, check iface.HealthCheckFunc) error {
	call := RegisterCheckCall{Name: name, Check: check, Return: nil}
	m.RegisterCheckCalls = append(m.RegisterCheckCalls, call)
	return call.Return
}

func (m *MockHealthChecker) RunChecks(ctx context.Context) map[string]iface.HealthCheckResult {
	call := RunChecksCall{Ctx: ctx, Return: make(map[string]iface.HealthCheckResult)}
	m.RunChecksCalls = append(m.RunChecksCalls, call)
	return call.Return
}

func (m *MockHealthChecker) IsHealthy(ctx context.Context) bool {
	call := IsHealthyCall{Ctx: ctx, Return: true}
	m.IsHealthyCalls = append(m.IsHealthyCalls, call)
	return call.Return
}

// MockSafetyChecker provides a mock implementation of the SafetyChecker interface.
type MockSafetyChecker struct {
	CheckContentCalls       []CheckContentCall
	RequestHumanReviewCalls []RequestHumanReviewCall
}

type CheckContentCall struct {
	Ctx         context.Context
	Error       error
	Content     string
	ContextInfo string
	Return      iface.SafetyResult
}

type RequestHumanReviewCall struct {
	Ctx         context.Context
	Error       error
	Content     string
	ContextInfo string
	Return      iface.ReviewDecision
	RiskScore   float64
}

func (m *MockSafetyChecker) CheckContent(ctx context.Context, content, contextInfo string) (iface.SafetyResult, error) {
	call := CheckContentCall{
		Ctx: ctx, Content: content, ContextInfo: contextInfo,
		Return: iface.SafetyResult{Content: content, Safe: true, RiskScore: 0.0},
		Error:  nil,
	}
	m.CheckContentCalls = append(m.CheckContentCalls, call)
	return call.Return, call.Error
}

func (m *MockSafetyChecker) RequestHumanReview(ctx context.Context, content, contextInfo string, riskScore float64) (iface.ReviewDecision, error) {
	call := RequestHumanReviewCall{
		Ctx: ctx, Content: content, ContextInfo: contextInfo, RiskScore: riskScore,
		Return: iface.ReviewDecision{Approved: true},
		Error:  nil,
	}
	m.RequestHumanReviewCalls = append(m.RequestHumanReviewCalls, call)
	return call.Return, call.Error
}

// MockEthicalChecker provides a mock implementation of the EthicalChecker interface.
type MockEthicalChecker struct {
	CheckContentCalls []EthicalCheckContentCall
}

type EthicalCheckContentCall struct {
	Ctx        context.Context
	Error      error
	Content    string
	EthicalCtx iface.EthicalContext
	Return     iface.EthicalAnalysis
}

func (m *MockEthicalChecker) CheckContent(ctx context.Context, content string, ethicalCtx iface.EthicalContext) (iface.EthicalAnalysis, error) {
	call := EthicalCheckContentCall{
		Ctx: ctx, Content: content, EthicalCtx: ethicalCtx,
		Return: iface.EthicalAnalysis{Content: content, OverallRisk: "low"},
		Error:  nil,
	}
	m.CheckContentCalls = append(m.CheckContentCalls, call)
	return call.Return, call.Error
}

// MockBestPracticesChecker provides a mock implementation of the BestPracticesChecker interface.
type MockBestPracticesChecker struct {
	ValidateCalls     []ValidateCall
	AddValidatorCalls []iface.Validator
}

type ValidateCall struct {
	Ctx       context.Context
	Data      any
	Component string
	Return    []iface.ValidationIssue
}

func (m *MockBestPracticesChecker) Validate(ctx context.Context, data any, component string) []iface.ValidationIssue {
	call := ValidateCall{Ctx: ctx, Data: data, Component: component, Return: []iface.ValidationIssue{}}
	m.ValidateCalls = append(m.ValidateCalls, call)
	return call.Return
}

func (m *MockBestPracticesChecker) AddValidator(validator iface.Validator) {
	m.AddValidatorCalls = append(m.AddValidatorCalls, validator)
}
