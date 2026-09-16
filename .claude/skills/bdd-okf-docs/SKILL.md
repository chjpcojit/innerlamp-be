---
name: bdd-okf-docs
description: Author behavior-driven development documentation — features, scenarios, Given/When/Then, acceptance criteria, ubiquitous language — as OKF (Open Knowledge Format) v0.2 conformant knowledge bundles, with Gherkin embedded in the concept body. Use this skill whenever the user mentions BDD, Gherkin, Cucumber, SpecFlow, Behave, feature files, scenarios, Given/When/Then, acceptance criteria, three amigos, specification by example, living documentation, or OKF/Open Knowledge Format — and also whenever they ask to write, review, restructure, or lint behavior specs for a feature, even if they never say "OKF" or "BDD" by name. Language- and framework-agnostic.
---

# BDD documentation as OKF concepts

BDD and OKF solve adjacent halves of the same problem. BDD gets teams to
agree on behavior in concrete examples. OKF makes a corpus of those
examples *trustable* over time: where each scenario came from, who agreed
to it, and whether it is still true. Writing BDD docs as OKF concepts
means a feature spec stops being a wiki page nobody dares delete and
becomes a document with provenance and a shelf life.

Your job with this skill is to produce a bundle where every feature is one
readable markdown file that a product person can review, a developer can
implement against, and an agent can traverse.

## The shape of the thing

A bundle is just a directory of markdown files. There is no schema
registry, no tooling requirement, no build step.

```
<bundle>/
  index.md                    # No frontmatter. Directory listing.
  log.md                      # No frontmatter. Dated change history.
  features/
    index.md
    checkout.md               # type: Feature — narrative + Gherkin
    refunds.md
  glossary/
    index.md
    basket.md                 # type: Glossary Term
  personas/
    index.md
    returning-customer.md     # type: Persona
```

Group by domain, not by artifact kind, once a bundle outgrows a handful of
features — `billing/`, `checkout/`, `identity/` each with their own
`index.md`. The directory structure is yours to choose; OKF does not care.

## Conformance is a low bar — clear it every time

Only three rules make a bundle conformant, and they are easy to satisfy
and easy to forget:

1. Every non-reserved `.md` file has a **parseable YAML frontmatter block**
   delimited by `---` on its own line, top and bottom.
2. Every frontmatter block has a **non-empty `type`**.
3. `index.md` and `log.md` are **reserved filenames**. They follow the
   structures below and are never used for concept documents.

The trap worth internalizing: **`index.md` and `log.md` carry no
frontmatter at all.** The single exception is a bundle-root `index.md`,
which may carry `okf_version: "0.2"` and nothing else. Adding a `type:`
block to an index file is the most common way to produce a non-conformant
bundle, because it feels like it should be right.

Everything else is optional. Consumers are required to tolerate unknown
types, unknown keys, missing fields, and broken links, so prefer writing
a useful document over an exhaustively annotated one. Run
`scripts/okf_lint.py` when you are done and fix what it reports.

## Type vocabulary

Type values are not registered anywhere — you pick them, and consumers
must tolerate whatever you choose. Use these unless the domain suggests
better, and stay consistent within a bundle:

| `type` | What it holds |
|---|---|
| `Feature` | One capability, its narrative, and its scenarios. The workhorse. |
| `Capability` | A grouping above features (an epic or journey) that links to them. Use only when features genuinely need a parent. |
| `Glossary Term` | One term from the ubiquitous language. |
| `Persona` | An actor that scenarios say "As a…" about. |
| `Rule` | A business rule complex enough to outgrow a heading inside a feature. Most rules should stay inside the feature. |

Resist inventing a type per scenario. A scenario is body content inside a
`Feature`, not its own document — splitting them scatters the narrative
that makes the examples legible.

## Frontmatter for a Feature

```yaml
---
type: Feature
title: Checkout
description: A shopper converts a basket into a paid order.
tags: [checkout, payments, revenue]
status: stable
generated: { by: human:rgamble, at: 2026-08-20T10:00:00Z }
verified:
  - { by: human:pchen, at: 2026-08-21T15:30:00Z }
  - { by: process:ci-acceptance, at: 2026-08-22T02:14:00Z }
stale_after: 2026-11-20
sources:
  - id: ac-431
    resource: https://tracker.acme/BILL-431
    title: "BILL-431 acceptance criteria"
    author: human:pchen
    last_modified: 2026-08-14
  - id: pay-policy
    resource: /policies/payment-capture.md
    title: Payment capture policy
---
```

