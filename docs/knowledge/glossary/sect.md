---
type: Glossary Term
title: Sect
description: A Buddhist tradition (tông phái) a sutra, mantra, or bodhisattva may belong to.
generated: { by: claude-sonnet-5/bdd-okf-docs, at: 2026-09-09T00:00:00Z }
sources:
  - id: content-spec
    resource: docs/specs/content/content.md
    title: "User Stories — Module content"
---

A sect (tông phái) is a Buddhist tradition: Nam Tông (Theravāda), Bắc Tông
(Mahāyāna), or Kim Cương Thừa (Vajrayāna). [Sutras](/glossary/sutra.md),
[mantras](/glossary/mantra.md), and [bodhisattvas](/glossary/bodhisattva.md)
may each optionally belong to one sect (`sect` field, nullable).

Content with no sect assigned is shown only in an unfiltered ("All") view,
never when a specific sect filter is applied.
