# Changelog — gallery Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v2.0.2 — 2026-10-15

Follows the product spec gallery v1.1 (be#40). Nothing in the backend contract changes.

- `requirements.md`: header cites product.md @ v1.1; a release-gate note records that the credit in every list
  (AC 11) now also gates the apps' viewer (product §2.7), so v2.0 code (Phase 6–7) is prioritised.
- `design.md`: version only. No requirement or design change; v1.1 changed client behaviour (error and empty
  blocks, copy, viewer), none of it on the server.

## v2.0.1 — 2026-10-15

Cleanup after the platform specs of all four repos were merged.

- `api-contract.md` deleted (superseded by `docs/specs/api/openapi.yaml`, tag Gallery).
- `requirements.md`, `design.md`: references to it removed. No requirement changed.

## v2.0 — 2026-10-15

Rebased on the product spec `docs/product/gallery/product.md` @ v1.0 (be#35) and rewritten in English per the
Language policy. MAJOR because the contract grows and the seeder refuses files it used to accept.

- `requirements.md`: header `Product spec` / `Contract` / `Implements`; every criterion numbered and tagged with
  its BR-/FR-GAL ID or `(platform)`. New (not implemented): AC 3 `total` on `top` (BR-GAL-07), AC 11 credit
  (`author`, `license`, `source_url`) on every list item (BR-GAL-04), AC 16 seeder refuses an empty author or
  licence (FR-GAL-009), AC 19 shared order vectors (BR-GAL-02). AC 20 content file in git — done in be#35.
  Old `⚠️ Not confirmed` items resolved or moved to two product-level items (production photo storage ADR,
  production content load).
- `design.md`: `Top` reuses `List(ctx, "", 0, top_limit)` to get the total (no new repository method;
  `ListTop` removed); DTO changes; new Step-by-step Flow "Featured photos"; seeder validation; vector test; E2E.
- `flow.md`: new sequence for `top`; existing two diagrams translated, seeder step 1 lists the new checks.
- `tasks.md`: Phase 6–8 added, all `[ ]`; content file item `[x]`.
- `api-contract.md`: marked superseded; deleted in the cleanup step.
- Follow-ups (not edited here): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` (Phase 7),
  `docs/knowledge/features/gallery.md` (after E2E). No migration, no ERD change.

## v1.1 — 2026-10-07

Ảnh không còn hotlink Wikimedia ở dev: tải 86 ảnh về `assets/images/gallery/{full,thumb}` và phục vụ từ file
system qua `/static/images/` bằng URL LAN IP, như `temple`/`deity`. Đổi: `requirements.md` (mô tả nguồn ảnh,
Out of scope, marker `⚠️ Not confirmed` về hotlink → về base URL production), `design.md` (thêm hàng "Ảnh dev",
Known gaps), `tasks.md` (2 mục `[x]` mới, đã kiểm bằng `curl`), `api-contract.md` (thuộc tính ảnh, ghi chú dev).
`flow.md` không đổi (không có bước mới trong hai flow). Không đổi contract HTTP/schema DB, nên `openapi.yaml`
chỉ sửa mô tả ảnh. Thêm `scripts/download_gallery_images.py` và gallery vào `make lan-urls`.

Yêu cầu "lọc theo type": đã có sẵn từ v1.0 (`GET /v1/gallery?type=`; bỏ trống = mọi loại), không đổi.

## v1.0 (initial) — 2026-10-06

Initial spec written **before implementation** (Prospective), then implemented the same day (Phase 1–4 `[x]`, xem `tasks.md`), as the work order for the public read-only
gallery catalogue built from `assets/data/buddhist_gallery.json`: `GET /v1/gallery/top`,
`GET /v1/gallery?type=&page=`, `GET /v1/gallery/{id}`, table `gallery_images` (migration `0017`),
seeder `cmd/seedgallery`. Mọi task `[ ]`.

Quyết định mặc định của spec (đánh `⚠️ Not confirmed`, đổi được): "lấy ảnh theo danh sách" = danh sách
phân trang có lọc `type` tuỳ chọn; `top` lấy theo `sort_order` xen kẽ 4 loại; detail thêm
`author`/`license`/`source_url` để ghi công CC.

Đã cập nhật cùng lượt: `docs/specs/api/openapi.yaml` (+ bản `web/docs/public/`), `migrations/README.md`,
`docs/specs/db/erd.mmd`, thêm `api-contract.md`. Còn lại: render `erd.svg`, E2E, và khái niệm OKF
`docs/knowledge/features/gallery.md` (do `il-okf-writer` tạo sau khi code được kiểm chứng).
