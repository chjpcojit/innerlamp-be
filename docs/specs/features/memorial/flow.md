# Flow — memorial

Version: 4.0
Last updated: 2026-10-14
Status: sơ đồ tạo (bản v1) và xoá đã được implement; v4.0 đổi Step C-3 của sơ đồ tạo và thêm sơ đồ Sửa, Báo múi giờ, Job gửi nhắc (chưa implement; J-5/J-7 chờ feature notification); sơ đồ tải/thay ảnh ⏸ hoãn (Phase 6).
Module: `memorial`

> See `changelog.md` in this folder for the version history of this spec.

Bản vẽ của sáu section "Step-by-step Flow" trong `design.md` (cùng tên, cùng thứ tự), cộng một flowchart cho nhánh quyết định của job (J-4..J-7). Mỗi message trace 1:1 về một Step; không thêm/bỏ bước.

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
    H->>H: DecodeJSONBodyStrict — field cũ timezone/add_to_os_reminder bị từ chối (Step C-2)
    H->>U: Create(ctx, userID, input)
    U->>U: validate AC3–AC9 (lead 0/1/3/7, giờ 06/07/12/19:00) + dựng PlanFunc (now, notBefore = hôm nay) (Step C-2)
    alt validation lỗi
        U--xH: validation error
        H-->>C: 400 VALIDATION_ERROR {details.field}
    end
    U->>R: Create(ctx, m, maxPerUser=100, plan)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step C-3)
        R->>DB: BEGIN
        R->>DB: SELECT timezone FROM users WHERE id=$1 FOR UPDATE
        R->>DB: SELECT count(*) FROM memorials WHERE user_id=$1
        alt count >= 100
            R--xU: ErrMemorialLimitReached → ROLLBACK
        else còn chỗ
            opt có reminder
                R->>R: LoadLocation(timezone) → plan(loc) → next_reminder_at, next_anniversary_date
            end
            R->>DB: INSERT INTO memorials (..., next_reminder_at, next_anniversary_date) RETURNING id, created_at, updated_at
            R->>DB: COMMIT
            R-->>U: Memorial
        end
    end

    alt thành công
        U-->>H: Memorial
        H-->>C: 201 Memorial (photo_url null, không có next_*) (Step C-4)
    else đủ 100 mục
        H-->>C: 409 MEMORIAL_LIMIT_REACHED
    else lỗi DB / plan
        H-->>C: 500 INTERNAL_ERROR
    end
```

## Sequence — Sửa một mục (PUT /v1/memorials/{id}) — v4.0

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant M as authMiddleware
    participant H as MemorialHandler<br/>(delivery/http)
    participant U as MemorialUsecase
    participant R as MemorialRepository
    participant DB as Postgres

    C->>M: PUT /v1/memorials/{id} + Bearer
    alt token không hợp lệ
        M--xC: 401
    end
    M->>H: userID trong ctx (Step U-1)
    H->>H: DecodeJSONBodyStrict (Step U-2)
    H->>U: Update(ctx, userID, id, input)
    U->>U: validate AC3–AC9 + dựng PlanFunc (Step U-2)
    alt validation lỗi
        H-->>C: 400 VALIDATION_ERROR (row không đổi)
    end
    U->>R: Update(ctx, m, plan)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step U-3)
        R->>DB: BEGIN
        R->>DB: SELECT timezone FROM users WHERE id=$userID FOR SHARE
        opt có reminder
            R->>R: LoadLocation → plan(loc) (thời điểm kỳ này đã qua → kỳ sau)
        end
        R->>DB: UPDATE memorials SET …, next_reminder_at, next_anniversary_date, updated_at WHERE id AND user_id RETURNING …
        Note over R,DB: nếu job đang khoá dòng (J-3) thì chờ job commit, rồi ghi đè kế hoạch
        alt 0 dòng
            R--xU: ErrMemorialNotFound → ROLLBACK
            H-->>C: 404 MEMORIAL_NOT_FOUND
        else lỗi DB / plan
            R--xU: error → ROLLBACK
            H-->>C: 500 INTERNAL_ERROR
        else đã ghi
            R->>DB: COMMIT
            R-->>U: Memorial
        end
    end

    U-->>H: Memorial
    H-->>C: 200 Memorial (Step U-4)
```

