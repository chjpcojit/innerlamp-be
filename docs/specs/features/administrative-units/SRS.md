# Software Requirements Specification — Administrative Units

| Field | Value |
|---|---|
| Document type | Software Requirements Specification (SRS) — IEEE 29148-2018 |
| Feature | Administrative Units (Vietnam provinces, districts and wards — old and new schemes, with conversion) |
| Product | InnerLamp |
| Version | 0.3 (draft) |
| Last updated | 2026-10-10 |
| Status | draft — core decisions confirmed 2026-10-10; two items still open (see Appendix A) |
| Owner / author | duc@wee.vn (BA assist: Claude) |
| Downstream handoff | `il-spec-writer` → `requirements.md` / `design.md` / `tasks.md` under `docs/specs/features/administrative-units/` |

---

## 1. Introduction

### 1.1 Purpose

This document specifies, in business terms, the reference data and lookups InnerLamp needs so that a person can pick a Vietnamese address down to ward level, under either the administrative scheme in force before 1 July 2025 or the one after, and so that the system can translate between the two. It is written for product, engineering and QA, and it drives the engineering spec under `docs/specs/features/administrative-units/`.

### 1.2 Scope

**In scope:**
- Reference data for both schemes, loaded into the system ahead of time: the new scheme (provinces and wards) and the old scheme (provinces, districts and wards).
- Public lookups that let a client present cascading pick-lists: province → wards (new scheme) and province → districts → wards (old scheme).
- Conversion between schemes at ward level, and at province level as derived from wards: old → new and new → old.
- Loading, versioning and integrity-checking of the reference dataset.

**Out of scope:**
- Conversion at district level (deferred by the owner, 2026-10-10).
- Wiring the lookups into existing records (temples, user profile, genealogy, memorial); confirmed out of this feature 2026-10-10.
- Anything below ward level: hamlet (ấp, thôn, bản), neighbourhood group (tổ dân phố, khu phố), house number and street are free text typed by the person and belong to the consuming feature (see 2.7).
- Storing a person's address (user profile, temple, genealogy, memorial). Each consuming feature owns that.
- Migrating existing data that uses free-text province names (for example the temple catalogue) to coded units.
- Geographic data (coordinates, boundaries, maps), postal codes, and address geocoding.
- Back-office screens for editing units by hand; units change only by loading a new dataset version.

### 1.3 Product overview

InnerLamp is a Buddhist-practice companion app. Several of its features need a place: where a temple stands, where a person lives or was ordained, where a family line originates. Today those places are free-text province names, and on 1 July 2025 Vietnam replaced its three-level system (63 provinces, districts, wards/communes) with a two-level one (34 provinces, wards/communes/special zones; no district level). People and existing records still use both vocabularies. This feature provides one authoritative, queryable source for both, plus the translation between them.

### 1.4 Definitions, acronyms, abbreviations

| Term | Definition |
|---|---|
| Scheme | One complete administrative system. **Old scheme**: before 1 July 2025 — province, district, ward. **New scheme**: from 1 July 2025 — province, ward. |
| Province | A first-level unit: province (tỉnh) or centrally-run city (thành phố trực thuộc trung ương). |
| District | A second-level unit of the old scheme only (quận, huyện, thị xã, thành phố thuộc tỉnh). Does not exist in the new scheme. |
| Ward | The lowest unit the official catalogue covers: commune (xã), ward (phường), town (thị trấn, old scheme only) or special zone (đặc khu, new scheme only). |
| Special zone | A new-scheme unit at ward level created for certain islands (13 in the dataset). |
| Mapping | A recorded link between one old-scheme ward and one new-scheme ward. |
| Relationship type | The kind of a mapping: whole old unit merged into the new one, only part of it moved there, renamed, unchanged, moved to a different province, or population split across several. |
| Candidate | One possible conversion target; a conversion may return several. |
| Dataset version | An identifier for one loaded snapshot of both schemes plus mappings, with its source date. |
| Free-text address line | What the person types below ward level: house number, street, hamlet, neighbourhood group. |
| Registration | The existing sign-up flow; a person in it has no session yet. |

