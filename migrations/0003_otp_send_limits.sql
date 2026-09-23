-- US-001 v5.0: rate limit cho việc GỬI OTP.
--
-- Bối cảnh: trước v5.0, trần gửi nằm ở `otp_codes.resend_count` và được
-- carry-forward mãi mãi từ hàng OTP mới nhất sang hàng mới. Vì không gì reset
-- nó, một số điện thoại gửi lại đủ 3 lần là bị chặn VĨNH VIỄN, không có đường
-- gỡ (xem docs/specs/features/onboarding/changelog.md v5.0).
--
-- v5.0 tách bộ đếm gửi ra bảng riêng theo (phone_number, purpose):
--   * nó phải sống lâu hơn hàng OTP (TTL 90 giây), và
--   * khoá-gửi KHÔNG được chặn `verify-otp` — nếu dùng chung `otp_codes.locked_until`
--     thì OTP hợp lệ cuối cùng vừa gửi sẽ không nhập được trong 90 giây còn hạn.
-- Chi tiết: docs/specs/features/onboarding/design.md, §"OTP send-limit policy".
--
-- File idempotent: CREATE TABLE IF NOT EXISTS + DROP COLUMN IF EXISTS, nên
-- apply lại không lỗi (khác 0002).

CREATE TABLE IF NOT EXISTS otp_send_limits (
    id           BIGSERIAL PRIMARY KEY,
    phone_number VARCHAR(20) NOT NULL,
    -- Cùng tập purpose với otp_codes; khai đủ bốn để login/reset_password dùng
    -- lại mà không phải ALTER constraint. Tập hằng ở domain/otp.go.
    purpose      VARCHAR(20) NOT NULL
                 CONSTRAINT chk_otp_send_limits_purpose
                 CHECK (purpose IN ('register', 'login', 'reset_password', 'verify_email')),
    -- Số lượt gửi ĐÃ TÍNH trong cycle hiện tại. Lần gửi đầu của cycle miễn phí
    -- (send_count = 0); khi đạt max_send_attempts thì set locked_until.
    send_count   INT NOT NULL DEFAULT 0,
    -- Mốc gửi gần nhất: vừa là mốc cooldown, vừa là mốc reset cycle (quá
    -- send_lock_duration không gửi thì cycle mới, send_count về 0).
    last_sent_at TIMESTAMPTZ,
    -- Khoá-GỬI (khác khoá-nhập-sai ở otp_codes.locked_until): chặn
    -- request-otp/resend-otp, KHÔNG chặn verify-otp.
    locked_until TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- UNIQUE (phone_number, purpose) là target của
    -- INSERT ... ON CONFLICT (phone_number, purpose) DO NOTHING trong
    -- applySendLimits, và cũng chính là index tra cứu — không cần index phụ.
    -- Khai inline (không ALTER ADD CONSTRAINT) để apply lại là no-op.
    UNIQUE (phone_number, purpose)
);

-- Bộ đếm cũ bỏ hẳn: giữ lại một cột không ai đọc là mời gọi dùng lại đúng cái
-- cơ chế carry-forward đã gây bug.
ALTER TABLE otp_codes DROP COLUMN IF EXISTS resend_count;
