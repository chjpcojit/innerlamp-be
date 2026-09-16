---
type: Glossary Term
title: Delta Sync
description: The offline-first sync strategy shared by sutra, mantra, and bodhisattva lists.
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

Delta sync is the offline-first caching strategy used by the content
lists ([Sutra List](/features/sutra-list.md),
[Mantra List](/features/mantra-list.md),
[Bodhisattva List](/features/bodhisattva-list.md)):

- The local cache is always shown first — the screen never blocks on the
  network.
- The client stores a `synced_at` timestamp locally and calls the list
  endpoint with `updated_since=<synced_at>`, so only changed records are
  transferred on subsequent opens.
- A background sync silently reconciles the local cache with the server
  without requiring user action.
- When offline, the last-synced cache is shown with no blocking error.
