# Changelog — profile

Lịch sử phiên bản của bộ spec `requirements.md` / `design.md` / `flow.md` / `tasks.md` (cùng một version tại mọi thời điểm). `SRS.md` có version riêng (đang là 0.3).

## v1.5 (2026-10-11)

`GET/PATCH /v1/users/me` có thêm `full_name`. Thêm field ở request và response, thêm mã lỗi `400 INVALID_FULL_NAME`; không đổi route.

- `requirements.md`: AC1/AC2/AC8 nhắc `full_name`; thêm AC30 (trim + NFC, rỗng hoặc quá `profile.full_name_max_chars` = 300 ký tự → `400 INVALID_FULL_NAME`) và AC31 (`null` bị từ chối, không xoá được); bảng API và hai ví dụ JSON cập nhật.
- `design.md`: `Profile`/`ProfilePatch`/DTO thêm `FullName`; bước `Update` thêm nhánh lỗi; ghi chú schema trỏ migration `0033`.
- `flow.md`: nhãn bước chuẩn hoá và nhãn lỗi thêm `full_name`/`INVALID_FULL_NAME`. Không thêm hay bớt bước.
- `tasks.md`: thêm Phase 7 (tất cả `[x]` trừ mục OKF); đã chạy `go test -short ./...`, test repository trên DB scratch và E2E profile.
- Reason: yêu cầu "bổ sung `full_name` vào `/me`" cùng đợt với đăng ký bắt buộc `full_name` (`onboarding` v7.5). Họ tên bắt buộc nên PATCH sửa được nhưng không xoá được. Ghi quyết định ở `docs/assumptions/onboarding/A-001-register-full-name.md`.
- Cần theo sau: OpenAPI + `web/docs` (cùng đợt); migration `0033`; OKF `profile.md` khi được làm mới.

## v1.4 (2026-10-11)

- `design.md`, `tasks.md`, `requirements.md`, `flow.md`: dev/test dùng container `fake-gcs-server` với **URL không ký**; ký thật là nợ khi deploy (`TD-001`). Không đổi AC, không đổi flow.
- `tasks.md`: tick test tích hợp adapter với emulator và E2E luồng avatar đầy đủ (hai mục đã tách ở v1.3); thêm 3 mục việc phát sinh (chế độ emulator của adapter, `storage.emulator_host`, service `gcs` trong compose).
- Reason: yêu cầu "dùng một test container cho upload ảnh, không cần ký; ký là nợ kỹ thuật khi deploy thật; container này cũng chạy chung với lệnh run server ở local". Quyết định kỹ thuật:
  - Công tắc là cấu hình (`storage.emulator_host`), không dò biến môi trường: để trống = ký thật như prod. Có giá trị thì `main` đặt `STORAGE_EMULATOR_HOST` cho GCS client và bật `WithEmulator` cho adapter. Thay thế cách đặt biến môi trường trong Makefile và e2e harness ở v1.3.
  - `fake-gcs-server` không có upload kiểu PUT; chỉ nhận `POST /upload/storage/v1/b/{bucket}/o?uploadType=media&name=…` (đã thử thật). Vì vậy cổng `AvatarStorage.SignUpload` đổi sang trả `SignedUpload{URL, Method, Headers}` và vé trả `method` do store quyết định, thay vì usecase cố định `PUT`. OpenAPI: `method` là `enum [PUT, POST]`.
  - Compose: service `gcs` (`fsouza/fake-gcs-server:1.56.1`, backend filesystem, volume `gcs-data` để ảnh sống qua `make down` như DB). Bucket do appService tạo khi khởi động ở chế độ emulator, với ngân sách thời gian riêng (3 giây, hằng trong `cmd`, chỉ dev) để emulator chưa lên không ăn thời gian seed; thiếu emulator chỉ cảnh báo, các route khác vẫn chạy.
  - E2E: một container emulator dùng chung cho cả package, ghi đè `storage.emulator_host` của mọi server test (không đụng emulator dev); mỗi test lưu object dùng bucket riêng.
