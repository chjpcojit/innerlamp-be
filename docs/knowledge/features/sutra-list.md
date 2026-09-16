---
type: Feature
title: Sutra List
description: A signed-in user browses the offline-first list of available sutras.
tags: [content, sutra, offline-first]
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

# Narrative

As a signed-in user
I want to see the list of sutras available in the app
So that I can pick which sutra to read or chant

# Rules

- The sutra list endpoint is public read (`GET /v1/content/sutras`) even though the screen only appears inside the signed-in app flow.[^content-spec]
- The list is offline-first: the local cache is always shown immediately, network is a background concern, not a blocking one.[^content-spec]
- Sync is delta-based — the client stores `synced_at` and requests only records changed `updated_since` that timestamp on later opens.[^content-spec]
- A user can filter the list by [sect](/glossary/sect.md) (Nam Tông / Bắc Tông / Kim Cương Thừa).

# Scenarios

## Viewing the list after it loads

```gherkin
Given a signed-in user opens the sutra list screen
When the app finishes loading
Then the user sees a list of sutras
And each entry shows at least its name and sect, when the sutra has one
```

## First-time load populates the cache

```gherkin
Given a signed-in user opens the sutra list screen for the first time
And no prior sync has happened
When the app calls the sutras API
Then the full list is downloaded
And the list is saved to the local cache
```

## Reopening shows the cache first, then syncs in the background

```gherkin
Given a signed-in user has previously synced the sutra list
When the user reopens the sutra list screen
Then the list is shown immediately from the local cache, without waiting on the network
And the app calls the API in the background to sync any changes
```

## Viewing the list while offline

```gherkin
Given a signed-in user has previously synced the sutra list
And the device has no network connection
When the user opens the sutra list screen
Then the previously cached list is still shown
And no full-screen blocking error is shown
```

## Filtering the list by sect

```gherkin
Given the sutra list contains sutras from multiple sects
When the user filters by a sect
Then only sutras belonging to that sect are shown
```

## Background sync picks up server-side changes

```gherkin
Given the server has a new or updated sutra since the last sync
When the app performs its background sync
Then the local cache is updated to reflect the change
And the user does not need to take any action
```

# Out of scope

- Keyword search within the list.
- Bookmarking / marking a sutra as favorite.
- Sorting by "most popular" or "recently read" — depends on tracking data from a journal module not yet specified.

[^content-spec]: User Stories — Module content, US-005
