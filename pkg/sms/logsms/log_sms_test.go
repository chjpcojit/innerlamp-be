package logsms

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/sms"
)

func newLogger() *logging.MockLogger {
	l := &logging.MockLogger{}
	l.On("WithFields", mock.Anything).Return(l)
	l.On("WithContext", mock.Anything).Return(l)
	l.On("Info", mock.Anything, mock.Anything).Return()
	return l
}

func TestRegisteredWithProviderLog(t *testing.T) {
	logger := newLogger()
	sender, err := sms.New(&sms.Config{Provider: sms.ProviderLog, Logger: logger})
	require.NoError(t, err)
	require.NotNil(t, sender)
}

func TestNew_RequiresLogger(t *testing.T) {
	_, err := New(&sms.Config{Provider: sms.ProviderLog})
	require.ErrorIs(t, err, sms.ErrInvalidConfig)
}

func TestSend_LogsAndNeverFails(t *testing.T) {
	logger := newLogger()
	sender, err := New(&sms.Config{Provider: sms.ProviderLog, Logger: logger})
	require.NoError(t, err)

	err = sender.Send(context.Background(), "+84900000000", "your code is 111111")
	require.NoError(t, err)

	logger.AssertCalled(t, "Info", "DEV MODE: sms not actually sent, logging instead", []any(nil))
}