### 1.5 References

- `docs/specs/features/temple/requirements.md` — temple catalogue; holds province as free text (US-TMP-07).
- `docs/specs/features/onboarding/requirements.md` — registration flow.
- `migrations/0024_temple_province.sql`, `migrations/0025_temple_province_34_units.sql` — current province handling for temples.
- Quyết định 19/2025/QĐ-TTg (administrative unit codes from 1 July 2025) and Nghị quyết 202/2025/QH15 (province merger), as cited by the candidate datasets; to be re-verified, see Appendix A.
- Candidate dataset: `vietnam-address-kit` (MIT-licensed code; data generated from the official conversion table, dated 2026-09-25). Cross-checked against `vietnamadminunits` (see 2.7).
- IEEE 29148-2018, *Systems and software engineering — Life cycle processes — Requirements engineering*.

### 1.6 Document organization

Section 2 describes the feature in context. Section 3 holds the requirements (functional with Gherkin acceptance criteria, then non-functional). Section 4 maps requirements to verification. Appendix A lists decisions still open.

---

## 2. Overall description

### 2.1 Product perspective

This is a new, shared reference-data capability inside the existing backend. It is consumed by client apps over the backend's HTTP API and, later, by other backend features that will store coded addresses. It replaces nothing today; the temple catalogue's free-text province column stays as is until a consuming feature is specified (Appendix A).

The reference data is static between dataset loads. No external system is called at request time.

### 2.2 Product functions

- Present the new-scheme provinces and, for a chosen province, its wards (`FR-001`, `FR-002`).
- Present the old-scheme provinces, the districts of a chosen province and the wards of a chosen district (`FR-003`, `FR-004`, `FR-005`).
- Offer all of the above without sign-in, including during registration (`FR-006`).
- Convert an old ward to its new ward(s) and a new ward to its old ward(s) (`FR-007`, `FR-008`).
- Convert at province level when only a province is known (`FR-009`).
- Return every possible target when a conversion is not unique, labelled by relationship type (`FR-010`).
- Search units by name ignoring diacritics (`FR-011`).
- Load a dataset version idempotently and verify its integrity (`FR-012`, `FR-013`).
- Give every unit a short name and a full name with its type prefix (`FR-014`).

### 2.3 User classes and characteristics

| Actor | Description | Characteristics / permissions |
|---|---|---|
| Visitor in registration | A person filling the sign-up flow, no session. | May use every lookup and conversion (confirmed: public). |
| Signed-in app user | A person using the app. | Same lookups; no extra rights. |
| Client application | The mobile/web client calling on behalf of either person. | Read-only. |
| Operator | An engineer or staff member who loads a dataset version. | The only actor that can change the data. |

### 2.4 Operating environment

- Platform: the existing appService backend, containerized (Cloud Run / GKE / Fargate), PostgreSQL.
- Clients: mobile app and any web client already calling the backend.
- Network: online; lookups are small, read-only and cacheable by clients.

### 2.5 Design and implementation constraints

- Follows the repository's Clean Architecture layout (domain → usecase → repository / delivery, wired from the `cmd` composition root) and its migration conventions.
- Reference data is loaded ahead of time, not fetched from a third party at request time.
- The same numeric code can denote different units in the old and new schemes (for example one code is both an old ward in District 1 and a different new ward). Any identifier must therefore be interpreted together with its scheme and the two schemes must never share one lookup namespace.
- The candidate dataset contains two labelling defects: a ward carries a province that disagrees with the province of its own district (ward "Phường 4" of Thành phố Tân An is labelled Tiền Giang, while the district belongs to Long An; one ward of Thành phố Hà Tiên is labelled An Giang, while the district belongs to Kiên Giang). This also produces two spurious duplicate district rows. A loader must detect such rows (`FR-013`); it must not treat them as real splits.

### 2.6 User documentation

**N/A** — this is an infrastructure/reference-data feature with no end-user manual. The API contract is documented under `docs/specs/api/`.

