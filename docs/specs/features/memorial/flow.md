# Flow — memorial

Version: 2.4
Last updated: 2026-10-07
Status: sơ đồ tạo và xoá đã được implement (v1, bước xoá-object ⏸ hoãn); sơ đồ tải/thay ảnh ⏸ hoãn (Phase 6).
Module: `memorial`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của ba section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự). Mỗi message trace 1:1 về một Step; không thêm/bỏ bước.

## Sequence — Thêm người vào sổ giỗ (POST /v1/memorials)

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as MemorialHandler<br/>(delivery/http)
    participant U as MemorialUsecase
    participant R as MemorialRepository
    participant DB as Postgres

    C->>M: POST /v1/memorials + Bearer access token
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx (Step C-1)
    H->>H: DecodeJSONBodyStrict (Step C-2)
    H->>U: Create(ctx, userID, input)
    U->>U: validate AC3–AC10: death_date đầy đủ 1900..hôm nay HOẶC ngày+tháng khi không rõ năm (AC6/6a/6b) (Step C-2)
    alt validation lỗi
        U--xH: validation error
        H-->>C: 400 VALIDATION_ERROR {details.field}
    end
    U->>R: Create(ctx, m, maxPerUser=100)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step C-3)
        R->>DB: BEGIN
        R->>DB: SELECT id FROM users WHERE id=$1 FOR UPDATE
        R->>DB: SELECT count(*) FROM memorials WHERE user_id=$1
        alt count >= 100
            R--xU: ErrMemorialLimitReached → ROLLBACK
        else còn chỗ
            R->>DB: INSERT INTO memorials (...) RETURNING id, created_at, updated_at
            R->>DB: COMMIT
            R-->>U: Memorial
        end
    end

    alt thành công
        U-->>H: Memorial
        H-->>C: 201 Memorial (photo_url null) (Step C-4)
    else đủ 100 mục
        H-->>C: 409 MEMORIAL_LIMIT_REACHED
    else lỗi DB
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Tải/thay ảnh (PUT /v1/memorials/{id}/photo)

> ⏸ **HOÃN (Phase 6)** — mirror của flow cùng tên trong `design.md`; v1 không implement.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as MemorialHandler<br/>(delivery/http)
    participant U as MemorialUsecase
    participant R as MemorialRepository
    participant DB as Postgres
    participant S as MemorialPhotoStore<br/>(pkg/storage → GCS)

    C->>H: PUT /v1/memorials/{id}/photo (multipart) + Bearer
    H->>U: SetPhoto(ctx, userID, id, openFile)
    U->>R: FindByIDAndUser(id, userID) (Step P-1)
    R->>DB: SELECT … WHERE id=$1 AND user_id=$2
    alt không có / của người khác
        R--xU: ErrMemorialNotFound
        H-->>C: 404 MEMORIAL_NOT_FOUND (body chưa đọc)
    end

    H->>H: MaxBytesReader 5MB + sniff content type (Step P-2)
    Note over H,S: ⚠️ Not confirmed: bucket/prefix riêng và quyền ký signed URL của Cloud Run SA (Step P-2)
    alt thiếu/rỗng part
        H-->>C: 400 VALIDATION_ERROR (details.field=photo)
    else > 5MB
        H-->>C: 413 PHOTO_TOO_LARGE
    else không phải jpeg/png/webp
        H-->>C: 415 PHOTO_UNSUPPORTED_TYPE
    end

    U->>S: Put(memorials/<userID>/<id>/<uuid>.<ext>) (Step P-3)
    alt ghi GCS lỗi
        S--xU: error
        H-->>C: 500 INTERNAL_ERROR (DB chưa đổi)
    end

    U->>R: SetPhotoKey(id, userID, newKey) (Step P-4)
    R->>DB: UPDATE memorials SET photo_object_key, updated_at … RETURNING old key
    alt lỗi DB hoặc 0 dòng
        R--xU: error / not found
        U->>S: Delete(newKey) best-effort
        H-->>C: 500 INTERNAL_ERROR / 404
    else thành công
        R-->>U: oldKey
    end

    opt oldKey khác rỗng (Step P-5)
        U->>S: Delete(oldKey) best-effort
        Note over U,S: thất bại chỉ log, không đổi kết quả
    end

    U->>S: SignedURL(newKey, 15m) (Step P-6)
    U-->>H: Memorial + photo_url
    H-->>C: 200 Memorial
```

## Sequence — Xoá một mục (DELETE /v1/memorials/{id})

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as MemorialHandler<br/>(delivery/http)
    participant U as MemorialUsecase
    participant R as MemorialRepository
    participant DB as Postgres
    participant S as MemorialPhotoStore<br/>(GCS)

    C->>M: DELETE /v1/memorials/{id} + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx (Step D-1)
    H->>U: Delete(ctx, userID, id)
    U->>R: Delete(id, userID) (Step D-2)
    R->>DB: DELETE FROM memorials WHERE id=$1 AND user_id=$2 RETURNING photo_object_key
    alt 0 dòng
        R--xU: ErrMemorialNotFound
        H-->>C: 404 MEMORIAL_NOT_FOUND
    else lỗi DB
        R--xU: error
        H-->>C: 500 INTERNAL_ERROR
    else đã xoá
        R-->>U: photo_object_key (có thể NULL)
    end

    opt key khác NULL (Step D-3 — ⏸ hoãn tới Phase 6; v1 key luôn NULL)
        U->>S: Delete(key) best-effort
        Note over U,S: thất bại chỉ log, vẫn trả 200
    end

    U-->>H: ok
    H-->>C: 200 {"success": true} (Step D-4)
```
