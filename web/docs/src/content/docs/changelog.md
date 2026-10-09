---
title: Changelog
description: Lịch sử thay đổi của API.
---

## v1 (hiện tại)

34 endpoint appService đang có:

- **Auth**: `register/request-otp`, `register/verify-otp`, `resend-otp`, `login`, `refresh`, `logout`.
- **Attestation**: `attestation/challenge`, `assertion/challenge`, `attestation/verify`.
- **Chùa**: `temples/top`, `temples`, `temples/{id}`.
- **Phật và Bồ tát**: `deities/top`, `deities`, `deities/{id}`.
- **Bài viết Phật pháp**: `articles/top`, `articles`, `articles/{id}`.
- **Sổ giỗ chạp** (cần đăng nhập): `POST/GET /v1/memorials`, `GET/PUT/DELETE /v1/memorials/{id}`.
- **Kinh, kệ, chú**: `sutras`, `sutras/{id}`, `sutras/{id}/lines`, `deities/{id}/sutras`.
- **Ảnh đẹp Phật giáo**: `gallery/top`, `gallery`, `gallery/{id}`.
- **Pháp thoại**: `dharma-talks`, `dharma-talks/{id}`.
- **Lịch âm**: `calendar/image`.
- **Ảnh tĩnh** (dev): `static/images/{path}`.

### 2026-10-07

- Tài liệu: thêm hướng dẫn tích hợp cho Sổ giỗ chạp, Kinh/kệ/chú, Ảnh đẹp, Pháp thoại và Ảnh theo ngày; bổ sung mã lỗi mới vào bảng Xử lý lỗi. Trang Xác thực ghi rõ `/v1/memorials` cần token.

### 2026-09-30

- **Thay đổi không tương thích**: `GET /v1/bodhisattvas?page=N` (chỉ Bồ tát) được thay bằng `GET /v1/deities?page=N`, danh sách phân trang gồm cả Phật và Bồ tát (mỗi mục có `type`, `total` là số mục của cả hai loại; dữ liệu hiện tại `total = 18`). Route cũ bị xoá, không có alias. Lỗi và cách phân trang không đổi (`400 INVALID_PAGE`, trang vượt cuối trả `items: []`).
- Thêm 3 endpoint bài viết Phật pháp (công khai, chỉ đọc): `GET /v1/articles/top` (5 bài), `GET /v1/articles?page=N` (10 bài mỗi trang), `GET /v1/articles/{id}`. Lỗi mới: `404 ARTICLE_NOT_FOUND`; `page` sai vẫn là `400 INVALID_PAGE`. `published_date` và `image_url` là `null` khi chưa có (hiện `image_url` là `null` ở mọi bài).
- `GET /v1/temples/{id}`: thêm 3 field nội dung `introduction`, `architecture_heritage` (chuỗi) và `visiting_notes` (mảng chuỗi). Chùa chưa có nội dung trả `""` / `[]`, không trả `null`.
- Tài liệu: nhóm Attestation được viết lại theo cùng mẫu (bảng field, example cho từng verdict, mã lỗi theo endpoint, `413`). Sửa tên key trong `signals` của Android: đúng là `app_recognition_verdict`, `device_recognition_verdict`, `app_licensing_verdict` (tài liệu cũ ghi `app_recognition`). Hướng dẫn Attestation thêm định dạng `clientData` và bảng lỗi `401` của assertion.
- Tài liệu: nhóm Auth trong API Reference được viết lại cho ngang với nhóm danh mục: bảng field của request và response, example cho response thành công, mã lỗi liệt kê theo từng endpoint kèm example, và bổ sung `413 PAYLOAD_TOO_LARGE`. Hướng dẫn Đăng ký và đăng nhập thêm bảng field và bảng mã lỗi. Không đổi route hay field nào.
- Tài liệu: API Reference liệt kê đầy đủ field trong `items` của `top`/danh sách và response chi tiết cho cả chùa lẫn Phật/Bồ tát, kèm example lấy từ server thật. Sửa example `type` của Phật/Bồ tát thành `BUDDHA` / `BODHISATTVA`.

## Backoffice (`:8081`)

5 endpoint admin: `admin/auth/login`, `admin/auth/change-password`, `admin/me`, `admin/users`, `admin/users/{id}/reset-password`. Xem mục Admin trong API Reference.

### Chưa có

Đã có spec nhưng chưa có route, nên chưa được tài liệu hoá: hồ sơ và avatar, đặt lại mật khẩu, nhật ký và thực hành.

:::note
Khi thêm hoặc đổi endpoint, cập nhật `docs/specs/api/openapi.yaml` (nguồn duy nhất; `public/openapi.yaml` được chép từ đó khi chạy `npm run dev`/`build`) và ghi một mục vào đây.
:::
