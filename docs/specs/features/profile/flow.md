# Flow — profile

Version: 1.5
Last updated: 2026-10-11
Status: implemented — mirror 1:1 của `design.md` v1.4 (không đổi bước nào so với v1.0).
Module: `profile`

> See `changelog.md` in this folder for the version history of this spec.

## Sequence — Đọc và sửa hồ sơ (GET / PATCH /v1/users/me)

Vẽ lại Step P-1 → P-7 của `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant L as userRateLimit
    participant H as ProfileHandler<br/>(delivery/http)
    participant U as ProfileUsecase
    participant R as ProfileRepository
    participant S as AvatarStorage
    participant DB as Postgres

    C->>M: GET/PATCH /v1/users/me (Bearer access token)
    alt token hợp lệ
        M->>L: userID trong context
    else thiếu / hết hạn / sai loại token
        M--xC: 401 UNAUTHORIZED
    end

    alt còn lượt
        L->>H: tiếp tục
    else vượt 10 request/phút/tài khoản
        L--xC: 429 RATE_LIMITED + Retry-After
    end

    opt PATCH
        H->>H: DecodeJSONBodyStrict (email/phone_number → field lạ)
        H->>U: Update(ctx, userID, patch)
        U->>U: trim + NFC full_name (null/blank/quá dài → lỗi), trim + NFC dharma_name, đếm ký tự, blank/null → xoá
        alt không hợp lệ hoặc có field lạ
            U--xC: 400 INVALID_FULL_NAME / INVALID_DHARMA_NAME / VALIDATION_ERROR + details.field
        end
        opt patch có ordination_temple_id khác rỗng
            U->>R: TempleExists(id)
            R->>DB: SELECT 1 FROM temples WHERE id
            alt chùa không có
                U--xC: 400 INVALID_TEMPLE (không ghi gì)
            end
        end
        U->>R: Update(userID, patch)
        R->>DB: UPDATE users SET (chỉ cột có trong patch), updated_at WHERE id
        Note over R,DB: một câu UPDATE, nguyên tử, lần ghi sau thắng
    end

    H->>R: Get(userID)
    R->>DB: SELECT users LEFT JOIN temples WHERE users.id
    alt tài khoản không còn
        R--xC: 401 UNAUTHORIZED
    end

    opt có avatar_object
        H->>S: SignRead(key, 15 phút)
    end
    H-->>C: 200 Profile (key vắng mặt khi chưa có giá trị)
```

## Sequence — Xin quyền upload avatar (POST /v1/users/me/avatar/upload-url)

Vẽ lại Step U-1 → U-6 của `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant L as userRateLimit
    participant H as ProfileHandler
    participant U as ProfileUsecase
    participant S as AvatarStorage
    participant G as Storage (GCS)

    C->>M: POST /v1/users/me/avatar/upload-url {content_type, size_bytes}
    alt token hợp lệ và còn lượt
        M->>L: userID
        L->>H: tiếp tục
    else 401 hoặc 429
        M--xC: 401 / 429
    end

    H->>U: RequestAvatarUpload(userID, req)
    U->>U: content_type ∈ cho phép, 0 < size_bytes ≤ 5 MB
    alt không hợp lệ
        U--xC: 400 VALIDATION_ERROR (content_type | size_bytes)
    end

    U->>S: SignUpload(tmpKey, content_type, 5 MB, 5 phút)
    Note over U,S: không ghi DB, không giữ trạng thái,<br/>tmpKey = tmp/{userID}/{upload_id}
    S-->>U: url + headers
    U-->>C: 200 AvatarUploadTicket {upload_id, upload_url, method, headers, expires_at}

    C->>G: PUT upload_url (đúng Content-Type, ≤ 5 MB, trước expires_at)
    alt hợp lệ
        G-->>C: 200 (object ở thư mục tạm)
    else hết hạn / sai Content-Type / quá độ dài
        G--xC: từ chối, avatar không đổi
    end
```

## Sequence — Xác nhận avatar (POST /v1/users/me/avatar/confirm)

Vẽ lại Step C-1 → C-8 của `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware + userRateLimit
    participant H as ProfileHandler
    participant U as ProfileUsecase
    participant R as ProfileRepository
    participant S as AvatarStorage
    participant DB as Postgres

    C->>M: POST /v1/users/me/avatar/confirm {upload_id}
    alt 401 hoặc 429
        M--xC: 401 / 429
    end
    M->>H: userID trong context

    H->>U: ConfirmAvatar(userID, upload_id)
    U->>U: upload_id là UUID?
    alt không hợp lệ
        U--xC: 400 VALIDATION_ERROR (upload_id)
    end

    U->>S: Stat(tmpKey)
    alt không có object
        S--xC: 404 AVATAR_UPLOAD_NOT_FOUND
    end

    U->>U: content type cho phép và size ≤ 5 MB?
    Note over U: chỉ đọc metadata đã ghi nhận,<br/>KHÔNG kiểm tra nội dung file
    alt không hợp lệ
        U--xC: 400 INVALID_AVATAR (avatar không đổi)
    end

    U->>S: Copy(tmpKey → avatars/{userID}/{uuid}.{ext})
    alt copy lỗi
        S--xC: 500 (DB chưa đổi, avatar cũ còn)
    end
    Note over U,S: object tạm giữ nguyên (dọn tạm hoãn)

    rect rgb(235, 240, 255)
        Note over U,DB: MỘT transaction
        U->>R: SwapAvatar(userID, newKey)
        R->>DB: SELECT avatar_object FROM users WHERE id FOR UPDATE
        R->>DB: UPDATE users SET avatar_object = newKey, updated_at
        R-->>U: oldKey (có thể rỗng)
    end

    opt có oldKey
        U->>S: Delete(oldKey)
        Note over U,S: best-effort, lỗi chỉ ghi log, vẫn 200<br/>chấp nhận file cũ thành rác (TD-002)
    end

    H->>R: Get(userID)
    H->>S: SignRead(newKey, 15 phút)
    H-->>C: 200 Profile (avatar_url mới)
```

## Sequence — Gỡ avatar (DELETE /v1/users/me/avatar)

Vẽ lại Step D-1 → D-4 của `design.md`.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware + userRateLimit
    participant H as ProfileHandler
    participant U as ProfileUsecase
    participant R as ProfileRepository
    participant S as AvatarStorage
    participant DB as Postgres

    C->>M: DELETE /v1/users/me/avatar
    alt 401 hoặc 429
        M--xC: 401 / 429
    end
    M->>H: userID trong context

    H->>U: RemoveAvatar(userID)
    rect rgb(235, 240, 255)
        Note over U,DB: MỘT transaction
        U->>R: ClearAvatar(userID)
        R->>DB: SELECT avatar_object FROM users WHERE id FOR UPDATE
        R->>DB: UPDATE users SET avatar_object = NULL, updated_at WHERE id AND avatar_object IS NOT NULL
        R-->>U: oldKey (rỗng nếu chưa có ảnh)
    end

    opt có oldKey
        U->>S: Delete(oldKey)
        Note over U,S: best-effort, lỗi chỉ ghi log, vẫn 200<br/>chấp nhận file cũ thành rác (TD-002)
    end

    H->>R: Get(userID)
    H-->>C: 200 Profile (không có avatar_url, chưa có ảnh vẫn 200)
```
