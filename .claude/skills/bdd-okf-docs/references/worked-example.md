# Worked example: a minimal conformant bundle

A complete five-file bundle to pattern-match against. It passes
`scripts/okf_lint.py` with no errors.

```
specs/
  index.md                    # root index — the one file that may carry okf_version
  log.md
  features/
    index.md
    checkout.md
  glossary/
    index.md
    basket.md
```

## `specs/index.md`

The bundle-root index is the only index that may carry frontmatter, and
`okf_version` is the only key permitted in it.

````markdown
---
okf_version: "0.2"
---

# Behavior specifications

* [Features](features/) - Capabilities described as scenarios.
* [Glossary](glossary/) - Ubiquitous language for the checkout domain.

# History

* [Change log](log.md) - Dated record of additions and revisions.
````

## `specs/log.md`

No frontmatter. Newest first. ISO date headings.

````markdown
# Specification Log

## 2026-08-21
* **Update**: Added the expired-saved-card scenario to [Checkout](/features/checkout.md).
* **Creation**: Established [Basket](/glossary/basket.md).

## 2026-08-20
* **Creation**: Established [Checkout](/features/checkout.md) from BILL-431.
* **Initialization**: Created the bundle structure.
````

## `specs/features/index.md`

No frontmatter. Descriptions lifted from each concept's `description`.

````markdown
# Features

* [Checkout](checkout.md) - A shopper converts a basket into a paid order.
````

## `specs/features/checkout.md`

The workhorse. Narrative, rules, scenarios, outline, and a footnote keyed
to a `sources` entry.

`````markdown
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
    resource: https://wiki.acme/finance/payment-capture
    title: Payment capture policy
---

# Narrative

As a returning customer
I want to pay with a saved card
So that I can check out without re-entering my details

# Rules

- A [basket](/glossary/basket.md) with zero items cannot be checked out.
- Payment is authorized at order placement and captured on dispatch.[^pay-policy]
- A saved card that has expired is treated as no saved card.[^ac-431]

# Scenarios

## Paying with a saved card

```gherkin
Given a returning customer with a valid saved card
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

## Paying with an expired saved card

```gherkin
Given a returning customer whose saved card has expired
And a basket containing 1 item
When they confirm the order
Then they are asked for payment details
And no order is created until payment succeeds
```

# Examples

```gherkin
Scenario Outline: Basket subtotal determines the shipping band
  Given a basket subtotal of <subtotal>
  When shipping is calculated
  Then the shipping cost is <shipping>

  Examples:
    | subtotal | shipping |
    | 10.00    | 4.99     |
    | 49.99    | 4.99     |
    | 50.00    | 0.00     |
```

# Out of scope

Guest checkout, which never involves a saved card.

# Open questions

- Should an expired card be removed from the account automatically, or
  only flagged?

[^ac-431]: BILL-431 acceptance criteria
[^pay-policy]: Payment capture policy
`````

## `specs/glossary/basket.md`

A glossary term is a concept like any other. Note the `Given a basket…`
lines across the feature all mean *this*, which is the point of writing it
down.

````markdown
---
type: Glossary Term
title: Basket
description: The mutable collection of items a shopper intends to buy.
tags: [checkout, domain-language]
status: stable
generated: { by: human:rgamble, at: 2026-08-21T09:00:00Z }
verified: { by: human:pchen, at: 2026-08-21T15:30:00Z }
---

# Definition

The mutable collection of items a shopper intends to buy, scoped to one
customer and one session. A basket becomes an order at the moment payment
is authorized; until then it holds no reservation on stock.

# Notes

- "Cart" appears in older tickets and in the checkout UI copy. Prefer
  "basket" in specifications so scenarios read consistently.
- An empty basket is a valid state, not an error — see
  [Checkout](/features/checkout.md).
````

## `specs/glossary/index.md`

````markdown
# Glossary

* [Basket](basket.md) - The mutable collection of items a shopper intends to buy.
````

## What makes this conformant

- Every concept file (`checkout.md`, `basket.md`) has parseable frontmatter
  with a non-empty `type`.
- No `index.md` or `log.md` carries frontmatter, except the root index
  carrying `okf_version` alone.
- Actor strings use `human:`, `process:`, or `producer/version`.
- Every footnote label (`ac-431`, `pay-policy`) resolves to a `sources[].id`.
- Log date headings are ISO `YYYY-MM-DD`, newest first.
- Cross-links are bundle-relative, beginning with `/`.

Note `basket.md` uses the bare-mapping form of `verified`. That is valid,
and consumers are required to read it as a one-element list.
