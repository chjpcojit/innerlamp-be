---
type: Feature
title: Mantra Detail
description: A signed-in user views the full text of one mantra, optionally with recitation audio.
tags: [content, mantra, offline-first]
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

# Narrative

As a signed-in user
I want to view the full content of one specific mantra
So that I can recite it correctly

# Rules

- Reached from [Mantra List](/features/mantra-list.md) by selecting an entry.
- The detail endpoint is `GET /v1/content/mantras/:slug` — this route did not exist in the original endpoint list and needs to be added.[^content-spec]
- When the mantra has a related [bodhisattva](/glossary/bodhisattva.md), the detail screen links to that bodhisattva's detail screen (see [Bodhisattva Detail](/features/bodhisattva-detail.md)).

# Scenarios

## Viewing full mantra content

```gherkin
Given a signed-in user selects a mantra from the mantra list
When the detail screen opens
Then the user sees the full text and name of the mantra
And, when a bodhisattva is linked, its info with a link to that bodhisattva's detail screen
```

## Mantra with recitation audio

```gherkin
Given a mantra has an audio URL
When the user opens its detail screen
Then a play button for the recitation audio is shown
```

## Mantra without recitation audio

```gherkin
Given a mantra has no audio URL
When the user opens its detail screen
Then no play button is shown
And no error is shown
```

## Reading a cached mantra while offline

```gherkin
Given a signed-in user has previously viewed a mantra
And the device has no network connection
When the user reopens that mantra's detail screen
Then the full content is still shown from the cache
```

# Out of scope

- Direct integration with the Mala Beads Counter from this screen — needs coordinated design with the `practice` module, to be specified separately.

[^content-spec]: User Stories — Module content, US-008