## Sequence — Báo múi giờ tài khoản (PATCH /v1/users/me, field timezone) — v4.0

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant PH as ProfileHandler + ProfileUsecase<br/>(thuộc profile)
    participant PR as ProfileRepository<br/>(thuộc profile)
    participant U as MemorialUsecase<br/>(ReplanUser)
    participant R as MemorialRepository
    participant DB as Postgres

    C->>PH: PATCH /v1/users/me {timezone} + Bearer
    Note over C,PH: thuộc profile, ngoài scope feature này (Step T-1, T-2)
    PH->>PH: auth + DecodeJSONBodyStrict + LoadLocation(timezone), từ chối "" / "Local" (Step T-1)
    alt token / timezone không hợp lệ
        PH-->>C: 401 / 400 (mã do spec profile chốt)
    end
    PH->>PR: lưu users.timezone (Step T-2)
    PR->>DB: UPDATE users SET timezone=$2 … WHERE id=$1
    alt lỗi DB
        PH-->>C: 500 INTERNAL_ERROR
    end

    PH->>U: ReplanUser(ctx, userID) (Step T-3, luôn chạy khi body có timezone)
    U->>R: ReplanUser(ctx, userID, plan)
    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction (Step T-3)
        R->>DB: BEGIN
        R->>DB: SELECT timezone FROM users WHERE id=$1 FOR SHARE
        R->>DB: SELECT … FROM memorials WHERE user_id=$1 AND reminder_days_before IS NOT NULL FOR UPDATE
        loop mỗi mục có nhắc (≤ 100)
            R->>R: plan(loc)
            R->>DB: UPDATE memorials SET next_reminder_at, next_anniversary_date WHERE id=$
        end
        alt lỗi
            R--xU: error → ROLLBACK (kế hoạch cũ giữ nguyên)
            PH-->>C: 500 INTERNAL_ERROR (timezone đã lưu, gửi lại cùng giá trị để tính lại)
        else xong
            R->>DB: COMMIT
        end
    end
    U-->>PH: ok
    PH-->>C: 200 Profile
