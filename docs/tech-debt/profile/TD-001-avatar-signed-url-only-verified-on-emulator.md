---
id: TD-001
feature: profile
title: "Avatar signed URLs are only exercised against a storage emulator / Signed URL avatar chỉ được chạy thử trên emulator"
severity: medium
status: open
created: 2026-10-11
owner: ducnguyen-wee
code: services/appService/storage/avatar_storage.go (SignUpload, SignRead), services/appService/config/config.dev.yaml (profile.bucket), services/appService/e2e/profile_e2e_test.go
---

# TD-001 — Avatar signed URLs are only exercised against a storage emulator / Signed URL avatar chỉ được chạy thử trên emulator

## 1. Debt / Nợ gì

**EN** — Local development and E2E tests for the avatar flow run against a storage emulator (such as `fake-gcs-server`), not a real GCS bucket. An emulator does not sign URLs the way GCS does: it does not enforce V4 signature checks, the signed `Content-Type` header, the `x-goog-content-length-range` limit or URL expiry. So a green E2E run does **not** prove that the real bucket accepts the upload ticket or rejects an expired, mistyped or oversized one.

**VI** — Dev local và E2E của luồng avatar chạy trên emulator storage (như `fake-gcs-server`), không phải bucket GCS thật. Emulator không ký URL giống GCS: không kiểm tra chữ ký V4, header `Content-Type` đã ký, giới hạn `x-goog-content-length-range` hay hạn của URL. Vì vậy E2E xanh **không** chứng minh bucket thật nhận vé upload, hay từ chối vé hết hạn, sai loại, quá dung lượng.

## 2. Why accepted / Vì sao chấp nhận

**EN** — Decided by the user on 2026-10-11: use an emulator for dev and test so no developer needs bucket credentials, and verify against the real bucket at deploy time. The text routes (read profile, edit dharma name and temple) do not depend on storage at all.

**VI** — Anh Đức quyết định ngày 2026-10-11: dùng emulator cho dev và test để dev không cần credential bucket, và kiểm tra trên bucket thật khi deploy. Các route text (xem hồ sơ, sửa pháp danh và chùa) không phụ thuộc storage.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — The first real upload may fail in production: a Cloud Run service account without `iam.serviceAccounts.signBlob` or a bucket missing CORS for a web client, or a header the client must send but the emulator never required. Worse, limits (5 MB, content type, 5-minute expiry) could be silently unenforced by the real signature if the headers are built wrongly, and the emulator would not show it.

**VI** — Lần upload thật đầu tiên có thể lỗi trên production: service account Cloud Run thiếu quyền `iam.serviceAccounts.signBlob`, bucket thiếu CORS cho client web, hoặc một header client bắt buộc gửi mà emulator không đòi. Tệ hơn, các giới hạn (5 MB, loại file, hạn 5 phút) có thể không được áp thật nếu header ký sai, và emulator không để lộ điều đó.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Before the first deploy that exposes the avatar routes, run the ticket → PUT → confirm flow once against the real bucket with the deploy service account / Trước lần deploy đầu có route avatar, chạy một lần ticket → PUT → confirm trên bucket thật với service account của deploy
- [ ] Verify a PUT is refused with the wrong `Content-Type`, after expiry, and above 5 MB (header `x-goog-content-length-range`) / Kiểm tra PUT bị từ chối khi sai `Content-Type`, quá hạn và quá 5 MB
- [ ] Confirm the service account can sign (`signBlob`) and the bucket has the CORS rules the web client needs / Xác nhận service account ký được và bucket có CORS cho client web
- [ ] Confirm the read URL expires after 15 minutes / Xác nhận URL đọc hết hạn sau 15 phút
- [ ] Deployed config leaves `storage.emulator_host` empty (it switches URL signing off) and the service account has `iam.serviceAccounts.signBlob` / Cấu hình deploy để trống `storage.emulator_host` (nó tắt việc ký URL) và service account có quyền `signBlob`
- [ ] Record the result and set `status: resolved` / Ghi kết quả và đổi `status: resolved`

## 5. Estimate / Ước lượng

**EN** — About half a day once the deploy bucket and service account exist.

**VI** — Khoảng nửa ngày khi đã có bucket và service account của môi trường deploy.
