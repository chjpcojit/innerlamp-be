---
type: Feature
title: Genealogy (gia phả)
description: A clan's family tree and chronicle owned by one user and shared read-only, by phone-number invitation, with relatives; the server computes age (Western and traditional), relatives, branch (chi), kinship term and the lunar equivalent of stored solar dates.
resource: services/appService/
tags: [genealogy, family-tree, sharing, invitation, kinship, lunar-calendar]
status: draft
generated: { by: claude-code, at: 2026-10-11T01:30:00+07:00 }
sources:
  - path: docs/specs/features/genealogy/SRS.md
    kind: spec
  - path: docs/specs/features/genealogy/requirements.md
    kind: spec
  - path: docs/specs/features/genealogy/design.md
    kind: spec
  - path: docs/specs/features/genealogy/tasks.md
    kind: spec
  - path: docs/specs/features/genealogy/ac-audit.md
    kind: spec
  - path: docs/specs/api/openapi.yaml
    kind: openapi
  - path: migrations/0028_genealogy.sql
    kind: migration
---

# Summary

A signed-in user creates a **gia phả**: a tree of people (blood members of the clan and their in-laws, *dâu* / *rể*) plus a chronicle (*phả ký*). The creator is the **owner** and the only person who can change anything. The owner picks one living member of the tree and invites them by phone number; the invitation waits as `pending`, the invitee finds it in their inbox and accepts (becoming a read-only **viewer**, standing at the invited member) or declines, and the owner sees the status change. People with no role cannot tell the tree exists (`404 GENEALOGY_NOT_FOUND`, byte-identical to a tree that does not exist).

Everything shown for a selected person is **computed on the server**: age, parents, spouses, siblings (half-siblings flagged), children, line of descent from the founder, branch path (*chi*), and what that person is to "me" (Bác, Chú, Thím, Anh họ…). "Me" is the member whose `phone_number` equals the caller's own account number. Chạp-day and death-anniversary dates are **stored solar** with a flag saying the user entered them as lunar; the server returns the lunar equivalent. Module `genealogy`; user stories US-GEN-01…10.

**Current limitation**: no invitation notification exists — the invitee only sees it by calling `GET /v1/invitations` (push is a separate, later, application-wide feature; SMS will not be used). People without an account see the invitation only after registering with that number, so the owner must tell them outside the app.

# Architecture

Clean Architecture inside `services/appService/`, dependency direction `delivery → usecase → domain` and `repository → domain`. Every write that needs an invariant (member cap, one clan head, one phone per member, viewer cap) runs in a repository transaction that first takes `SELECT … FOR UPDATE` on the `genealogies` row, so writes to one tree are serialised and the invariants hold under concurrency. Tunable numbers come from `config.GenealogyConfig`; schema-fixed values (field lengths, `kind`/`status` enums) are constants.

| Layer | Type | File |
|---|---|---|
| Domain / Entity | Domain Entity | `services/appService/domain/genealogy.go` — `Genealogy`, `Chronicle`, `GenealogyMember`, `Viewer`, `Invitation`, computed `MemberProfile`/`Kinship`, the repository ports and sentinel errors |
| Domain / Mock | test support | `services/appService/domain/mock_genealogy_repository.go` — five hand-written mocks |
| Config | configuration | `services/appService/config/config.go` — `GenealogyConfig` (`max_per_user`, `max_members`, `max_generation`, `max_spouses`, `max_viewers`, `invitation_ttl`) with `Validate()`; `cmd/main.go` stops the service when it is invalid |
| Usecase | UseCase | `services/appService/usecase/genealogy_usecase.go` — create / list / get / rename / delete / chronicle; role resolution (`resolveRole`, `requireOwner`) |
| Usecase | UseCase | `services/appService/usecase/genealogy_member_usecase.go` — list / get / add / update / delete / move; clan-head rules; `warnings` |
| Usecase | UseCase | `services/appService/usecase/genealogy_access_usecase.go` — invite, owner's invitation list and cancel, viewers list and `RemoveViewer` |
| Usecase | UseCase | `services/appService/usecase/genealogy_invitation_usecase.go` — the invitee's inbox, accept, decline |
| Usecase | UseCase | `services/appService/usecase/genealogy_profile_usecase.go` — loads the tree once and computes the profile |
| Usecase | pure functions | `services/appService/usecase/genealogy_kinship.go` (age, relatives, lineage, chi, kinship), `genealogy_lunar.go` (`SolarToLunar`, UTC+7, years 1900–2199), `genealogy_validation.go` (per-field rules) |
| Repository | Repository | `services/appService/repository/postgres_genealogy_repository.go` — trees, chronicle, `ResolveRole`; `postgres_genealogy_member_repository.go` — members, `Move`; `postgres_genealogy_access_repository.go` — viewers, invitations, `NewUserPhoneLookup` |
| Delivery / Handler | Handler | `services/appService/delivery/http/genealogy_handler.go`, `genealogy_member_handler.go`, `genealogy_access_handler.go` — routes, DTOs, strict body decoding; error mapping in `errors.go` |
| Migration | Migration | `migrations/0028_genealogy.sql` — five tables; composite FKs keep every relationship inside one tree |
| Wiring | composition root | `services/appService/cmd/main.go` — builds the five repositories, five use cases and four handlers, registers 21 routes each wrapped by `authMiddleware.AuthenticateFunc`; touches no `⚠️ DEV ONLY` block |
| Verification | E2E | `services/appService/e2e/genealogy_e2e_test.go` — real binary + Postgres; acceptance map in `docs/specs/features/genealogy/ac-audit.md` |

