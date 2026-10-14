# IEEE 29148-2018 Structure for a Multi-Platform Product Spec

The product spec follows ISO/IEC/IEEE 29148-2018, adapted for one product shipped on four platforms
(backend, website, iOS, Android). Use this map to fill `assets/templates/product.md.template`. The
SRS is written at **system level**: "THE SYSTEM" is InnerLamp as the user experiences it, not any one
codebase.

## What never belongs in this layer

| Kind | Example | Where it goes instead |
|---|---|---|
| Contract | `POST /v1/memorials`, `death_date`, `INVALID_PHONE`, HTTP status | `docs/specs/api/openapi.yaml` + backend platform spec |
| Storage / persistence technology | Postgres table, Core Data, Room, localStorage, cookie | platform spec `design.md` |
| Code structure | class, ViewModel, use case, handler names | platform spec `design.md` |
| Platform gestures and controls | "tap", "swipe", a button id | Gherkin uses "chooses", "submits", "opens" |
| Per-platform `if` | "on iOS the app also…" inside a requirement | §3.6 Platform variance |

## Section map

### 1. Introduction

| Subsection | What goes in it | Notes |
|---|---|---|
| 1.1 Purpose | What the document is, who reads it, and that it is the single product truth for every platform. | |
| 1.2 Scope | In scope / out of scope, from the user's point of view. Deferred items are out of scope with a "later" note. | |
| 1.3 Product overview | One non-technical paragraph on InnerLamp and where the feature sits. | |
| 1.4 Glossary | Every domain term (giỗ, giỗ đầu, pháp danh, quy y, tháng thiếu…) with a definition and the nearby term it must not be confused with. Pick one canonical English term and use it everywhere. | Platform specs reuse these terms. |
| 1.5 References | Prototype file + screens, legacy SRS/notes, and the folder name the feature has on each platform. | Record which copy of the prototype description was used. |

### 2. Overall description

| Subsection | What goes in it | Notes |
|---|---|---|
| 2.1 Product perspective | New / enhancement / replacement; other features touched; external systems by role. | |
| 2.2 Product functions | One line per major function with its `FR-*` IDs. | A map, not the detail. |
| 2.3 User classes | Actors and permissions (guest, signed-in user, temple admin, back-office). | |
| 2.4 Operating environment | The platform table (in scope / later / no) and the network expectation. | OS minimum versions belong in platform specs. |
| 2.5 Constraints | Business, legal, policy, religious-content constraints. | Not architecture. |
| 2.6 User documentation | Usually **N/A** for a consumer feature. | |
| 2.7 Assumptions and dependencies | Explicit assumptions; other features and external services needed. | Hidden coupling surfaces here. |

### 3. Specific requirements

| Subsection | What goes in it | Notes |
|---|---|---|
| 3.1 External interfaces | 3.1.1 user surfaces at product level; 3.1.2 device capabilities by role (or N/A); 3.1.3 capabilities needed from the backend in plain words; 3.1.4 user-visible channels (SMS, push) or N/A. | Never restate the contract. |
| 3.2 Business rules | `BR-*`, one decision each, with examples — `references/business_rules.md`. | The part platforms most often drift on. |
| 3.3 Functional requirements | `FR-*`: priority, platforms, EARS description, linked BRs, verification method, source, Gherkin scenarios. | Grouped by feature area or actor. |
| 3.4 Non-functional | `NFR-*`, user-perceived and quantified where possible. | Unknown target → `⚠️ Not confirmed`. |
| 3.5 Data requirements | Conceptual data items: meaning, owner (account on server / this device), retention, privacy. | IEEE "logical database requirements", without schema. |
| 3.6 Platform variance | Every intentional difference between platforms, with reason. | Empty table = identical everywhere. |
| 3.7 System attributes | Reliability, availability (backend down), security & privacy, maintainability/portability. | Privacy is usually the important one. |

### 4. Verification & traceability

| Subsection | What goes in it |
|---|---|
| 4.1 Verification approach | How requirements are verified; computational rules via shared vectors on every platform. |
| 4.2 Traceability matrix | ID × platform status — `references/traceability.md`. |
| 4.3 Test vectors | Vector files and their case counts. |

### 5. Appendices

| Subsection | What goes in it |
|---|---|
| A — Open questions | `⚠️ Not confirmed` items (with blocked platforms) and the resolved list with dates. |
| B — Contract handoff | What the backend contract must provide, in plain words; removed once covered. |

## Marking N/A vs leaving a gap

**N/A** means the section cannot apply to this feature by nature (hardware interfaces for a reading
list). If it could apply but the answer is unknown, keep the section with a `⚠️ Not confirmed` note.
