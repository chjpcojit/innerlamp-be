# Changelog — memorial Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v2.4 (2026-10-07)

- `requirements.md`: câu trả lời cho HEIC chuyển từ "cách hiểu của spec" sang đã xác nhận — client đổi HEIC/HEIF sang JPEG, backend chỉ nhận jpeg/png/webp. AC27 không đổi.
- `design.md`: no change needed, verified — Step P-2 đã đúng (chỉ nhận jpeg/png/webp, sniff nội dung).
- `flow.md`: no change needed, verified.
- `tasks.md`: no change needed, verified — Phase 6 không có việc BE nào cho HEIC; phần client ghi ở `docs/tech-debt/memorial/TD-001`.
- Reason: anh Đức xác nhận client tự đổi sang JPEG. `docs/assumptions/memorial/A-001` đổi `status` sang `confirmed`. Bump minor vì không đổi hành vi.
- Vẫn mở: marker bucket GCS (chỉ chặn Phase 6).

## v2.3 (2026-10-07)

- `requirements.md`: no change to behaviour; Status/Pending cập nhật — `erd.svg` đã sinh lại nên bỏ khỏi Pending.
- `design.md`: mục "Known gaps" về `erd.svg` đổi từ "chưa sinh lại" sang "đã sinh lại"; không đổi thiết kế.
- `flow.md`: no change needed, verified.
- `tasks.md`: tick mục sinh lại `erd.svg`; bỏ chú thích "chưa kiểm cú pháp"; Status chỉ còn marker bucket.
- Reason: xử lý các điểm còn lại sau v2.2. `erd.svg` render lại bằng `mmdc` từ `erd.mmd` (render thành công = cú pháp hợp lệ; bản cũ còn thiếu `gallery_images`). Hai file `openapi.yaml` được bỏ ignore trong `.gitignore` (`!docs/specs/api/openapi.yaml`, `!web/docs/public/openapi.yaml`) nên giờ hiện trong `git status` và diff được.
- Vẫn mở (cần anh quyết, không tự làm): marker bucket GCS; cách hiểu HEIC của `A-001` chưa được anh xác nhận. OKF memorial chưa viết theo yêu cầu.

## v2.2 (2026-10-07)

- `requirements.md`: v1 đã implement — cập nhật Status, đoạn mở đầu và mục Pending (chỉ còn ảnh và `erd.svg`); contract API đã có trong `openapi.yaml`. AC không đổi.
- `design.md`: sửa các dòng code đã làm sai: validation dùng `MemorialValidationError` thay vì "cơ chế có sẵn"; `MemorialRepository.Delete` ở v1 chỉ trả `error` và `SetPhotoKey` chưa khai báo (đổi sang Phase 6); Step D-2 dùng `RowsAffected` ở v1; mục "Known gaps" về middleware đã được giải quyết bằng E2E. Số bước và thứ tự các flow không đổi.
- `flow.md`: no change needed, verified — chỉ cập nhật header/Status; sơ đồ vẫn khớp từng bước với `design.md`.
- `tasks.md`: tick Phase 1–5 (trừ `erd.svg` và marker bucket); sửa chữ của mục interface cho khớp chữ ký thật; thêm mục "Sinh lại `erd.svg`"; ghi chú toàn bộ test sau khi chạy.
- Reason: implement v1 theo `golang-clean-arch-tdd`. Kiểm chứng: unit (usecase, handler), repository trên Postgres thật (gồm trần 100 dưới tải song song và CHECK của DB), E2E trên binary thật + testcontainers Postgres (5 route đều 401 khi thiếu/rác/refresh token; người lạ nhận 404 byte-giống id không tồn tại; trần 100 từ config thật), `go vet ./...` và `go test -short ./...` sạch.
- Ghi nhận khi chạy đủ test không `-short`: `TestSutraRepository_ShippedData` fail (số `DeityLinks` 191 thay vì 151) — không liên quan memorial, chưa sửa.
- Nợ kỹ thuật mới: `docs/tech-debt/memorial/TD-001-photo-upload-blocked-on-gcs.md` (ảnh/GCS hoãn theo quyết định của anh Đức).
- Cross-file follow-ups: `docs/specs/db/erd.svg` cần sinh lại từ `erd.mmd` (đã thêm bảng `memorials`, chưa render để kiểm cú pháp); `openapi.yaml` (+ bản `web/`) đã cập nhật nhưng bị `.gitignore` (`*.yaml`) bỏ qua như đã ghi ở `docs/tech-debt/sutra/TD-001`; concept OKF `docs/knowledge/features/memorial.md` do `il-okf-writer` tạo khi anh muốn (flow đã chạy E2E nhưng anh chưa yêu cầu), không thuộc skill này; `docs/knowledge/index.md`/`log.md` chưa đụng.

## v2.1 (2026-10-07)

- `requirements.md`: chốt thứ tự danh sách `GET /v1/memorials` là `created_at` DESC, `id` DESC (đúng như AC13 đang viết) — marker chuyển sang "Open questions (resolved)"; còn 1 marker (bucket GCS, chỉ chặn Phase 6).
- `design.md`: no change needed, verified — index `idx_memorials_user (user_id, created_at DESC, id DESC)` đã khớp thứ tự này.
- `flow.md`: no change needed, verified — không có flow nào phụ thuộc thứ tự danh sách.
- `tasks.md`: sửa mục Phase 5 về marker còn lại; không thêm/bỏ task, mọi mục vẫn `[ ]`.
- Reason: anh Đức đồng ý với thứ tự mặc định spec đề xuất. Bump minor vì chỉ chốt một câu hỏi, không đổi hành vi.

