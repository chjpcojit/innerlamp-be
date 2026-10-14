# ADR-0003: Accounts are identified by email and verified by an email OTP; SMS is removed

| Field | Value |
|---|---|
| Status | Accepted |
| Date | 2026-10-14 |
| Deciders | anh Đức (decides) · Claude (`adr-writer`, proposes) |
| Features | `onboarding` (register), `login` (login, lockout, forgot password), `genealogy` (invitations), `profile` |
| Platforms | Backend, Website, iOS, Android |
| Supersedes | — |
| Related | ADR-0001 (Firebase is added to the apps for push) · `/mnt/project-files/reports/security/2026-10-14-danh-gia-pentest.md` (C-1, H-2, M-3, M-9, item 19) · `/mnt/project-files/reports/infra/2026-10-14-thiet-ke-gcp.md` (no SMS, no Cloud NAT) · `/mnt/project-files/reports/product/auth/2026-10-14-cau-hoi-truoc-adr.md` (questions and answers) |

## Context

InnerLamp is a personal project. On 2026-10-14 the owner decided to **drop SMS completely**: no SMS
provider of any kind, because per-message cost is too high for the project. Registration today
depends on SMS, so the way an account is proven and recovered has to change.

What exists today (checked 2026-10-14 on `main` @ 157d853):

- **Register (US-001)** is a two-step flow: step 1 stores the payload in `pending_registration`
  (`UNIQUE phone_number`, password already bcrypt-hashed, TTL `pending_registration_ttl: 24h`) and
  sends a 6-digit OTP; step 2 verifies the OTP and creates the `users` row plus a session in one
  transaction. No `users` row ever exists unverified (`migrations/0001_register_onboarding.sql`,
  `services/appService/usecase/register_usecase.go`).
- The OTP is **SMS-only and fixed**: `generateOTP()` returns `111111`
  (`services/appService/usecase/otp_service.go`), sent by `repository/sms_otp_sender.go` through
  `pkg/sms` → `pkg/sms/logsms` (logs instead of sending). `otp_codes` is keyed by `phone_number`
  with `purpose IN ('register','login','reset_password','verify_email')`, bcrypt `otp_hash`,
  attempts / resend cap / lock in the row (config `ttl: 90s`, `max_attempts: 3`, `lock_duration: 30m`).
- **Users**: `users.phone_number NOT NULL UNIQUE` is the verified identity; `users.email NOT NULL
  UNIQUE` is never verified (`email_verified_at` is always NULL). Collected at registration: phone,
  email, password, full name, dharma name (pháp danh), ordination temple
  (`domain/pending_registration.go`, `ordination_temple_id`).
- **Login (US-002)** is phone + password (`FindUserByPhone`, `usecase/login_usecase.go`), with a
  per-phone lockout in `login_attempts` (PK `phone_number`, `migrations/0002_…`). Sessions: JWT
  access 15m / refresh 168h, refresh rotation with reuse detection, `user_sessions` +
  `user_devices`. This part is sound (pentest report §1) and is not changed by this ADR.
- **Genealogy (gia phả) invitations** are addressed to a phone number and matched against the
  caller's own `users.phone_number` on list and accept
  (`usecase/genealogy_invitation_usecase.go`, `repository/postgres_genealogy_access_repository.go`);
  viewers are also linked to tree members by equal phone numbers. Today this relies on the phone
  being verified — which, with the fixed OTP, it effectively is not (pentest C-1).
- **Forgot password** is spec-only (`docs/specs/features/password-reset/`, SMS OTP), no code.
- `pkg/mailer` exists with only a dev `LogMailer` (`SendPasswordResetEmail`,
  `SendAccountInviteEmail`); the back office already needs a real mailer for admin invites.
- `pkg/ratelimit` has `memory` and `redis` stores; none of `/v1/auth/*` is rate limited
  (`services/appService/cmd/main.go:479-493`, pentest H-2).
- Clients: iOS has `SignUp`, `Otp`, `Login`, `ForgotPassword`; the website has `(auth)/signup` and
  `(auth)/login`; Android login is a stub.
- There are **no real users** in any running environment (infrastructure is not built), so a schema
  change needs no data backfill (confirmed by the owner 2026-10-14).

Constraints: under 10k users in the first year (ADR-0001); monthly cost and operational simplicity
weigh heavily; a solo developer. Inferred: well under 1,000 verification/reset emails per month.

### Why this needs an ADR