```

## Sequence — Job gửi nhắc giỗ hằng giờ (cmd/memorialreminders) — v4.0

```mermaid
sequenceDiagram
    autonumber
    participant CS as Cloud Scheduler
    participant J as Job<br/>(cmd/memorialreminders)
    participant U as MemorialReminderJob<br/>(usecase)
    participant R as MemorialReminderStore<br/>(repository)
    participant DB as Postgres
    participant N as Feature notification<br/>(inbox + ReminderPusher)

    CS->>J: chạy Cloud Run Job memorial-reminders ("0 * * * *")
    J->>J: nạp config/secrets, DB Connect, dựng dependency, now := time.Now() (Step J-1)
    alt lỗi config / DB
        J--xCS: exit ≠ 0 (retry an toàn)
    end
    J->>U: Run(ctx, now)

    loop tới khi hết mục chưa có kế hoạch (Step J-2)
        rect rgb(235, 240, 255)
            Note over R,DB: một transaction mỗi lô
            U->>R: PlanUnplanned(ctx, PlanBatch, plan)
            R->>DB: SELECT m.…, u.timezone … WHERE reminder_days_before IS NOT NULL AND next_reminder_at IS NULL LIMIT n FOR UPDATE OF m SKIP LOCKED
            R->>DB: UPDATE memorials SET next_* (PlanReminder, notBefore = hôm nay)
            alt plan lỗi một dòng
                R--xU: ROLLBACK lô, id vào failedIDs, đếm failed
            else xong lô
                R->>DB: COMMIT
            end
        end
    end

    loop tới khi hết mục đến hạn hoặc đủ MaxPerRun
        rect rgb(235, 240, 255)
            Note over R,DB: MỘT transaction cho MỘT mục (Step J-3 → J-6)
            U->>R: ProcessNextDue(ctx, now, failedIDs, handle)
            R->>DB: BEGIN → SELECT m.…, u.timezone … WHERE next_reminder_at <= now AND id ∉ failedIDs ORDER BY next_reminder_at LIMIT 1 FOR UPDATE OF m SKIP LOCKED (Step J-3)
            alt không còn dòng
                R->>DB: COMMIT
            else có dòng
                R->>U: handle(due) → ShouldDeliver + Next = PlanReminder(notBefore = next_anniversary_date + 1) (Step J-4)
                opt deliver = true
                    Note over R,N: ⚠️ Not confirmed: bảng inbox + hàm ghi tx-aware thuộc spec notification chưa có
                    R->>DB: INSERT INTO inbox (…, idempotency_key = memorial:id:date) ON CONFLICT DO NOTHING RETURNING id (Step J-5)
                end
                R->>DB: UPDATE memorials SET next_reminder_at, next_anniversary_date WHERE id (Step J-6)
                alt lỗi DB / plan
                    R--xU: ROLLBACK, id vào failedIDs, đếm failed
                else ổn
                    R->>DB: COMMIT
                    R-->>U: created? + notificationID
                end
            end
        end
        opt created = true (Step J-7)
            Note over U,N: thuộc feature notification (ADR-0001), ngoài scope feature này, ⚠️ Not confirmed: cổng ReminderPusher
            U->>N: Push(ctx, notificationID) — mọi token của user, công tắc tài khoản tắt → không push, inbox giữ nguyên
            N-->>U: ok / lỗi (chỉ log id)
        end
    end

    U->>DB: SELECT count(*) WHERE next_reminder_at < now − 1h (Step J-8)
    U-->>J: RunStats (claimed, sent, skipped_late, already_sent, failed, overdue_1h)
    J-->>CS: exit 0
```

## Flowchart — Quyết định cho một mục đến hạn (Step J-4 → J-7)

```mermaid
flowchart TB
    A[Đã khoá một mục đến hạn - J-3] --> B{Ngày giỗ đã hết<br/>theo múi giờ tài khoản? - J-4}
    B -- Có --> C[Không tạo lời nhắc<br/>đếm skipped_late]
    B -- Không --> D[INSERT inbox ON CONFLICT DO NOTHING - J-5]
    D --> E{Có id mới?}
    E -- Có --> F[created = true, đếm sent]
    E -- Không --> G[Đã gửi trước đó<br/>đếm already_sent, không push]
    C --> H[Cuộn kế hoạch sang kỳ sau - J-6]
    F --> H
    G --> H
    H --> I{COMMIT được?}
    I -- Không --> J[ROLLBACK, id vào failedIDs<br/>lượt sau thử lại]
    I -- Có --> K{created?}
    K -- Có --> L[Push qua feature notification<br/>best-effort - J-7]
    K -- Không --> M[Mục tiếp theo]
    L --> M
    J --> M
```

## Sequence — Tải/thay ảnh (PUT /v1/memorials/{id}/photo)

> ⏸ **HOÃN (Phase 6)** — mirror của flow cùng tên trong `design.md`; chưa implement.

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
    Note over R,DB: kế hoạch nhắc (next_*) mất cùng dòng, nếu job đang khoá dòng thì chờ job commit (AC36)
    alt 0 dòng
        R--xU: ErrMemorialNotFound
        H-->>C: 404 MEMORIAL_NOT_FOUND
    else lỗi DB
        R--xU: error
        H-->>C: 500 INTERNAL_ERROR
    else đã xoá
        R-->>U: photo_object_key (có thể NULL)
    end

    opt key khác NULL (Step D-3 — ⏸ hoãn tới Phase 6, v1 key luôn NULL)
        U->>S: Delete(key) best-effort
        Note over U,S: thất bại chỉ log, vẫn trả 200
    end

    U-->>H: ok
    H-->>C: 200 {"success": true} (Step D-4)
```