### 2.7 Assumptions and dependencies

- **Assumptions:**
  - The dataset facts below hold for dataset 2026-09-25 and may change when new resolutions are issued:
    - New scheme: 34 provinces (25 provinces, 9 centrally-run cities), 3,321 wards (2,598 communes, 710 wards, 13 special zones). Largest province: Hồ Chí Minh, 168 wards; smallest has 38.
    - Old scheme: 63 provinces, 696 districts (after removing two duplicate rows, see 2.5), 10,033 wards (7,686 communes, 1,731 wards, 616 towns). Largest district: 47 wards.
    - 441 old wards (about 4.4%) correspond to more than one new ward; every old ward has at least one mapping and every new ward is the target of at least one.
    - Province level is many-to-one: each old province corresponds to exactly one new province (provinces were merged whole under Nghị quyết 202/2025/QH15). The raw dataset appears to contradict this for Tiền Giang (1 of 165 wards pointing at Tây Ninh), but that ward is a labelling defect (see 2.5), not a split. A new province corresponds to one or more old provinces (Hồ Chí Minh ← 3).
    - Five old districts have no wards (Bạch Long Vĩ, Cồn Cỏ, Hoàng Sa, Lý Sơn, Côn Đảo); district-level conversion is deferred (confirmed 2026-10-10: "decide later").
  - Below ward level, the official catalogue has no data; the person types it (confirmed in conversation 2026-10-10).
  - Selecting a province presents its wards as a list the person can choose from (confirmed 2026-10-10).
- **Dependencies:**
  - Registration flow and in-app screens call these lookups (confirmed: public for both).
  - Consuming features (temple catalogue, profile, genealogy, memorial) are separate specs.
  - The candidate dataset and its licence (Appendix A).

---

## 3. Specific requirements

### 3.1 External interface requirements

#### 3.1.1 User interfaces

**N/A** — this feature has no screens of its own. Client screens built by consuming features are expected to show, for the new scheme, a province pick-list followed by a ward pick-list, and for the old scheme a province, district and ward pick-list, followed by a free-text address line. When a conversion returns more than one candidate, the client lets the person choose.

#### 3.1.2 Hardware interfaces

**N/A** — a backend reference-data feature with no hardware interface.

#### 3.1.3 Software interfaces

A read-only HTTP API at contract level (paths and shapes are fixed later in the engineering spec and `docs/specs/api/openapi.yaml`):

- List provinces, per scheme.
- List the wards of a new-scheme province.
- List the districts of an old-scheme province, and the wards of an old-scheme district.
- Convert a ward (or a province) from one scheme to the other, returning candidates with their relationship type.
- Search units by name within a scheme (optional, `FR-011`).

Access rule (confirmed 2026-10-10): the lookups use the same web API key check as the registration and temple-suggestion routes. A web client sends the shared key header; a request without the header (mobile, which cannot keep a secret) is also served; a request with a wrong key is refused. The key identifies the calling application, not a person, and does not stop a caller who omits the header.

#### 3.1.4 Communication interfaces

HTTPS, JSON, UTF-8 Vietnamese text. Responses follow the project's existing response envelope convention (`docs/specs/features/api-response-envelope`).

### 3.2 Functional requirements

#### Selecting a unit

**FR-001 — List new-scheme provinces**
- **Priority:** Must
- **Description:** WHEN a client asks for the provinces of the new scheme THE SYSTEM SHALL return all 34, each with its name and type.
- **Verification method:** integration test against the seeded dataset.
- **Source:** conversation 2026-10-10 ("user chooses a new address").
- **Acceptance criteria:**
  ```gherkin
  Scenario: List the new provinces
    Given the dataset has been loaded
    When a client asks for the provinces of the new scheme
    Then it receives 34 provinces
    And "Hồ Chí Minh" and "Đồng Tháp" are among them
    And "Bình Dương" and "Tiền Giang" are not among them

  Scenario: Provinces are returned in a stable order
    Given the dataset has been loaded
    When a client asks twice for the provinces of the new scheme
    Then both responses list the provinces in the same order
  ```
  > Confirmed 2026-10-10: lists are returned in Vietnamese alphabetical order of the short name (digits compared as numbers, so "Phường 2" precedes "Phường 10").

