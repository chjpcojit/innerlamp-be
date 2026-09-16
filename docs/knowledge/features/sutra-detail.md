---
type: Feature
title: Sutra Detail
description: A signed-in user reads the full text of one sutra, optionally with chanting audio.
tags: [content, sutra, offline-first]
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

# Narrative

As a signed-in user
I want to view the full text of one specific sutra
So that I can read or chant along with its complete content

# Rules

- Reached from [Sutra List](/features/sutra-list.md) by selecting an entry.
- The detail endpoint is public read (`GET /v1/content/sutras/:slug`).[^content-spec]
- The client compares local `version`/`updated_at` against the server response to decide whether the cache is still valid, so a stale cached body is never shown once the network is available.[^content-spec]
- Font size is adjustable and persists for the reading session, to support long reading sessions, particularly for older users.[^content-spec]

# Scenarios

## Viewing full sutra content

```gherkin
Given a signed-in user selects a sutra from the sutra list
When the detail screen opens
Then the user sees the full text, title, and sect of the sutra
```

## Sutra with chanting audio

```gherkin
Given a sutra has an audio URL
When the user opens its detail screen
Then a play button for the chanting audio is shown
```

## Sutra without chanting audio

```gherkin
Given a sutra has no audio URL
When the user opens its detail screen
Then no play button is shown
And no error is shown
```

## Reading a cached sutra while offline

```gherkin
Given a signed-in user has previously viewed a sutra
And the device has no network connection
When the user reopens that sutra's detail screen
Then the full content is still shown from the cache
```

## Reopening after the sutra was updated on the server

```gherkin
Given a signed-in user has a cached version of a sutra
And the server has a newer version of that sutra
When the user reopens the detail screen with network available
Then the app fetches and shows the newer version
And the outdated cached version is not shown
```

## Adjusting reading font size

```gherkin
Given a signed-in user is reading a sutra
When the user increases or decreases the font size
Then the content is redisplayed at the chosen size
```

# Out of scope

- Auto wooden-fish-tap synchronized to reading pace (text scroll + audio timing) — belongs to a future `practice` module feature.
- Bookmarking / marking a sutra as favorite.
- Sharing a sutra outside the app.

[^content-spec]: User Stories — Module content, US-006
