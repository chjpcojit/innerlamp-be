package domain

// LoginInput is the use-case input for US-002.
type LoginInput struct {
	Email    string
	Password string
	Device   DeviceInfo
}

// LoginResult mirrors RegisterResult — the spec requires an identical
// response shape for login and register (US-002 "Ghi chú kỹ thuật").
type LoginResult = RegisterResult
