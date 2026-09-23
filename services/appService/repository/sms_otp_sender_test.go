package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

type stubSMSSender struct {
	gotPhoneNumber string
	gotMessage     string
	err            error
}

func (s *stubSMSSender) Send(ctx context.Context, phoneNumber, message string) error {
	s.gotPhoneNumber = phoneNumber
	s.gotMessage = message
	return s.err
}

func TestSMSOTPSender_SendOTP(t *testing.T) {
	t.Run("forwards the otp code and phone number", func(t *testing.T) {
		stub := &stubSMSSender{}
		sender := NewSMSOTPSender(stub)

		err := sender.SendOTP(context.Background(), "+84900000000", "111111", domain.OTPPurposeRegister)
		require.NoError(t, err)
		require.Equal(t, "+84900000000", stub.gotPhoneNumber)
		require.Contains(t, stub.gotMessage, "111111")
	})

	t.Run("propagates the underlying sender's error", func(t *testing.T) {
		wantErr := errors.New("provider down")
		stub := &stubSMSSender{err: wantErr}
		sender := NewSMSOTPSender(stub)

		err := sender.SendOTP(context.Background(), "+84900000000", "111111", domain.OTPPurposeRegister)
		require.ErrorIs(t, err, wantErr)
	})
}