- Hệ quả đã chấp nhận: ở dev, store không ép 5 MB / loại file / hạn URL. Usecase vẫn kiểm lại loại và dung lượng đã ghi nhận ở confirm (AC24) nên ảnh sai loại vẫn bị `INVALID_AVATAR`, nhưng AC21 và việc ép giới hạn trước khi upload chỉ kiểm được trên bucket thật.
- Web docs: khung cảnh báo của `catalogues/profile.md` và ví dụ cho `method` cập nhật theo; `docs/development/local-stack.md` thêm dòng service `gcs`.

### Việc theo sau (cập nhật v1.4)

- Hai bản `openapi.yaml` (`docs/specs/api` và `web/docs/public`) đang lệch nhau ở phần calendar do thay đổi dở dang của việc khác, không phải của feature này; phần profile giống nhau.
- `TD-001`: thêm vào tiêu chí trả nợ việc bật ký thật (để trống `storage.emulator_host`) và cấp quyền `signBlob`; vẫn `open`.

## v1.3 (2026-10-11)

- `tasks.md`: Phase 1–5 xong và tick; Phase 6 tick OpenAPI, web docs, E2E (không gồm luồng avatar đầy đủ) và `vet`/`test`. Tách hai mục còn mở thành mục riêng thay vì tick nửa vời: test tích hợp adapter với emulator (Phase 4) và E2E luồng avatar đầy đủ (Phase 6). Thêm 4 mục việc phát sinh khi làm (`storage.timeout`, CORS `PATCH`, `registerProfileRoutes`, `STORAGE_EMULATOR_HOST`).
- `design.md`, `requirements.md`, `flow.md`: sửa các câu "chưa có code" thành trạng thái thật; ghi các kiểu domain thực tế (`OptionalString`, `ProfileView`); không đổi flow, không đổi AC.
- Reason: implement theo TDD (`golang-clean-arch-tdd`). Các quyết định kỹ thuật do implement tự chọn, chưa có trong spec:
  - Kiểu tuỳ chọn là `domain.OptionalString{Set, Value}` thay vì một generic; `ProfileUsecase` trả `ProfileView` (đã ký địa chỉ ảnh) chứ không trả `Profile` rồi để handler ký, để handler không biết storage.
  - `upload_id` chỉ nhận dạng UUID chuẩn chữ thường (`uuid.Parse` một mình còn nhận `{...}` và `urn:uuid:`, đưa chữ lạ vào khoá object).
  - `ErrUserNotFound` → `401 UNAUTHORIZED` chỉ trong handler profile; không thêm vào `writeDomainError` dùng chung vì sẽ đổi mã lỗi của route khác (hiện trả `500`).
  - Phần mở rộng object khoá chính theo content type (`.jpg`/`.png`/`.webp`) là hằng trong usecase (do giao thức quy định), không phải config.
  - `UserRateLimit` gặp request không có tài khoản thì trả `401` và không đếm, thay vì gom vào một bucket khoá rỗng sẽ 429 người dùng thật.
  - `Update` rỗng không ghi gì, kể cả `updated_at`.
  - Thêm `storage.timeout` + `StorageConfig.Validate`; service `Connect` storage lúc khởi động (fail fast). GCS client không dựng được khi thiếu ADC, nên `make run` và e2e harness đặt `STORAGE_EMULATOR_HOST` nếu chưa có.
  - `pkg/middleware.CORS` thêm method `PATCH` (phát hiện khi nối route: không có thì web client không gọi được `PATCH /v1/users/me`).
