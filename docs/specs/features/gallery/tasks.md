# Tasks — gallery

Version: 1.1
Last updated: 2026-10-07
Status: implemented (2026-10-06) cho Phase 1–4; Phase 5 còn thiếu E2E tự động và OKF. Mọi `[x]` bên dưới đã kiểm chứng ngày 2026-10-06: `go build ./...`, `go vet ./services/appService/...`, `go test -short ./services/appService/...` sạch; repository test chạy trên Postgres thật (DB tạm `innerlamp_test`, đã xoá) pass 5/5; migration + seed chạy trên DB dev (86 ảnh, chạy lần hai vẫn 86); ba route gọi bằng `curl` trên binary thật.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

Mọi mục dưới đây trace về một hàng của bảng "Layers & components" hoặc một Step của "Step-by-step Flow" trong `design.md`.

## Phase 1 — Domain

- [x] `GalleryImage`, `GalleryPage`, hằng `GalleryChua`/`GalleryTuongPhat`/`GalleryThien`/`GalleryLeHoi`, `GalleryTypes`, `IsValidGalleryType` trong `services/appService/domain/gallery.go` (không tag)
- [x] `ErrGalleryImageNotFound`, `ErrInvalidGalleryType` trong `domain/gallery.go`; tái dùng `ErrInvalidPage`
- [x] Interface `GalleryRepository` (4 method, `List` nhận `imageType`; thứ tự `sort_order ASC, id ASC` nằm trong hợp đồng)
- [x] `domain/mock_gallery_repository.go`

## Phase 2 — Usecase

- [x] `usecase/gallery_usecase.go`: `Top`, `List` (chặn `page`, kiểm `type`, offset, `total_pages`, `items` nil → `[]`), `Get`
- [x] Unit tests `usecase/gallery_usecase_test.go` — `TestGalleryUsecase_Top`, `_List` (lọc type, page lạ, type lạ), `_Get`, dùng mock tay

## Phase 3 — Repository, config và dữ liệu

- [x] `migrations/0017_gallery_images.sql` — bảng, `CHECK` trên `type`, hai index
- [x] `repository/postgres_gallery_repository.go`: `ListTop`, `List` (lọc `type` bằng tham số), `FindByID`, `Upsert` một transaction
- [x] `config`: `GalleryConfig` + block `gallery:` (10 / 20) trong `config.dev.yaml`
- [x] `cmd/seedgallery/main.go` + `parse.go`: kiểm id/type/title/URL, `sort_order` xen kẽ 4 loại, `Upsert`; target `make seed-gallery`
- [x] Tests: `cmd/seedgallery/parse_test.go` (file thật 86 mục, validation, xen kẽ); `repository/postgres_gallery_repository_test.go` (upsert/find, not found, atomic, ordering, phân trang + lọc type) — cần Postgres, bỏ qua khi `-short`; thêm `0017` và `gallery_images` vào `applyMigration`/`truncateAll` của harness
- [x] Seed vào DB dev: 86 ảnh (41 `CHUA`, 21 `TUONG_PHAT`, 12 `THIEN`, 12 `LE_HOI`)

## Phase 4 — Delivery và tài liệu

- [x] `delivery/http/gallery_handler.go`: `Top`, `List`, `Get` + DTO summary/detail
- [x] `delivery/http/errors.go`: `404 GALLERY_IMAGE_NOT_FOUND`, `400 INVALID_GALLERY_TYPE` (`details.field = type`)
- [x] `cmd/main.go`: dựng dây và đăng ký 3 route
- [x] Handler tests `delivery/http/gallery_handler_test.go` + `mock_gallery_usecase_test.go`
- [x] `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: 3 route, tag `Gallery`, 5 schema (parse YAML được)
- [x] Ghi `0017` vào `migrations/README.md`; bảng `gallery_images` trong `docs/specs/db/erd.mmd`
- [ ] Render lại `docs/specs/db/erd.svg` từ `erd.mmd` (chưa chạy mermaid-cli)
- [x] `api-contract.md` cho client iOS

## Phase 5 — Integration & documentation

- [ ] E2E black-box trong `services/appService/e2e/`: ba route, lọc `type`, `INVALID_PAGE`, `INVALID_GALLERY_TYPE`, `GALLERY_IMAGE_NOT_FOUND`, bảng rỗng trả `items: []`
- [x] `go build ./...` và `go vet ./services/appService/...` sạch
- [ ] Khái niệm OKF `docs/knowledge/features/gallery.md` — do `il-okf-writer` tạo sau khi có E2E; chưa có file

- [x] `scripts/download_gallery_images.py` + `assets/images/gallery/{full,thumb}` (86 + 86 ảnh jpg, ~32 MB) + `credits.json`; JSON trỏ URL LAN; thêm `buddhist_gallery.json` và `seedgallery` vào `make lan-urls`
- [x] Kiểm chứng: `curl` `http://192.168.1.186:18080/static/images/gallery/{full,thumb}/...` trả `200 image/jpeg`; đủ 86/86 file mỗi loại

## Deferred (separate feature, not blocking gallery)

- Đưa ảnh lên GCS/CDN thay file system dev — chặn bởi `⚠️ Not confirmed` trong `requirements.md`
- Quy trình nạp dữ liệu ngoài dev — chặn bởi `⚠️ Not confirmed` trong `requirements.md`
