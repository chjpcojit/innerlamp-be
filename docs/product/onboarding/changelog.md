# Changelog — onboarding product spec

## v1.3 — 2026-10-15

- References only, no requirement changed: the Downstream row and §1.5 now name the one spec folder per
  platform (all four approved: backend, website, iOS, Android), and the prior documents are marked
  rewritten or deleted. Platform specs citing v1.2 stay valid.
- Platforms to follow up: none.

## v1.2 — 2026-10-14

- Resolved (anh Đức, "đồng ý đề xuất của em"): the street line of a proposed temple is optional
  (BR-ONB-10, §3.6). No open items left.
- Platforms to follow up: Website, iOS, Android (temple proposal form). Backend already treats it as
  optional.

## v1.1 — 2026-10-14

- Resolved (anh Đức): skipping the intro counts as seen (BR-ONB-01); a temple proposal needs province and
  ward from the administrative-unit list, required (BR-ONB-10); one terms text for every client
  (BR-ONB-11, §3.6, §4.2 now drift on all three clients).
- Resolved (anh Đức): a signed-in user opening the registration page on the website goes to the home
  page (§3.6).
- Still open: street line of a proposed temple.
- Platforms to follow up: iOS, Android (skip), all clients (terms source, temple proposal).

## v1.0 — 2026-10-14

- First product spec for the first-run experience and registration, retroactive mode. Replaces the
  product parts of backend `docs/specs/features/{onboarding,otp-mechanism}/`, website
  `Docs/specs/features/register/` and iOS `Docs/Specs/{SignUp,Otp,Onboarding,Splash}/`.
- **Verification moves from SMS to email** (ADR-0003, Accepted 2026-10-14): BR-ONB-08, BR-ONB-09,
  FR-ONB-003, FR-ONB-007…010. Phone stays required and unique, never verified (BR-ONB-04).
- Settled cross-platform disagreements (anh Đức, 2026-10-14): dharma name optional, full name required
  (BR-ONB-07); password 8–64 characters, strength hint advice only (BR-ONB-06); per-country phone
  length with one 12-country list (BR-ONB-04); one email rule (BR-ONB-05); launch routing to sign-in
  after the intro, intro-seen record on the device (BR-ONB-01, BR-ONB-02); intro choices stay on the
  device (BR-ONB-03); code valid 10 minutes, a backend setting (BR-ONB-09).
- Added vectors: `rules/phone-number.json` (13), `rules/email.json` (11), `rules/password.json` (10),
  `rules/names.json` (11).
- Open: skip counts as intro seen; temple-proposal address; terms text source; website redirect for
  signed-in users (Appendix A).
- Platforms to follow up: all four (`spec-writer` in each repo). Android builds registration and code
  entry for the first time.
