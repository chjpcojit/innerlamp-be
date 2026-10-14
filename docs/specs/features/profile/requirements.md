# Requirements — profile

Version: 1.5
Last updated: 2026-10-11
Status: implemented (v1.5) — **v1.5 (2026-10-11): `full_name` đọc và sửa được qua `/me` (AC30–AC31, đã implement + test)**; AC1–AC29 có code. Các AC của luồng upload avatar (AC18–AC25) đã được kiểm chứng đầu-cuối trên emulator với URL không ký; AC21 (storage từ chối PUT quá hạn/sai loại/quá dung lượng) và phần ép giới hạn ở phía storage chưa được kiểm chứng, chỉ có thể làm trên bucket thật (`TD-001`). Nguồn nghiệp vụ [`SRS.md`](SRS.md) v0.3.
Module: `profile`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code) cho tính năng **Hồ sơ cá nhân**: người dùng đã đăng nhập xem hồ sơ của chính mình, sửa pháp danh và chùa quy y, đặt/đổi/gỡ ảnh đại diện. Nguồn nghiệp vụ là [`SRS.md`](SRS.md) v0.3 (IEEE 29148, 22 câu đã chốt với anh Đức ngày 2026-10-11); các AC dưới đây bám theo `FR-001`…`FR-009` và `NFR-001`…`NFR-003` của SRS.

Spec này **thay thế** mục `US-004` (mới ở mức user story, chưa design) trong `docs/specs/features/onboarding/requirements.md`. Mục cũ vẫn gọi nơi quy y là text; từ onboarding v7.0 nó là `ordination_temple_id`.

## Quyết định nền tảng (đã chốt, 2026-10-11)

- **Chỉ "của tôi".** Mọi route nằm dưới `/v1/users/me`, chủ sở hữu luôn là `user_id` trong access token; không có route nào nhận user id từ client và không có route xem hồ sơ người khác.
- **Ba trường sửa được:** pháp danh, chùa quy y (`ordination_temple_id`), ảnh đại diện. Email và số điện thoại chỉ đọc; request có `email`/`phone_number` bị từ chối cả request.
- **Cập nhật từng phần.** Trường vắng mặt = giữ nguyên; trường gửi `null` hoặc rỗng/toàn dấu cách = xoá. Một request hợp lệ hoàn toàn hoặc không đổi gì.
- **Hai đầu nhập–xuất của pháp danh dùng chung luật với đăng ký:** trim, rỗng lưu `NULL`, tối đa 255 **ký tự** (không phải byte).
- **Chùa chỉ chọn được từ danh sách chùa của catalogue** (`GET /v1/temples/options`); id lạ là `400 INVALID_TEMPLE`, cùng mã với đăng ký.
- **Ảnh đại diện đi thẳng lên storage bằng signed URL.** Backend không nhận byte ảnh. File vào **thư mục tạm** trước, khi client xác nhận thì server **copy sang thư mục chính**. JPEG/PNG/WebP, tối đa 5 MB. Server **không** kiểm tra nội dung file và **không** kiểm duyệt (rủi ro đã chấp nhận).
- **Đọc ảnh qua signed URL ngắn hạn** (15 phút), không có URL công khai cố định. Quyền upload sống 5 phút. Cả hai là giá trị config.
- **Hai thiết bị sửa cùng lúc: lần ghi sau thắng**, không có kiểm tra phiên bản.
- **Giới hạn tần suất:** 10 request/phút/tài khoản, tính mọi thao tác của nhóm này kể cả đọc.
- **Dọn thư mục tạm: hoãn có chủ đích** ("anh sẽ tính sau"). Xem mục Pending.

---

## US-PROFILE-01: Xem hồ sơ của tôi

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem thông tin hồ sơ của chính mình**,
> Để **biết hệ thống đang lưu gì về tôi và quyết định có cần chỉnh sửa hay không**.

## Acceptance Criteria (EARS notation)

