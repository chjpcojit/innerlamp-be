# Changelog — temple product spec

## v1.2 — 2026-10-15

Answers to the three product questions the four platform specs raised. No new rule IDs.

- Resolved (anh Đức): a temple without Rằm & Mùng Một hours shows the normal hours on that row
  (BR-TPL-09). Already covered by the vector "Rằm without holiday hours uses normal hours"; no vector
  change.
- Resolved (anh Đức): fields that come later (architecture details, "Kinh sách", photo set) stay visible
  with "Đang cập nhật"; "Kinh sách" is no longer "when known" (BR-TPL-06, FR-TPL-007).
- Resolved (anh Đức): when the featured temples cannot be loaded, the home block shows the same empty
  state as an empty directory, with no error text and no sample temples (FR-TPL-001, new scenario).
- Platforms to follow up: Website, iOS and Android specs remove the matching ⚠️ markers; the website
  drops its sample temples on error.

## v1.1 — 2026-10-15

- Resolved (anh Đức): the backend adds Rằm & Mùng Một hours, contacts, Buddhist sangha body and update
  month now; architecture details, "Kinh sách" and the photo set later (BR-TPL-06, FR-TPL-007).
- Resolved (anh Đức): open/closed is judged in Vietnam time and uses the Rằm & Mùng Một hours on lunar
  days 1 and 15 (BR-TPL-09); new vectors `rules/temple-open-status.json` (10 cases).
- Resolved (anh Đức): the follow action is hidden until the follow feature exists (BR-TPL-08).
- Resolved (anh Đức): the directory is loaded from the JSON directory file in every environment
  (FR-TPL-010); Appendix B asks the backend to keep that file under version control.
- Open: crediting sources and image rights (Appendix A).
- Platforms to follow up: Backend (new optional fields, commit the directory file), all clients
  (BR-TPL-09 in Vietnam time).

## v1.0 — 2026-10-15

- First product spec for the temple directory, written retroactively from the backend spec
  (`temple` v1.14), the iOS specs (`Temple` SRS v0.1 and v2.0, `Temples` v1.0), the website and Android
  code and the prototype, with anh Đức's answers of 2026-10-15.
- Decisions: no ADR; scope = featured top N, directory pages of 12, server search on name + aliases,
  region filter North / Central / South; temple page kept; follow is a later feature; report wrong
  information dropped; offering hidden; distance only with the user's location; back-office editing out
  of scope; featured order unchanged.
- IDs: BR-TPL-01…09, FR-TPL-001…010, NFR-TPL-01…02. Vectors: `temple-order`, `temple-search-match`,
  `temple-distance`.
- Open: temple-page fields to add and open/closed time zone (A1), follow action hidden (A2), content
  operations (A3).
- Platforms to follow up: Backend (search + region, page size 12), Website (search, region filter,
  remove follow), iOS (server search and paging instead of loading all pages, remove follow / offering /
  report, location for distance), Android (from sample data to the backend).
