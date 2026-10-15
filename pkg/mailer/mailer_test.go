package mailer

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/logging/formatter"
)

func newBufferLogger(buf *bytes.Buffer) logging.Logger {
	return logging.New(logging.Config{Level: logging.DebugLevel, Formatter: formatter.NewText(), Output: buf})
}

func TestLogMailer_Send_LogsWholeMessage(t *testing.T) {
	var buf bytes.Buffer
	m := NewLogMailer(newBufferLogger(&buf))

	err := m.Send(context.Background(), Message{
		From:    "no-reply@innerlamp.local",
		To:      "user@example.com",
		Subject: "Mã xác minh",
		Text:    "Mã của bạn là 123456",
		HTML:    "<p>123456</p>",
	})

	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "no-reply@innerlamp.local")
	assert.Contains(t, out, "user@example.com")
	assert.Contains(t, out, "Mã xác minh")
	assert.Contains(t, out, "Mã của bạn là 123456")
	assert.Contains(t, out, "<p>123456</p>")
}

func TestMockMailer_Send(t *testing.T) {
	msg := Message{To: "a@b.c", Subject: "s", Text: "t"}
	wantErr := errors.New("provider down")

	m := new(MockMailer)
	m.On("Send", context.Background(), msg).Return(wantErr).Once()

	err := m.Send(context.Background(), msg)

	require.ErrorIs(t, err, wantErr)
	m.AssertExpectations(t)
}
