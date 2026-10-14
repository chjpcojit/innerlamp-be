# Gathering Questions

The skill is two-way: it asks a short, prioritized set of questions *before* writing the final SRS.
This file defines the required categories, the phrasing style, and the "never skip" rule.

## Phrasing style

- **Ask in Vietnamese** (anh/em). Questions to the user are always Vietnamese; only the SRS file is
  English. Keep technical terms (endpoint, cache, JWT…) as-is inside the Vietnamese sentence.
- **One topic per question.** Split compound questions; the user should never have to untangle "do
  you want X and if so Y and how about Z".
- **Concrete and answerable.** Prefer a specific choice over an open prompt. Offer a suggested
  default or an example whenever you can.
- **Short.** Aim for one or two lines. If a question needs background, give one sentence of context,
  then the question.
- **Say why it matters.** Each question should make its blocking effect legible ("this decides
  whether we need an offline cache").

## Required categories (ask the ones that apply — do not force all of them)

### 1. Edge cases & boundary behavior
The highest-value category for a BA. Ask about:
- Empty / missing / invalid input (empty phone, malformed email, no dharma name).
- Boundary values (minimum password length, maximum list size, retry limits).
- Duplicate actions (double-tap submit, re-triggering an OTP while one is pending).
- Offline / no-network / slow-network / timeout behavior.
- Concurrency (two devices, two sessions, backgrounding mid-flow).
- Error and empty states (what the screen shows when the list is empty or the call fails).

### 2. Business rules
- Validation rules and their exact parameters (lengths, formats, allowed characters).
- Permissions and roles (who can see/do what).
- Business constraints (e.g. one OTP per phone per N minutes; resend cooldown).
- Rules with exceptions ("always X, unless Y") — force the "unless" to be spelled out.

### 3. Non-functional requirements
- Performance targets (response time, launch, list rendering).
- Availability / reliability expectations (what happens when the backend is down).
- Security & privacy (PII handled, where it is stored, token handling, device attestation) — see the
  project's `Core/Logging` + PII-masking rules in AGENTS.md.
- Compliance / App Store policy constraints.

### 4. Data & persistence
- What data is captured, what is persisted, where (local Core Data vs UserDefaults vs Keychain vs
  server), and what is transient.
- Retention / deletion (GDPR-style: does the user's data get deleted with the account?).
- Sync/conflict behavior if data exists both locally and server-side.

### 5. Dependencies & integration
- External systems (backend APIs, third-party SDKs) and their status (confirmed vs not).
- Cross-feature coupling (does this depend on Onboarding, SignUp, an existing OTP flow?).
- Ordering constraints (must run after X, before Y).

### 6. Technical feasibility & risks (PM/BA flagging, not deep design)
- Platform constraints that could block the feature (background execution limits, App Store review
  risk, required permissions).
- Cost/effort flags the team should see before committing (e.g. "this implies offline sync, which is
  a large effort").

## The "never skip" rule

Every question in the required set that applies to this feature is either answered by the user, or —
if the user says "skip" or leaves it unanswered — recorded inline in the SRS as:

```
> ⚠️ Not confirmed: <the question, in one sentence> — <what it blocks / what depends on it>.
```

Never silently invent the answer, and never drop the question because it is awkward. When the user
later answers, move it to "Open questions (resolved)" with date + resolution, and note the downstream
effect.

## Prioritization

Present the most **blocking** questions first (those whose answers change the shape of the feature or
the architecture). If there are many, batch them by category rather than dumping everything at once.
