package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Level defines the log level
type Level int

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in production.
	DebugLevel Level = iota
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	case PanicLevel:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// Fields is a map of key-value pairs to log
type Fields map[string]interface{}

// Formatter interface for log formatting
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

// Logger interface defines the behavior of the logger
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	Panic(msg string, keysAndValues ...interface{})

	WithFields(fields Fields) Logger
	WithContext(ctx context.Context) Logger
	WithError(err error) Logger
}

// Config holds the configuration for the logger
type Config struct {
	Level              Level
	Formatter          Formatter
	Output             io.Writer
	CorrelationIDField string
	ServiceName        string
	Environment        string
}

// logger implements the Logger interface
type logger struct {
	config Config
	pool   *sync.Pool
	fields Fields
	ctx    context.Context
}

// Entry represents a log entry
type Entry struct {
	Level         Level
	Time          time.Time
	Message       string
	Fields        Fields
	Context       context.Context
	CallerFile    string
	CallerLine    int
	CallerFunc    string
	CorrelationID string
}

// New creates a new Logger instance
func New(config Config) Logger {
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.CorrelationIDField == "" {
		config.CorrelationIDField = "correlation_id"
	}

	return &logger{
		config: config,
		pool: &sync.Pool{
			New: func() interface{} {
				return new(Entry)
			},
		},
		fields: make(Fields),
		ctx:    context.Background(),
	}
}

func (l *logger) log(level Level, msg string, keysAndValues ...interface{}) {
	if level < l.config.Level {
		return
	}

	entry := l.pool.Get().(*Entry)
	defer l.pool.Put(entry)

	entry.Level = level
	entry.Time = time.Now()
	entry.Message = msg
	entry.Context = l.ctx

	// Capture caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		entry.CallerFile = file
		entry.CallerLine = line
		if fn := runtime.FuncForPC(pc); fn != nil {
			entry.CallerFunc = fn.Name()
		}
	}

	// Initialize fields with logger's persistent fields
	entry.Fields = make(Fields, len(l.fields)+len(keysAndValues)/2+1)
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add structured fields from keysAndValues
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				key = fmt.Sprintf("%v", keysAndValues[i])
			}
			entry.Fields[key] = keysAndValues[i+1]
		}
	}

	// Add service name and environment if configured
	if l.config.ServiceName != "" {
		entry.Fields["service"] = l.config.ServiceName
	}
	if l.config.Environment != "" {
		entry.Fields["environment"] = l.config.Environment
	}

	// Extract or generate correlation ID
	correlationID := l.getOrCreateCorrelationID()
	if correlationID != "" {
		entry.CorrelationID = correlationID
		entry.Fields[l.config.CorrelationIDField] = correlationID
	}

	// Format and write
	if l.config.Formatter != nil {
		data, err := l.config.Formatter.Format(entry)
		if err == nil {
			_, _ = l.config.Output.Write(data) //nolint:errcheck
		} else {
			// Fallback to simple print if formatter fails
			fmt.Fprintf(l.config.Output, "Error formatting log: %v\n", err)
			fmt.Fprintf(l.config.Output, "%s: %s\n", level, msg)
		}
	} else {
		// Default simple format
		fmt.Fprintf(l.config.Output, "%s [%s] %s %v\n", entry.Time.Format(time.RFC3339), level, msg, entry.Fields)
	}

	if level == FatalLevel {
		os.Exit(1)
	}
	if level == PanicLevel {
		panic(msg)
	}
}

func (l *logger) Debug(msg string, keysAndValues ...interface{}) {
	l.log(DebugLevel, msg, keysAndValues...)
}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	l.log(InfoLevel, msg, keysAndValues...)
}

func (l *logger) Warn(msg string, keysAndValues ...interface{}) {
	l.log(WarnLevel, msg, keysAndValues...)
}

func (l *logger) Error(msg string, keysAndValues ...interface{}) {
	l.log(ErrorLevel, msg, keysAndValues...)
}

func (l *logger) Fatal(msg string, keysAndValues ...interface{}) {
	l.log(FatalLevel, msg, keysAndValues...)
}

func (l *logger) Panic(msg string, keysAndValues ...interface{}) {
	l.log(PanicLevel, msg, keysAndValues...)
}

func (l *logger) WithFields(fields Fields) Logger {
	newFields := make(Fields, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &logger{
		config: l.config,
		pool:   l.pool,
		fields: newFields,
		ctx:    l.ctx,
	}
}

func (l *logger) WithContext(ctx context.Context) Logger {
	return &logger{
		config: l.config,
		pool:   l.pool,
		fields: l.fields,
		ctx:    ctx,
	}
}

func (l *logger) WithError(err error) Logger {
	if err == nil {
		return l
	}
	return l.WithFields(Fields{"error": err.Error()})
}

// getOrCreateCorrelationID extracts correlation ID from context or generates a new one
func (l *logger) getOrCreateCorrelationID() string {
	if l.ctx == nil {
		// No context, generate new correlation ID
		return uuid.New().String()
	}

	// Try standard key "correlation_id" first (used by middleware)
	if val := l.ctx.Value("correlation_id"); val != nil {
		if cid, ok := val.(string); ok && cid != "" {
			return cid
		}
	}

	// Fallback: try configured field name if different
	if l.config.CorrelationIDField != "correlation_id" {
		if val := l.ctx.Value(l.config.CorrelationIDField); val != nil {
			if cid, ok := val.(string); ok && cid != "" {
				return cid
			}
		}
	}

	// If no correlation ID found in context, generate a new one
	return uuid.New().String()
}
