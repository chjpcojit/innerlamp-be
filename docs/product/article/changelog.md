# Changelog — article product spec

## v1.0 — 2026-10-15

- First product spec for the Dharma articles, written retroactively from the backend spec (`article`
  v2.0, assumptions A-001 and A-002), the iOS specs (`Article` v1.3, `Articles` v1.1), the website and
  Android code, and the prototype ("Bài viết Phật pháp hay", "Tất cả bài viết", "Bài viết Phật pháp"),
  with anh Đức's answers of 2026-10-15.
- Decisions: no ADR for the reading list; shared links follow ADR-0004 (website URL, opens the app when
  installed); no Save action; featured = first 5 in editorial order (website home shows 4); pages of 10;
  "Đọc tiếp" = 2 articles of the same category picked by the backend, completed from the featured order;
  image order on cards and on the page; source and summary notice always shown; content not translated,
  apps Vietnamese only; no offline copy, empty block when nothing loads; category is a label only;
  everything public; content changes only through the content file load.
- IDs: BR-ART-01…09, FR-ART-001…009, NFR-ART-01…02. Vectors: `article-order` (4 cases),
  `article-related` (7 cases).
- Open: content and image rights (A1), website domain for shared links (A2), retry and text of the empty
  block (A3), correcting or removing a loaded article (A4).
- Platforms to follow up: Backend (related-articles read, commit the content file, rewrite spec v2.0 in
  English), Website (Share with link, "Đọc tiếp", association files of ADR-0004, empty block instead of
  the static home fallback), iOS (drop Save, show the summary notice, page image order, "Đọc tiếp" from
  the backend, Share with link, open shared links), Android (from sample data to the backend, pages of
  10, drop Save, "Đọc tiếp" from the backend, Share with link, open shared links, empty block).
