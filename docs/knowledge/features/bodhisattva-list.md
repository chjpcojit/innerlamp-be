---
type: Feature
title: Bodhisattva List
description: A signed-in user browses the offline-first list of Buddhas and Bodhisattvas.
tags: [content, bodhisattva, offline-first]
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

# Narrative

As a signed-in user
I want to see the list of Buddhas and Bodhisattvas in the library
So that I can learn about them and choose one to focus my practice on

# Rules

- The list endpoint is public read (`GET /v1/content/bodhisattvas`) and follows the same delta-sync pattern as [sutras](/features/sutra-list.md) and [mantras](/features/mantra-list.md).[^content-spec]
- An entry with no image shows a default placeholder image rather than an error or a broken layout.
- A [bodhisattva](/glossary/bodhisattva.md) with no [sect](/glossary/sect.md) assigned (`sect` = NULL, e.g. one venerated across multiple sects) only appears under "All", never under a specific sect filter.[^content-spec]

# Scenarios

## Viewing the list after it loads

```gherkin
Given a signed-in user opens the Buddhas & Bodhisattvas list screen
When the app finishes loading
Then the user sees a list of entries
And each entry shows a name and an image
```

## Entry without an image

```gherkin
Given a bodhisattva has no image URL
When it is shown in the list
Then a default placeholder image is shown for it
And no error is shown
```

## Filtering by sect

```gherkin
Given the list contains entries from multiple sects
When the user filters by a sect
Then only entries assigned to that sect are shown
And entries with no sect assigned are not shown
```

## Viewing entries with no sect filter applied

```gherkin
Given the list contains an entry with no sect assigned
When the user views the list under "All"
Then that entry is shown
```

## Viewing the list while offline

```gherkin
Given a signed-in user has previously synced the list
And the device has no network connection
When the user opens the Buddhas & Bodhisattvas list screen
Then the previously cached list is still shown
```

## Background sync picks up server-side changes

```gherkin
Given the server has updated the list since the last sync
When the app performs its background sync
Then the local cache is updated via delta sync
```

[^content-spec]: User Stories — Module content, US-009