- OpenAPI: sửa ba `message` ví dụ (`size_bytes`, `upload_id`, `INVALID_AVATAR`) cho khớp thông điệp thật; hai bản `openapi.yaml` vẫn giống nhau.
- Web docs: bỏ khung "Đang là spec, chưa có ở server" của `catalogues/profile.md` và dòng tương ứng ở `changelog.md`; thay bằng cảnh báo đúng: upload ảnh chưa kiểm chứng trên storage thật.

### Việc theo sau (cập nhật v1.3)

- `docs/specs/features/http-middleware-wiring/requirements.md` (AC3) và `docs/knowledge/packages/middleware.md` vẫn ghi danh sách method CORS cũ (`GET, POST, PUT, DELETE, OPTIONS`). Thuộc `il-spec-writer` / `il-okf-writer`, chưa sửa ở đây.
- `docs/knowledge/features/profile.md` + `index.md` + `log.md`: viết bằng `il-okf-writer` sau khi luồng avatar được dùng thật end-to-end (chưa: chỉ phần text đã được kiểm chứng).
- Test tích hợp storage với emulator và E2E luồng avatar: `tasks.md` Phase 4 và Phase 6.
- `TD-001`, `TD-002`: giữ nguyên `open`.

## v1.2 (2026-10-11)

- `requirements.md`: đóng 3 marker `⚠️ Not confirmed` cuối cùng và chuyển sang "Open questions (resolved)": xoá file cũ lỗi thì chấp nhận để lại rác (log, vẫn `200`, không thử lại); không đặt mục tiêu thời gian phản hồi; storage dev dùng emulator. Mục Pending trỏ tới hai nợ kỹ thuật mới.
- `design.md`: bỏ hai marker ở Step C-7 và D-3 (thay bằng ghi quyết định); thêm dòng "Storage dev dùng emulator" vào "Deliberately-stubbed-for-now". Không đổi flow.
- `flow.md`: hai Note của bước xoá file cũ đổi từ `⚠️ Not confirmed` sang "chấp nhận file cũ thành rác (TD-002)". Không thêm hay bớt bước.
- `tasks.md`: Phase 4 chọn emulator cho test tích hợp, thêm mục ghi chú `TD-001`/`TD-002` trong code; Phase 6 tách "kiểm tra trên bucket thật" (TD-001) và "dọn rác" (TD-002) thành hai mục.
- Reason: anh Đức trả lời 3 câu cuối ngày 2026-10-11. Hai quyết định có hệ quả vận hành nên được ghi thành nợ kỹ thuật theo `docs/tech-debt/README.md`: `TD-001` (signed URL chỉ chạy thử trên emulator, phải kiểm tra trên bucket thật khi deploy) và `TD-002` (object mồ côi không được dọn).
- `SRS.md` lên 0.4: cùng các quyết định trên, không còn mục mở.

## v1.1 (2026-10-11)

- `requirements.md`: mã lỗi của pháp danh quá dài đổi từ `400 VALIDATION_ERROR` sang `400 INVALID_DHARMA_NAME` (AC9 và bảng contract), cùng mã với đăng ký. Ghi chú "khoảng trống ở đăng ký" đổi thành đã làm.
- `design.md`: dòng Error map, Domain/Errors, Config và "Registration (sửa nhỏ)" cập nhật; Step P-3 đổi nhánh lỗi. Không đổi flow.
- `flow.md`: nhãn lỗi của bước validate PATCH đổi theo. Không thêm hay bớt bước.
- `tasks.md`: tick 3 mục đã implement cùng đăng ký (config `DharmaNameMaxChars`, hàm chuẩn hoá dùng chung, sửa `register_usecase`); phần còn lại của `ProfileConfig` vẫn `[ ]`.
- Reason: sửa khoảng trống pháp danh ở đăng ký (onboarding v7.4, AC20) và chốt **một mã lỗi cho một luật**: đăng ký dùng mã riêng theo từng field (`INVALID_EMAIL`, `INVALID_PASSWORD`, `INVALID_TEMPLE`), nên pháp danh cũng có `INVALID_DHARMA_NAME` chung cho cả hai endpoint thay vì `VALIDATION_ERROR` ở một bên và `500` ở bên kia.
- Đã đồng bộ theo: `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`), `web/docs/.../catalogues/profile.md` và `register-and-login.mdx`, `migrations/0029_users_avatar.sql` + `migrations/README.md`, `docs/specs/db/erd.mmd` + `erd.svg`.

