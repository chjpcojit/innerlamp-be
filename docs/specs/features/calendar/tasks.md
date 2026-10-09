# Tasks — calendar

Version: 1.1
Last updated: 2026-10-07
Status: implemented + verified against a running server (curl, port tạm)
Module: `calendar`

> See `changelog.md` in this folder for the version history of this spec.

## Phase 1 — Domain

- [x] `domain/calendar.go`: `CalendarImage`, `ErrInvalidDate` (không tag JSON)

## Phase 2 — Repository (manifest parser) + Assets

- [x] `assets/embed.go`: nhúng `images/home-calendar/manifest.json`
- [x] `repository/calendar_manifest.go`: `ParseCalendarManifest` (AC10)
- [x] `repository/calendar_manifest_test.go`: bảng lỗi + manifest nhúng thật khớp file ảnh trên đĩa

## Phase 3 — Config + Usecase

- [x] `config/config.go` + `config.dev.yaml`: `CalendarConfig{Timezone, ImageBaseURL}`; cập nhật `config_test.go` nếu có kiểm tra khoá
- [x] `usecase/calendar_usecase.go`: `NewCalendarUsecase`, `ImageForDate` (AC1–AC5, AC10)
- [x] `usecase/calendar_usecase_test.go`: cùng ngày cùng ảnh, ngày liên tiếp khác ảnh, bỏ trống = hôm nay theo múi giờ, ngày sai, biên năm, năm < 1970, wrap

## Phase 4 — Delivery

- [x] `delivery/http/errors.go`: `ErrInvalidDate` → `400 INVALID_DATE`
- [x] `delivery/http/calendar_handler.go` (AC1, AC6–AC9)
- [x] `delivery/http/mock_calendar_usecase_test.go` + `calendar_handler_test.go`

## Phase 5 — Wiring & docs

- [x] `cmd/main.go`: parse manifest, dựng usecase/handler, route `GET /v1/calendar/image` công khai (AC8)
- [x] `docs/specs/api/openapi.yaml`: thêm route + schema
- [x] `api-contract.md` cho iOS (đã đối chiếu với server thật)
- [x] Chạy server thật + `curl` kiểm tra AC1–AC9; `go build ./... && go vet ./...`

## Deferred

- Thumbnail ảnh (chờ iOS xác nhận có cần).
- Ảnh theo ngày lễ (cần lịch âm ở BE).
