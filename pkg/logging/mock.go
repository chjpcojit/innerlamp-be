package logging

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockLogger is a mock implementation of Logger for testing
type MockLogger struct {
	mock.Mock
}

// Debug mocks the Debug method
func (m *MockLogger) Debug(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// Info mocks the Info method
func (m *MockLogger) Info(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// Warn mocks the Warn method
func (m *MockLogger) Warn(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// Error mocks the Error method
func (m *MockLogger) Error(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// Fatal mocks the Fatal method
func (m *MockLogger) Fatal(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// Panic mocks the Panic method
func (m *MockLogger) Panic(msg string, keysAndValues ...interface{}) {
	m.Called(msg, keysAndValues)
}

// WithFields mocks the WithFields method
func (m *MockLogger) WithFields(fields Fields) Logger {
	args := m.Called(fields)
	if logger := args.Get(0); logger != nil {
		return logger.(Logger)
	}
	return m
}

// WithContext mocks the WithContext method
func (m *MockLogger) WithContext(ctx context.Context) Logger {
	args := m.Called(ctx)
	if logger := args.Get(0); logger != nil {
		return logger.(Logger)
	}
	return m
}

// WithError mocks the WithError method
func (m *MockLogger) WithError(err error) Logger {
	args := m.Called(err)
	if logger := args.Get(0); logger != nil {
		return logger.(Logger)
	}
	return m
}