1. WHEN a `GET /v1/users/me` request arrives with a valid access token, THE SYSTEM SHALL return `200` with the profile of the token's user: `email`, `email_verified`, `phone_number`, `full_name`, `dharma_name`, `ordination_temple` (`id`, `name`) and `avatar_url`. (FR-001)
2. WHEN the user has no full name (tài khoản tạo trước v7.5 của đăng ký), no dharma name, no ordination temple or no avatar, THE SYSTEM SHALL omit that key from `data` (không trả `null`, không trả chuỗi rỗng). `email`, `email_verified` and `phone_number` are always present. (FR-001)
3. WHEN the user's email has never been verified (`users.email_verified_at IS NULL`), THE SYSTEM SHALL return `email_verified: false`; otherwise `true`. Hiện mọi tài khoản đều `false` vì chưa có luồng xác thực email. (FR-001)
4. THE SYSTEM SHALL return `phone_number` in full to its owner, unmasked. (FR-001)
5. WHEN the user has an avatar, THE SYSTEM SHALL return `avatar_url` as a signed read address valid for `profile.avatar_read_url_ttl` (dev: 15 minutes) together with `avatar_url_expires_at` (RFC 3339 UTC); THE SYSTEM SHALL NOT return a permanent public address, and a later request SHALL return a fresh address. (FR-008)
6. THE SYSTEM SHALL NOT return `password_hash`, the storage object key, `created_at`/`updated_at` or any other user's data in any profile response. (FR-001)
7. WHEN the request carries no valid access token (thiếu, hết hạn, sai chữ ký, hoặc refresh token dùng làm bearer), THE SYSTEM SHALL return `401 UNAUTHORIZED` and SHALL NOT read or return anything. (FR-009)

## US-PROFILE-02: Sửa pháp danh và chùa quy y

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **cập nhật pháp danh và nơi quy y/xuất gia của mình, hoặc gỡ chúng đi**,
> Để **hồ sơ tu tập phản ánh đúng thông tin cá nhân, đặc biệt khi tôi mới quy y sau khi đã tạo tài khoản**.

## Acceptance Criteria (EARS notation)

8. WHEN a `PATCH /v1/users/me` request arrives with a valid access token and a body containing `full_name`, `dharma_name` and/or `ordination_temple_id`, THE SYSTEM SHALL update only the fields present, and SHALL return `200` with the full profile as in AC1–AC6, reflecting the change. (FR-002, FR-003, FR-004)
9. WHEN `dharma_name` is a string, THE SYSTEM SHALL trim leading and trailing whitespace, normalise it to NFC and store the result; WHEN the result is longer than `profile.dharma_name_max_chars` (dev: 255) characters, THE SYSTEM SHALL return `400 INVALID_DHARMA_NAME` with `details.field = "dharma_name"` (cùng mã và cùng luật với đăng ký) and SHALL NOT change anything. Độ dài đếm theo ký tự, không theo byte; chữ có dấu tính một ký tự. (FR-002)
10. WHEN `dharma_name` is `null`, an empty string or only whitespace, THE SYSTEM SHALL clear the stored dharma name (lưu `NULL`, không lưu chuỗi rỗng) and SHALL NOT return the key. (FR-002, FR-004)
11. WHEN `ordination_temple_id` is a non-empty string that is not the id of a temple in the catalogue (`temples.id`), THE SYSTEM SHALL return `400 INVALID_TEMPLE` with `details.field = "ordination_temple_id"` and SHALL NOT change any field of the same request. (FR-003, FR-004)
12. WHEN `ordination_temple_id` is `null`, an empty string or only whitespace, THE SYSTEM SHALL clear the stored ordination temple. (FR-003, FR-004)
13. WHEN a field is absent from the body, THE SYSTEM SHALL leave the stored value unchanged. WHEN the body is `{}`, THE SYSTEM SHALL return `200` with the unchanged profile (không lỗi). (FR-004)
14. WHEN any field of the request is invalid, THE SYSTEM SHALL apply none of the request's changes (tất cả hoặc không gì). (FR-004)
15. WHEN the body contains `email`, `phone_number` or any other field not listed in AC8 (kể cả `id`, `avatar_url`, `email_verified`), THE SYSTEM SHALL reject the whole request with `400 VALIDATION_ERROR` and SHALL NOT apply the other fields; WHEN the body is empty, is not valid JSON, or has the wrong `Content-Type`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (cùng cơ chế `DecodeJSONBodyStrict` như các route khác). (FR-005)
16. WHEN two requests of the same user are processed concurrently, THE SYSTEM SHALL let the one written last win and SHALL NOT reject either as out of date. (FR-004)
17. WHEN the request carries no valid access token, THE SYSTEM SHALL return `401 UNAUTHORIZED` and SHALL NOT change anything. (FR-009)
30. WHEN `full_name` is a string, THE SYSTEM SHALL trim leading and trailing whitespace, normalise it to NFC and store the result; WHEN the result is empty or longer than `profile.full_name_max_chars` (dev: 300) characters, THE SYSTEM SHALL return `400 INVALID_FULL_NAME` with `details.field = "full_name"` and SHALL apply none of the request's changes (AC14). Cùng luật và cùng mã lỗi với đăng ký (`docs/specs/features/onboarding/`, AC21). (Mới v1.5. Đánh số 30–31 để giữ AC1–AC29 ổn định.)
31. WHEN `full_name` is `null`, THE SYSTEM SHALL return `400 INVALID_FULL_NAME` and SHALL NOT clear the stored value: họ tên là trường bắt buộc, sửa được nhưng không xoá được (khác `dharma_name`, AC10). (Mới v1.5.)

