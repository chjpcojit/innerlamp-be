package formatter

import (
	"encoding/json"
	"time"

	"base_project/pkg/logging"
)

// JSONFormatter formats logs as JSON
type JSONFormatter struct {
	TimestampFormat string
}

// NewJSON creates a new JSONFormatter
func NewJSON() *JSONFormatter {
	return &JSONFormatter{
		TimestampFormat: time.RFC3339,
	}
}

// Format renders a single log entry
func (f *JSONFormatter) Format(entry *logging.Entry) ([]byte, error) {
	data := make(map[string]interface{}, len(entry.Fields)+4)

	for k, v := range entry.Fields {
		data[k] = v
	}

	data["level"] = entry.Level.String()
	data["time"] = entry.Time.Format(f.TimestampFormat)
	data["message"] = entry.Message

	if entry.CallerFile != "" {
		data["caller"] = map[string]interface{}{
			"file": entry.CallerFile,
			"line": entry.CallerLine,
			"func": entry.CallerFunc,
		}
	}

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return append(serialized, '\n'), nil
}