**`status` tracks agreement, not automation.** `draft` means the team has
not yet agreed this is the behavior they want. `stable` means they have.
`deprecated` means the behavior is retired but the document stays for its
links and history. A scenario can be fully automated and still be `draft`
if nobody has agreed it describes the right thing, and that distinction is
exactly what stops a green test suite from being mistaken for a correct
one. Absent `status` means `stable`.

**`verified` is where the three amigos land.** Each entry is a `{ by, at }`
verification event, and multiple entries capture independent checks. This
is deliberately separate from `generated`, because whoever wrote the
scenarios need not be whoever confirmed them. Consumers read a trust tier
straight off it:

- no `verified` key → **unverified**
- `verified` by non-`human:` actors only → **machine-confirmed**
- any `human:<id>` entry → **human-reviewed**

So a feature whose only verifier is `process:ci-acceptance` is visibly
"tests pass, nobody read it" — the single most useful signal this format
gives a BDD corpus. Record the human sign-off when it happens; do not
invent one.

**Actor strings follow a fixed convention** and the `human:` prefix is
what drives the trust tier, so it has to be right: `human:<id>` for a
person, `process:<id>` for automation, `<producer>/<version>` for an agent
or tool (e.g. `claude-opus-5/bdd-okf-docs`). When you author scenarios
yourself, you are the producer — use your own actor string in
`generated.by`, and do not write a `human:` verifier on your own output.

**`stale_after` is an absolute date**, never a relative TTL, so staleness
is a plain date comparison. Set it when the behavior is tied to something
with a known review cycle — a pricing rule, a regulatory window, a
contract. Leave it off for behavior with no natural expiry rather than
inventing a date.

**`sources`** records what the scenarios derive from: tickets, acceptance
criteria, interview notes, policy documents. Give an entry an `id` when
the body cites it. A `resource` may be an absolute URL, a bundle-relative
path beginning with `/`, or a scope descriptor a consumer cannot follow
(`"discovery interviews, Q2 2026"`) — the last is legitimate and better
than omitting provenance.

For details on the optional field families — credibility signals,
`usage_window`, path forms — read `references/okf-fields.md`.

## Body structure

Favor structural markdown over prose; it is what makes a document both
human-readable and agent-retrievable. Use these headings:

````markdown
# Narrative

As a returning customer
I want to pay with a saved card
So that I can check out without re-entering details

# Rules

- A basket with zero items cannot be checked out.
- Payment is captured on dispatch, not on order placement.[^pay-policy]

# Scenarios

## Paying with a saved card

```gherkin
Given a returning customer with a saved card
And a basket containing 2 items
When they confirm the order
Then the order is placed
And the payment is authorized but not captured
```

## Checking out an empty basket

```gherkin
Given a signed-in customer
And an empty basket
When they attempt to check out
Then they are told the basket is empty
And no order is created
```

# Examples

```gherkin
Scenario Outline: Basket totals apply the right shipping band
  Given a basket subtotal of <subtotal>
  When shipping is calculated
  Then the shipping cost is <shipping>

  Examples:
    | subtotal | shipping |
    | 10.00    | 4.99     |
    | 50.00    | 0.00     |
```

# Out of scope

Guest checkout is covered by [guest checkout](/features/guest-checkout.md).

# Open questions

- Does a saved card that expires mid-session block the order?

[^pay-policy]: Payment capture policy
````

`# Examples` is a heading OKF already blesses, which makes it the natural
home for `Scenario Outline` tables. `# Narrative`, `# Rules`,
`# Scenarios`, `# Out of scope`, and `# Open questions` are this skill's
conventions — consistent within a bundle is what matters, since consumers
tolerate anything.

Put each scenario under its own `##` heading with a sentence-style name,
then a ```gherkin fence. The heading is what a reader scans and what a
link can target; the fence is what a parser lifts. Keep the `Given/When/
Then` lines bare inside the fence — the `Feature:` and `Scenario:`
keywords are redundant when the document *is* the feature and the heading
*is* the scenario name. The exception is `Scenario Outline`, which needs
its keyword to make the `Examples:` table meaningful.

## Citing a source for a specific claim

Attribute individual claims with a markdown footnote whose label is a
`sources[].id`:

```markdown
Payment is captured on dispatch, not on order placement.[^pay-policy]

