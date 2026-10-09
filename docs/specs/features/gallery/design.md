# Design — gallery

Version: 1.1
Last updated: 2026-10-07
Status: implemented (2026-10-06) — mirror `requirements.md`.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Ba route `GET` công khai (không `RequireAuth`), chỉ đọc. Chạm đủ bốn layer Clean Architecture (domain / usecase / repository / delivery), sở hữu migration `0017_gallery_images.sql`. Bám sát hình dạng `deity` (top + danh sách phân trang + chi tiết, seeder `cmd/seed*` chạy tay) cộng thêm bộ lọc `type` tuỳ chọn cho danh sách.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/gallery.go` | `GalleryImage` (id, type, sortOrder, title, location, thumbnailURL, imageURL, author, license, sourceURL), `GalleryPage`, hằng `GalleryChua`/`GalleryTuongPhat`/`GalleryThien`/`GalleryLeHoi`, `IsValidGalleryType`. Không tag. |
| Domain / Errors | `services/appService/domain/gallery.go` | `ErrGalleryImageNotFound`, `ErrInvalidGalleryType`; tái dùng `ErrInvalidPage` (ở `domain/temple.go`). |
| Domain / Repository interface | `services/appService/domain/gallery.go` | `GalleryRepository`: `ListTop(ctx, limit)`, `List(ctx, imageType, offset, limit)`, `FindByID`, `Upsert`. Thứ tự `sort_order ASC, id ASC` là một phần hợp đồng. |
| Domain / Mock | `services/appService/domain/mock_gallery_repository.go` | Mock tay cho test usecase. |
| Usecase | `services/appService/usecase/gallery_usecase.go` | `Top`, `List(ctx, imageType, page)`, `Get`. Chặn `page`, kiểm `type`, tính `total_pages`. |
| Repository | `services/appService/repository/postgres_gallery_repository.go` | SQL, `Upsert` một transaction, scan. |
| Delivery / Handler + DTO | `services/appService/delivery/http/gallery_handler.go` | Parse `page`/`type`, gọi usecase, DTO summary/detail (JSON tag chỉ ở đây). |
| Delivery / Errors | `services/appService/delivery/http/errors.go` | `ErrGalleryImageNotFound` → 404 `GALLERY_IMAGE_NOT_FOUND`; `ErrInvalidGalleryType` → 400 `INVALID_GALLERY_TYPE` (`details.field = type`). |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `GalleryConfig{TopLimit, PageSize}`, block `gallery:` (10 / 20). |
| Seeder | `services/appService/cmd/seedgallery/{main,parse}.go` | Đọc `assets/data/buddhist_gallery.json`, kiểm id/type/title/URL, tính `sort_order` xen kẽ, gọi `Upsert`. |
| Ảnh dev | `scripts/download_gallery_images.py`, `assets/images/gallery/` | Tải ảnh Wikimedia về file system, ghi `credits.json`, viết lại URL trong JSON thành `http://<LAN IP>:8080/static/images/gallery/...`; phục vụ bằng route `/static/images/` có sẵn (`delivery/http/static_images.go`). ⚠️ DEV ONLY. |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler, đăng ký 3 route (không gọi seed). |
| Migration | `migrations/0017_gallery_images.sql` | Bảng `gallery_images` + `CHECK` trên `type` + index `(sort_order, id)` và `(type, sort_order, id)`. |
| API spec | `docs/specs/api/openapi.yaml` | 3 route + schema + hai mã lỗi mới. |
| API contract cho client | `docs/specs/features/gallery/api-contract.md` | Tài liệu cho iOS. |

## Request flow

`top` và `detail` chỉ một câu SQL; `list` có hai câu — xem Step-by-step Flow.

```
GET /v1/gallery/top
  └─ public
       └─ GalleryHandler.Top
            └─ GalleryUsecase.Top(ctx)
                 └─ GalleryRepository.ListTop(ctx, cfg.TopLimit)
                      └─ SELECT … FROM gallery_images ORDER BY sort_order, id LIMIT $1

GET /v1/gallery/{id}
  └─ public
       └─ GalleryHandler.Get
            └─ GalleryUsecase.Get(ctx, id)
                 └─ GalleryRepository.FindByID → SELECT … WHERE id = $1 (no rows → ErrGalleryImageNotFound)
```

