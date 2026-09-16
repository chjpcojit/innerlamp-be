---
type: Feature
title: Bodhisattva Detail
description: A signed-in user views one Buddha/Bodhisattva's profile and the mantras associated with them.
tags: [content, bodhisattva, offline-first]
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

# Narrative

As a signed-in user
I want to view details and related mantras for one specific Buddha/Bodhisattva
So that I can understand them better and easily find a mantra to recite in their name

# Rules

- Reached from [Bodhisattva List](/features/bodhisattva-list.md) by selecting an entry.
- The detail endpoint is `GET /v1/content/bodhisattvas/:slug`. Whether related mantras are returned inline in this response (a `mantras` field) or fetched separately by the client is still open — leaning toward inline to save a round trip, to be settled when the response schema is finalized.[^content-spec]

# Scenarios

## Viewing bodhisattva detail

```gherkin
Given a signed-in user selects a bodhisattva from the list
When the detail screen opens
Then the user sees the name, description, image, and a list of related mantras, when any exist
```

## Bodhisattva with no related mantras

```gherkin
Given a bodhisattva has no mantras assigned
When the user views its detail screen
Then the related-mantras section is hidden or shows a gentle "no related mantras yet" message
And this is not treated as an error
```

## Navigating to a related mantra

```gherkin
Given a signed-in user is viewing a bodhisattva's detail screen
And the related-mantras list is not empty
When the user selects a mantra from that list
Then the user is taken to that mantra's detail screen
```

## Viewing detail while offline

```gherkin
Given a signed-in user has previously viewed a bodhisattva's detail screen
And the device has no network connection
When the user reopens that screen
Then the full information is still shown from the cache
```

[^content-spec]: User Stories — Module content, US-010
