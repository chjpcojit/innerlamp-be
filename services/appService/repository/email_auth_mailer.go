package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"chjpcoj.com.innerlamp/pkg/mailer"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// emailAuthMailer implements domain.AuthMailer on top of pkg/mailer.
//
// pkg/mailer only delivers a Message; the Vietnamese wording of each auth
// email is a service concern, so it lives here (onboarding design.md
// "Mailer"). Bodies carry at most a code and its validity — never a password
// (AC19, NFR-ONB-03).
type emailAuthMailer struct {
	mailer mailer.Mailer
	from   string
}

// NewEmailAuthMailer returns the domain.AuthMailer backed by m. from is the
// configured sender address (config mailer.from), set on every Message.
func NewEmailAuthMailer(m mailer.Mailer, from string) domain.AuthMailer {
	return &emailAuthMailer{mailer: m, from: from}
}

const (
	subjectRegisterCode      = "InnerLamp — Mã xác minh đăng ký"
	subjectAccountExists     = "InnerLamp — Email của bạn đã có tài khoản"
	subjectPasswordResetCode = "InnerLamp — Mã đặt lại mật khẩu"
)

// SendRegisterCode emails the registration code and its validity in minutes.
func (s *emailAuthMailer) SendRegisterCode(ctx context.Context, to, code string, ttl time.Duration) error {
	text := fmt.Sprintf(
		"Xin chào,\n\n"+
			"Mã xác minh đăng ký tài khoản InnerLamp của bạn là: %s\n\n"+
			"Mã có hiệu lực trong %d phút. Không chia sẻ mã này với bất kỳ ai.\n\n"+
			"Nếu bạn không yêu cầu đăng ký, hãy bỏ qua email này.\n\nInnerLamp",
		code, ttlMinutes(ttl))
	return s.send(ctx, to, subjectRegisterCode, text)
}

// SendAccountExistsNotice tells the owner that the email already has an
// account and suggests logging in or resetting the password. No code.
func (s *emailAuthMailer) SendAccountExistsNotice(ctx context.Context, to string) error {
	const text = "Xin chào,\n\n" +
		"Có người vừa dùng địa chỉ email này để đăng ký InnerLamp, nhưng email này đã có tài khoản.\n\n" +
		"Nếu đó là bạn, hãy đăng nhập bằng tài khoản hiện có. " +
		"Nếu bạn quên mật khẩu, hãy dùng chức năng \"Quên mật khẩu\" trong ứng dụng.\n\n" +
		"Nếu không phải bạn, hãy bỏ qua email này — tài khoản của bạn không bị thay đổi.\n\nInnerLamp"
	return s.send(ctx, to, subjectAccountExists, text)
}

// SendPasswordResetCode emails the password-reset code and its validity in minutes.
func (s *emailAuthMailer) SendPasswordResetCode(ctx context.Context, to, code string, ttl time.Duration) error {
	text := fmt.Sprintf(
		"Xin chào,\n\n"+
			"Mã đặt lại mật khẩu InnerLamp của bạn là: %s\n\n"+
			"Mã có hiệu lực trong %d phút. Không chia sẻ mã này với bất kỳ ai.\n\n"+
			"Nếu bạn không yêu cầu đặt lại mật khẩu, hãy bỏ qua email này — mật khẩu của bạn không bị thay đổi.\n\nInnerLamp",
		code, ttlMinutes(ttl))
	return s.send(ctx, to, subjectPasswordResetCode, text)
}

func (s *emailAuthMailer) send(ctx context.Context, to, subject, text string) error {
	if err := s.mailer.Send(ctx, mailer.Message{From: s.from, To: to, Subject: subject, Text: text}); err != nil {
		return fmt.Errorf("send auth email %q: %w", subject, err)
	}
	return nil
}

// ttlMinutes rounds up so a 90s code reads "2 phút", never "1 phút" (or 0).
func ttlMinutes(ttl time.Duration) int {
	return int(math.Ceil(ttl.Minutes()))
}