It changes the sign-in method and the verified identity of every account (gate: "a new auth factor
or sign-in method", "changes the trust boundary"). Undoing it later means migrating user identities,
changing the `/v1/auth/*` contract and re-releasing both mobile apps.

## Decision drivers

- No SMS, no SMS provider (owner, 2026-10-14).
- Near-zero monthly cost at < 10k users.
- Every account must have **one channel the user proved they control**, used for registration and
  for account recovery.
- Keep what registration collects today; only the verification channel changes (owner answer 1).
- Smallest change to the working code and client screens (two-step flow, OTP screen, sessions).
- Close the pentest findings on auth: C-1 (fixed OTP), H-2 (no rate limit), M-3 (counter races),
  M-9 (account enumeration), item 19 (safe password reset).

## Options considered

Costs are list prices checked on 2026-10-14 and must be re-checked when the provider is picked.

### Option 0: Keep the status quo (SMS OTP on the phone number)

Keep phone + SMS OTP; plug in a real SMS provider before launch.

| Dimension | Assessment |
|---|---|
| Complexity | Low — code exists; needs a provider driver and rate limits |
| Cost | Per-message SMS cost, unbounded under abuse (SMS pumping / toll fraud, pentest H-2) |
| Reversibility | — |
| Cross-platform impact | None |
| Security & privacy | Phone is a verified identity; SMS is weak against SIM swap |
| Reliability | Depends on SMS delivery in Vietnam |
| Fit with current stack | Full |
| Familiarity | High |

**Pros:** nothing to rebuild. **Cons:** violates the owner's no-SMS constraint; cost and fraud risk
scale with abuse, not with real users. **Rejected by constraint.**

### Option A: Email + password, email verified by a 6-digit OTP sent by email

Email becomes the sign-in identifier and the verified channel. Registration keeps its two steps and
the same collected fields; the OTP goes to the email address instead of the phone. The phone number
is still collected but not verified. Forgot password sends an OTP to the verified email. Mail goes
out through a transactional email API behind `pkg/mailer` (e.g. Resend: free 3,000/month, 100/day;
Amazon SES: about $0.10 per 1,000).

```
Client ──register(email, phone, password, …)──► BE ──pending_registration(email)──► Postgres
                                                  └─ OTP (crypto/rand, bcrypt) ──► Mailer ──► user inbox
Client ──verify(email, otp)──► BE ──create users (email_verified_at = now) + session──► tokens
```

| Dimension | Assessment |
|---|---|
| Complexity | Low–Med — re-key OTP / pending / lockout from phone to email, add a mailer driver, rate limits |
| Cost | ~$0/month at the expected volume; a sending domain is required (not yet owned) |
| Reversibility | Med — identifiers change in DB and contract; adding social sign-in later is additive |
| Cross-platform impact | BE: auth usecases, schema, contract. Web/iOS: field label and OTP copy, login form; Android builds it fresh |
| Security & privacy | Verified email for recovery; OTP random + hashed; email inbox is the recovery root |
| Reliability | Depends on the email provider and deliverability (spam folders); outage blocks register/reset, not login |
| Fit with current stack | High — reuses `otp_codes`, the two-step flow, `pkg/mailer`, sessions unchanged |
| Familiarity | High |

**Pros:** cheapest working option with a recovery channel; least rework; one mail provider also
serves the back office. **Cons:** deliverability work (SPF/DKIM/DMARC, domain); phone becomes an
unverified claim, which weakens the phone-based genealogy invitations (accepted risk, Decision rule 4).

### Option B: Google + Apple sign-in only (no password)

The backend verifies a Google / Apple ID token and issues its own session.

| Dimension | Assessment |
|---|---|
| Complexity | Med — token verification per provider, Apple on Android/web via web flow, nonce handling |
| Cost | $0 (Apple Developer Program is already needed for iOS) |
| Reversibility | High cost — accounts keyed by provider subject IDs |
| Cross-platform impact | High — new sign-in UI and SDKs on all three clients; register/OTP screens removed |
| Security & privacy | Strong (provider handles credentials, MFA); Apple can hide the email (private relay) |
| Reliability | Depends on Google/Apple availability |
| Fit with current stack | Low — password, OTP and pending-registration code become dead |
| Familiarity | Low |

**Pros:** no password or recovery to run. **Cons:** users without (or not remembering) a Google or
Apple account are locked out; hidden relay emails; most rework; the owner chose to keep the current
registration fields and flow.

### Option C: Option A + Google/Apple sign-in, linked by verified email

| Dimension | Assessment |
|---|---|
| Complexity | High — A plus B plus account linking rules |
| Cost | ~$0 |
| Reversibility | Med |
| Cross-platform impact | Highest — three sign-in methods on three clients |
| Security & privacy | Good; linking by verified email must be done carefully (account takeover risk) |
| Reliability | As A and B |
| Fit with current stack | Med |
| Familiarity | Low |

**Pros:** best UX. **Cons:** most work now. The owner deferred social sign-in to a separate,
later ADR (answer 2), which can add it on top of A without migrating identities again.

### Option D: Delegate auth to Firebase Authentication / Identity Platform

Firebase handles email/password, email links and social providers; the backend verifies Firebase ID
tokens. Free up to 50,000 MAU for standard providers. Firebase SDKs will already be in the apps for
push (ADR-0001).

| Dimension | Assessment |
|---|---|
| Complexity | Med — replaces the session/rotation layer; SvelteKit server must handle Firebase tokens |
| Cost | $0 up to 50k MAU |
| Reversibility | Low — user records live in Firebase; leaving means exporting password hashes |
| Cross-platform impact | High — every client changes its auth stack |
| Security & privacy | Good, but identity data moves to a third party |
| Reliability | Depends on Firebase |
| Fit with current stack | Low — discards working JWT/refresh/reuse-detection code |
| Familiarity | Low |

**Pros:** no auth infrastructure to run. **Cons:** throws away the part the pentest rated good;
vendor lock-in for the core identity.

### Option E: No verification at all

Email (or phone) + password with no proof of ownership; abuse controlled only by rate limits and
mobile attestation.

| Dimension | Assessment |
|---|---|
| Complexity | Low |
| Cost | $0 |
| Reversibility | Med — later verification must handle unverified legacy accounts |
| Cross-platform impact | Low — OTP screen removed |
| Security & privacy | Weak — anyone can squat an email or phone; no recovery except manual |
| Reliability | No external dependency |
| Fit with current stack | High |
| Familiarity | High |

**Pros:** cheapest and simplest. **Cons:** a forgotten password loses the account; squatting;
genealogy sharing by contact becomes unsafe. Passkeys (WebAuthn) were also considered but are not
a standalone option: they still need a recovery channel, so they fit as a later add-on to A.

## Trade-off analysis

The real choice is between a channel the project runs itself (A) and one delegated to a provider
(B, D). A is the only option that meets every driver: $0, a proven channel for recovery, the current
fields and two-step flow kept, sessions untouched. B and D cost more rework than they save at this
scale and move identity outside the project. E saves the email domain and provider, but leaves no
recovery and weakens every feature that shares data between people. C is A plus deferred work.

## Decision

**Decided:** Option A — email + password, email verified by an email OTP; SMS removed.

Rules the decision carries (owner answers 2026-10-14 included):

1. **Identifier.** Sign-in is email + password. Email is unique (case-insensitive, normalized) and
   is the only verified channel (`email_verified_at` set at account creation).
2. **Registration** keeps its two steps and every field it collects today (email, phone, password,
   full name, dharma name, ordination temple). The `users` row is created only after the email OTP
   is verified, in the same transaction as the first session (unchanged invariant).
3. **Phone number** is still collected and stays **required and unique across the system**
   (owner, 2026-10-14), but is **never verified** and never used as a credential or an identifier
   for sign-in or recovery. Accepted risk: someone can register first with another person's number
   and block its real owner from registering; the real owner contacts support and an admin frees
   the number manually (same path as rule 5).
4. **Genealogy invitations** keep matching on the phone number exactly as today, with **no extra
   proof** such as an invite code (owner, 2026-10-14). Accepted risk, recorded so it is a conscious
   choice: because the phone is unverified, a person who registers first with the invitee's number
   receives and can accept that invitation and read the tree (the pentest C-1 chain stays open for
   this path). Revisit if a tree holds sensitive data of many people or the app goes public
   (see Consequences).
5. **Recovery.** Forgot password = OTP to the verified email (one-time, hashed, short TTL, rate
   limited, all sessions revoked after reset, identical response whether or not the email exists).
   A user who lost access to their email contacts support and an admin handles it manually through
   the back office (owner answer 6).
6. **OTP.** 6 digits from `crypto/rand`, bcrypt-hashed, short TTL, attempt cap and lock, resend cap
   — the existing `otp_codes` mechanics, keyed by email. A fixed OTP may exist only behind a dev-only
   flag that **fails startup** outside the dev environment.
7. **Email delivery** goes through `pkg/mailer` with a real provider driver (provider picked in the
   BE spec). For now it stays **mocked** (owner, 2026-10-14): `LogMailer` plus the fixed dev OTP,
   allowed only in the dev environment (rule 6). Buying a sending domain and configuring
   SPF/DKIM/DMARC is a prerequisite for production, tracked under Follow-ups → Infrastructure.
8. **Abuse controls (required, not optional):** rate limits per IP and per email on every
   `/v1/auth/*` route (register, verify, resend, login, refresh, forgot, reset) with a shared store
   when Cloud Run runs several instances; a global daily email-send budget; atomic counters for OTP
   attempts and login lockout (`UPDATE … WHERE attempts < max RETURNING`, pentest M-3); no
   "email already registered" signal from register or forgot (pentest M-9); mobile attestation on
   register where available (`pkg/attestation`). Blocking disposable-email domains is optional.
9. **Out of scope:** Google/Apple sign-in (later ADR, owner answer 2); back-office admin login
   lockout/MFA (pentest H-1, different trust boundary).

## Consequences

- **Easier:** no SMS cost or SMS-fraud exposure; no Cloud NAT for an SMS egress IP; one mail provider
  serves the app and the back office; recovery exists from day one; social sign-in or passkeys can
  be added later without re-keying accounts.
- **Harder:** email deliverability (domain, SPF/DKIM/DMARC, spam folders) becomes a launch
  dependency; the phone number is only a claim, so every feature that trusts it must be reviewed
  (today: genealogy invitations and viewer↔member linking).
- **Accepted risks:** an unverified but unique phone can be squatted (rule 3); genealogy invitations
  matched on that phone can be accepted by the squatter (rule 4).
- **Revisit when:** a sending domain is bought; the first support case about a squatted phone
  number or a wrongly accepted genealogy invitation; users report not receiving OTP emails; before
  public launch (Google/Apple sign-in ADR); if email volume nears the provider's free tier.

### What must be removed or changed in the current code

| Where | Change |
|---|---|
| `pkg/sms`, `pkg/sms/logsms`, `repository/sms_otp_sender.go`, SMS wiring in `cmd/main.go` | **Remove** |
| `usecase/otp_service.go` | OTP sent by email; random `crypto/rand`; fixed value dev-only with fail-fast; atomic attempt counter |
| `otp_codes` (keyed by `phone_number`) | Re-key to email |
| `usecase/register_usecase.go`, `pending_registration` (`UNIQUE phone_number`) | Pending payload keyed by email; account created with `email_verified_at = now()` |
| `users.phone_number NOT NULL UNIQUE` | Kept as is; only the verification meaning goes away (rule 3) |
| `usecase/login_usecase.go` (`FindUserByPhone`), `login_attempts` (PK `phone_number`) | Login by email; lockout keyed by email |
| `usecase/genealogy_invitation_usecase.go`, `repository/postgres_genealogy_access_repository.go` | No change (rule 4, accepted risk) |
| `pkg/mailer` | Add OTP mail methods and a real provider driver |
| `cmd/main.go` `/v1/auth/*` routes | Wrap with rate limiters |
| Docs: `docs/specs/features/{onboarding,login,otp-mechanism,password-reset}/`, `docs/knowledge/features/{register,login}.md`, `docs/specs/api/openapi.yaml` (phone-based auth) | Rewritten in the follow-up steps, old ones deleted after approval |

## Follow-ups

| Layer | Follow-up |
|---|---|
| Product spec (`ba-writer`) | Two specs citing ADR-0003: `onboarding` (register + email OTP + ordination temple) and `login` (login, refresh, logout, lockout, forgot password); genealogy product spec notes the accepted risk of rule 4 |
| API contract (`spec-writer`, BE) | `/v1/auth/*` request/response move from phone to email; resend/verify by email; new forgot/reset endpoints; uniform error responses (M-9); 429 for rate limits |
| Backend spec / code | Items in the table above; mailer provider choice; rate-limit store (Redis vs Postgres) |
| Website / iOS / Android | Sign-in form uses email; OTP screen copy says "email"; forgot-password flow; Android builds login/register on the new contract |
| Infrastructure | Sending domain + DNS (SPF/DKIM/DMARC); mail provider API key in Secret Manager; no SMS egress |

## Open questions

None.

### Resolved

- 2026-10-14 — Sign-in identifier → email; registration keeps the other fields; OTP is sent to and verifies the email (anh Đức).
- 2026-10-14 — Google/Apple sign-in → not in this round, separate ADR later (anh Đức).
- 2026-10-14 — Phone number → still used, just not verified, because SMS costs too much (anh Đức).
- 2026-10-14 — Existing data → no real users yet, no backfill (anh Đức).
- 2026-10-14 — Lost email access → support + admin handles it manually (anh Đức).
- 2026-10-14 — Account is created only after the email OTP is verified (anh Đức: "verify otp qua email rồi thì tạo").
- 2026-10-14 — Phone uniqueness → "Sdt vẫn là duy nhất trên hệ thống" (anh Đức), against the recommendation to drop UNIQUE; squatting risk accepted (rule 3).
- 2026-10-14 — Proof for genealogy invitations → "Không cần" (anh Đức); risk accepted (rule 4).
- 2026-10-14 — Sending domain → "Tạm mock" (anh Đức); production needs a domain first (rule 7).

## Decision record

- 2026-10-14 — Proposed by Claude (`adr-writer`).
- 2026-10-14 — Open questions answered by anh Đức; merged as Proposed (be#13).
- 2026-10-14 — Accepted by anh Đức: "Xác nhận" in reply to "Anh xác nhận giúp ADR-0003 là Accepted nhé?".
