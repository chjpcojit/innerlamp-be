---
type: Feature
title: Mantra List
description: A signed-in user browses the offline-first list of available mantras.
tags: [content, mantra, offline-first]
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

# Narrative

As a signed-in user
I want to see the list of mantras available in the app
So that I can pick which mantra to recite

# Rules

- The mantra list endpoint is public read (`GET /v1/content/mantras`) and follows the same delta-sync pattern as [sutras](/features/sutra-list.md).[^content-spec]
- A mantra may optionally be linked to one [bodhisattva](/glossary/bodhisattva.md); when it isn't, the list shows the mantra without related-bodhisattva info and does not error.
- A user can filter the list by related bodhisattva, and separately by [sect](/glossary/sect.md).
- A mantra with no sect assigned (`sect` = NULL) does not appear when filtering by a specific sect.[^content-spec]

# Scenarios

## Viewing the list after it loads

```gherkin
Given a signed-in user opens the mantra list screen
When the app finishes loading
Then the user sees a list of mantras
And each entry shows its name and related bodhisattva, when one is assigned
```

## Mantra without a related bodhisattva

```gherkin
Given a mantra has no bodhisattva assigned
When it is shown in the list
Then no related-bodhisattva information is shown for it
And no error is shown
```

## Filtering by related bodhisattva

```gherkin
Given the mantra list contains mantras linked to different bodhisattvas
When the user filters by one bodhisattva
Then only mantras linked to that bodhisattva are shown
```

## Filtering by sect

```gherkin
Given the mantra list contains mantras from multiple sects
When the user filters by a sect
Then only mantras belonging to that sect are shown
And mantras with no sect assigned are not shown
```

## Viewing the list while offline

```gherkin
Given a signed-in user has previously synced the mantra list
And the device has no network connection
When the user opens the mantra list screen
Then the previously cached list is still shown
```

## Background sync picks up server-side changes

```gherkin
Given the server has a new or updated mantra since the last sync
When the app performs its background sync
Then the local cache is updated via delta sync, using the `updated_since` marker
```

# Out of scope

- Recitation counting (Mala Beads Counter) — belongs to a `practice` module, not `content`.
- Keyword search within the list.

[^content-spec]: User Stories — Module content, US-007