## US-PROFILE-03: Đặt, đổi và gỡ ảnh đại diện

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **tải lên, thay hoặc gỡ ảnh đại diện của mình**,
> Để **ảnh của tôi hiển thị đúng ở mọi nơi trong app**.

## Acceptance Criteria (EARS notation)

18. WHEN a `POST /v1/users/me/avatar/upload-url` request arrives with a valid access token and a body `{content_type, size_bytes}`, THE SYSTEM SHALL return `200` with `upload_id`, `upload_url`, `method` (`PUT`), `headers` (các header client phải gửi nguyên văn khi PUT) and `expires_at`; the URL SHALL allow exactly one object, owned by the token's user, in the **temporary folder**, valid for `profile.upload_url_ttl` (dev: 5 minutes). (FR-006)
19. WHEN `content_type` is not one of `image/jpeg`, `image/png`, `image/webp`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "content_type"` and SHALL NOT issue a URL. (FR-006)
20. WHEN `size_bytes` is not a positive integer, or is greater than `profile.avatar_max_bytes` (dev: 5 242 880), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "size_bytes"` and SHALL NOT issue a URL; a value equal to the maximum SHALL be accepted. (FR-006)
21. WHEN the client PUTs to `upload_url` after `expires_at`, or with a different `Content-Type` than the one it asked for, THE SYSTEM (storage) SHALL refuse the upload; the avatar SHALL be unchanged. (FR-006)
22. WHEN a `POST /v1/users/me/avatar/confirm` request arrives with a valid access token and a body `{upload_id}` of an object that exists in the user's temporary folder, whose stored content type is allowed and whose stored size is within the limit, THE SYSTEM SHALL copy it to the **permanent folder** under a new key, set it as the user's avatar and return `200` with the full profile (AC1–AC6) carrying the new `avatar_url`. (FR-006)
23. WHEN `upload_id` is not in the form this endpoint issues, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "upload_id"`; WHEN the user's temporary folder holds no object for it (chưa upload, hoặc id của người khác), THE SYSTEM SHALL return `404 AVATAR_UPLOAD_NOT_FOUND` and SHALL NOT change the avatar. (FR-006)
24. WHEN the stored object's content type is not allowed or its size exceeds the limit, THE SYSTEM SHALL return `400 INVALID_AVATAR` and SHALL NOT change the avatar. THE SYSTEM SHALL check only the content type and size recorded by storage and SHALL NOT inspect the file's content. (FR-006)
25. WHEN the user already has an avatar and a new one is confirmed, THE SYSTEM SHALL make the new image the avatar, and AFTER the user record has been updated SHALL delete the previous image from storage. WHEN a replacement is rejected, THE SYSTEM SHALL leave the previous avatar and its image untouched. (FR-007)
26. WHEN `DELETE /v1/users/me/avatar` arrives with a valid access token, THE SYSTEM SHALL clear the user's avatar, delete the stored image, and return `200` with the full profile without `avatar_url`; WHEN the user has no avatar, THE SYSTEM SHALL return `200` with the unchanged profile (không lỗi). (FR-008)
27. WHEN the request to any avatar route carries no valid access token, THE SYSTEM SHALL return `401 UNAUTHORIZED` and SHALL NOT issue a URL, copy or delete anything. (FR-009)

## NFR-003: Giới hạn tần suất

28. WHEN one account makes more than `profile.rate_limit.max` (dev: 10) requests to the routes of this feature within `profile.rate_limit.window` (dev: 1 minute), THE SYSTEM SHALL return `429 RATE_LIMITED` with the `Retry-After` header and `details.retry_after_seconds`, and SHALL NOT process the request. Reads, edits, upload-URL requests, confirms and removals all count. (NFR-003)
29. THE SYSTEM SHALL count requests per account (user id of the token), not per IP, so that one account's use SHALL NOT limit another's. WHEN the request has no valid access token, THE SYSTEM SHALL return `401` (xác thực chạy trước bộ giới hạn). (NFR-003)

## API contract (confirmed surface for this feature)

> Tên route do kỹ thuật chọn (SRS §3.1.3 giao quyền này cho `design.md`/OpenAPI). Đã đưa vào `docs/specs/api/openapi.yaml` (tag `Profile`, 5 route, schema `Profile`/`ProfileUpdateRequest`/`AvatarUploadRequest`/`AvatarUploadTicket`/`AvatarConfirmRequest`) và bản sao `web/docs/public/openapi.yaml`. Envelope `{"success", "data"}` / `{"success": false, "error", "details"?}` như mọi route. Đã đăng ký trong `cmd/main.go` (`registerProfileRoutes`).

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| GET | `/v1/users/me` | `authMiddleware` (access token) | `200` `Profile` | 401, 429 `RATE_LIMITED`, 500 |
| PATCH | `/v1/users/me` | `authMiddleware` (access token) | `200` `Profile` | 400 `INVALID_FULL_NAME`, 400 `INVALID_DHARMA_NAME`, 400 `INVALID_TEMPLE`, 400 `VALIDATION_ERROR` (body/field lạ), 401, 413, 429, 500 |
| POST | `/v1/users/me/avatar/upload-url` | `authMiddleware` (access token) | `200` `AvatarUploadTicket` | 400 `VALIDATION_ERROR` (`content_type`, `size_bytes`), 401, 413, 429, 500 |
| POST | `/v1/users/me/avatar/confirm` | `authMiddleware` (access token) | `200` `Profile` | 400 `VALIDATION_ERROR` (`upload_id`), 400 `INVALID_AVATAR`, 401, 404 `AVATAR_UPLOAD_NOT_FOUND`, 413, 429, 500 |
| DELETE | `/v1/users/me/avatar` | `authMiddleware` (access token) | `200` `Profile` | 401, 429, 500 |

**Response `Profile`** (key vắng mặt khi chưa có giá trị):

```json
{
  "success": true,
  "data": {
    "email": "an@example.com",
    "email_verified": false,
    "phone_number": "0901234567",
    "full_name": "Nguyễn Văn An",
    "dharma_name": "Thích Minh An",
    "ordination_temple": { "id": "temple_north_011", "name": "Chùa Bích Động" },
    "avatar_url": "https://storage.googleapis.com/…&X-Goog-Expires=900…",
    "avatar_url_expires_at": "2026-10-11T12:15:00Z"
  }
}
```

**Request `PATCH /v1/users/me`** — vắng mặt = giữ nguyên, `null`/rỗng = xoá:

```json
{ "full_name": "Nguyễn Văn An", "dharma_name": "Thanh Liên", "ordination_temple_id": null }
```

**Request `POST /v1/users/me/avatar/upload-url`**: `{ "content_type": "image/jpeg", "size_bytes": 2097152 }`.

**Response `AvatarUploadTicket`**: `upload_id`, `upload_url`, `method` (`PUT` trên bucket thật; `POST` trên emulator dev, client phải dùng giá trị này), `headers` (`{"Content-Type": "image/jpeg"}`), `expires_at`.

**Request `POST /v1/users/me/avatar/confirm`**: `{ "upload_id": "0b9c…" }`.

## Out of scope (this feature boundary, not tracked here)

- Đổi email hoặc số điện thoại (cần luồng xác thực lại riêng).
- Đổi mật khẩu khi đang đăng nhập (khác "quên mật khẩu" ở `password-reset`).
- Xoá tài khoản (và việc xoá ảnh đại diện theo tài khoản).
- Xem hồ sơ của người dùng khác.
- Xác thực email (hồ sơ chỉ báo trạng thái `email_verified`); nút "Xác thực email" của US-004 cũ thuộc feature đó.
- Thêm trường hồ sơ mới ngoài ba trường đã chốt.
- Kiểm duyệt và kiểm tra nội dung ảnh (đã quyết định không làm ở v1).
- Resize, tạo thumbnail, xử lý ảnh phía server.

## Pending (not yet implemented)

- **Đã implement** (xem `tasks.md`). Còn mở: kiểm tra trên bucket thật trước deploy, gồm chữ ký, AC21 và giới hạn do storage ép (`TD-001`); dọn file mồ côi (`TD-002`).
- **Hoãn có chủ đích:** dọn thư mục tạm. Upload không được xác nhận nằm lại trong thư mục tạm, dung lượng tăng dần cho đến khi có cơ chế dọn. Cần xem lại **trước go-live** (`tasks.md` Phase 6).
- Đồng bộ concept OKF `docs/knowledge/features/profile.md` (`il-okf-writer`, sau khi flow được kiểm chứng) — `tasks.md` Phase 6.
- Sinh lại `docs/specs/db/erd.svg` sau khi thêm cột avatar — `tasks.md` Phase 3.
- Mục `US-004` trong `docs/specs/features/onboarding/requirements.md` cần trỏ sang spec này (việc của người duyệt spec, chưa sửa ở đây).

Không còn marker `⚠️ Not confirmed`. Ba câu cuối đã chốt ngày 2026-10-11 (xem "Open questions (resolved)"); hai hệ quả được ghi thành nợ kỹ thuật:

- [`docs/tech-debt/profile/TD-001`](../../../tech-debt/profile/TD-001-avatar-signed-url-only-verified-on-emulator.md): signed URL chỉ được chạy thử trên emulator, **phải kiểm tra trên bucket thật khi deploy**.
- [`docs/tech-debt/profile/TD-002`](../../../tech-debt/profile/TD-002-avatar-orphan-objects-not-cleaned.md): object mồ côi (thư mục tạm, avatar cũ xoá lỗi) không được dọn.

## Ghi chú kỹ thuật liên quan

- **`users` hiện chưa có cột ảnh.** Cần migration `0029` thêm `avatar_object VARCHAR(255)` (khoá object trong thư mục chính, `NULL` = chưa có). Lưu **khoá object**, không lưu URL: URL đọc là signed URL sinh lúc trả response và hết hạn sau 15 phút, nên không thể là dữ liệu lưu.
- **Chưa có storage trong `cmd/main.go`.** `pkg/storage` (interface `Client`: `SignedURL`, `Copy`, `Delete`, `GetObjectInfo`) và driver `pkg/storage/gcs` đã có, chưa được wiring. Dựng client qua factory, gọi `Connect(ctx)` lúc khởi động để lỗi cấu hình/credential hiện ngay (đúng quy ước vòng đời kết nối của repo), không phải lúc có request.
- **Mọi giới hạn nằm ở config, không là hằng trong domain/usecase:** `profile.dharma_name_max_chars`, `profile.avatar_max_bytes`, `profile.avatar_content_types`, `profile.upload_url_ttl`, `profile.avatar_read_url_ttl`, `profile.bucket`, `profile.avatar_tmp_prefix`, `profile.avatar_prefix`, `profile.rate_limit.{max,window}` (theo nguyên tắc giá trị không nhạy cảm đổi theo môi trường ở `config/`). `dharma_name_max_chars` chỉ có thể **hạ xuống** so với cột `VARCHAR(255)`.
- **Luật pháp danh dùng chung với đăng ký (đã làm ở v1.1).** Trước đây `dharma_name` không được validate độ dài khi đăng ký (quá 255 ký tự là `500`, khoảng trống ghi ở `docs/HANDOFF.md`). Từ v1.1 đăng ký đã trả `400 INVALID_DHARMA_NAME` (onboarding v7.4, AC20) với `normalizeDharmaName` và `ProfileConfig.DharmaNameMaxChars`; `ProfileUsecase` gọi lại đúng hàm và giới hạn đó nên một trường không có hai luật hay hai mã lỗi.
- **CORS cho `PATCH`.** `pkg/middleware.CORS` hard-code danh sách method; từ v1.3 gồm `PATCH`, nếu không preflight của `PATCH /v1/users/me` từ web bị chặn. Đổi này chung cho cả hai service dùng package đó.
- **Quan hệ với `lamp`:** đèn chụp pháp danh lúc thắp, nên đổi pháp danh ở đây **không** đổi tên trên đèn đã thắp (`lamp` AC12). Không cần sửa gì ở `lamp`.
- **Quyền riêng tư:** email, số điện thoại, pháp danh là PII — không ghi vào log (chỉ `user_id`); response profile gửi `Cache-Control: no-store` (cùng cơ chế `NoStore` của các route trả token), vì có PII và signed URL.
- **Giới hạn tần suất theo tài khoản** cần một bộ giới hạn khoá theo `user_id` đặt **sau** `authMiddleware`; bộ hiện có (`NewClientIPRateLimit`) khoá theo IP nên không dùng lại nguyên trạng. Khi một lần đổi avatar tốn khoảng 3 request (xin URL, confirm, đọc lại), 10/phút đủ cho một lần đổi avatar kèm một lần sửa; xem lại nếu client chạm trần trong dùng bình thường.

## Open questions (resolved)

- ~~Trường nào sửa được?~~ → Pháp danh, chùa quy y, ảnh đại diện; email và số điện thoại chỉ đọc. Resolved 2026-10-11.
- ~~Vắng mặt và xoá khác nhau thế nào?~~ → Vắng mặt = giữ nguyên; `null`/rỗng = xoá. Resolved 2026-10-11.
- ~~Đọc hồ sơ trả gì?~~ → Email + trạng thái xác thực, số điện thoại đầy đủ, pháp danh, chùa (id + tên), ảnh. Resolved 2026-10-11.
- ~~Luật pháp danh?~~ → Như đăng ký (tuỳ chọn, trim, tối đa 255 **ký tự**, rỗng lưu `NULL`). Resolved 2026-10-11.
- ~~Ảnh đến storage bằng cách nào?~~ → Signed upload URL, thẳng lên storage; thư mục tạm rồi copy sang thư mục chính khi xác nhận. Resolved 2026-10-11.
- ~~Giới hạn ảnh?~~ → JPEG/PNG/WebP, tối đa 5 MB; không kiểm tra nội dung file; không kiểm duyệt. Resolved 2026-10-11.
- ~~Gỡ ảnh và file cũ?~~ → Cho phép gỡ; file cũ bị xoá. Resolved 2026-10-11.
- ~~Chùa không tồn tại / ẩn?~~ → Từ chối cả request; chỉ chọn từ danh sách chùa của catalogue. Resolved 2026-10-11.
- ~~Sửa đồng thời?~~ → Lần ghi sau thắng. Resolved 2026-10-11.
- ~~Ảnh đọc công khai hay ký?~~ → Signed URL ngắn hạn, 15 phút. Resolved 2026-10-11.
- ~~TTL quyền upload?~~ → 5 phút, trong config. Resolved 2026-10-11.
- ~~Tên route?~~ → Do kỹ thuật chọn: `/v1/users/me` và `/v1/users/me/avatar/*`. Resolved 2026-10-11.
- ~~Giới hạn tần suất?~~ → 10 request/phút/tài khoản, tính mọi thao tác kể cả đọc. Resolved 2026-10-11.
- ~~Request có email/phone?~~ → Từ chối cả request. Resolved 2026-10-11.
- ~~Dọn thư mục tạm?~~ → Hoãn, anh Đức sẽ tính sau (nợ kỹ thuật `TD-002`). Resolved (deferred) 2026-10-11.
- ~~Xoá file ảnh cũ bị lỗi thì làm gì?~~ → Chấp nhận để lại file thành rác; chỉ ghi log, vẫn trả `200`, không thử lại (nợ kỹ thuật `TD-002`). Resolved 2026-10-11.
- ~~Mục tiêu thời gian phản hồi (NFR-001)?~~ → Không đặt mục tiêu ở giai đoạn này, chậm một chút được chấp nhận. Resolved 2026-10-11.
- ~~Storage cho dev/local?~~ → Dùng emulator; việc kiểm tra signed URL trên bucket thật làm khi deploy (nợ kỹ thuật `TD-001`). Resolved 2026-10-11.

Không còn câu hỏi mở. Hai nợ kỹ thuật `TD-001` và `TD-002` được theo dõi ở `docs/tech-debt/profile/`.
