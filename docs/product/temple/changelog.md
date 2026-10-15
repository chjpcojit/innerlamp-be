# Changelog — temple product spec

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
