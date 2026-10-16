# Changelog — onboarding product spec

## v1.7 — 2026-10-17

- Resolved (anh Đức, "đồng ý"): the phone number and the chosen temple are checked again when the account
  is created. If the number was taken, or the temple removed, while the user waited for the code, no
  account is created and the user returns from the code view to the registration form, every value kept,
  with the error at the field. BR-ONB-08 gains the rule and an example; FR-ONB-005 and FR-ONB-004 gain one
  scenario each. Raised by the website review; the website, iOS and Android specs already assumed it.
- The backend already behaves this way (onboarding backend spec AC18). Platform specs citing v1.6 stay
  valid; clients add the return path where they have none.
- Platforms to follow up: Website, iOS, Android (state the return path in the platform spec). Backend: none.

## v1.6 — 2026-10-16

- Resolved (anh Đức, "Theo đề xuất của iOS"): what the user sees when a registration whose code is still
  valid is submitted again with different details. BR-ONB-08 now says an earlier pending registration is
  replaced only when its code has expired or the details are the same; otherwise the form stays open with
  every value kept, "Vui lòng chờ N giây rồi thử lại" (N from the backend), no email, nothing changed.
  FR-ONB-003 gains the sentence and two scenarios. Raised by the iOS spec (assumption AS-Onboarding-001)
  and the Android spec (criterion 15a, was `⚠️ Not confirmed`).
- The rule is already how the backend behaves (onboarding backend spec v8.2, AC24, security review
  2026-10-16); until now the product spec was silent, so no client had a defined screen. Because the
  user-visible answer is new, clients citing v1.5 must add the screen.
- Platforms to follow up: Website, iOS, Android (the refusal view; iOS and Android already hold the same
  assumption). Backend: none.

## v1.5 — 2026-10-16

- Traceability matrix only (§4.2), no requirement changed: the Backend column is `done` for BR-ONB-04,
  BR-ONB-05, BR-ONB-06, BR-ONB-08, BR-ONB-09, FR-ONB-003, FR-ONB-007, FR-ONB-008, NFR-ONB-01 and
  NFR-ONB-02 (email registration and per-IP limits shipped on the backend, 2026-10-16); five notes
  reworded. Platform specs citing v1.2–v1.4 stay valid.
- Platforms to follow up: none for the backend. Website, iOS, Android: their own columns are unchanged.

## v1.4 — 2026-10-15

- Clarified (anh Đức, after the registration UI audit): "Bỏ qua" (skip) is shown on all three intro
  steps, matching iOS and Android today (§3.1.1, FR-ONB-001). The registration submit button reads
  "Đăng ký", not "Gửi mã xác thực" (§3.1.1, FR-ONB-003; §4.2 note).
- No business rule changed; platform specs citing v1.2/v1.3 stay valid.
- Platforms to follow up: Website, iOS, Android (rename the submit button to "Đăng ký"). iOS and
  Android intro: nothing to change.

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
