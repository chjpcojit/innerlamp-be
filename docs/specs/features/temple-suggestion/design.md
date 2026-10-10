# Design — temple-suggestion

Version: 2.2
Last updated: 2026-10-10
Status: not started — spec viết trước khi có code. Không còn marker `⚠️ Not confirmed`.
Module: `content` (appService) + `backoffice` (backofficeService)

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Một route công khai ở `appService` (ghi một dòng `temple_suggestions`, `pending`) và ba route có token ở `backofficeService` (liệt kê, xem, tạo chùa); route thứ tư — từ chối — thêm ở phase sau (Phase 6). Hai service dùng chung Postgres; migration `0022` sở hữu bảng mới. Route gửi đề xuất theo hình dạng của `/v1/auth/*` (public + `webAPIKey`); ba route duyệt theo hình dạng của `/v1/admin/users` (token access + `RequireActiveSession` + role). Phần duy nhất có nhiều bước ghi là `create-temple` — nằm ở Step-by-step Flow và `flow.md`.

## Layers & components

### appService (gửi đề xuất)

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple_suggestion.go` (not yet created) | `TempleSuggestion{ID, Name, Province, Address, Status, CreatedAt}`; hằng `SuggestionPending`/`SuggestionCreated`; `TempleSuggestionValidationError{Field, Message}`. Không JSON tag. |
| Domain / Repository interface | `domain/temple_suggestion.go` | `TempleSuggestionRepository.Create(ctx, s) (id, err)`. |
| Domain / Mock | `domain/mock_temple_suggestion_repository.go` (not yet created) | Mock tay cho test usecase. |
| Usecase | `usecase/temple_suggestion_usecase.go` (not yet created) | Trim, kiểm rỗng/độ dài (AC2–AC3), gọi repository. |
| Repository | `repository/postgres_temple_suggestion_repository.go` (not yet created) | `INSERT … RETURNING id`. |
| Delivery | `delivery/http/temple_suggestion_handler.go` (not yet created) | `DecodeJSONBodyStrict` → usecase → `201 {id, status}`; DTO ở đây. |
| Config | `config/config.go` (needs modification) | `temple_suggestion.rate_limit` (dev: 5 / giờ / IP). |
| Wiring | `cmd/main.go` (needs modification) | `mux.HandleFunc("POST /v1/temple-suggestions", webAPIKey.WrapFunc(...))`, bọc `pkg/ratelimit/middleware` theo IP client. |

### backofficeService (duyệt)

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/backofficeService/domain/temple_suggestion.go` (not yet created) | `TempleSuggestion` (đủ `TempleID`, `ReviewedBy`, `ReviewedAt`), `NewTemple` (input tạo chùa), `TemplePageInfo`. Sentinel `ErrSuggestionNotFound`, `ErrSuggestionAlreadyCreated`, `ErrSuggestionAlreadyHandled` (Phase 6), `ErrReopenForbidden`, `ErrInvalidStatus`, `ErrInvalidPage`, `TempleInputError{Field, Message}`; `HeritagePriority(rank)` là **bản chép** của `appService/domain/temple.go` (N7). Test đối chiếu `heritage_priority_sync_test.go` import `appService/domain` và so từng hạng, nên lệch là test đỏ. |
| Domain / Repository interface | `domain/temple_suggestion.go` | `List(ctx, status, offset, limit)`, `FindByID`, `CreateTemple(ctx, suggestionID, adminID, role, NewTemple) (*Temple, error)` — một transaction (xem Step-by-step Flow); `Reject(ctx, suggestionID, adminID, reason *string)` (Phase 6). |
| Domain / Mock | `domain/mock_temple_suggestion_repository.go` (not yet created) | Mock tay. |
| Usecase | `usecase/temple_suggestion_usecase.go` (not yet created) | Validate status/page; validate `NewTemple` (AC18–AC20); suy ra `heritage_priority`; gọi repo. |
| Repository | `repository/postgres_temple_suggestion_repository.go` (not yet created) | SQL list/find; `CreateTemple` với `SELECT … FOR UPDATE`, sinh id, `INSERT temples`, `UPDATE temple_suggestions`. |
| Delivery / Middleware | `delivery/http/errors.go` (needs modification) | `RequireAnyRole(roles ...string)` cạnh `RequireRole`; handler truyền role từ claims xuống usecase để kiểm AC30. |
| Delivery / Handler | `delivery/http/temple_suggestion_handler.go` (not yet created) | Parse → usecase → map sentinel sang status; adminID từ claims của token. |
| Config | `config/config.go` (needs modification) | `temple_suggestion.page_size` (dev: 20). |
| Wiring | `cmd/main.go` (needs modification) | 3 route `/v1/admin/temple-suggestions…` qua `authMiddleware` + `RequireActiveSession` + role. |

### Chung