**FR-002 — List wards of a new-scheme province**
- **Priority:** Must
- **Description:** WHEN a client chooses a new-scheme province THE SYSTEM SHALL return all wards of that province, each selectable, including communes, wards and special zones.
- **Verification method:** integration test; counts reconciled with the dataset.
- **Source:** conversation 2026-10-10 ("choose a province, many wards come out, can be selected").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Choose a province and list its wards
    Given the dataset has been loaded
    When a client asks for the wards of "Hồ Chí Minh" in the new scheme
    Then it receives 168 wards
    And "Phường Tân Định" is among them
    And "Phường Thủ Dầu Một" and "Phường Vũng Tàu" are among them

  Scenario: Wards of one province do not include another province's wards
    Given the dataset has been loaded
    When a client asks for the wards of "Hà Nội" in the new scheme
    Then none of the returned wards belongs to another province

  Scenario: Unknown province
    Given the dataset has been loaded
    When a client asks for the wards of a province that does not exist
    Then it receives a clear "not found" outcome
    And no wards are returned
  ```

**FR-003 — List old-scheme provinces**
- **Priority:** Must
- **Description:** WHEN a client asks for the provinces of the old scheme THE SYSTEM SHALL return all 63.
- **Verification method:** integration test.
- **Source:** conversation 2026-10-10 ("choose an old address").
- **Acceptance criteria:**
  ```gherkin
  Scenario: List the old provinces
    Given the dataset has been loaded
    When a client asks for the provinces of the old scheme
    Then it receives 63 provinces
    And "Bình Dương", "Bà Rịa - Vũng Tàu" and "Tiền Giang" are among them
  ```

**FR-004 — List districts of an old-scheme province**
- **Priority:** Must
- **Description:** WHEN a client chooses an old-scheme province THE SYSTEM SHALL return its districts.
- **Verification method:** integration test.
- **Source:** conversation 2026-10-10.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Choose an old province and list its districts
    Given the dataset has been loaded
    When a client asks for the districts of the old province "Thành phố Hồ Chí Minh"
    Then "Quận 1" is among them
    And no district of "Bình Dương" is among them

  Scenario: Unknown old province
    Given the dataset has been loaded
    When a client asks for the districts of a province that does not exist
    Then it receives a clear "not found" outcome
  ```

