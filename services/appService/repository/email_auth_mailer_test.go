package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/mailer"
)

const testMailFrom = "no-reply@innerlamp.test"

// sentMessage wires a MockMailer that captures the one Message sent.
func sentMessage(t *testing.T, send func(m *mailer.MockMailer)) mailer.Message {
	t.Helper()
	m := new(mailer.MockMailer)
	var got mailer.Message
	m.On("Send", mock.Anything, mock.AnythingOfType("mailer.Message")).
		Run(func(args mock.Arguments) { got = args.Get(1).(mailer.Message) }).
		Return(nil).Once()
	send(m)
	m.AssertExpectations(t)
	return got
}

// assertNoPasswordWording guards AC19: no auth email talks about "your
// password is ..." or carries a password field.
func assertNoPasswordWording(t *testing.T, msg mailer.Message) {
	t.Helper()
	body := strings.ToLower(msg.Text + msg.HTML)
	assert.NotContains(t, body, "mật khẩu của bạn là")
	assert.NotContains(t, body, "mật khẩu:")
	assert.NotContains(t, body, "password")
}

func TestEmailAuthMailer_SendRegisterCode(t *testing.T) {
	msg := sentMessage(t, func(m *mailer.MockMailer) {
		err := NewEmailAuthMailer(m, testMailFrom).SendRegisterCode(context.Background(), "user@example.com", "482913", 10*time.Minute)
		require.NoError(t, err)
	})

	assert.Equal(t, testMailFrom, msg.From)
	assert.Equal(t, "user@example.com", msg.To)
	assert.Contains(t, msg.Subject, "Mã xác minh đăng ký")
	assert.Contains(t, msg.Text, "482913")
	assert.Contains(t, msg.Text, "10 phút")
	assertNoPasswordWording(t, msg)
}

func TestEmailAuthMailer_SendRegisterCode_RoundsValidityUp(t *testing.T) {
	msg := sentMessage(t, func(m *mailer.MockMailer) {
		require.NoError(t, NewEmailAuthMailer(m, testMailFrom).SendRegisterCode(context.Background(), "user@example.com", "111111", 90*time.Second))
	})

	assert.Contains(t, msg.Text, "2 phút")
}

func TestEmailAuthMailer_SendAccountExistsNotice(t *testing.T) {
	msg := sentMessage(t, func(m *mailer.MockMailer) {
		require.NoError(t, NewEmailAuthMailer(m, testMailFrom).SendAccountExistsNotice(context.Background(), "owner@example.com"))
	})

	assert.Equal(t, testMailFrom, msg.From)
	assert.Equal(t, "owner@example.com", msg.To)
	assert.Contains(t, msg.Subject, "đã có tài khoản")
	assert.Contains(t, msg.Text, "đã có tài khoản")
	assert.Contains(t, msg.Text, "đăng nhập")
	assert.Contains(t, msg.Text, "Quên mật khẩu")
	assert.NotRegexp(t, `\d{6}`, msg.Text, "the notice must not carry a code")
	assertNoPasswordWording(t, msg)
}

func TestEmailAuthMailer_SendPasswordResetCode(t *testing.T) {
	msg := sentMessage(t, func(m *mailer.MockMailer) {
		require.NoError(t, NewEmailAuthMailer(m, testMailFrom).SendPasswordResetCode(context.Background(), "user@example.com", "730155", 10*time.Minute))
	})

	assert.Equal(t, testMailFrom, msg.From)
	assert.Equal(t, "user@example.com", msg.To)
	assert.Contains(t, msg.Subject, "đặt lại mật khẩu")
	assert.Contains(t, msg.Text, "730155")
	assert.Contains(t, msg.Text, "10 phút")
	assertNoPasswordWording(t, msg)
}

func TestEmailAuthMailer_PropagatesSendError(t *testing.T) {
	wantErr := errors.New("provider down")
	m := new(mailer.MockMailer)
	m.On("Send", mock.Anything, mock.Anything).Return(wantErr)
	am := NewEmailAuthMailer(m, testMailFrom)
	ctx := context.Background()

	require.ErrorIs(t, am.SendRegisterCode(ctx, "a@b.c", "123456", time.Minute), wantErr)
	require.ErrorIs(t, am.SendAccountExistsNotice(ctx, "a@b.c"), wantErr)
	require.ErrorIs(t, am.SendPasswordResetCode(ctx, "a@b.c", "123456", time.Minute), wantErr)
}