[^pay-policy]: Payment capture policy
```

The label is the join key into `sources` — consumers resolve attribution
through the matching entry, not by reading the footnote prose. Labels are
keyed rather than positional precisely because these documents get
rewritten constantly, and a positional index misattributes silently the
moment the list is reordered.

## Linking

Use bundle-relative links beginning with `/` — they survive a document
being moved within its subdirectory:

```markdown
The basket is defined in [basket](/glossary/basket.md).
```

Link a term the first time a feature uses it, and link between features
that share behavior. A link asserts a relationship; the kind of
relationship is carried by the surrounding prose, not by the link. Broken
links are explicitly fine — they often mark knowledge not yet written —
so link to a glossary term you intend to write rather than skipping it.

## index.md and log.md

`index.md` supports progressive disclosure: it lets a reader see what
exists before opening anything. No frontmatter; descriptions lifted from
the linked concepts.

```markdown
# Features

* [Checkout](checkout.md) - A shopper converts a basket into a paid order.
* [Refunds](refunds.md) - A customer returns an item and is repaid.

# Glossary

* [Glossary](../glossary/) - Ubiquitous language for the checkout domain.
```

`log.md` is a flat list of dated entries, newest first, with ISO `YYYY-MM-DD`
headings:

```markdown
# Feature Log

## 2026-08-21
* **Update**: Added the expired-saved-card scenario to [Checkout](/features/checkout.md).
* **Deprecation**: Retired [Legacy guest checkout](/features/guest-checkout-v1.md).

## 2026-08-14
* **Creation**: Established [Checkout](/features/checkout.md) from BILL-431.
```

Update `log.md` whenever you add or meaningfully change a concept. The
bold leading word is a convention, not a requirement.

## Writing scenarios that stay true

The failure mode of BDD documentation is scenarios that describe the
interface instead of the behavior. Those break whenever a button moves,
which teaches the team the specs are noise.

- **Declarative over imperative.** "When they confirm the order", not
  "When they click `#submit-btn`". The behavior is the durable part; the
  selector is not.
- **One behavior per scenario.** If a scenario needs a second `When`,
  it is two scenarios.
- **Ubiquitous language everywhere.** Use the domain's words, and put
  them in `glossary/` so the words themselves have definitions with
  provenance. If a scenario needs a term the glossary lacks, write the
  glossary entry.
- **`Background` for genuinely shared setup only.** Context repeated in
  every scenario belongs in `Background`; context that matters to the
  example belongs in the scenario where a reader can see it.
- **Scenarios come from sources.** If you cannot name what a scenario
  derives from, that is a signal to ask rather than to invent. Record it
  in `sources` and cite it.

## Workflow

1. **Read what exists.** If the bundle is there, read its root `index.md`,
   the relevant `index.md` files, and neighbouring features. Match their
   type vocabulary, heading conventions, and directory layout — a bundle
   that is internally consistent beats one that is individually perfect.
2. **Gather sources before writing.** Tickets, acceptance criteria, policy
   docs, existing tests. These become `sources` entries, and their absence
   is worth surfacing to the user rather than papering over.
3. **Draft the narrative and rules first, scenarios second.** Scenarios
   that precede an agreed narrative tend to encode the implementation
   someone already had in mind.
4. **Write the concept files**, then any new glossary terms they reference.
5. **Update `index.md` and `log.md`** for every directory you touched.
6. **Lint**: `python3 scripts/okf_lint.py <bundle>`. Fix real errors;
   warnings are advisory and some are legitimately ignorable.
7. **Report what you could not source.** Scenarios you inferred rather
   than derived, and `verified` entries you deliberately left empty, are
   the things a human needs to act on.

## Bundled resources

- `scripts/okf_lint.py` — conformance checker. Validates frontmatter,
  required `type`, reserved-file rules, actor convention, footnote/source
  resolution, log date formats, and link targets. Run it before you call
  the work done; it catches the index-frontmatter mistake reliably.
- `references/okf-fields.md` — the full frontmatter field catalog with
  every optional family, path-value rules, and the v0.1 fields you may
  encounter in older bundles. Read it when a bundle uses a field this page
  does not cover, or when you need the exact semantics of a credibility
  signal.
- `references/worked-example.md` — a complete six-file bundle (root index,
  log, feature, glossary term, and both directory indexes) to pattern-match
  against when starting fresh.
