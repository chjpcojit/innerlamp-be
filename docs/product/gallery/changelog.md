# Changelog — gallery product spec

## v1.0 — 2026-10-15

- First product spec for the Buddhist photo gallery, written retroactively from the backend spec
  (`gallery` v1.1, `api-contract.md`, assumption A-001), the iOS `Photos` v1.2 and `PhotoViewer` v1.0
  specs with their assumptions and tech-debt entries, the website and Android code, and the prototype
  (`photos`, viewer block, home section), with anh Đức's answers of 2026-10-15 (Q1–Q14).
- Decisions: no ADR for the feature (production photo storage left to a separate cross-feature ADR);
  one name "Ảnh đẹp Phật giáo"; prototype topic labels and order; credit required on every large
  photo and carried in the lists; save on the apps only; no favourite, wallpaper or share; uniform 4:3
  tiles; viewer pages through the whole filtered gallery; home "+N" counts the whole gallery; public;
  Vietnamese content; content file committed with the backend, back office later.
- IDs: BR-GAL-01…09, FR-GAL-001…009, NFR-GAL-01…03. Vectors: `gallery-order` (7 cases),
  `gallery-home-more` (5 cases).
- Also in this change: `assets/data/buddhist_gallery.json` committed with an exception in `.gitignore`.
- Platforms to follow up: Backend (credit in the lists, loader requires author and licence), Website
  (labels, chip order, lightbox across pages), iOS (name, credit, viewer range, "+N", remove stubs, real
  save), Android (everything; sample data today).