`pkg/` imports relied on: `pkg/jwt` (access-token claims → user id), `pkg/utils` (envelope and strict JSON decoding), `pkg/validation` (`NormalizePhone`), `pkg/database`.

# API surface

All 21 routes require `Authorization: Bearer <access_token>`; the acting user is the token's `user_id`.

| Method | Path | Who | Handler |
|---|---|---|---|
| `POST` / `GET` | `/v1/genealogies` | any user | `GenealogyHandler.Create` / `List` |
| `GET` | `/v1/genealogies/{id}` | owner, viewer | `GenealogyHandler.Get` |
| `PUT` / `DELETE` | `/v1/genealogies/{id}` | owner | `GenealogyHandler.Update` / `Delete` |
| `PUT` | `/v1/genealogies/{id}/chronicle` | owner | `GenealogyHandler.UpdateChronicle` |
| `GET` | `/v1/genealogies/{id}/members` | owner, viewer | `GenealogyMemberHandler.List` |
| `POST` | `/v1/genealogies/{id}/members` | owner | `GenealogyMemberHandler.Add` |
| `GET` | `/v1/genealogies/{id}/members/{memberId}` | owner, viewer | `GenealogyMemberHandler.Get` |
| `PUT` / `DELETE` | `/v1/genealogies/{id}/members/{memberId}` | owner | `GenealogyMemberHandler.Update` / `Delete` |
| `PUT` | `/v1/genealogies/{id}/members/{memberId}/relation` | owner | `GenealogyMemberHandler.Move` |
| `GET` | `/v1/genealogies/{id}/members/{memberId}/profile` | owner, viewer | `GenealogyMemberHandler.Profile` |
| `POST` / `GET` | `/v1/genealogies/{id}/invitations` | owner | `GenealogyAccessHandler.Invite` / `ListInvitations` |
| `DELETE` | `/v1/genealogies/{id}/invitations/{invitationId}` | owner | `GenealogyAccessHandler.CancelInvitation` |
| `GET` | `/v1/genealogies/{id}/viewers` | owner | `GenealogyAccessHandler.ListViewers` |
| `DELETE` | `/v1/genealogies/{id}/viewers/{userId}` | owner, or the viewer themself | `GenealogyAccessHandler.RemoveViewer` |
| `GET` | `/v1/invitations` | any user (own number) | `GenealogyInvitationHandler.ListMine` |
| `POST` | `/v1/invitations/{id}/accept` · `/decline` | the invited number | `GenealogyInvitationHandler.Accept` / `Decline` |

Error mapping (`delivery/http/errors.go`): `GENEALOGY_NOT_FOUND` 404, `GENEALOGY_FORBIDDEN` 403 (a viewer on a write route), `GENEALOGY_LIMIT_REACHED` / `MEMBER_LIMIT_REACHED` / `VIEWER_LIMIT_REACHED` / `MEMBER_HAS_DESCENDANTS` / `FOUNDER_NOT_DELETABLE` / `FOUNDER_NOT_MOVABLE` / `MEMBER_PHONE_ALREADY_USED` / `VIEWER_ALREADY_EXISTS` / `INVITATION_NOT_PENDING` / `INVITATION_ALREADY_PENDING` 409, `MEMBER_NOT_FOUND` / `VIEWER_NOT_FOUND` / `INVITATION_NOT_FOUND` 404, and `VALIDATION_ERROR` 400 with `details.field`. The wire contract is in `docs/specs/api/openapi.yaml` (tags `Genealogies`, `GenealogyMembers`, `GenealogyAccess`). No `API Endpoint` concepts exist yet for these routes.

# Request flow

```
POST /v1/genealogies/{id}/members                          (owner writes a member)
  └─ authMiddleware.AuthenticateFunc                        (access token → user_id claim)
       └─ GenealogyMemberHandler.Add                        (delivery/http)
            ├─ DecodeJSONBodyStrict (unknown / server-owned fields → 400)
            └─ GenealogyMemberUsecase.Add                   (usecase)
                 ├─ requireOwner → ResolveRole              (one query; no role → 404, viewer → 403)
                 ├─ FindByID(related)                       (not in this tree → 404)
                 ├─ child rules: related is a male blood member, generation ≤ cap
                 ├─ buildMember                             (per-field validation)
                 ├─ GenealogyMemberRepository.Add           (repository, ONE transaction)
                 │    ├─ SELECT … FROM genealogies FOR UPDATE
                 │    ├─ member cap, spouse cap, default birth_order, clan-head swap
                 │    └─ INSERT … RETURNING (unique phone per tree → ErrMemberPhoneInUse)
                 └─ warnings: child born before parent → advisory, never an error
```