## Step-by-step Flow — Danh sách phân trang (`GET /v1/gallery?type=&page=`)

```
Step 1 — Parse query (delivery)
- Trigger: client gọi GET /v1/gallery
- Pre-condition: không
- Call: page rỗng → 1, ngược lại strconv.Atoi; type = query "type" (rỗng = mọi loại)
    - Success → GalleryUsecase.List(ctx, type, page)
    - Failure (page không phải số nguyên) → 400 INVALID_PAGE

Step 2 — Validate (usecase)
- Trigger: Step 1 thành công
- Pre-condition: không
- Call: page < 1 hoặc > MaxInt32 → ErrInvalidPage; type != "" và không thuộc 4 giá trị → ErrInvalidGalleryType
    - Success → GalleryRepository.List(ctx, type, offset, page_size)
    - Failure → 400 INVALID_PAGE / 400 INVALID_GALLERY_TYPE

Step 3 — Đếm (repository)
- Trigger: Step 2 thành công
- Pre-condition: không
- Call: SELECT count(*) FROM gallery_images [WHERE type = $1]
    - Success → total
    - Failure (lỗi DB) → 500 INTERNAL_ERROR
    - Transaction boundary: Step 3 và Step 4 là hai câu độc lập, KHÔNG chung transaction/snapshot (như temple/deity)

Step 4 — Lấy trang (repository)
- Trigger: Step 3 thành công
- Pre-condition: không
- Call: SELECT <cột> FROM gallery_images [WHERE type = $1] ORDER BY sort_order ASC, id ASC OFFSET $ LIMIT $
    - Success → items (rỗng nếu vượt cuối)
    - Failure (lỗi DB) → 500 INTERNAL_ERROR

Step 5 — Dựng trang (usecase)
- Trigger: Step 4 thành công
- Pre-condition: không
- Call: items nil → []; total_pages = ceil(total / page_size)
    - Success → 200 {items, page, page_size, total, total_pages}
```

## Step-by-step Flow — Nạp dữ liệu bằng `cmd/seedgallery`

```
Step 1 — Đọc và parse file (seedgallery/parse.go)
- Trigger: operator chạy `go run ./services/appService/cmd/seedgallery [-file ...]`
- Pre-condition: file tồn tại
- Call: json.Unmarshal → kiểm id duy nhất, type hợp lệ, title / image_url / thumbnail_url không rỗng
    - Success → danh sách GalleryImage
    - Failure → log.Fatalf, không chạm DB

Step 2 — Tính sort_order (parse.go)
- Trigger: Step 1 thành công
- Pre-condition: không
- Call: xen kẽ theo vòng CHUA → TUONG_PHAT → THIEN → LE_HOI, giữ thứ tự file trong từng loại; sort_order = vị trí sau xen kẽ
    - Success → mỗi ảnh có SortOrder

Step 3 — Upsert (repository)
- Trigger: Step 2 thành công
- Pre-condition: DB kết nối được bằng config dev, migration 0017 đã apply
- Call: GalleryRepository.Upsert → INSERT … ON CONFLICT (id) DO UPDATE
    - Success → log "seeded N images"
    - Failure → rollback toàn bộ, log.Fatalf
    - Transaction boundary: mọi hàng chung MỘT transaction — tất cả hoặc không hàng nào
```

## Known gaps carried by this feature

- Chưa có E2E black-box; `erd.svg` chưa render lại từ `erd.mmd`.
- Ảnh nằm trên file system của máy dev và URL mang LAN IP (`⚠️ DEV ONLY`): đổi mạng thì chạy `make lan-urls`; chưa có CDN (xem `⚠️ Not confirmed` trong `requirements.md`).
- Seeder dùng cấu hình dev (`⚠️ DEV ONLY`), như `seeddeities`.
