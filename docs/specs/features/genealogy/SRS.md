# Software Requirements Specification — Genealogy (Gia phả)

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Genealogy (family tree + clan chronicle) |
| Product | InnerLamp |
| Version | 1.2 |
| Last updated | 2026-10-11 |
| Status | draft — 1 non-blocking open item (Appendix A: performance targets); every other question is resolved |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `tasks.md` (already exist at v5.12 — see 2.7 for the mapping and Appendix B for the cross-reference) |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, how a person using the InnerLamp app builds and keeps the family tree and chronicle of their clan (**gia phả**), and how they let relatives look at it. It is written for stakeholders and non-technical readers. It is the business-facing source for the engineering spec set under `docs/specs/features/genealogy/` (v5.12), which stays the authority on how the feature is built.

### 1.2 Scope

**In scope:**
- Creating a family tree together with its progenitor (thuỷ tổ) and his spouses.
- Building the tree: adding children and spouses, editing and deleting members, correcting a mistaken relationship, ordering siblings.
- Recording per-person information: living/deceased status, years, residence, Buddhist name, tablet location, burial place, death anniversary, phone number, note.
- Recording the clan chronicle (phả ký): surname, location, origin, history, ancestral hall, annual clan commemoration day, family teachings.
- Marking one living male member as the clan head (trưởng tộc) as a remembered flag.
- Inviting a specific person in the tree to view the tree by phone number; accepting or declining; seeing invitation status.
- Showing every person in the tree what is known about a selected member, computed for them: age, relatives, line of descent, branch (chi), and how that person relates to "me".
- Keeping the tree private to the owner and the people the owner invited.

**Out of scope:**
- Letting anyone other than the owner edit the tree (no co-editors, no clan-head authority).
- Transferring ownership of a tree.
- Photos (ancestral-hall plaque, member portraits).
- Push or SMS notification of invitations — a separate application-wide notification feature, built later. SMS will not be used (cost).
- Recurring reminders for death anniversaries or the clan commemoration day (belongs to the memorial feature).
- Maternal-side kinship terms (ông bà ngoại, cậu, dì) — returned as "unknown" in this version.
- Public sharing links, PDF/print export, search, edit history, undo, GEDCOM import, merging two trees.

### 1.3 Product overview

InnerLamp is a mobile app for Vietnamese Buddhist practice and family remembrance. Alongside meditation and temple content, it already offers a private memorial book (sổ giỗ chạp) per user. Genealogy adds a **shared but private** family tree: unlike the memorial book, several relatives can look at the same tree, but only its creator can change it.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Genealogy / family tree (gia phả) | The clan record owned by one user: a tree of people plus a chronicle. |
| Owner | The user who created the tree. Sees and edits everything. Exactly one per tree; cannot be changed. |
| Viewer | A user who accepted an invitation. Sees the tree, the chronicle and member profiles; cannot change anything. |
| Member | One person in the tree. Either a **blood member** (belongs to the clan) or an **in-law** (spouse of a blood member). |
| Progenitor / founder (thuỷ tổ) | The first generation blood member, who has no parent in the tree. Must be male. |
| In-law (dâu / rể) | A spouse who does not belong to the clan: *dâu* is the wife of a male blood member, *rể* the husband of a female blood member. Has no generation of their own. |
| Generation (đời) | Depth in the tree; the founder is generation 1. |
| Branch (chi) | A line that starts wherever a person has two or more sons; each son heads a branch, and branches nest. |
| Clan head (trưởng tộc) | A flag marking one living male blood member as the head of the clan. Not a role; grants no permission. |
| Chronicle (phả ký) | The clan's general record: surname, location, origin, history, ancestral hall, commemoration day, family teachings. |
| Chạp họ | The annual clan commemoration day (ngày chạp họ). |
| Death anniversary (ngày giỗ) | The yearly remembrance day of a deceased member. |
| Family teachings (gia huấn) | Hoành phi (horizontal plaque) and câu đối (parallel couplet), each with a reading and a meaning, plus a free-text teaching. |
| Invitation | An owner's request, addressed to a phone number and tied to one member, to view the tree. States: pending, accepted, declined, expired, cancelled. |
| "Me" / my position | The member in a tree whose phone number equals the signed-in user's own phone number. |
| Solar / lunar date | Dương lịch / âm lịch. Dates are stored as solar; a flag remembers the user entered them as lunar. |

### 1.5 References

- `docs/specs/features/genealogy/requirements.md`, `design.md`, `flow.md`, `tasks.md`, `changelog.md` (v5.12) — the engineering spec this SRS feeds.
- `docs/specs/features/memorial/` — the private memorial book, the nearest existing feature.
- `docs/assumptions/genealogy/A-002-invitation-defaults.md` — invitation assumptions.
- IEEE 29148-2018, *Systems and software engineering — Life cycle processes — Requirements engineering*.

### 1.6 Document organization

Section 2 describes the feature and its users. Section 3 holds the interfaces, the numbered functional requirements (each with acceptance scenarios), and non-functional requirements. Section 4 gives verification and the traceability matrix. Appendix A lists open questions; Appendix B cross-references requirement IDs to the engineering spec's acceptance criteria.

---

## 2. Overall description

### 2.1 Product perspective