## v2.0 (2026-10-07)

- `requirements.md`: **ảnh (US-MEM-05) chuyển sang Deferred** vì chưa làm được GCS — AC25–AC32 và các nhánh ảnh của AC16/AC23 không thuộc v1, `photo_url` luôn `null`, hai route ảnh đánh dấu ⏸; thêm mục "Deferred". Chốt: mã US tự chọn, tháng nhuận theo đề xuất (dùng tháng thường cùng số), HEIC → client đổi sang JPEG (cách hiểu của spec, ghi ở `docs/assumptions/memorial/A-001-heic-client-converts.md`), cờ OS đa thiết bị client tự xử lý, dọn object mồ côi "tính sau"; mỗi mục tối đa 1 ảnh tuỳ chọn. Còn 2 marker (thứ tự danh sách; bucket GCS — chỉ chặn Phase 6).
- `design.md`: v1 làm 5 route CRUD, không import `pkg/storage`, không dựng storage client; các layer/flow liên quan ảnh đánh dấu ⏸ Phase 6 (port `MemorialPhotoStore`, adapter GCS, config ảnh, flow tải/thay ảnh, Step D-3). Giữ cột `photo_object_key` (NULL) trong `0018` để khỏi cần migration khi làm ảnh. Số bước của flow tạo/xoá không đổi.
- `flow.md`: sơ đồ tải/thay ảnh và nhánh xoá-object trong sơ đồ xoá đánh dấu hoãn; không thêm/bỏ bước.
- `tasks.md`: Phase 1–5 thu về v1 (5 route, bỏ phần storage/ảnh); thêm Phase 6 — Ảnh ⏸ gom mọi mục ảnh, mọi mục vẫn `[ ]`.
- Reason: anh Đức quyết định tạm bỏ qua GCS. Đây là thay đổi scope (ảnh ra khỏi v1) nên bump major.
- Follow-up: `openapi.yaml` (+ bản `web/`) chỉ cần 5 route cho v1; `migrations/0018_memorials.sql` gồm 3 cột ngày-mất mới (v1.1) và `photo_object_key`; `docs/specs/db/erd.*`; OKF chưa tạo nên chưa có gì để làm lại.

## v1.1 (2026-10-07)

- `requirements.md`: năm mất thành tuỳ chọn — sửa AC6, thêm AC6a/6b, sửa AC17, thêm field `death_day`/`death_month`/`death_leap_month` vào `Memorial`; AC1 ghi rõ mục mới tạo không có ảnh; `OTHER` chỉ là một giá trị (không nhãn tự do); xác nhận "bà ngoại, cha, mẹ" là ba giá trị; ảnh tuỳ chọn. Hai marker (năm không rõ, nhãn `OTHER`) chuyển sang "Open questions (resolved)"; còn 7 marker.
- `design.md`: entity `DeathDate *time.Time` + `DeathDay/Month/LeapMonth`; schema `death_date` nullable + 3 cột mới + CHECK loại trừ hai dạng ngày mất; nhánh validate ở Step C-2. Số bước của các flow không đổi.
- `flow.md`: chỉ đổi nhãn validate ở Step C-2 của diagram tạo; không thêm/bỏ bước.
- `tasks.md`: bổ sung validation hai dạng ngày và CHECK vào Phase 2/3; cập nhật danh sách marker cần chốt ở Phase 5. Mọi mục vẫn `[ ]`.
- Reason: anh Đức cho năm mất tuỳ chọn. Vì ngày âm không đổi sang dương được khi thiếu năm, quyết định "chỉ lưu dương" giữ nguyên khi có năm, còn thiếu năm thì lưu ngày+tháng theo lịch đã nhập. Bump minor vì phần còn lại của luồng không đổi.
- Follow-up: `openapi.yaml` (+ `web/docs/public/openapi.yaml`), `migrations/0018_memorials.sql` + `docs/specs/db/erd.*` phải phản ánh 3 cột mới; OKF chưa tạo nên không có gì cần làm lại.

## v1.0 (2026-10-06)

- `requirements.md`: spec đầu tiên cho Sổ giỗ chạp, viết trước khi có code (Prospective): 5 user story `US-MEM-01..05`, 32 AC (EARS), contract 7 route đều `RequireAuth`, 9 marker `⚠️ Not confirmed`.
- `design.md`: 4 layer + storage adapter + migration `0018_memorials.sql`; ba Step-by-step Flow (tạo có khoá `users` để chặn vượt trần 100 khi chạy song song; upload ảnh DB-trước-GCS-sau; xoá).
- `flow.md`: ba sequence diagram mirror 1:1 các flow trên.
- `tasks.md`: 5 Phase, mọi mục `[ ]`.
- Reason: yêu cầu mới của anh Đức. Đã chốt trong phiên: chỉ lưu ngày dương + cờ `death_calendar` (client quy đổi âm↔dương), nhắc nhở local-only, upload ảnh multipart qua BE → GCS, giới hạn mặc định đề xuất.
- Follow-up chưa làm (không thuộc skill này): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`, `migrations/0018_memorials.sql` + `docs/specs/db/erd.*`, concept OKF `docs/knowledge/features/memorial.md` (sau khi code kiểm chứng).