| Layer | File | Responsibility |
|---|---|---|
| Migration | `migrations/0022_temple_suggestions.sql` (not yet created) | Bảng + CHECK + index bên dưới. |
| DB docs | `docs/specs/db/erd.mmd`, `erd.svg`, `migrations/README.md` | Thêm bảng, quan hệ `temple_suggestions N—1 temples`, `N—1 admins`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 4 route, tag mới `TempleSuggestions`. |

## Schema

```
temple_suggestions
  id           BIGSERIAL PRIMARY KEY
  name         VARCHAR(255) NOT NULL
  province     VARCHAR(100) NOT NULL
  address      VARCHAR(500) NOT NULL
  status       VARCHAR(16)  NOT NULL DEFAULT 'pending'
               CHECK (status IN ('pending','created','rejected'))
  temple_id    VARCHAR(64)  REFERENCES temples(id)       -- chỉ khi created
  reviewed_by  BIGINT       REFERENCES admins(id)        -- NULL khi pending
  reviewed_at  TIMESTAMPTZ                               -- NULL khi pending
  reject_reason VARCHAR(500)                             -- chỉ khi rejected, có thể NULL (lý do tuỳ chọn)
  created_at, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()

CHECK (chk_temple_suggestions_review:
         (status = 'pending'  AND temple_id IS NULL     AND reviewed_by IS NULL     AND reviewed_at IS NULL     AND reject_reason IS NULL)
      OR (status = 'created'  AND temple_id IS NOT NULL AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND reject_reason IS NULL)
      OR (status = 'rejected' AND temple_id IS NULL     AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL))
UNIQUE INDEX uq_temple_suggestions_temple ON (temple_id) WHERE temple_id IS NOT NULL
INDEX idx_temple_suggestions_queue ON (status, created_at, id)   -- phục vụ AC11
```

Không lưu IP/thiết bị người gửi (quyết định 2026-10-10); đề xuất giữ vĩnh viễn, không có job dọn. Khi admin tạo chùa từ đề xuất `rejected`, `reject_reason` bị xoá (CHECK `created` đòi NULL). Idempotent (`IF NOT EXISTS`) như mọi file trong `migrations/`.

## Request flow

> Route gửi đề xuất, hai route đọc và (phase sau) route từ chối chỉ một câu SQL nên giữ dạng ASCII; `create-temple` có nhiều bước ghi nên nằm ở Step-by-step Flow và `flow.md`.

```
POST /v1/temple-suggestions
  └─ webAPIKey → rate-limit theo IP (5/giờ) → 429 nếu vượt
       └─ TempleSuggestionHandler.Create   (appService/delivery/http)
            ├─ DecodeJSONBodyStrict → 400 nếu hỏng
            └─ TempleSuggestionUsecase.Create(ctx, name, province, address)
                 ├─ trim + kiểm rỗng/độ dài → TempleSuggestionValidationError (400)
                 └─ TempleSuggestionRepository.Create → INSERT temple_suggestions … RETURNING id
                      └─ 201 {id, status: "pending"}

GET /v1/admin/temple-suggestions[/{id}]
  └─ authMiddleware → RequireActiveSession → RequireAnyRole
       └─ TempleSuggestionHandler.List / Get   (backofficeService/delivery/http)
            └─ usecase: validate status/page → repository
                 ├─ List: SELECT COUNT(*) + SELECT … WHERE status=$1 ORDER BY created_at, id LIMIT/OFFSET
                 └─ Get : SELECT … WHERE id=$1 (sql.ErrNoRows → ErrSuggestionNotFound → 404)

POST /v1/admin/temple-suggestions/{id}/reject   (Phase 6 — route tạm)
  └─ authMiddleware → RequireActiveSession → RequireAnyRole(admin, content_editor)
       └─ TempleSuggestionHandler.Reject → usecase: reason ≤ 500 ký tự (400)
            └─ UPDATE temple_suggestions SET status='rejected', reviewed_by, reviewed_at, reject_reason
                 WHERE id=$1 AND status='pending' RETURNING …
                 ├─ 1 dòng → 200
                 └─ 0 dòng → phân biệt không có dòng (404) / đã xử lý (409 SUGGESTION_ALREADY_HANDLED)
```

## Step-by-step Flow — Tạo chùa từ đề xuất (POST /v1/admin/temple-suggestions/{id}/create-temple)