A new feature that sits beside the memorial book. It depends on existing sign-up and sign-in (every action needs a signed-in user, and a user's phone number is their account identity). It exposes no data publicly. It does not send any message outside the app.

### 2.2 Product functions

- Create a tree with its founder and the founder's spouses (`FR-001`–`FR-003`).
- See my trees and open one; owner renames or deletes it (`FR-004`–`FR-007`).
- Build and correct the tree (`FR-008`–`FR-017`).
- Record dates in solar or lunar form (`FR-018`–`FR-019`).
- Record the clan chronicle (`FR-020`–`FR-021`).
- Mark the clan head (`FR-022`).
- Invite, accept, decline, cancel, expire, and remove viewers (`FR-023`–`FR-031`).
- Know where "I" am in the tree (`FR-032`–`FR-034`).
- See a computed profile of any member (`FR-035`–`FR-036`).
- Keep the tree private and read-only for viewers (`FR-037`–`FR-038`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Owner | The signed-in user who created the tree. | Full read and write; the only person who invites, removes viewers, and deletes the tree. |
| Viewer | A signed-in user who accepted an invitation. | Read-only: tree, chronicle, member profiles, own position. May leave the tree. Sees phone numbers and residence of living members. |
| Invitee | A person holding a pending invitation; may not have an account yet. | Sees only the tree's name, the name of the member they were invited as, and the inviter's phone number, until they accept. |
| Other user | Any signed-in user with no role in a tree. | Cannot see the tree, and cannot tell whether it exists. |

### 2.4 Operating environment

- Platform: server-side Go services consumed by the iOS app (and later other clients).
- Devices: not applicable to this document.
- Network: online required for every action; no offline editing.

### 2.5 Design and implementation constraints

- Clean Architecture layering per the project `CLAUDE.md`; the feature lives in the app-facing service.
- Every route requires a signed-in user; the acting user always comes from the sign-in, never from the request body.
- Phone numbers are normalized with the same rule as sign-up, so equality of numbers is reliable.
- Photos are excluded until cloud object storage exists.

### 2.6 User documentation

N/A — the app guides the flow in-screen; no manual is required.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - A user has exactly one phone number, and a member has at most one phone number.
  - The tree is clan-centred: only descendants of male blood members appear as blood members; children of a daughter belong to another clan's tree.
  - The owner tells relatives outside the app that an invitation is waiting, since no SMS or push is sent in this version.
  - The ceiling of 20 viewers per tree reuses the number first agreed for editors.
  - Two members of the same tree cannot share a phone number (a consequence of "one member, one account"); the same number may appear in different trees.
  - A pending invitation for a deceased member is not possible; an already-sent invitation is cancelled when its member's phone number is cleared or changed.
- **Dependencies:**
  - Sign-up and sign-in (phone number identity) — existing.
  - Memorial feature for recurring reminders — separate.
  - Application-wide notification feature (push) — to be built; this feature does not wait for it.
- **Mapping to the engineering spec:** this SRS introduces no change to `requirements.md` v5.12; Appendix B maps each `FR-*` to its acceptance-criteria numbers.

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

Screens are designed separately against the Design System. This feature needs: a tree list with an invitation badge; a tree overview; a tree view with a member profile sheet; add/edit member forms; chronicle editor; invitation management (owner) and invitation inbox (invitee). No pixel-level requirement is stated here.

#### 3.1.2 Hardware interfaces

**N/A** — no custom hardware interface.

#### 3.1.3 Software interfaces

The app talks to the back-end over 21 routes, all requiring sign-in. At contract level:

| Area | Routes | Who may call |
|---|---|---|
| Tree | create, list mine, open, rename, delete | create/list: any user; open: owner, viewer; rename/delete: owner |
| Chronicle | replace chronicle | owner |
| Members | list, open one, add, edit, delete, change relationship | list/open: owner, viewer; the rest: owner |
| Member profile | computed profile of one member | owner, viewer |
| Viewers | list viewers, remove a viewer (or leave) | owner; a viewer may remove only themself |
| Invitations (owner) | invite, list, cancel | owner |
| Invitations (invitee) | list mine, accept, decline | any signed-in user, for their own number |

The exact request/response shapes, error codes, and status values are in the engineering spec's API contract table and the OpenAPI document (not yet updated).

#### 3.1.4 Communication interfaces

HTTPS, JSON, bearer access token on every request. No other channel is used; in particular no SMS and no email.

### 3.2 Functional requirements

#### Creating a family tree

**FR-001 — Create a tree with its founder**
- **Priority:** Must
- **Description:** WHEN a signed-in user submits a tree name, a hometown and the founder's details, THE SYSTEM SHALL create the tree, an empty chronicle and the founder as generation 1 owned by that user, all together or not at all.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief (create tree and enter progenitor).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Create a tree with a founder
    Given I am signed in
    When I create a tree named "Họ Nguyễn" with hometown "Nam Định" and a founder "Nguyễn Văn A"
    Then the tree appears in my list
    And the founder is shown as generation 1
    And I am its owner

  Scenario: Tree name is blank
    Given I am signed in
    When I create a tree with a blank name
    Then I am told the name is required
    And no tree is created

  Scenario: Founder information is incomplete
    Given I am signed in
    When I create a tree without saying whether the founder is alive
    Then I am told that information is required
    And no tree is created
  ```

**FR-002 — The founder must be male and may have spouses**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL accept only a male founder, and WHEN spouses are listed THE SYSTEM SHALL record up to five of them as in-laws of generation 1, defaulting each spouse's gender to the opposite of the founder's.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10 (founder must be male); founding-couple rule.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Female founder is rejected
    Given I am signed in
    When I create a tree whose founder is female
    Then I am told the founder must be male
    And no tree is created

  Scenario: Founder with spouses
    Given I am signed in
    When I create a tree with a male founder and two spouses
    Then both spouses are shown beside the founder as in-laws
    And their gender defaults to female

  Scenario: Too many spouses
    Given I am signed in
    When I create a tree listing six spouses for the founder
    Then I am told at most five are allowed
    And no tree is created
  ```

**FR-003 — Limit of 10 trees per owner**
- **Priority:** Must
- **Description:** WHEN a user who already owns 10 trees tries to create another, THE SYSTEM SHALL refuse, including when two creations are submitted at the same moment.
- **Verification method:** unit test, concurrency test
- **Source:** Decision 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Eleventh tree
    Given I already own 10 trees
    When I create another tree
    Then I am told I reached the limit
    And no tree is created

  Scenario: Two creations at once at the limit
    Given I own 9 trees
    When two creation requests arrive at the same time
    Then exactly one tree is created
  ```

#### Seeing and managing trees

**FR-004 — List my trees**
- **Priority:** Must
- **Description:** WHEN a signed-in user asks for their trees, THE SYSTEM SHALL list every tree they own or are a viewer of, newest first, each marked with the user's role.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Owner and viewer trees together
    Given I own one tree and I am a viewer of another
    When I open my trees
    Then I see both, each marked with my role

  Scenario: No trees
    Given I own no tree and am a viewer of none
    When I open my trees
    Then I see an empty list
  ```

**FR-005 — Open a tree overview**
- **Priority:** Must
- **Description:** WHEN the owner or a viewer opens a tree, THE SYSTEM SHALL show its general information, the chronicle, the clan head if any, which member is "me" if any, and counts of generations and members computed at that moment.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Overview shows counts
    Given a tree with a founder, two spouses and three children
    When I open the tree
    Then it shows 2 generations and 6 members
    And the founder's name is shown

  Scenario: Overview as a viewer
    Given I am a viewer of a tree
    When I open the tree
    Then I see the same information as the owner
    And I am told my role is viewer
  ```

**FR-006 — Only the owner renames the tree**
- **Priority:** Must
- **Description:** WHEN the owner changes the tree name or hometown, THE SYSTEM SHALL save it; WHEN a viewer tries, THE SYSTEM SHALL refuse and change nothing.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10 (viewer is read-only).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Owner renames
    Given I own a tree
    When I change its name
    Then the new name is shown

  Scenario: Viewer tries to rename
    Given I am a viewer of a tree
    When I try to change its name
    Then I am told I may not change this tree
    And the name is unchanged
  ```

**FR-007 — Only the owner deletes the tree**
- **Priority:** Must
- **Description:** WHEN the owner deletes the tree, THE SYSTEM SHALL permanently remove the tree with its members, chronicle, viewers and invitations; WHEN a viewer tries, THE SYSTEM SHALL refuse.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Owner deletes
    Given I own a tree with viewers
    When I delete the tree
    Then the tree is gone for me and for every viewer

  Scenario: Viewer tries to delete
    Given I am a viewer of a tree
    When I try to delete the tree
    Then I am told I may not do this
    And the tree remains
  ```

#### Building the tree

**FR-008 — Add a child**
- **Priority:** Must
- **Description:** WHEN the owner adds a child to a male blood member, THE SYSTEM SHALL create a blood member one generation below, optionally naming which spouse is the other parent and giving a birth order.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief (create people in the tree).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Add a child
    Given the founder is generation 1
    When I add a child to the founder
    Then the child is shown in generation 2 under the founder

  Scenario: Child with the other parent named
    Given the founder has two spouses
    When I add a child and name the second spouse as the other parent
    Then the child is shown as that spouse's child

  Scenario: Other parent is not a spouse of the parent
    Given a blood member and an in-law who is not that member's spouse
    When I add a child and name that in-law as the other parent
    Then I am told the other parent must be the parent's spouse
    And no member is added

  Scenario: Birth order defaults to last
    Given a member already has two children
    When I add a third child without a birth order
    Then the new child is placed third
  ```

**FR-009 — Children of daughters are not added**
- **Priority:** Must
- **Description:** WHEN the owner tries to add a child under a female blood member or under an in-law, THE SYSTEM SHALL refuse.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Child under a daughter
    Given a female blood member
    When I add a child under her
    Then I am told children of a daughter are not part of this tree
    And no member is added

  Scenario: Child under an in-law
    Given an in-law member
    When I add a child under that in-law
    Then I am told a child must be attached to a blood member
    And no member is added
  ```

**FR-010 — Add a spouse**
- **Priority:** Must
- **Description:** WHEN the owner adds a spouse to a blood member, THE SYSTEM SHALL create an in-law of the same generation, allowing up to five spouses per member, and SHALL NOT accept a spouse of an in-law.
- **Verification method:** unit test
- **Source:** Genealogy brief; remarriage and polygamy rule.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Add a spouse
    Given a blood member in generation 3
    When I add a spouse
    Then the spouse is shown in generation 3 as an in-law
    And the label reads daughter-in-law for a male member, son-in-law for a female member

  Scenario: Sixth spouse
    Given a blood member with five spouses
    When I add another spouse
    Then I am told at most five are allowed

  Scenario: Spouse of an in-law
    Given an in-law member
    When I add a spouse to the in-law
    Then I am told this is not allowed
  ```

**FR-011 — Size limits**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL refuse to add a member when the tree already has 5000 members, or when the addition would create generation 61, including when two additions are submitted at the same moment.
- **Verification method:** unit test, concurrency test
- **Source:** Decision 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Tree is full
    Given a tree with 5000 members
    When I add one more member
    Then I am told the tree is full

  Scenario: Too deep
    Given a member in generation 60
    When I add a child
    Then I am told the tree cannot go deeper
  ```

**FR-012 — Edit a member**
- **Priority:** Must
- **Description:** WHEN the owner edits a member, THE SYSTEM SHALL save name, gender, birth year, living status, death year, residence, Buddhist name, tablet location, burial place, death anniversary, phone number, note and sibling order; THE SYSTEM SHALL refuse information that contradicts the person's status.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief; US-GEN-04.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Edit residence
    Given a living member
    When I change the residence
    Then the new residence is shown

  Scenario: Living member given a death year
    Given a member marked as living
    When I enter a death year
    Then I am told only a deceased person has a death year
    And nothing is saved

  Scenario Outline: Reject impossible years
    Given a member
    When I enter birth year <birth> and death year <death>
    Then I am told the years are not valid

    Examples:
      | birth | death |
      | 999   | 1900  |
      | 1950  | 1940  |

  Scenario: Gender change would orphan children
    Given a male member who has children
    When I change the member's gender to female
    Then I am told a female member cannot have children in this tree
  ```

**FR-013 — Delete a member only from the bottom up**
- **Priority:** Must
- **Description:** WHEN the owner deletes a blood member with no children, THE SYSTEM SHALL delete that member and their in-law spouses; WHEN the member has children THE SYSTEM SHALL refuse; WHEN the member is the founder THE SYSTEM SHALL refuse.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10 ("block deletion; delete along the tree").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Delete a leaf
    Given a blood member without children and with one spouse
    When I delete the member
    Then the member and the spouse are gone

  Scenario: Delete a member with children
    Given a blood member with a child
    When I delete the member
    Then I am told to remove the descendants first
    And nothing is deleted

  Scenario: Delete the founder
    Given the founder of a tree
    When I delete the founder
    Then I am told to delete the whole tree instead
    And the founder remains

  Scenario: Delete an in-law
    Given an in-law member
    When I delete the in-law
    Then only the in-law is removed
    And their children keep their blood parent but lose the link to this spouse
  ```

**FR-014 — Warn when a child is born before a parent**
- **Priority:** Should
- **Description:** WHEN a saved birth year makes a child older than their parent, THE SYSTEM SHALL still save it and SHALL return a warning naming both people; otherwise it SHALL return no warning.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10 ("warn, do not block").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Child born before parent
    Given a parent born in 1950
    When I add a child born in 1945
    Then the child is saved
    And I see a warning that the child was born before the parent

  Scenario: Same birth year
    Given a parent born in 1950
    When I add a child born in 1950
    Then the child is saved
    And I see no warning

  Scenario: Parent's year edited
    Given a child born in 1970 and a parent born in 1950
    When I change the parent's birth year to 1990
    Then the change is saved
    And I see a warning about the child
  ```

**FR-015 — Correct a mistaken relationship**
- **Priority:** Must
- **Description:** WHEN the owner moves a blood member under a different male blood member, or moves an in-law to a different blood member, THE SYSTEM SHALL apply the move as a whole, shifting the generation of the whole branch, and SHALL refuse a move that would create a loop, exceed 60 generations, or move the founder.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10 ("users may enter wrong data; must be changeable").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Move a branch to another parent
    Given a member with two descendants under parent A
    When I move the member under parent B who is one generation deeper
    Then the member and both descendants each appear one generation deeper

  Scenario: Move under own descendant
    Given a member with a child
    When I move the member under that child
    Then I am told this would create a loop
    And nothing changes

  Scenario: Move the founder
    Given the founder
    When I try to move the founder
    Then I am told the founder cannot be moved

  Scenario: Move would exceed 60 generations
    Given a branch whose deepest member would land in generation 61
    When I move it
    Then I am told the tree cannot go that deep
    And nothing changes

  Scenario: Move an in-law to another spouse-holder
    Given an in-law whose children name them as the other parent
    When I attach the in-law to a different blood member
    Then those children remain with their blood parent
    And are no longer linked to this in-law
  ```

**FR-016 — Sibling order**
- **Priority:** Should
- **Description:** THE SYSTEM SHALL keep a birth order among children of the same parent, set when adding or editing; equal numbers are ordered by creation.
- **Verification method:** unit test
- **Source:** Kinship needs elder/younger among siblings, independent of birth year.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Set order by hand
    Given two siblings
    When I set the younger one as first
    Then the siblings are listed with that one first

  Scenario: Order for founder and in-laws
    Given the founder or an in-law
    When I give a birth order
    Then I am told birth order does not apply to this member
  ```
  (Resolved 2026-10-11: editing the order one member at a time is accepted; no re-order action in v1.)

**FR-017 — View the whole tree**
- **Priority:** Must
- **Description:** WHEN the owner or a viewer opens the tree, THE SYSTEM SHALL provide every member (up to 5000) so the app can draw the tree, ordered by generation then creation.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief (viewer sees the tree).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Full tree is returned
    Given a tree of 300 members
    When I open the tree view
    Then all 300 members are available

  Scenario: Member of another tree
    Given a member that exists only in a tree I cannot see
    When I ask for that member
    Then I am told it was not found
  ```

#### Dates

**FR-018 — Death anniversary stored as a solar date with a "entered as lunar" flag**
- **Priority:** Must
- **Description:** WHEN the owner records a death anniversary of a deceased member, THE SYSTEM SHALL store it as a solar date (the death year plus month and day) whichever calendar the user typed it in, remember whether the user chose lunar, and SHALL show the matching lunar date computed by the server.
- **Verification method:** unit test (conversion table), end-to-end test
- **Source:** Decision 2026-10-10 ("store solar, keep a lunar flag, server converts back").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Anniversary entered as lunar
    Given a deceased member with a death year
    When I enter a lunar anniversary and the app sends the converted solar date with the lunar flag
    Then the member shows the solar date
    And shows the lunar date computed by the system
    And the app knows to display it as lunar first

  Scenario: Month and day must come together
    Given a deceased member
    When I enter a month without a day
    Then I am told both are required

  Scenario: Living member
    Given a member marked as living
    When I enter an anniversary
    Then I am told only a deceased person has one

  Scenario: Date outside the conversion range
    Given a member who died in a year before 1900
    When I view the member
    Then no lunar date is shown
    And the solar date remains
  ```

**FR-019 — Clan commemoration day stored the same way**
- **Priority:** Must
- **Description:** WHEN the owner records the chạp họ day, THE SYSTEM SHALL require year, month and day together, validate the day against the month, store them as a solar date with the "entered as lunar" flag, and show the lunar date computed by the server.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Valid day
    Given the chronicle of a tree
    When I record 31 January 2026 as the commemoration day
    Then the chronicle shows that solar date and its lunar equivalent

  Scenario: Day that does not exist
    Given the chronicle of a tree
    When I record 30 February
    Then I am told the date is not valid

  Scenario: Clearing
    Given a chronicle with a commemoration day
    When I save the chronicle without it
    Then no commemoration day is shown
  ```

#### Chronicle

**FR-020 — Record the chronicle**
- **Priority:** Must
- **Description:** WHEN the owner saves the chronicle, THE SYSTEM SHALL replace all of it with what was sent, so an omitted field is cleared; THE SYSTEM SHALL provide read-only totals (founder's name, number of generations, number of members).
- **Verification method:** unit test, end-to-end test
- **Source:** US-GEN-05.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Save the chronicle
    Given a tree with an empty chronicle
    When I enter surname, current location, origin and history
    Then the chronicle shows them

  Scenario: Omitted field is cleared
    Given a chronicle with an origin text
    When I save it again without the origin
    Then the origin is empty

  Scenario: Text too long
    Given the chronicle editor
    When I enter a history longer than 20000 characters
    Then I am told it is too long
    And nothing is saved

  Scenario: Viewer tries to edit
    Given I am a viewer
    When I try to save the chronicle
    Then I am told I may not do this
  ```

**FR-021 — Family teachings with reading and meaning**
- **Priority:** Should
- **Description:** THE SYSTEM SHALL record the hoành phi and each of the two couplet lines with their own reading and meaning, plus a free-text teaching.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10 ("both").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Couplet with reading and meaning
    Given the chronicle editor
    When I enter the upper couplet line with its reading and meaning
    Then all three are saved with that line

  Scenario: Reading too long
    Given the chronicle editor
    When I enter a plaque reading longer than 200 characters
    Then I am told it is too long
  ```

#### Clan head

**FR-022 — Mark one living male blood member as clan head**
- **Priority:** Should
- **Description:** WHEN the owner marks a member as clan head, THE SYSTEM SHALL accept only a living male blood member, SHALL clear the flag of the previous holder in the same step so there is at most one per tree, and SHALL grant that member no permission.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10 ("a remembered flag; one per tree; male and living").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Mark a clan head
    Given a living male blood member
    When I mark the member as clan head
    Then the tree shows that member as clan head

  Scenario: Replace the clan head
    Given member A is clan head
    When I mark member B as clan head
    Then B is clan head
    And A is no longer clan head

  Scenario Outline: Ineligible member
    Given a member who is <state>
    When I mark the member as clan head
    Then I am told the clan head must be a living male member of the clan

    Examples:
      | state      |
      | female     |
      | deceased   |
      | an in-law  |

  Scenario: Clan head dies
    Given the clan head
    When I mark the member as deceased without clearing the clan-head flag
    Then I am told to clear or reassign the clan head first

  Scenario: Clan head has no extra permission
    Given the account linked to the clan head is a viewer
    When that viewer tries to edit
    Then they are refused like any viewer
  ```

#### Inviting people to view

**FR-023 — Invite a member by phone number**
- **Priority:** Must
- **Description:** WHEN the owner chooses a living member and gives a phone number (or uses the number already on that member), THE SYSTEM SHALL create a pending invitation tied to that member and SHALL write the number onto the member if it has none.
- **Verification method:** unit test, end-to-end test
- **Source:** Owner's scenario: "I choose a person and invite them by phone; status is pending".
- **Acceptance criteria:**
  ```gherkin
  Scenario: Invite a member with no number yet
    Given a living member without a phone number
    When I invite that member using a phone number
    Then a pending invitation is created for that member
    And the member now shows that phone number

  Scenario: Invite using the number already on the member
    Given a living member with a phone number
    When I invite that member without typing a number
    Then a pending invitation is created for that number

  Scenario: Deceased member
    Given a deceased member
    When I try to invite the member
    Then I am told only a living person can be invited

  Scenario: Different number from the one on the member
    Given a member with a phone number
    When I invite them using another number
    Then I am told the number does not match the one recorded

  Scenario: Number belongs to another member
    Given a number already recorded on another member of this tree
    When I invite a different member using that number
    Then I am told the number is already used in this tree

  Scenario: Already invited
    Given a pending invitation exists for a number
    When I invite the same number again
    Then I am told an invitation is already waiting
  ```

**FR-024 — The same outcome whether or not the person has an account**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL answer an invitation request identically whether the phone number belongs to a registered user or not, so the owner cannot learn who is registered.
- **Verification method:** unit test, security review
- **Source:** Privacy decision 2026-10-08.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Registered and unregistered numbers look alike
    Given one invited number has an account and another has none
    When I invite each of them
    Then both invitations are created and shown the same way

  Scenario: Person registers later
    Given an invitation waits for a number with no account
    When a person registers with that number and signs in
    Then they see the invitation in their inbox
  ```

**FR-025 — Invitee sees the invitation**
- **Priority:** Must
- **Description:** WHEN a signed-in user opens their invitation inbox, THE SYSTEM SHALL list the pending, unexpired invitations addressed to their own phone number, showing the tree name, the name of the member they were invited as, and the inviter's phone number, and nothing else of the tree.
- **Verification method:** unit test, end-to-end test
- **Source:** Owner's scenario ("the person receives the invitation in notifications").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Invitation appears
    Given an owner invited my phone number as "Nguyễn Văn B"
    When I open my invitations
    Then I see the tree name, "Nguyễn Văn B" and the inviter's number
    And I do not see the tree's members or chronicle

  Scenario: Someone else's invitation
    Given an invitation addressed to another number
    When I open my invitations
    Then I do not see it

  Scenario: Expired invitation
    Given an invitation older than 30 days
    When I open my invitations
    Then I do not see it
  ```

**FR-026 — Accept an invitation**
- **Priority:** Must
- **Description:** WHEN the invitee accepts a pending, unexpired invitation, THE SYSTEM SHALL make them a viewer of the tree, mark the invitation accepted, and place them at the invited member; the owner SHALL see the status change.
- **Verification method:** unit test, end-to-end test
- **Source:** Owner's scenario ("agree, then the status is updated").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Accept
    Given I have a pending invitation
    When I accept it
    Then the tree appears in my list as viewer
    And I am placed at the member I was invited as
    And the owner sees the invitation as accepted

  Scenario: Accept twice
    Given I already accepted an invitation
    When I accept it again
    Then I am told it is no longer waiting

  Scenario: Accept an invitation that is not mine
    Given an invitation addressed to another number
    When I try to accept it
    Then I am told it was not found

  Scenario: Tree is full of viewers
    Given a tree that already has 20 viewers
    When I accept my invitation
    Then I am told the tree has reached its limit
  ```

**FR-027 — Decline an invitation**
- **Priority:** Must
- **Description:** WHEN the invitee declines, THE SYSTEM SHALL mark the invitation declined and grant nothing; the owner SHALL see it as declined.
- **Verification method:** unit test
- **Source:** Owner's scenario.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Decline
    Given I have a pending invitation
    When I decline it
    Then I do not gain access to the tree
    And the owner sees the invitation as declined
  ```
  (Resolved 2026-10-11: the owner may invite the same number again immediately after a decline.)

**FR-028 — Owner sees invitation statuses and may cancel**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL let the owner list invitations with their status (pending, accepted, declined, expired) and cancel a pending one; a cancelled invitation disappears from the list.
- **Verification method:** unit test
- **Source:** Owner's scenario.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Status list
    Given invitations that are pending, accepted, declined and expired
    When I open the invitations of my tree
    Then I see all four with their status

  Scenario: Cancel
    Given a pending invitation
    When I cancel it
    Then the invitee no longer sees it
    And it no longer appears in my list

  Scenario: Cancel a finished invitation
    Given an accepted invitation
    When I try to cancel it
    Then I am told it was not found
  ```

**FR-029 — Invitations expire after 30 days**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL treat an invitation as expired 30 days after it was created; an expired invitation cannot be accepted and the number may be invited again.
- **Verification method:** unit test
- **Source:** Decision 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Accept after expiry
    Given an invitation created 31 days ago
    When I accept it
    Then I am told it is no longer waiting

  Scenario: Re-invite after expiry
    Given an expired invitation for a number
    When I invite the same number again
    Then a new pending invitation is created
  ```
  (Resolved 2026-10-11: the risk is accepted — an account is tied to its unique phone number, so a person who changed number can no longer sign in with the old one.)

**FR-030 — At most 20 viewers**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL count viewers plus pending invitations against a ceiling of 20 per tree and refuse a new invitation or an acceptance that would exceed it.
- **Verification method:** unit test, concurrency test
- **Source:** Decision 2026-10-10 (number carried over from editors).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Twenty-first
    Given a tree with 19 viewers and 1 pending invitation
    When I invite another person
    Then I am told the limit is reached

  Scenario: Pending invitations count
    Given a tree with 10 viewers and 10 pending invitations
    When I invite another person
    Then I am told the limit is reached
  ```

**FR-031 — Remove a viewer, or leave**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL let the owner remove a viewer, and let a viewer leave a tree, taking effect on the person's very next action; the owner cannot leave.
- **Verification method:** unit test, end-to-end test
- **Source:** Genealogy brief; access control.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Owner removes a viewer
    Given a person is a viewer
    When I remove them
    Then they can no longer open the tree on their next action

  Scenario: Viewer leaves
    Given I am a viewer
    When I leave the tree
    Then the tree is no longer in my list

  Scenario: Owner tries to leave
    Given I own a tree
    When I try to leave it as a viewer would
    Then I am told this is not possible

  Scenario: Phone number cleared on a pending invitation
    Given a pending invitation for a member
    When I clear or change that member's phone number
    Then the invitation is cancelled automatically
  ```

#### Knowing where "I" am

**FR-032 — My position is found by phone number**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL treat as "me", in a given tree, the member whose recorded phone number equals the signed-in user's own phone number, for the owner and for viewers alike, and SHALL show no position when none matches.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Viewer lands on their member
    Given I accepted an invitation as "Nguyễn Văn B"
    When I open the tree
    Then "Nguyễn Văn B" is marked as me

  Scenario: No matching member
    Given no member carries my phone number
    When I open the tree
    Then no member is marked as me
    And relationships "to me" are not shown

  Scenario: Another tree
    Given I am a viewer of two trees and my number is on a member of each
    When I open each tree
    Then I am a different member in each
  ```

**FR-033 — A phone number belongs to one member per tree**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL refuse to record a phone number on a member when another member of the same tree already carries it, including within one creation request.
- **Verification method:** unit test, repository test
- **Source:** Decision 2026-10-10 ("one node, one account, within one tree").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Duplicate number on edit
    Given member A carries a phone number
    When I record the same number on member B of the same tree
    Then I am told the number is already used in this tree

  Scenario: Duplicate within creation
    Given I am creating a tree
    When the founder and a spouse have the same phone number
    Then I am told the number is repeated
    And no tree is created

  Scenario: Same number in another tree
    Given a number is on a member of my first tree
    When I record it on a member of my second tree
    Then it is accepted
  ```

**FR-034 — The viewer cannot change their own position**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL offer no way for a viewer to choose or change which member is "me"; only the owner changes it, by editing the member's phone number.
- **Verification method:** code review, end-to-end test
- **Source:** Decision 2026-10-10 ("they may not change it; the tree is mine").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Owner moves a viewer to another member
    Given a viewer is at member B
    When I clear B's phone number and record it on member C
    Then the viewer is at member C on their next view

  Scenario: Position lost when the member is deleted
    Given a viewer is at a member I delete
    When the viewer opens the tree
    Then they can still view the tree
    And no member is marked as me
  ```
  (Resolved 2026-10-11: accepted — the viewer keeps viewing without a position until the owner records their number again.)

#### Member profile

**FR-035 — Computed profile of a member**
- **Priority:** Must
- **Description:** WHEN the owner or a viewer selects a member, THE SYSTEM SHALL compute and return the person's age (or age at death) both as Western age and as traditional age (Western age plus one), father, mother, spouses, siblings (marking half-siblings), children, line of descent from the founder, branch (chi) path, and relationship to "me".
- **Verification method:** unit test (computed rules), end-to-end test
- **Source:** Decision 2026-10-07; branch and kinship rules 2026-10-07/10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Age of a living member
    Given a living member born in 1980
    When I open the profile in 2026
    Then the age is 46

  Scenario: Age at death
    Given a deceased member born in 1930 and died in 2005
    When I open the profile
    Then the age shown is 75

  Scenario: Unknown age
    Given a member without a birth year
    When I open the profile
    Then no age is shown

  Scenario: Line of descent
    Given a member in generation 3
    When I open the profile
    Then the line lists the founder, the parent and the member in order

  Scenario: Branch nesting
    Given the founder has sons Lộc and Nghĩa, and Lộc has sons A and B
    When I open the profile of a descendant of A
    Then the branch path shows Lộc and then A
    And the current branch is A

  Scenario: Daughters do not start a branch
    Given a father with one son and two daughters
    When I open the daughter's profile
    Then she belongs to her father's branch

  Scenario: Half-siblings
    Given two children of the same father by different spouses
    When I open one child's profile
    Then the other is shown as a half-sibling

  Scenario: Another tree's member
    Given a member of a tree I cannot see
    When I ask for that profile
    Then I am told it was not found
  ```

**FR-036 — Kinship term follows generation, not age**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL name how a selected member relates to "me" by their position in the tree: members one or more generations above are of senior rank regardless of age; between brothers or between a father and his brothers the elder/younger comparison uses birth order, never birth year; distant branches add the word "họ"; maternal-side relations are reported as unknown.
- **Verification method:** unit test (table of cases)
- **Source:** Kinship table approved 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Kinship terms
    Given I am the member in the tree
    When I open the profile of <person>
    Then the relationship shown is "<term>"

    Examples:
      | person                                   | term            |
      | my father                                | Cha             |
      | my father's elder brother                | Bác             |
      | my father's younger brother              | Chú             |
      | my father's sister                       | Cô              |
      | my father's younger brother's wife       | Thím            |
      | my father's sister's husband             | Dượng           |
      | the son of my father's elder brother     | Anh họ          |
      | my own elder brother's wife              | Chị dâu         |
      | my son's wife                            | Con dâu         |
      | my daughter's husband                    | Con rể          |

  Scenario: Younger in age but senior in rank
    Given my uncle is younger in years than I am
    When I open my uncle's profile
    Then he is shown as my uncle, not as a peer

  Scenario: Cousin older by rank only
    Given my father's elder brother's son is younger in years than I am
    When I open his profile
    Then he is shown as my elder cousin

  Scenario: Maternal relative
    Given a person connected to me only through my mother's family
    When I open that person's profile
    Then the relationship is shown as unknown

  Scenario: No position
    Given no member is marked as me
    When I open any profile
    Then no relationship to me is shown
  ```

#### Privacy and permissions

**FR-037 — The tree is invisible to everyone else**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL answer "not found" to any request about a tree from a user who is neither its owner nor a viewer, identical to a tree that does not exist, for every part of the tree.
- **Verification method:** unit test, end-to-end test, security review
- **Source:** Owner's scenario: "other users must not see the tree I created".
- **Acceptance criteria:**
  ```gherkin
  Scenario: Stranger opens a tree
    Given a tree I do not own and am not a viewer of
    When I try to open it
    Then I am told it was not found

  Scenario: Existence is not revealed
    Given a tree that exists and a tree number that does not
    When a stranger asks for each
    Then both answers are identical

  Scenario: Stranger asks for a member, the chronicle or the viewers
    Given a tree I cannot see
    When I ask for any part of it
    Then I am told it was not found
  ```

**FR-038 — Viewer is read-only**
- **Priority:** Must
- **Description:** THE SYSTEM SHALL refuse every change from a viewer — tree information, members, relationships, chronicle, invitations, viewers, deletion — and leave all data unchanged.
- **Verification method:** unit test, end-to-end test
- **Source:** Decision 2026-10-10 ("view only; no other roles").
- **Acceptance criteria:**
  ```gherkin
  Scenario Outline: Viewer attempts a change
    Given I am a viewer of a tree
    When I try to <action>
    Then I am told I may not do this
    And nothing changes

    Examples:
      | action                           |
      | add a member                     |
      | edit a member                    |
      | delete a member                  |
      | change a relationship            |
      | save the chronicle               |
      | invite someone                   |
      | remove another viewer            |
      | mark a clan head                 |

  Scenario: Viewer may leave and answer own invitations
    Given I am a viewer
    When I leave the tree
    Then it succeeds
  ```

### 3.3 Performance requirements

- **NFR-001 — Whole-tree load:** opening a tree of up to 5000 members SHALL return the full member list in one request within 2 seconds at the 95th percentile under normal load.
- **NFR-002 — Computed profile:** a member profile SHALL be returned within 1 second at the 95th percentile for a tree of 5000 members.
- **NFR-003 — Atomic writes:** every multi-step change (create tree, add member, move a branch, accept invitation) SHALL be applied completely or not at all.
- **NFR-004 — Concurrent safety of limits:** the limits of 10 trees per owner, 5000 members per tree and 20 viewers per tree SHALL hold under simultaneous requests.

### 3.4 Logical database requirements

- The server stores: trees, one chronicle per tree, members (with parent, spouse and other-parent links, generation, birth order, clan-head flag, solar dates and the "entered as lunar" flags), viewer grants, and invitations (tied to a member, with status and expiry).
- A viewer's position and the lunar dates are **derived at read time and not stored**.
- Deleting a tree removes everything under it. Deleting a member removes its spouses and any pending invitation tied to it.
- A phone number is unique among the members of one tree.
- Deleting an owner's account is blocked while the tree exists.

### 3.5 Design constraints

- Existing validation of phone numbers and the API response envelope are reused.
- Lunar conversion uses the Vietnamese lunar calendar (UTC+7) and is valid for years 1900–2199; outside that range no lunar date is shown.
- Personal data (phone numbers, residence of living people, notes) must not be written to logs.

### 3.6 Software system attributes

- **Reliability:** all multi-step writes are transactional (NFR-003).
- **Availability:** no stricter target than the rest of the app-facing service.
- **Security:** only the owner and viewers can read a tree; "not found" is uniform; viewers cannot write; the acting user always comes from the sign-in, never the request body; access is rechecked on every request so removal is immediate; invitation responses do not reveal whether a number is registered.
- **Maintainability:** kinship, branch, age and lunar-conversion rules are pure functions with table-driven tests.
- **Portability:** computed values are produced on the server so every client shows the same result.
- **Concurrency note:** if two people edit the same member at the same moment, the last write wins (see Appendix A).

---

## 4. Verification & traceability

### 4.1 Verification approach

Unit tests cover business rules and computed values; repository tests run against a real database for atomicity, uniqueness and cascading effects; end-to-end tests run the real service against a real database for permissions (owner, viewer, stranger), the invitation journey, and uniform "not found". Manual review covers the privacy points in FR-024 and FR-037.

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | Brief: create tree + founder | → `requirements.md` AC1–AC12 | 3 scenarios |
| FR-002 | Decision 2026-10-10 | → AC4, AC9 | 3 scenarios |
| FR-003 | Decision 2026-10-10 | → AC10 | 2 scenarios |
| FR-004 | Brief | → AC13 | 2 scenarios |
| FR-005 | Brief | → AC14, AC41 | 2 scenarios |
| FR-006 | Decision 2026-10-10 | → AC16, AC81 | 2 scenarios |
| FR-007 | Brief | → AC17 | 2 scenarios |
| FR-008 | Brief | → AC18–AC19, AC56–AC57 | 4 scenarios |
| FR-009 | Decision 2026-10-10 | → AC19–AC20 | 2 scenarios |
| FR-010 | Brief | → AC20 | 3 scenarios |
| FR-011 | Decision 2026-10-10 | → AC24–AC25 | 2 scenarios |
| FR-012 | Brief | → AC30–AC33, AC4–AC8 | 4 scenarios |
| FR-013 | Decision 2026-10-10 | → AC34–AC37, AC58 | 4 scenarios |
| FR-014 | Decision 2026-10-10 | → AC27 | 3 scenarios |
| FR-015 | Decision 2026-10-10 | → AC76–AC80 | 5 scenarios |
| FR-016 | Kinship needs | → AC56 | 2 scenarios |
| FR-017 | Brief | → AC28–AC29 | 2 scenarios |
| FR-018 | Decision 2026-10-10 | → AC55 | 4 scenarios |
| FR-019 | Decision 2026-10-10 | → AC40 | 3 scenarios |
| FR-020 | US-GEN-05 | → AC38–AC39, AC41–AC43 | 4 scenarios |
| FR-021 | Decision 2026-10-10 | → AC39 | 2 scenarios |
| FR-022 | Decision 2026-10-10 | → AC82 | 5 scenarios |
| FR-023 | Owner's scenario | → AC44, AC46, AC60 | 6 scenarios |
| FR-024 | Decision 2026-10-08 | → AC45, AC75 | 2 scenarios |
| FR-025 | Owner's scenario | → AC70 | 3 scenarios |
| FR-026 | Owner's scenario | → AC71 | 4 scenarios |
| FR-027 | Owner's scenario | → AC72 | 1 scenario |
| FR-028 | Owner's scenario | → AC73–AC74 | 3 scenarios |
| FR-029 | Decision 2026-10-10 | → AC75 | 2 scenarios |
| FR-030 | Decision 2026-10-10 | → AC47 | 2 scenarios |
| FR-031 | Brief | → AC48–AC50, AC54, AC30 | 4 scenarios |
| FR-032 | Decision 2026-10-10 | → AC59, AC62 | 3 scenarios |
| FR-033 | Decision 2026-10-10 | → AC60 | 3 scenarios |
| FR-034 | Decision 2026-10-10 | → AC59–AC63 | 2 scenarios |
| FR-035 | Decision 2026-10-07 | → AC64–AC68 | 8 scenarios |
| FR-036 | Decision 2026-10-10 | → AC69 | 4 scenarios (+ 10 examples) |
| FR-037 | Owner's scenario | → AC15 | 3 scenarios |
| FR-038 | Decision 2026-10-10 | → AC53, AC81 | 2 scenarios (+ 8 examples) |
| NFR-001 | Assumed from 5000-member cap | → design Step P-2 | TBD (load test) |
| NFR-002 | Assumed | → design Step P-2/P-3 | TBD (load test) |
| NFR-003 | Brief | → design transaction steps | covered by repository tests |
| NFR-004 | Decision 2026-10-10 | → design row locks | covered by concurrency tests |

> NFR-001 and NFR-002 targets are proposed by the BA, not agreed. See Appendix A.

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)


> ⚠️ Not confirmed: are the response-time targets in NFR-001 and NFR-002 acceptable — proposed by the BA; blocks only the performance tests.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Who can see and edit a tree? | Owner edits; invited people only view; no editor and no other role | 2026-10-10 |
| Is there a clan-head role? | No role; a single flag on a living male blood member, one per tree | 2026-10-10 |
| How is a user placed in the tree? | By phone number: the member whose phone equals the user's number; one member per number per tree; the viewer cannot change it | 2026-10-10 |
| How is someone invited? | Owner picks a living member and invites by phone; pending until accepted or declined; status visible to owner | 2026-10-10 |
| Can a deceased person be invited? | No | 2026-10-10 |
| What happens to a pending invitation when the member's number is cleared? | It is cancelled automatically | 2026-10-10 |
| Which calendar are anniversary and commemoration dates stored in? | Solar, with a flag that the user entered lunar; the server converts back to lunar | 2026-10-10 |
| Reading and meaning for family teachings | Both the plaque and each couplet line | 2026-10-10 |
| May a member with children be deleted? | No; delete from the bottom up | 2026-10-10 |
| Birth year of a child earlier than the parent's | Warn, do not block | 2026-10-10 |
| Can a wrongly entered relationship be changed? | Yes, by a dedicated action that moves the whole branch | 2026-10-10 |
| Children of daughters; female founder | Not added; the founder must be male | 2026-10-10 |
| Numeric limits | 10 trees per owner, 5000 members, 60 generations, 20 viewers (pending included), invitation lifetime 30 days | 2026-10-10 |
| How are invitees notified? | In-app invitation inbox only; push notification is a separate later feature; no SMS | 2026-10-10 |
| Transfer of ownership | Not provided | 2026-10-10 |
| Kinship terms and rank | Approved table; rank by generation, never age; "họ" added for distant lines | 2026-10-10 |
| How age is shown (Western vs. traditional) | Both: Western age and traditional age (Western + 1) | 2026-10-11 |
| Editing sibling order | One member at a time through the normal edit; no "re-order siblings" action in v1 | 2026-10-11 |
| May the owner invite again right after a decline | Yes, immediately | 2026-10-11 |
| Reassigned phone number seeing an old invitation | Risk accepted: an account is tied to its unique number; a person who changed number cannot sign in with the old one | 2026-10-11 |
| Viewer whose member was deleted or number cleared | Keeps viewing without a position until the owner records the number again | 2026-10-11 |
| Child with no named other parent | Always sits in the father's branch; the other parent is optional and the client does not distinguish the mother | 2026-10-11 |
| Convert a blood member to an in-law or the reverse | Not supported; delete and re-create | 2026-10-11 |
| Two edits at the same moment | Last write wins (only the owner can write) | 2026-10-11 |

### Appendix B — Cross-reference to the engineering spec

`FR-*` → `docs/specs/features/genealogy/requirements.md` acceptance criteria: see the "Design" column of 4.2. The engineering spec at v5.12 is complete for every `FR-*` above; this SRS introduces no new behavior. Items still open here are the same as the ones marked `⚠️ Not confirmed` in that spec.
