# Changelog — login product spec

## v1.2 — 2026-10-14

- Resolved (anh Đức, "Gỡ như em đề xuất"): a successful password reset also lifts the sign-in lock on
  that email (BR-LGN-05, example added). Raised by the backend spec (`login` v2.0).
- Platforms to follow up: Backend (one extra write in the reset). Clients: no change.

## v1.1 — 2026-10-14

- Resolved (anh Đức): sign-out while offline completes on the device at once (BR-LGN-07).
- Resolved (anh Đức): website sends signed-in users from the sign-in page to the home page, and after
  sign-in returns a guest to the page they were heading to (FR-LGN-007, 3 scenarios). No open items.
- Platforms to follow up: Website, iOS, Android.

## v1.0 — 2026-10-14

- First product spec for sign-in, session, sign-out, forgot password and guest mode, retroactive mode.
  Replaces the product parts of backend `docs/specs/features/{login,password-reset}/` (incl.
  `login/SRS.md`), website `Docs/specs/features/login/` and iOS `Docs/Specs/{Login,ForgotPassword}/`.
- **Sign-in moves from phone to email** (ADR-0003, Accepted 2026-10-14): BR-LGN-01, BR-LGN-02,
  FR-LGN-001…003. The one-time-code sign-in button is removed.
- Added: password reset by email code (BR-LGN-05, FR-LGN-008, FR-LGN-009), every device signed out after
  a reset; session-ended message (BR-LGN-04, FR-LGN-005); guest mode rules (BR-LGN-06, FR-LGN-010).
- Open: offline sign-out; website return-to-page after sign-in (Appendix A).
- Platforms to follow up: all four. Android builds sign-in, sessions and reset for the first time.