```
Step C-1 — Xác thực & role (backofficeService)
- Trigger: request tới route.
- Pre-condition: không có.
- Call: authMiddleware.AuthenticateFunc → RequireActiveSession → RequireAnyRole(admin, content_editor)
    - Success → claims (admin id, role) vào context.
    - Failure → 401 (token thiếu/sai/refresh/bị thu hồi) hoặc 403 (role không được phép, vd viewer) — chưa đọc body (AC27).

Step C-2 — Parse & validate (this module)
- Trigger: C-1 qua.
- Pre-condition: {id} là số nguyên dương; body JSON hợp lệ.
- Call: handler DecodeJSONBodyStrict → usecase.CreateTemple → validate NewTemple (AC18–AC20), suy heritage_priority qua HeritagePriority(rank).
    - Success → NewTemple đã chuẩn hoá (chuỗi trim, nil slice → []).
    - Failure ({id} sai → 404; body/field sai → 400 VALIDATION_ERROR) → dừng, chưa chạm DB.

Step C-3 — Khoá & kiểm đề xuất (this module, trong transaction)
- Trigger: C-2 qua; repository.CreateTemple mở transaction.
- Pre-condition: có transaction.
- Call: SELECT … FROM temple_suggestions WHERE id=$1 FOR UPDATE
    - Success (status = pending, hoặc status = rejected và role = admin) → giữ khoá dòng tới hết transaction.
    - Failure (không có dòng) → rollback, ErrSuggestionNotFound → 404.
    - Failure (status = created) → rollback, ErrSuggestionAlreadyCreated → 409. Request thứ hai trong cuộc đua dừng ở đây sau khi request đầu commit (AC22).
    - Failure (status = rejected và role ≠ admin) → rollback, ErrReopenForbidden → 403 (AC30). Kiểm sau khi đọc status vì cần biết trạng thái.

Step C-4 — Sinh id chùa (this module, cùng transaction)
- Trigger: C-3 thấy pending.
- Pre-condition: region đã validate.
- Call: pg_advisory_xact_lock(hash của 'temples:'+region) rồi SELECT max(n) của id dạng temple_<region>_<n> → id mới, dạng `temple_<region>_<NNN>` (đã chốt).
    - Success → id mới, vd temple_south_180.
    - Failure (lỗi DB) → rollback, 500.
    - Ghi chú: khoá advisory chỉ để hai lần tạo cùng region không sinh trùng id; không liên quan khoá dòng ở C-3.

Step C-5 — Chèn chùa (this module, cùng transaction)
- Trigger: C-4 có id.
- Pre-condition: dữ liệu đã validate; CHECK của `temples` là lưới an toàn thứ hai.
- Call: INSERT INTO temples (cột tường minh, heritage_priority từ C-2)
    - Success → hàng chùa mới, hiện ngay với API đọc (AC25).
    - Failure (vi phạm unique/CHECK/DB) → rollback, 500 (AC24); id trùng không thể xảy ra khi C-4 đúng.

Step C-6 — Đánh dấu đề xuất (this module, cùng transaction)
- Trigger: C-5 thành công.
- Pre-condition: dòng đề xuất vẫn đang bị khoá từ C-3.
- Call: UPDATE temple_suggestions SET status='created', temple_id=$2, reviewed_by=$3, reviewed_at=now(), reject_reason=NULL, updated_at=now() WHERE id=$1
    - Success → COMMIT; trả {suggestion, temple} → 201.
    - Failure → rollback cả C-5 (chùa không được tạo), 500.
    - Transaction boundary: C-3 → C-6 chung MỘT transaction; lỗi bất kỳ thì không có chùa và đề xuất vẫn pending.
```

## Known gaps carried by this feature

- **Ảnh chỉ là đường dẫn.** Backend không kiểm file tồn tại và không nhận upload; chùa mới có thể trỏ tới ảnh chưa được đặt vào `static/images/`. Upload thật chờ GCS (cùng lý do `memorial/TD-001`).
- **Hành động từ chối chưa có ở bản đầu.** Schema và nhánh xem lại/tạo chùa từ `rejected` (AC30–AC31) đã sẵn, nhưng route `reject` (AC28–AC29) làm ở Phase 6, nên trước đó đề xuất rác vẫn nằm ở `pending`.
- **Lý do từ chối bị mất khi mở lại.** Tạo chùa từ đề xuất `rejected` xoá `reject_reason`; chưa có bảng lịch sử duyệt.
- **Trùng id chùa giữa hai service.** Backoffice sinh id còn `cmd/seedtemples` upsert theo id của file nguồn; nếu file seed sau này chứa id trùng với chùa tạo từ đề xuất thì seed sẽ ghi đè (dạng id đã chốt, rủi ro này vẫn còn).
- **Rate-limit chưa được wire trong appService.** `pkg/ratelimit` có sẵn nhưng chưa dùng; Phase 2 phải wire theo IP client (phía sau Cloud Run phải lấy IP thật từ header proxy, không phải IP của proxy). IP dùng chung có thể bị giới hạn chung — rủi ro đã chấp nhận.
- **Ánh xạ priority có hai bản.** Backoffice chép bảng 8 hạng từ appService; chỉ test đối chiếu giữ chúng khớp. Thêm hạng mới phải sửa cả hai nơi (migration CHECK + hai map), test sẽ đỏ nếu quên một bên.
- **Role chưa thành quyền thật.** 3 role là nhãn (xem `admin/requirements.md`); spec này chỉ thêm kiểm "một trong các role" và kiểm admin cho nhánh mở lại, chưa có RBAC đầy đủ.