## v1.0 (2026-10-11)

- `requirements.md`: bản đầu. 3 user story (`US-PROFILE-01..03`), 29 AC theo EARS, bám `FR-001`…`FR-009` và `NFR-003` của `SRS.md` v0.3. Bảng hợp đồng 5 route dưới `/v1/users/me`. Pending liệt kê phần hoãn có chủ đích (dọn thư mục tạm) và ba marker `⚠️ Not confirmed`.
- `design.md`: bản đầu. Bảng Layers, schema (`users.avatar_object`), vị trí object tạm/chính, Step-by-step Flow cho đọc/sửa hồ sơ (P-1…P-7), xin quyền upload (U-1…U-6), xác nhận avatar (C-1…C-8) và gỡ avatar (D-1…D-4). Phần "Deliberately-stubbed-for-now" ghi ba thứ cố ý chưa làm.
- `flow.md`: bản đầu. Bốn sơ đồ sequence mirror 1:1 các Step ở `design.md`.
- `tasks.md`: bản đầu. 6 phase, tất cả `[ ]` (chưa có code). Hai mục tài liệu (OpenAPI, web/docs) ghi rõ đã viết cùng spec nhưng chờ đối chiếu với handler thật.
- Reason: yêu cầu "client lấy được profile và chỉnh sửa thông tin cá nhân" từ SRS đã chốt 22 câu với anh Đức ngày 2026-10-11. Thay thế mục `US-004` (user story chưa design) trong `onboarding/requirements.md`. Những quyết định kỹ thuật do spec tự chọn (SRS giao cho design): tên route `/v1/users/me` và `/v1/users/me/avatar/*`; tách `ProfileUsecase` khỏi `LoginUsecase`/`RegisterUsecase`; lưu khoá object chứ không lưu URL; khoá object mới cho mỗi lần xác nhận; copy **trước** khi ghi DB, xoá file cũ **sau**; mã lỗi `INVALID_AVATAR` và `AVATAR_UPLOAD_NOT_FOUND`; bộ giới hạn khoá theo `user_id` đặt sau `authMiddleware`.

### Việc theo sau (cập nhật 2026-10-11)

- `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`: **đã cập nhật cùng v1.0** (tag `Profile`, 5 route, 5 schema). Đối chiếu lại khi handler thật xong.
- `web/docs/src/content/docs/guides/catalogues/profile.md`, `astro.config.mjs`, `changelog.md`: **đã cập nhật cùng v1.0**.
- `migrations/0029_users_avatar.sql` + `migrations/README.md`: **đã viết ở v1.1.**
- `docs/specs/db/erd.mmd` / `erd.svg`: **đã cập nhật ở v1.1** (cột `avatar_object`, `erd.svg` sinh lại bằng `npx @mermaid-js/mermaid-cli`).
- `docs/specs/features/onboarding/requirements.md`: mục `US-004` **đã trỏ sang spec này ở v1.1** (onboarding v7.4).
- `docs/knowledge/features/profile.md` + `index.md` + `log.md`: viết bằng `il-okf-writer` khi flow đã được kiểm chứng; chưa có gì để ghi.
- Sửa nhỏ ngoài feature: validate độ dài pháp danh ở đăng ký — **đã làm ở v1.1** (onboarding v7.4).
- Ba marker còn mở ở v1.0 **đã chốt ở v1.2**: xoá file cũ lỗi (chấp nhận rác, `TD-002`), mục tiêu thời gian phản hồi (không đặt), storage dev (emulator, `TD-001`).
