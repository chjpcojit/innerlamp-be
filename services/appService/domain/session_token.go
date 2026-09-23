package domain

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashRefreshToken returns the value stored in user_sessions.refresh_token_hash
// for a refresh token: its SHA-256 digest as lowercase hex.
//
// It lives in domain, not in repository, because two layers now need the same
// answer: the repository writes the column, and the use case has to compute the
// hash of the token a client presented before it can ask for the session. A
// private helper in repository could not be reached from usecase, and a second
// copy of the rule there would be a duplicate definition of a security
// contract — the day one of them changed, /refresh would silently stop finding
// every session.
//
// The token itself is a bearer credential and is never persisted; only this
// digest is. A stolen database dump therefore yields no usable token, which is
// also why a plain SHA-256 is enough here: the input is 200+ bits of signed
// token, not a guessable password, so there is nothing to brute-force.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
