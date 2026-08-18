package formatter

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"base_project/pkg/logging"
)

// TextFormatter formats logs as text
type TextFormatter struct {
	TimestampFormat string
	DisableColors   bool
}

// NewText creates a new TextFormatter
func NewText() *TextFormatter {
	return &TextFormatter{
		TimestampFormat: time.RFC3339,
	}
}

// Format renders a single log entry
func (f *TextFormatter) Format(entry *logging.Entry) ([]byte, error) {
	var b *bytes.Buffer
	// We can't easily reuse a buffer here without passing it in, so we allocate.
	// In a high-perf scenario, we might want a pool here too, but for now this is fine.
	b = &bytes.Buffer{}

	timestamp := entry.Time.Format(f.TimestampFormat)
	level := entry.Level.String()

	if !f.DisableColors {
		level = f.colorize(entry.Level, level)
	}

	fmt.Fprintf(b, "%s [%s] %s", timestamp, level, entry.Message)

	if len(entry.Fields) > 0 {
		b.WriteString(" ")
		f.writeFields(b, entry.Fields)
	}

	if entry.CorrelationID != "" {
		fmt.Fprintf(b, " correlation_id=%s", entry.CorrelationID)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *TextFormatter) writeFields(b *bytes.Buffer, fields logging.Fields) {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			b.WriteByte(' ')
		}
		v := fields[k]
		fmt.Fprintf(b, "%s=%v", k, v)
	}
}

func (f *TextFormatter) colorize(level logging.Level, text string) string {
	switch level {
	case logging.DebugLevel:
		return fmt.Sprintf("\x1b[36m%s\x1b[0m", text) // Cyan
	case logging.InfoLevel:
		return fmt.Sprintf("\x1b[32m%s\x1b[0m", text) // Green
	case logging.WarnLevel:
		return fmt.Sprintf("\x1b[33m%s\x1b[0m", text) // Yellow
	case logging.ErrorLevel:
		return fmt.Sprintf("\x1b[31m%s\x1b[0m", text) // Red
	case logging.FatalLevel, logging.PanicLevel:
		return fmt.Sprintf("\x1b[31;1m%s\x1b[0m", text) // Bold Red
	default:
		return text
	}
}
