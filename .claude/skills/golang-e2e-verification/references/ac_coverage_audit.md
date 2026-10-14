# AC coverage audit — process and format

## Where ACs live

`docs/specs/features/<feature>/requirements.md`, written by `il-spec-be` in EARS notation, numbered `AC1`, `AC2`, ... `tasks.md` and the code's own test comments reference these numbers — that's the thread to pull.

## Search strategy

For each AC, search in this order and stop at the first test that genuinely exercises it:

1. **Test name or comment referencing the AC number** — grep the feature's test files for the literal AC id (`AC3`, `AC-3`, or the requirement's key phrase). `golang-clean-arch-tdd`'s verification step already asks implementers to "point to the specific test proving each AC," so a disciplined implementation often already has this thread; don't assume it does.
2. **Behavioral match without an explicit reference** — read the AC's condition/action/response triple (EARS shape: "WHEN \<trigger\> THE SYSTEM SHALL \<response\>") and check whether any existing `t.Run` subtest's input/assertion pair actually matches it. A test titled `TestVerifyOTP_Success` that only asserts a 200 status does not cover an AC about *lockout after N failed attempts* even though both live in the same handler's test file.
3. **No match** — record it as a gap. Do not mark an AC "probably covered by the general happy-path test" — that is exactly the kind of unverified assumption this audit exists to catch.

Check `usecase`, `repository`, `delivery`, and any existing `e2e` test files for the feature — an AC about wiring (auth required on a route) will never be provable at the `usecase` or `repository` layer no matter how thorough those tests are; it can only be covered by delivery-layer or E2E tests.

## Gap table format

Report the audit as a table, one row per AC:

| AC | Requirement (short) | Covered by | Status |
|----|---------------------|-----------|--------|
| AC1 | Register with valid phone+email sends OTP | `usecase/register_usecase_test.go:42` `TestRequestOTP_Success` | ✅ covered |
| AC4 | Locked account rejects further OTP attempts | — | ❌ gap — usecase layer, not E2E |
| AC7 | `/v1/users/me` requires a valid access token | — | ❌ gap — wiring, needs E2E (Step 2 in SKILL.md) |

The "Status" column must say *why* an uncovered AC belongs where it belongs (which layer, or E2E) — that's what lets the reader route the gap to the right follow-up instead of re-auditing it.

## What counts as "proving" an AC

A test proves an AC when its assertion would fail if the AC's behavior regressed. A test that only checks "no error returned" does not prove an AC that specifies a particular status code, response field, or side effect (a row written, a session revoked, a lockout timestamp set). When in doubt, ask: if someone reverted the code that implements this AC, would this test go red? If not, it isn't proof, even if it happens to pass today.