```
Invitation journey
  owner  POST …/invitations {member_id, phone_number?}  → tx: lock tree, member alive,
                                                          write the number onto the member,
                                                          cap = viewers + pending, INSERT pending
  invitee GET /v1/invitations                           → rows whose phone_number = my account number
  invitee POST /v1/invitations/{id}/accept              → tx: lock tree, then lock invitation;
                                                          INSERT genealogy_viewers; status=accepted
  owner  GET …/invitations, GET …/viewers               → status accepted; viewer.member_id = the node
```

# Validation rules

- **Per field** (`usecase/genealogy_validation.go`, same rules for founder, spouses, `POST` and `PUT`): `full_name` 1–100; `gender` male/female (spouse default = opposite of the attached member); `is_alive` required; years 1000..current year (Asia/Ho_Chi_Minh), death ≥ birth; text lengths (residence 200, dharma name 100, tablet 200, burial 200, note 1000); `phone_number` normalised by `validation.NormalizePhone`; anniversary month+day together with `death_year`, day valid for the month (February counts 29); `birth_order` 1..1000 and not for the founder or an in-law. A living member cannot carry death year, tablet, burial or anniversary.
- **Tree rules** (use case, repeated under lock in the repository): founder must be male; children attach only to **male blood** members (a daughter's children are not in the tree); ≤ `max_spouses` spouses; ≤ `max_generation` generations; `other_parent_id` must be an in-law spouse of the parent; a member cannot move under itself or its descendants; the founder cannot be deleted or moved.
- **Clan head** (`is_clan_head`): one per tree, only a living male blood member; changing the holder's `gender`/`is_alive` without clearing the flag is refused; an absent flag on `PUT` keeps it.
- **Chronicle**: field lengths per `buildChronicle`; the chạp date is `chap_year`+`chap_month`+`chap_day` all or none; the derived fields (`founder_name`, counts, `*_lunar_*`) are refused if sent.
- **Role**: format of ids is checked after the role is resolved, so a stranger cannot distinguish a malformed id from a missing tree.

# Pending work

- **Notifications**: nothing tells an invitee they were invited; `GET /v1/invitations` is the only channel. Push (FCM/APNs) is planned as a separate application-wide feature; SMS is deliberately not used.
- **Dev placeholders that this feature's E2E rides on**: users are registered through the dev OTP (`111111`) and log-only SMS (see [Register](/features/register.md)); the genealogy code itself touches no `⚠️ DEV ONLY` block.
- **Spec questions**: none open. Settled on 2026-10-11: age is shown both as Western age (`age`) and traditional age (`traditional_age` = Western + 1); a declined number may be re-invited at once; a reassigned phone number is an accepted risk; sibling order is edited one member at a time; a viewer whose node was deleted keeps viewing without a position; a child's mother is optional (the client picks only the father); no blood↔in-law conversion; concurrent edits are last-write-wins.
- **Not measured**: NFR-001/002 (tree load ≤ 2 s, profile ≤ 1 s at 5000 members) are BA-proposed targets with no load test.
- **Expired invitations are not swept**: a `pending` row past `expires_at` is *reported* as `expired` on read and only rewritten when the same number is invited again (the unique pending index needs it); there is no cleanup job.
- **Photos** (hoành phi, member portraits) are out of v1, blocked on object storage.
- **Docs**: `docs/specs/db/erd.svg` still needs re-rendering from `erd.mmd`; the `web/docs` guide for this feature is not written yet.
- Promote to `stable` once the open questions above are answered and the invitation journey has been exercised against a deployment that uses real OTP/SMS rather than the dev placeholders.

# Known gaps vs. design

- `design.md` first planned `Revoke` and `Leave` as separate use-case methods; the code has one `RemoveViewer` (the owner removes anyone, a viewer only themself, the owner asking for themself gets `VIEWER_NOT_FOUND`).
- `design.md` named migration `0020`; the numbers `0020`–`0027` were already used, so the file is `0028_genealogy.sql`.
- The spec's `GenealogyDetail` mentions a `stats` object; the wire shape puts `founder_name`, `generations_count` and `members_count` inside `chronicle` (as the contract table says), with no separate `stats`.

# Related concepts

- [Register](/features/register.md) — creates the accounts the owner and invitees use; the account's `phone_number` is what both the invitation and the "me" position are matched on.
- [Login](/features/login.md) — issues the access token every genealogy route requires.
- [packages/middleware.md](/packages/middleware.md) — the chain that wraps the mux and provides the 404/405 envelope for unknown or wrong-method paths.
- No `Feature` concept exists yet for `memorial` (the private memorial book this feature sits beside), and no `Entity`, `UseCase`, `API Endpoint` or `Migration` concepts exist for genealogy.
