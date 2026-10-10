# IEEE 29148-2018 SRS Structure

The SRS follows ISO/IEC/IEEE 29148-2018. Use this as the authoritative section-by-section map for
filling `assets/templates/SRS.md.template`. InnerLamp is an iOS app, so several classic sections are
**N/A** — mark them N/A with a one-line reason instead of deleting them or inventing filler; a
recognizably IEEE-complete SRS is easier for stakeholders and downstream tools to trust.

## Section map

### 1. Introduction

| Subsection | What goes in it | iOS note |
|---|---|---|
| 1.1 Purpose | One or two sentences: what this document is, who it is for (stakeholders, PM, engineering), and what it is meant to drive. | State the downstream handoff (`il-spec-writer`). |
| 1.2 Scope | What the feature does and, equally important, what it explicitly does **not** do. Name the product (InnerLamp). | Keep a crisp "Out of scope" list. |
| 1.3 Product overview | A short, non-technical description of the product and where this feature sits in it. | One paragraph. |
| 1.4 Definitions, acronyms, abbreviations | A glossary table. Every term the input uses ambiguously or in a domain-specific sense goes here. | Include project terms (dharma name, OTP, ordination place) if the feature uses them. |
| 1.5 References | Any BRD/PRD/spec/confluence links, API docs, or prior SRS this document depends on. | Link `Docs/Specs/<Feature>/…` if relevant. |
| 1.6 Overview (document organization) | A sentence or two on how the rest of the document is organized. | Boilerplate is fine. |

### 2. Overall description

| Subsection | What goes in it | iOS note |
|---|---|---|
| 2.1 Product perspective | How the feature relates to the rest of InnerLamp (is it new, an enhancement, a replacement?), and what external systems it touches. | Mention backend/API if any. |
| 2.2 Product functions | A summary list of the major functions, one line each — a map, not the full detail (that is §3.2). | Reference the `FR-*` IDs. |
| 2.3 User classes and characteristics | Actors: who uses this, their characteristics, and their permissions/experience level. | e.g. visitor, signed-in user, monastic user, admin. |
| 2.4 Operating environment | Hardware/OS/environment constraints. | iOS minimum version, iPhone/iPad, offline expectations. |
| 2.5 Design and implementation constraints | Any constraint imposed by policy, platform, or existing architecture (Swift, Clean Architecture, App Store policy, backend contracts). | Quote AGENTS.md constraints where they bind (e.g. no hardcoded text/color). |
| 2.6 User documentation | What help/manual content is required. | Usually **N/A** for a consumer app feature. |
| 2.7 Assumptions and dependencies | Explicit assumptions the spec relies on, and dependencies on other features/systems/teams. | This is where hidden coupling surfaces — list it honestly. |

### 3. Specific requirements

This is the heart of the SRS. The subsections below are the IEEE set; an iOS app will use a subset.

| Subsection | What goes in it | iOS note |
|---|---|---|
| 3.1 External interface requirements | 3.1.1 User interfaces, 3.1.2 Hardware interfaces, 3.1.3 Software interfaces, 3.1.4 Communication interfaces. | **Hardware interfaces → N/A.** Software/communication interfaces = the backend REST API (endpoints, request/response shapes, error codes) — keep it at contract level, or mark "contract not yet confirmed". |
| 3.2 Functional requirements | The full set of `FR-*` requirements: **ID, priority (Must/Should/Could), description, and verification method** for each. Use EARS-style phrasing (WHEN … THE SYSTEM SHALL …). Each FR also carries **Gherkin acceptance criteria** (see `references/gherkin_acceptance.md`). | Group by feature area/actor. |
| 3.3 Performance requirements | `NFR-*`: response time, launch time, list size, animation smoothness. | Quantify where possible (e.g. "OTP submit ≤ X ms"), mark "not confirmed" otherwise. |
| 3.4 Logical database requirements | Data the system must persist and its relationships. | For iOS, describe local persistence (Core Data / UserDefaults / Keychain) and what data lives server-side. |
| 3.5 Design constraints | Standards compliance, platform constraints, mandated libraries/patterns. | Map to InnerLamp's Design System + Clean Architecture rules if they bind. |
| 3.6 Software system attributes | Non-functional qualities: **Reliability, Availability, Security, Maintainability, Portability**. | Security is usually the biggest one for InnerLamp (PII, tokens, device attestation) — see `references/gathering_questions.md`. |

### 4. Verification & traceability

| Subsection | What goes in it |
|---|---|
| 4.1 Verification approach | How each requirement will be verified (unit test, UI test, manual QA, code review). Map to the verification method already on each `FR-*`. |
| 4.2 Traceability matrix | A table mapping each requirement ID → source (which user story / which part of the input doc) → design → test. If the design/test columns are still empty (SRS stage), leave them as "→ il-spec-writer" / "TBD". |

### 5. Appendices

| Subsection | What goes in it |
|---|---|
| Appendix A — Open questions | The running list of `⚠️ Not confirmed` items plus "Open questions (resolved)" with dates. |

## Marking N/A vs leaving a gap

The rule: **mark N/A only when the section genuinely cannot apply to a mobile app feature** (e.g.
hardware interfaces for a screen, or user documentation for a consumer flow). If a section *could*
apply but you simply don't have the answer yet, do **not** mark it N/A — leave it with the relevant
`⚠️ Not confirmed` note. N/A means "irrelevant by nature"; `⚠️` means "unanswered, still relevant".
