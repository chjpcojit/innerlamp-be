# Changelog — calendar

## v3.1 (2026-10-15) — sau khi viết spec ba client

- `requirements.md` / `design.md` / `flow.md`: E23 đổi: request có `X-API-Key` đúng (website) không bị đếm vào giới hạn theo IP, vì website gọi từ một IP server (phát hiện khi viết spec website). Step 1b và sequence vẽ lại nhánh này. `tasks.md` thêm một task cho nhánh bỏ qua giới hạn.
- Header cite product spec v1.2 (không đổi ID Backend). Resolved: mã nhóm `buddhist` ổn định (BR-CAL-09 v1.2).
- Xoá `SRS.md` và `api-contract.md` cũ (đã bị product spec + `openapi.yaml` thay thế); sửa các chỗ trỏ tới chúng ở `docs/knowledge/` và `docs/specs/features/temple-event/SRS.md`. Các mục changelog cũ nhắc tới hai file này giữ nguyên làm lịch sử.

## v3.0 (2026-10-15) — theo product spec `calendar` v1.1

US-CAL-002 chuyển sang cite product spec `docs/product/calendar/product.md` v1.1 (be#21) thay cho `SRS.md`. Bump major vì đổi hợp đồng (route công khai, trường mới) và đổi schema. US-CAL-001 không đổi.

- `requirements.md`: header `Product spec` / `Contract` / `Implements`; cột "SRS" thành "Product" (BR-/FR-/NFR-CAL). Đổi E4 (route công khai), E13 (thêm `is_major`, `sort_order`, `recurrence`; giữ nguyên xuống dòng), E17 (`private` → `public`), E19 (thêm `is_major`). Thêm E22 (`X-API-Key` sai → 401), E23 (429 theo IP), E24 (seed nạp `is_major`), E25 (seed chặn Rằm/Mùng 1 không xếp cuối), E26 (seed vào git), E27 (chạy test vector của product). User story đổi sang khách + người đăng nhập trên ba client. Hai quyết định "Xác thực" và "cờ lễ lớn" chuyển sang resolved 2026-10-15.
- Mặc định do spec này chọn (merge PR = duyệt): bảo vệ route công khai bằng `webAPIKey` + giới hạn theo IP riêng `calendar.events_rate_limit`, cùng hình dạng `administrative-units`; trường `recurrence` để client nhận biết Rằm / Mùng 1 cho khối Home (BR-CAL-15); `is_major` tuỳ chọn trong seed (thiếu = `false`, giữ `format: 1`).
- `design.md`: các dòng **v3.0** trong Layers & components (domain, repository, handler, config, migration `0037`, seed CLI, data, test vectors, wiring, openapi, E2E); mô hình dữ liệu thêm `is_major`; response shape thêm ba trường; Step-by-step Flow lấy sự kiện thay bước JWT bằng Step 1a (`webAPIKey`) + 1b (giới hạn IP), Step 6 đổi `Cache-Control`; flow seed thêm luật Verify và `is_major`; Known gaps thêm rủi ro route công khai.
- `flow.md`: sequence lấy sự kiện thay participant `authMiddleware` bằng `webAPIKey` + `calendarEventLimit`; seed thêm `is_major` và luật Verify mới. Mirror 1:1 `design.md` v3.0.
- `tasks.md`: thêm Phase 14–18, mọi item `[ ]`. Phase 1–13 giữ `[x]` (đã đối chiếu code: `cmd/main.go` vẫn bọc `authMiddleware`, handler vẫn `private`).
- Reason: product spec v1.1 (quyết định 2026-10-15: calendar công khai, cờ lễ lớn, seed vào git, nội dung seed sửa theo BR-CAL-05/10/14).

Cross-file follow-ups (chưa làm ở đây):
- `docs/specs/api/openapi.yaml`: bỏ `security`, thêm `X-API-Key`, `401`, `429`, ba trường (Phase 18).
- `migrations/0037_*.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd` / `erd.svg` (Phase 14).
- `.gitignore` + `assets/data/calendar-events.json` (Phase 16).
- `docs/knowledge/features/calendar.md`: chưa có; `il-okf-writer` sau khi kiểm chứng.
- `SRS.md`, `api-contract.md` trong thư mục này: đã bị product spec + `openapi.yaml` thay thế, xoá ở bước dọn tài liệu cũ.
- Product spec §4.2: cột Backend của FR-CAL-001 và BR-CAL-10 đổi sang `done` sau khi triển khai (`ba-writer`).

## v2.1 (2026-10-11) — US-CAL-002 triển khai

Triển khai toàn bộ US-CAL-002 (Phase 6–13). Không đổi AC; sửa các chỗ spec lệch thực tế khi code.

- `requirements.md` / `design.md` / `flow.md` / `tasks.md` / `api-contract.md`: Status đổi từ "spec only" sang đã triển khai; `api-contract.md` thay ví dụ minh hoạ bằng trích response thật của server dev (năm 2026: 51 sự kiện, 75 occurrence).
- **Số migration `0031`, không phải `0030`**: `0030` đã dùng cho `temple_suggestions_address_codes`. Sửa mọi tham chiếu.
- **Tên entity `CalendarLunarDate`**, không phải `LunarDate`: `genealogy.go` đã có `LunarDate` (không có năm). Handler DTO cũng dùng `calendarLunarDateResponse`.
- **`ReplaceCatalogue` không ghi `is_active`**: sự kiện mới mặc định `true`, sự kiện đã có giữ giá trị back-office đặt, để chạy lại seed không bật lại thứ BO đã tắt (hiểu E18 "ghi đè nội dung" là nội dung file, không gồm cờ BO). Có test khoá.
- **`CalendarCatalogue.Verify()` nằm ở `domain`** (cùng mẫu `Dataset.Verify`); `parse.go` chỉ giải mã strict, kiểm tham số theo kind (có/không có), NFC, và bắt buộc `source`.
- **Ngày vía cuối tháng** (Địa Tạng 30/7, Dược Sư 30/9) dùng kind `lunar_last_day` (29 hoặc 30), theo quyết định "lùi về 29" — không cần đổi quy tắc `lunar`.
- Thêm `calendar.default_language` và `calendar.events_cache_max_age` vào config (bắt buộc, `CalendarConfig.Validate`); thêm target `ensure-calendar-events` / `seed-calendar-events` vào `Makefile`.
- Hạ tầng E2E: `harness.go` chờ server 30s thay vì 10s (khởi động mất đúng 10s ngân sách kết nối storage khi emulator trỏ vào cổng chết) và gom stdout vào log lỗi khởi động.
- `docs/specs/api/openapi.yaml`: bỏ nhãn "đang là spec", sửa message mẫu của `INVALID_YEAR`. `docs/specs/features/memorial/requirements.md`: ghi chú server đã có bộ chuyển đổi âm ↔ dương nhưng memorial chưa dùng.
- Còn lại (`erd.svg` đã sinh lại sau đó, 2026-10-11): đối chiếu nội dung seed với nguồn chính thức, bảng tham chiếu độc lập cho E20, `docs/knowledge/features/calendar.md` (gọi `il-okf-writer` khi anh yêu cầu). Phát hiện: `usecase/genealogy_lunar.go` trùng thuật toán với `domain/lunarcal` và trả `Day: 0` ở 2054-05-07, 2062-04-09 — chưa sửa (ngoài phạm vi).

## v2.0 (2026-10-11) — US-CAL-002 sự kiện lịch (prospective)

Thêm user story US-CAL-002: sự kiện lịch Phật giáo / Việt Nam / dân gian, trả theo năm, kèm cả ngày dương và ngày âm. Nguồn nghiệp vụ là `SRS.md` v0.3 (IEEE 29148, đã chốt hết câu hỏi). Bump major vì có scope mới, bảng/migration mới, hai Step-by-step Flow mới và `flow.md` mới. US-CAL-001 (ảnh theo ngày) **không đổi**.

- `requirements.md`: thêm US-CAL-002 và AC `E1…E21` (đánh số riêng để các tham chiếu AC1–AC10 của US-CAL-001 không đổi nghĩa), cột "SRS" truy vết sang `FR-*`/`NFR-*`. Sửa phần "Out of scope": câu cũ "tính ngày âm lịch / ngày lễ: client tự làm, server không biết lịch âm" không còn đúng cho sự kiện — nay server tính; route ảnh vẫn không dùng lịch âm. Thêm bảng API cho `GET /v1/calendar/events`, các Open questions đã giải quyết (2026-10-11).
- `design.md`: thêm mục US-CAL-002 vào Layers & components (domain, `domain/lunarcal`, resolver, repository, usecase tách riêng, handler, config, migration, seed CLI, data); mô hình dữ liệu bốn bảng; bảng tham số theo `kind`; thuật toán "Sinh ngày theo quy tắc" (xét năm âm `Y−1` và `Y`); hình dạng response chuẩn hoá theo sự kiện; hai Step-by-step Flow (lấy sự kiện của năm; nạp seed). Mục Request flow của route ảnh giữ nguyên. Các quyết định thiết kế tự đặt (ghi để người review biết): usecase/handler tách khỏi `CalendarUsecase` để không đổi chữ ký route ảnh; `lunarcal` đặt ở `domain/` thay vì `pkg/`; response chuẩn hoá `events` + `occurrences`; sự kiện được đọc từ DB mỗi request, không cache bộ nhớ; múi giờ lịch âm cố định UTC+7 (định nghĩa lịch, không phải cấu hình).
- `flow.md`: **mới** — hai sequence diagram mirror 1:1 hai Step-by-step Flow của `design.md`.
- `tasks.md`: giữ nguyên Phase 1–5 `[x]` của US-CAL-001; thêm Phase 6–13 cho US-CAL-002, mọi item `[ ]`; mở rộng Deferred.
- `api-contract.md`: thêm mục `GET /v1/calendar/events` (đề xuất theo spec, chưa đối chiếu server thật); mục route ảnh không đổi.
- Reason: yêu cầu mới từ chủ sản phẩm ngày 2026-10-11 (lịch sự kiện, API theo năm + "load more"), đã qua BA (`SRS.md`).

Cross-file follow-ups (chưa làm ở đây, theo rule của skill):
- `docs/specs/api/openapi.yaml`: thêm `/v1/calendar/events`, schema `CalendarEventsResponse`/`CalendarOccurrence`/`LunarDate`, code `INVALID_YEAR` (Phase 13).
- `migrations/0031_calendar_events.sql` + `migrations/README.md` + `docs/specs/db/erd.mmd`/`erd.svg` (Phase 8).
- `docs/specs/features/memorial/requirements.md`: còn câu "server không biết lịch âm" — cần sửa khi US-CAL-002 triển khai (Phase 13).
- `docs/knowledge/features/calendar.md`: chưa có; gọi `il-okf-writer` sau khi code + kiểm chứng.
- `assets/data/calendar-events.json`: dữ liệu seed do team thu thập (Phase 10) — chưa tồn tại.
- Không đổi `docs/specs/features/calendar/SRS.md`.

## v1.1 (2026-10-07) — implemented

Code xong và kiểm tra trên server thật: domain, parser manifest, config, usecase, handler, wiring, openapi; mọi task `[x]`. Chỉ đổi `Status`/checkbox/`Pending` và ví dụ trong `api-contract.md` (lấy từ server thật); AC không đổi. Phát hiện khi code: `time.LoadLocation("")` âm thầm trả UTC nên usecase từ chối `calendar.timezone` rỗng (khớp AC10, không đổi spec).

Cross-file follow-ups:
- `docs/specs/api/openapi.yaml`: đã thêm `/v1/calendar/image` (done).
- `docs/knowledge/features/calendar.md`: chưa viết, gọi `il-okf-writer` khi được yêu cầu.

## v1.0 (2026-10-07) — initial

Spec đầu tiên cho `GET /v1/calendar/image` (ảnh Phật theo ngày cho màn Lịch âm), theo yêu cầu của session iOS. Prospective: mọi task `[ ]`.

Cross-file follow-ups:
- `docs/specs/api/openapi.yaml`: thêm route + schema (Phase 5).
- `docs/knowledge/features/calendar.md`: viết bằng `il-okf-writer` sau khi chạy end-to-end (chưa làm).
- Không có migration / ERD.
