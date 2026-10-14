# Tasks — temple-announcement

Version: 1.2
Last updated: 2026-10-14
Status: implemented — còn lại: OKF concept (sau khi người kiểm chứng) và dựng lại `erd.svg`
Module: `temple-announcement`

> See `changelog.md` in this folder for the version history of this spec.

Thứ tự theo `design.md`: schema trước, rồi phía ghi (backoffice), rồi phía đọc (app), rồi wiring và tài liệu. Mỗi phase viết test cùng lúc (`golang-clean-arch-tdd`); đối chiếu TA1–TA33 ở `requirements.md`. Repository test đặt `TEST_DB_NAME=innerlamp_test`.

## Phase 1 — Schema

- [x] `migrations/0036_temple_announcements.sql`: `temple_issuing_bodies` (+ unique `(temple_id, lower(name))`), `temple_announcements` (+ CHECK, index `idx_temple_announcements_active`), cột `users.receive_temple_announcements` (TA12, TA28, TA30, TA31)
- [x] `migrations/README.md`: ghi `0036`
- [x] `docs/specs/db/erd.mmd`: hai bảng và cột `users` (`erd.svg` **chưa** dựng lại: repo không có `mmdc`)

## Phase 2 — Backoffice domain + config

- [x] `backofficeService/domain/temple_announcement.go`: entity, `AnnouncementState`, input, sentinel, hai interface repository
- [x] `backofficeService/config/config.go` + `config.dev.yaml`: `TempleAnnouncementConfig` + `Validate()` + `config_test.go` (TA20, TA21, TA31)

## Phase 3 — Backoffice usecase

- [x] `backofficeService/usecase/temple_announcement_usecase.go`: `Publish` (P-3…P-6), `Edit`, `Revoke`, `List`, `CreateIssuingBody`, `ListIssuingBodies` (TA17, TA19–TA29)
- [x] `usecase/temple_announcement_usecase_test.go`: bảng cho trường rỗng/quá dài (đếm rune, tiếng Việt có dấu)/`expires_at` quá khứ/quá `max_validity_days`, chùa không có, ban không thuộc chùa, sửa bản đã thu hồi, thu hồi hai lần, tên ban trùng khác hoa thường; `state` suy ra đúng ở biên `expires_at == now`

## Phase 4 — Backoffice repository

- [x] `backofficeService/repository/postgres_temple_announcement_repository.go`
- [x] `repository/postgres_temple_announcement_repository_test.go` (Postgres thật): FK cascade khi xoá chùa, unique tên ban, CHECK `expires_at > published_at`, phân trang ổn định

## Phase 5 — Backoffice delivery + wiring

- [x] `backofficeService/delivery/http/temple_announcement_handler.go` + test handler (strict decode, mã lỗi, `details.field`)
- [x] `backofficeService/delivery/http/errors.go`: map sentinel mới
- [x] `backofficeService/cmd/main.go`: sáu route (design cũ ghi "năm", đã sửa thành sáu) bọc `reviewer`
- [x] `backofficeService/e2e/temple_announcement_e2e_test.go`: viewer bị `403`, không token `401`, đăng → sửa → thu hồi

## Phase 6 — App domain + config + usecase

- [x] `appService/domain/temple_announcement.go` + `mock_temple_announcement_repository.go`
- [x] `appService/config/config.go` + `config.dev.yaml`: `temple_announcement.{page_size, max_page_size, rate_limit.{max, window}}` + test (TA10, TA32)
- [x] `appService/usecase/temple_announcement_usecase.go`: `ListForUser` (R-1…R-5), `GetReceive`, `SetReceive` (TA1, TA3–TA11, TA12–TA16)
- [x] `usecase/temple_announcement_usecase_test.go`: không chùa → rỗng; tắt công tắc → rỗng; hết hạn bị loại ở biên; thu hồi bị loại; đổi chùa A→B→A; công tắc giữ nguyên khi đổi chùa

## Phase 7 — App repository + delivery + wiring

- [x] `appService/repository/postgres_temple_announcement_repository.go` + test (Postgres thật): chỉ chùa của user, thứ tự `published_at DESC, id`, `published_at` tương lai không trả
- [x] `appService/delivery/http/temple_announcement_handler.go` + test handler (body công tắc strict: thiếu `receive`, sai kiểu, khoá lạ → `400`)
- [x] `appService/delivery/http/errors.go`: `ErrInvalidPage` → `400 INVALID_PAGE`
- [x] `appService/cmd/main.go` + `cmd/temple_announcement_routes.go` (helper `registerTempleAnnouncementRoutes`) + `temple_announcement_routes_test.go`: ba route bọc `authMiddleware` + bộ giới hạn riêng `NewUserRateLimit` (TA32, TA33)
- [x] `appService/e2e/temple_announcement_e2e_test.go`: không token `401`; vượt giới hạn `429 RATE_LIMITED` và không ảnh hưởng budget của `/v1/users/me`; user không chùa rỗng; hai thiết bị cùng danh sách; đổi chùa qua `PATCH /v1/users/me` rồi đọc lại

## Phase 8 — Tài liệu và hợp đồng

- [x] `docs/specs/api/openapi.yaml`: chín route, schema, mã lỗi mới (tag `TempleAnnouncements`; YAML parse OK)
- [x] `docs/specs/features/temple-announcement/api-contract.md` cho client web/app
- [x] `web/docs`: `guides/catalogues/temple-announcements.md`, sidebar, changelog, `guides/error-handling.md`
- [ ] OKF concept qua `il-okf-writer` **sau khi** code đã kiểm chứng
- [x] `go build ./...`, `go vet ./...`, `go test -short ./services/...` và E2E cả hai service (`TEST_DB_NAME=innerlamp_test`) đều pass