**FR-005 — List wards of an old-scheme district**
- **Priority:** Must
- **Description:** WHEN a client chooses an old-scheme district THE SYSTEM SHALL return its wards; WHEN the district has none THE SYSTEM SHALL return an empty list rather than an error.
- **Verification method:** integration test.
- **Source:** conversation 2026-10-10; dataset analysis (five ward-less island districts).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Choose an old district and list its wards
    Given the dataset has been loaded
    When a client asks for the wards of the old district "Quận 1" of "Thành phố Hồ Chí Minh"
    Then "Phường Tân Định" and "Phường Đa Kao" are among them

  Scenario: District with no wards
    Given the dataset has been loaded
    When a client asks for the wards of the old district "Huyện Bạch Long Vĩ"
    Then it receives an empty list
    And it is not treated as an error

  Scenario: A district lists only wards of its own province
    Given the dataset has been loaded
    When a client asks for the wards of the old district "Thành phố Tân An" of "Long An"
    Then every returned ward belongs to that district
    And "Phường 4" is among them
  ```

**FR-006 — Public access**
- **Priority:** Must
- **Description:** WHEN a client calls any lookup or conversion THE SYSTEM SHALL serve it without a signed-in session, including during registration.
- **Verification method:** end-to-end test calling without a session token.
- **Source:** conversation 2026-10-10 ("API is public, for both registration and in-app use").
- **Acceptance criteria:**
  ```gherkin
  Scenario: A person in registration loads the province list
    Given a visitor in the middle of registration with no session
    When the client asks for the provinces of the new scheme
    Then the provinces are returned

  Scenario: A signed-in user uses the same lookup
    Given a signed-in user
    When the client asks for the wards of a province
    Then the wards are returned, identical to what a visitor receives

  Scenario: Web client with the correct key
    Given a web client that sends the correct shared key
    When it asks for the provinces of the new scheme
    Then the provinces are returned

  Scenario: Client without a key
    Given a mobile client that sends no key
    When it asks for the provinces of the new scheme
    Then the provinces are returned

  Scenario: Wrong key
    Given a client that sends a wrong shared key
    When it asks for the provinces of the new scheme
    Then the request is refused as having an invalid key
    And no province is returned
  ```
  > Confirmed 2026-10-10: public lookups are rate-limited per client IP, as the temple-suggestion route is; the limit and window are configuration.

#### Converting between schemes

**FR-007 — Convert an old ward to the new scheme**
- **Priority:** Must
- **Description:** WHEN a client supplies an old-scheme ward THE SYSTEM SHALL return the new-scheme ward(s) it maps to, each with its new province and relationship type.
- **Verification method:** integration test; sampled against the official conversion table.
- **Source:** conversation 2026-10-10 ("from an old address, derive the new one").
- **Acceptance criteria:**
  ```gherkin
  Scenario: Old ward merged wholly into one new ward
    Given the dataset has been loaded
    When a client converts the old ward "Phường Cầu Kho" of "Quận 1", "Thành phố Hồ Chí Minh"
    Then exactly one candidate is returned
    And it is "Phường Cầu Ông Lãnh" in "Hồ Chí Minh"

  Scenario: Old ward in a province that has been merged into another
    Given the dataset has been loaded
    When a client converts the old ward "Phường Hiệp Thành" of "Thành phố Thủ Dầu Một", "Bình Dương"
    Then the candidate's province is "Hồ Chí Minh"

  Scenario: Unknown ward
    Given the dataset has been loaded
    When a client converts a ward that does not exist
    Then it receives a clear "not found" outcome
  ```

**FR-008 — Convert a new ward to the old scheme**
- **Priority:** Must
- **Description:** WHEN a client supplies a new-scheme ward THE SYSTEM SHALL return every old-scheme ward that maps to it, each with its district and province, so the person can choose.
- **Verification method:** integration test.
- **Source:** conversation 2026-10-10 ("from a new address, derive the old one").
- **Acceptance criteria:**
  ```gherkin
  Scenario: New ward formed from several old wards
    Given the dataset has been loaded
    When a client converts the new ward "Phường Cầu Ông Lãnh" of "Hồ Chí Minh"
    Then the candidates include "Phường Cô Giang", "Phường Nguyễn Cư Trinh" and "Phường Cầu Kho"
    And each candidate shows its old district and old province

  Scenario: New ward formed from one old ward
    Given the dataset has been loaded
    When a client converts a new ward that corresponds to exactly one old ward
    Then exactly one candidate is returned
  ```

**FR-009 — Convert at province level**
- **Priority:** Should
- **Description:** WHEN a client supplies only a province THE SYSTEM SHALL return the province(s) it corresponds to in the other scheme, derived from the ward mappings: exactly one for an old province, one or more for a new province.
- **Verification method:** integration test.
- **Source:** conversation 2026-10-10; dataset analysis.
- **Acceptance criteria:**
  ```gherkin
  Scenario: New province corresponds to several old provinces
    Given the dataset has been loaded
    When a client converts the new province "Hồ Chí Minh" to the old scheme
    Then the candidates are "Thành phố Hồ Chí Minh", "Bình Dương" and "Bà Rịa - Vũng Tàu"

  Scenario: Old province corresponds to one new province
    Given the dataset has been loaded
    When a client converts the old province "Bình Dương" to the new scheme
    Then exactly one candidate is returned, "Hồ Chí Minh"

  Scenario: Old province whose raw rows mention two new provinces because of a labelling defect
    Given the dataset has been loaded
    When a client converts the old province "Tiền Giang" to the new scheme
    Then exactly one candidate is returned, "Đồng Tháp"
  ```

**FR-010 — Return all candidates, labelled**
- **Priority:** Must
- **Description:** WHEN a conversion has more than one possible target THE SYSTEM SHALL return all of them, each labelled with its relationship type, and SHALL NOT silently choose one.
- **Verification method:** integration test; unit test on the candidate builder.
- **Source:** conversation 2026-10-10 ("many wards can come out, can be chosen"); dataset analysis (441 one-to-many old wards).
- **Acceptance criteria:**
  ```gherkin
  Scenario: Old ward only partly moved into each of two new wards
    Given the dataset has been loaded
    When a client converts the old ward "Phường Đa Kao" of "Quận 1", "Thành phố Hồ Chí Minh"
    Then the candidates are "Phường Tân Định" and "Phường Sài Gòn"
    And each is labelled as a partial move

  Scenario: A unique conversion is not padded with extra candidates
    Given the dataset has been loaded
    When a client converts the old ward "Phường Cầu Kho" of "Quận 1", "Thành phố Hồ Chí Minh"
    Then exactly one candidate is returned
  ```
  > Confirmed 2026-10-10: the client presents all candidates and lets the person choose; the system never chooses for them.

**FR-011 — Search by name**
- **Priority:** Could
- **Description:** WHEN a client submits a name fragment and a scheme THE SYSTEM SHALL return matching units regardless of diacritics and letter case.
- **Verification method:** integration test.
- **Source:** analyst-proposed; not yet requested.
- **Acceptance criteria:**
  ```gherkin
  @not-confirmed
  Scenario: Search without diacritics
    Given the dataset has been loaded
    When a client searches the new scheme for "ha noi"
    Then "Hà Nội" is among the results
  ```
  > ⚠️ Not confirmed: whether name search is needed at all, and whether it is scoped to a province — it decides whether this requirement exists and how large the response can be.

#### Managing the dataset

**FR-012 — Load a dataset version**
- **Priority:** Must
- **Description:** WHEN an operator loads a dataset THE SYSTEM SHALL store both schemes and all mappings together with a dataset version and source date, and loading the same dataset again SHALL change nothing.
- **Verification method:** integration test (load twice, compare); manual run by an operator.
- **Source:** conversation 2026-10-10 ("data loaded into the DB in advance"); dataset volatility (2026 amendments).
- **Acceptance criteria:**
  ```gherkin
  Scenario: First load
    Given an empty system
    When an operator loads dataset version 2026-09-25
    Then the new scheme has 34 provinces and 3,321 wards
    And the old scheme has 63 provinces, 696 districts and 10,033 wards
    And the dataset version is recorded

  Scenario: Loading the same dataset again
    Given dataset version 2026-09-25 is already loaded
    When an operator loads it again
    Then no unit or mapping is added, changed or removed
  ```
  > Confirmed 2026-10-10: an operator runs the load by hand when the dataset changes; loading only adds or updates units, so units absent from a newer version are left untouched.

**FR-013 — Verify dataset integrity at load**
- **Priority:** Must
- **Description:** WHEN a dataset is loaded THE SYSTEM SHALL verify that every old ward has at least one mapping, every new ward is the target of at least one mapping and every mapping points to a known unit, and SHALL refuse the load with a report if any check fails.
- **Verification method:** unit test with corrupted fixtures; integration test.
- **Source:** dataset analysis; risk of hand-transcribed 2026 amendments.
- **Acceptance criteria:**
  ```gherkin
  Scenario: Complete dataset passes
    Given a dataset in which every unit is linked
    When an operator loads it
    Then the load succeeds

  Scenario: Old ward without a mapping
    Given a dataset in which one old ward has no mapping
    When an operator loads it
    Then the load is refused
    And the report names the ward

  Scenario: Mapping to an unknown new ward
    Given a dataset in which one mapping points to a new ward that does not exist
    When an operator loads it
    Then the load is refused
    And the report names the mapping

  Scenario: Ward whose province disagrees with its district's province
    Given a dataset in which one old ward is labelled with a different province than its district
    When an operator loads it
    Then the load is refused
    And the report names the ward and both provinces
  ```
  > Confirmed 2026-10-10: the two labelling defects are corrected in the data file itself and recorded in its metadata; the loader refuses data that still has them.
  > Confirmed 2026-10-10: reconciliation against the official conversion table (including the "Huyện Kim Sơn → Xã Kim Đông" row) is deferred by the owner; the row is kept out of the loaded mappings and listed as unresolved in the data file.

**FR-014 — Short and full names**
- **Priority:** Should
- **Description:** WHEN the system returns a unit THE SYSTEM SHALL provide both its short name ("Tân Định") and its full name with type prefix ("Phường Tân Định").
- **Verification method:** integration test.
- **Source:** dataset structure; needed to tell a ward from a commune of the same name.
- **Acceptance criteria:**
  ```gherkin
  Scenario: A ward shows both names
    Given the dataset has been loaded
    When a client asks for the wards of "Hồ Chí Minh" in the new scheme
    Then the ward "Tân Định" also carries the full name "Phường Tân Định"
  ```

### 3.3 Performance requirements

- **NFR-001 — Lookup latency:** a province, district or ward list is returned within a quantified time. Largest list is 168 wards (new) or 47 wards (old district).
  > Confirmed 2026-10-10: target p95 ≤ 200 ms.
- **NFR-002 — Cacheability:** reference data changes only on a dataset load, so responses should be cacheable by clients and intermediaries.
  > Confirmed 2026-10-10: clients may cache lookup responses for up to 24 hours.
- **NFR-003 — No pagination needed:** every list the pick-lists need fits in one response (largest 168 items), so no paging is required.

### 3.4 Logical database requirements

- The server persists both schemes' provinces and wards, the old scheme's districts, and the ward-to-ward mappings with relationship type and note.
- A mapping is many-to-many between old wards and new wards. A province-level correspondence is derived from ward mappings, not stored separately.
- Old and new units are held separately so identical numeric codes never collide (2.5).
- The dataset version and source date are stored with the data.
- Retention: nothing is deleted on a new dataset load without an explicit decision (Appendix A).

### 3.5 Design constraints

- Existing backend conventions: Clean Architecture layers, migrations under `migrations/`, an idempotent seed command in the style of `cmd/seedtemples`.
- The data is attributed to its source; the licence of the official data is reviewed before production (Appendix A).
- Names and statuses are Vietnamese; the response envelope follows `docs/specs/features/api-response-envelope`.

### 3.6 Software system attributes

- **Reliability:** lookups depend only on the local database; an outage of any third-party dataset host does not affect them.
- **Availability:** same as the backend; registration depends on this feature once a client screen uses it, so it should not be a single point of failure for sign-up.
- **Security:** no personal data is handled or stored. All endpoints are read-only and public behind the web API key check; because a caller may simply omit the key header, they remain exposed to scraping and request floods (see the rate-limit question in Appendix A).
- **Maintainability:** the dataset can be replaced by loading a new version without a code change.
- **Portability:** the seed format is a plain file, independent of the database vendor.

---

## 4. Verification & traceability

### 4.1 Verification approach

- Unit tests for candidate building and integrity checks.
- Integration tests against a seeded PostgreSQL (counts and the worked examples in the scenarios above).
- End-to-end test calling the public endpoints with no session.
- A one-off reconciliation of the loaded data against the official conversion table (FR-013 question).

### 4.2 Traceability matrix

| Requirement ID | Source | Design | Test |
|---|---|---|---|
| FR-001 | Conversation 2026-10-10 | → il-spec-writer | 2 scenarios |
| FR-002 | Conversation 2026-10-10 | → il-spec-writer | 3 scenarios |
| FR-003 | Conversation 2026-10-10 | → il-spec-writer | 1 scenario |
| FR-004 | Conversation 2026-10-10 | → il-spec-writer | 2 scenarios |
| FR-005 | Conversation + dataset analysis | → il-spec-writer | 3 scenarios |
| FR-006 | Conversation 2026-10-10 (public, web API key) | → il-spec-writer | 5 scenarios |
| FR-007 | Conversation 2026-10-10 | → il-spec-writer | 3 scenarios |
| FR-008 | Conversation 2026-10-10 | → il-spec-writer | 2 scenarios |
| FR-009 | Conversation + dataset analysis | → il-spec-writer | 3 scenarios |
| FR-010 | Conversation + dataset analysis | → il-spec-writer | 2 scenarios |
| FR-011 | Analyst-proposed | → il-spec-writer | 1 scenario (not confirmed) |
| FR-012 | Conversation + dataset volatility | → il-spec-writer | 2 scenarios |
| FR-013 | Dataset analysis | → il-spec-writer | 4 scenarios |
| FR-014 | Dataset structure | → il-spec-writer | 1 scenario |
| NFR-001 | Analyst-proposed | → il-spec-writer | not confirmed |
| NFR-002 | Analyst-proposed | → il-spec-writer | not confirmed |
| NFR-003 | Dataset analysis | → il-spec-writer | 1 integration check |

---

## 5. Appendices

### Appendix A — Open questions

#### Open (unconfirmed)

> ⚠️ Not confirmed: which consuming screens or fields use the lookups first (registration profile, temples, genealogy, memorial) and whether the free-text address line is required with a maximum length — it decides which requirements move to those features.

> ⚠️ Not confirmed: whether name search without diacritics is needed and its scope — it decides whether `FR-011` exists.

#### Open questions (resolved)

| Question | Resolution | Date |
|---|---|---|
| Are the lookups public or authenticated? | Public, for both registration and in-app use. | 2026-10-10 |
| When a province is chosen, are wards presented for selection? | Yes — a province yields a list of wards the person can choose from. | 2026-10-10 |
| What language is the SRS written in? | English. | 2026-10-10 |
| Is the feature only the metadata + lookup + conversion API, or does it also wire into existing data? | Only the API and the dataset; wiring into existing records is separate work. | 2026-10-10 |
| In what order are lists returned? | Vietnamese alphabetical order of the short name. | 2026-10-10 |
| What does a client do with several candidates? | Show them all and let the person choose. | 2026-10-10 |
| Rate limiting? | Per client IP, like the temple-suggestion route; limit is configuration. | 2026-10-10 |
| Who loads a new dataset version? | An operator, by hand, when the dataset changes. | 2026-10-10 |
| Latency and caching? | p95 ≤ 200 ms; clients may cache for 24 hours. | 2026-10-10 |
| How are the two data-label defects handled? | Corrected in the data file with a recorded note ("Phường 4" of Tân An → Long An; "Xã Tiên Hải" of Hà Tiên → Kiên Giang). | 2026-10-10 |
| Reconcile with the official conversion table before production? | Deferred by the owner ("can be fixed later"); recorded in the data file's metadata. | 2026-10-10 |
| District-level conversion? | Deferred ("decide later"). | 2026-10-10 |
| Is an API key required? | Yes, the existing web API key check: key header correct → served, header absent (mobile) → served, header wrong → refused. | 2026-10-10 |
| Is an old province converted to more than one new province (Tiền Giang)? | No. The dataset shows 1 of 165 Tiền Giang wards pointing at Tây Ninh, but that ward belongs to Thành phố Tân An (Long An): a labelling defect, not a split. Old province → new province is always exactly one; the temple migration `0025` comment ("none was split") stands. | 2026-10-10 |
| Province-only conversion: list or ward required first? | Old → new is exactly one; new → old is a list. No uncertainty flag is needed. | 2026-10-10 |
| Is the level below ward (hamlet, street, house number) covered by metadata? | No — free text typed by the person. | 2026-10-10 |
| Which data source is used? | `vietnam-address-kit`, to be reconciled against the official table (reconciliation itself remains open). | 2026-10-10 |
