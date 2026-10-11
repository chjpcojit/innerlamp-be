# Design — temple-suggestion

Version: 3.4
Last updated: 2026-10-11
Status: **v3.4 (2026-10-11): Phase 3–5 và Phase 8 đã implement và kiểm chứng** — hàng chờ, chi tiết, `create-temple` (Step C-1..C-7 gồm C-3 đối chiếu mã) ở backoffice; unit, repository (Postgres thật), E2E đều pass. Phase 6 (từ chối) chưa làm.
Module: `content` (appService) + `backoffice` (backofficeService)

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Một route công khai ở `appService` (ghi một dòng `temple_suggestions`, `pending`) và ba route có token ở `backofficeService` (liệt kê, xem, tạo chùa); route thứ tư — từ chối — thêm ở phase sau (Phase 6). Hai service dùng chung Postgres; migration `0022` sở hữu bảng mới. Route gửi đề xuất theo hình dạng của `/v1/auth/*` (public + `webAPIKey`); ba route duyệt theo hình dạng của `/v1/admin/users` (token access + `RequireActiveSession` + role). Từ v3.0, route gửi đề xuất không còn nhận tỉnh dạng chuỗi: nó nhận `province_code` + `ward_code` của hệ mới, đối chiếu với bộ dữ liệu `administrative-units` rồi lưu mã kèm tên chụp tại thời điểm gửi (Step S-1..S-4). Từ v3.2, `create-temple` cũng nhận `province_code` + `ward_code` + `address`, kiểm mã với cùng dataset (Step C-3) rồi ghi chúng cùng tên chụp vào `temples` — cột mới do spec `temple` v1.7 (migration `0032`) sở hữu. Hai flow có nhiều bước là gửi đề xuất và `create-temple` — nằm ở Step-by-step Flow và `flow.md`.

## Layers & components

### appService (gửi đề xuất)

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/temple_suggestion.go` *(đã làm)* | `TempleSuggestion{ID, Name, ProvinceCode, Province, WardCode, Ward, Address, Status, CreatedAt}` (v3.0: `Province`/`Ward` là tên chụp từ dataset, không phải đầu vào của client); hằng `SuggestionPending`/`SuggestionCreated`; `TempleSuggestionValidationError{Field, Message}`. Không JSON tag. |
| Domain / Repository interface | `domain/temple_suggestion.go` | `TempleSuggestionRepository.Create(ctx, s) (id, err)`; v3.0 thêm `TempleSuggestionInput{Name, ProvinceCode, WardCode, Address}` và port hẹp `WardResolver.ResolveCurrentWard(ctx, provinceCode, wardCode) (*ResolvedWard{ProvinceName, WardFullName}, error)` — `ErrAdministrativeUnitNotFound` khi tỉnh không tồn tại, `ErrWardNotInProvince` khi xã không có hoặc thuộc tỉnh khác (cài bằng `AdministrativeUnitRepository`; một câu SELECT `addr_new_provinces LEFT JOIN addr_new_wards` nên phân biệt được hai trường hợp mà không cần câu thứ hai). |
| Domain / Mock | `domain/mock_temple_suggestion_repository.go` *(đã làm)* | Mock tay cho test usecase. |
| Usecase | `usecase/temple_suggestion_usecase.go` *(đã làm)* | Trim, kiểm rỗng/độ dài (AC2–AC3); v3.0: gọi `WardResolver` để kiểm mã và lấy tên (AC3a, AC5) → `TempleSuggestionValidationError{Field: province_code|ward_code}`; rồi gọi repository. |
| Repository | `repository/postgres_temple_suggestion_repository.go` *(đã làm)* | `INSERT … RETURNING id`; v3.0 ghi thêm `province_code`, `ward_code`, `province`, `ward`. |
| Delivery | `delivery/http/temple_suggestion_handler.go` *(đã làm; v3.0: DTO request `{name, province_code, ward_code, address}`)* | `DecodeJSONBodyStrict` → usecase → `201 {id, status}`; DTO ở đây. Field `province` cũ trở thành field lạ → 400 (AC3b). |
| Config | `config/config.go`, `config.dev.yaml` *(đã làm)* | `temple_suggestion.rate_limit` = `{max, window, trust_forwarded_for}` + `Validate()` (dev: 5 / 1h / false). |
| Delivery / Rate limit | `delivery/http/client_ip_rate_limit.go` *(đã làm)* | `ClientIPRateLimit` bọc `pkg/ratelimit/middleware` theo IP client (TCP peer, hoặc entry cuối `X-Forwarded-For` khi `trust_forwarded_for`). |
| Wiring | `cmd/main.go` *(đã làm; v3.0: `NewTempleSuggestionUsecase(repo, administrativeUnitRepo)`)* | `POST /v1/temple-suggestions` qua `webAPIKey.WrapFunc(limit.WrapFunc(handler.Create))`; `Validate()` sau `Load`. Repository hành chính đã được dựng sẵn cho `administrative-units`, dùng lại. |

### backofficeService (duyệt)

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/backofficeService/domain/temple_suggestion.go` *(đã làm)* | `TempleSuggestion` (đủ `TempleID`, `ReviewedBy`, `ReviewedAt`), `NewTemple` (input tạo chùa; v3.2 thêm `ProvinceCode`, `WardCode` từ client và `Province`, `Ward` do usecase điền từ dataset), port hẹp `WardResolver.ResolveCurrentWard(ctx, provinceCode, wardCode) (*ResolvedWard, error)` *(v3.2, bản riêng của backoffice — không import `appService/domain`)*, `TemplePageInfo`. Sentinel `ErrSuggestionNotFound`, `ErrSuggestionAlreadyCreated`, `ErrSuggestionAlreadyHandled` (Phase 6), `ErrReopenForbidden`, `ErrInvalidStatus`, `ErrInvalidPage`, `TempleInputError{Field, Message}`; `HeritagePriority(rank)` là **bản chép** của `appService/domain/temple.go` (N7). Test đối chiếu `heritage_priority_sync_test.go` import `appService/domain` và so từng hạng, nên lệch là test đỏ. |
| Domain / Repository interface | `domain/temple_suggestion.go` | `List(ctx, status, offset, limit)`, `FindByID`, `CreateTemple(ctx, suggestionID, adminID, role, NewTemple, priority) (*CreatedTemple, error)` — một transaction (xem Step-by-step Flow); `Reject(ctx, suggestionID, adminID, reason *string)` (Phase 6). |
| Domain / Mock | `domain/mock_temple_suggestion_repository.go` *(đã làm)* | Mock tay. |
| Usecase | `usecase/temple_suggestion_usecase.go` *(đã làm)* | Validate status/page; validate `NewTemple` (AC18–AC20); **v3.2:** gọi `WardResolver` để kiểm mã và lấy tên (AC18a–AC18b) → `TempleInputError{Field: province_code|ward_code}`; suy ra `heritage_priority`; gọi repo. |
| Repository | `repository/postgres_temple_suggestion_repository.go` *(đã làm)* | SQL list/find; `CreateTemple` với `SELECT … FOR UPDATE`, sinh id, `INSERT temples` (v3.2: thêm `province_code`, `ward_code`, `province`, `ward`), `UPDATE temple_suggestions`; `ResolveCurrentWard` = một SELECT `addr_new_provinces LEFT JOIN addr_new_wards` (cùng câu với `appService`, bản chép riêng). |
| Delivery / Middleware | `delivery/http/errors.go` *(đã làm)* | `RequireAnyRole(roles ...string)` cạnh `RequireRole`; handler truyền role từ claims xuống usecase để kiểm AC30. |
| Delivery / Handler | `delivery/http/temple_suggestion_handler.go` *(đã làm)* | Parse → usecase → map sentinel sang status; adminID từ claims của token. |
| Config | `config/config.go` *(đã làm)* | `temple_suggestion.page_size` (dev: 20). |
| Wiring | `cmd/main.go` *(đã làm)* | 3 route `/v1/admin/temple-suggestions…` qua `authMiddleware` + `RequireActiveSession` + role. |

### Chung

| Layer | File | Responsibility |
|---|---|---|
| Migration | `migrations/0022_temple_suggestions.sql`  | Bảng + CHECK + index bên dưới. |
| Migration (v3.0) | `migrations/0030_temple_suggestions_address_codes.sql` *(đã tạo)* | Thêm `province_code`, `ward_code`, `ward` (nullable), CHECK mã đi cặp; idempotent (`ADD COLUMN IF NOT EXISTS`). Không backfill dòng cũ. |
| Migration (v3.2) | `migrations/0032_temple_address_codes.sql` *(của spec `temple` v1.7, chưa tạo)* | Ba cột `temples.province_code`, `ward_code`, `ward`; `create-temple` phụ thuộc nó. Spec này không thêm migration. |
| DB docs | `docs/specs/db/erd.mmd`, `erd.svg`, `migrations/README.md` | Thêm bảng, quan hệ `temple_suggestions N—1 temples`, `N—1 admins`. |
| API spec | `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) | 4 route, tag mới `TempleSuggestions`. |

## Schema

```
temple_suggestions
  id           BIGSERIAL PRIMARY KEY
  name         VARCHAR(255) NOT NULL
  province     VARCHAR(100) NOT NULL                     -- v3.0: tên tỉnh chụp từ dataset; dòng cũ giữ chuỗi người dùng gõ
  province_code VARCHAR(16)                              -- v3.0 (0030): mã tỉnh hệ mới; NULL ở dòng cũ
  ward_code    VARCHAR(16)                               -- v3.0 (0030): mã xã hệ mới; NULL ở dòng cũ
  ward         VARCHAR(255)                              -- v3.0 (0030): tên đầy đủ của xã chụp từ dataset; NULL ở dòng cũ
  address      VARCHAR(500) NOT NULL                     -- v3.0: chỉ phần dưới cấp xã
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
CHECK (chk_temple_suggestions_address_codes, v3.0:
         (province_code IS NULL AND ward_code IS NULL AND ward IS NULL)
      OR (province_code IS NOT NULL AND ward_code IS NOT NULL AND ward IS NOT NULL))
UNIQUE INDEX uq_temple_suggestions_temple ON (temple_id) WHERE temple_id IS NOT NULL
INDEX idx_temple_suggestions_queue ON (status, created_at, id)   -- phục vụ AC11
```

**Không đặt FK** từ `province_code`/`ward_code` sang `addr_*` (nạp lại dữ liệu theo kiểu upsert; mã chỉ có nghĩa cùng hệ) — tính hợp lệ được kiểm ở usecase lúc gửi, còn tên chụp giữ cho đề xuất đọc được dù dataset đổi. Dòng gửi trước v3.0 để cột mới `NULL`. Không lưu IP/thiết bị người gửi (quyết định 2026-10-10); đề xuất giữ vĩnh viễn, không có job dọn. Khi admin tạo chùa từ đề xuất `rejected`, `reject_reason` bị xoá (CHECK `created` đòi NULL). Idempotent (`IF NOT EXISTS`) như mọi file trong `migrations/`.

## Request flow

> Route gửi đề xuất, hai route đọc và (phase sau) route từ chối chỉ một câu SQL nên giữ dạng ASCII; `create-temple` có nhiều bước ghi nên nằm ở Step-by-step Flow và `flow.md`.

```
POST /v1/temple-suggestions
  └─ webAPIKey → rate-limit theo IP (5/giờ) → 429 nếu vượt
       └─ TempleSuggestionHandler.Create   (appService/delivery/http)
            ├─ DecodeJSONBodyStrict → 400 nếu hỏng
            └─ TempleSuggestionUsecase.Create(ctx, name, provinceCode, wardCode, address)
                 ├─ trim + kiểm rỗng/độ dài → TempleSuggestionValidationError (400)
                 ├─ WardResolver.ResolveCurrentWard(provinceCode, wardCode) → tên tỉnh/xã; không có/lệch tỉnh → 400 (province_code | ward_code)
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

## Step-by-step Flow — Gửi đề xuất (POST /v1/temple-suggestions, v3.0)

```
Step S-1 — Cổng vào (appService)
- Trigger: request tới route.
- Pre-condition: không có.
- Call: webAPIKey.WrapFunc → ClientIPRateLimit.WrapFunc
    - Success → đi tiếp (header key vắng hoặc đúng; chưa vượt 5/IP/giờ).
    - Failure → 401 INVALID_API_KEY hoặc 429 RATE_LIMITED; chưa đọc body, chưa chạm DB (AC7).

Step S-2 — Parse & validate hình dạng (this module)
- Trigger: S-1 qua.
- Pre-condition: không có.
- Call: handler DecodeJSONBodyStrict → usecase.Create: trim, kiểm rỗng và độ dài của name/address, kiểm rỗng của province_code/ward_code (AC2–AC4).
    - Success → input đã chuẩn hoá.
    - Failure (JSON hỏng, field lạ kể cả `province` cũ, rỗng, quá dài) → 400, chưa chạm DB.

Step S-3 — Đối chiếu mã với metadata hành chính (this module, đọc)
- Trigger: S-2 qua.
- Pre-condition: province_code và ward_code không rỗng.
- Call: WardResolver.ResolveCurrentWard(provinceCode, wardCode) — một SELECT trên addr_new_provinces LEFT JOIN addr_new_wards ON ward.code=$2 AND ward.province_code=province.code WHERE province.code=$1
    - Success → (province.name, ward.full_name) làm tên chụp (AC5).
    - Failure (ErrAdministrativeUnitNotFound) → 400 VALIDATION_ERROR; một câu duy nhất đã phân biệt `province_code` (tỉnh không tồn tại → `ErrAdministrativeUnitNotFound`) với `ward_code` (xã không có hoặc thuộc tỉnh khác → `ErrWardNotInProvince`) (AC3a); chưa ghi gì.
    - Failure (lỗi DB) → 500.

Step S-4 — Ghi đề xuất (this module, một câu INSERT)
- Trigger: S-3 trả tên.
- Pre-condition: tên tỉnh/xã lấy từ dataset, không từ client.
- Call: TempleSuggestionRepository.Create — INSERT temple_suggestions (name, province_code, province, ward_code, ward, address, status='pending') RETURNING id
    - Success → 201 {id, status: "pending"}.
    - Failure (DB / CHECK) → 500 INTERNAL_ERROR, không lộ nguyên nhân.
    - Ghi chú: S-3 và S-4 không chung transaction; dataset chỉ đổi khi vận hành nạp bản mới nên cửa sổ lệch là chấp nhận được — tên đã chụp, mã giữ nguyên.
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

Step C-3 — Đối chiếu mã địa chỉ (this module, đọc, ngoài transaction) *(v3.2)*
- Trigger: C-2 qua.
- Pre-condition: province_code và ward_code không rỗng (AC18).
- Call: WardResolver.ResolveCurrentWard(provinceCode, wardCode) — một SELECT trên addr_new_provinces LEFT JOIN addr_new_wards ON ward.code=$2 AND ward.province_code=province.code WHERE province.code=$1
    - Success → (province.name, ward.full_name) gán vào NewTemple.Province / NewTemple.Ward (AC18b).
    - Failure (ErrAdministrativeUnitNotFound / ErrWardNotInProvince) → 400 VALIDATION_ERROR, details.field = province_code | ward_code (AC18a); chưa mở transaction, đề xuất không đổi.
    - Failure (lỗi DB) → 500.
    - Ghi chú: chạy trước transaction vì chỉ đọc dataset; không chung snapshot với C-4…C-7 — cùng lập luận S-3/S-4 (dataset chỉ đổi khi vận hành nạp bản mới, tên đã chụp, mã giữ nguyên).

Step C-4 — Khoá & kiểm đề xuất (this module, trong transaction)
- Trigger: C-3 qua; repository.CreateTemple mở transaction.
- Pre-condition: có transaction.
- Call: SELECT … FROM temple_suggestions WHERE id=$1 FOR UPDATE
    - Success (status = pending, hoặc status = rejected và role = admin) → giữ khoá dòng tới hết transaction.
    - Failure (không có dòng) → rollback, ErrSuggestionNotFound → 404.
    - Failure (status = created) → rollback, ErrSuggestionAlreadyCreated → 409. Request thứ hai trong cuộc đua dừng ở đây sau khi request đầu commit (AC22).
    - Failure (status = rejected và role ≠ admin) → rollback, ErrReopenForbidden → 403 (AC30). Kiểm sau khi đọc status vì cần biết trạng thái.

Step C-5 — Sinh id chùa (this module, cùng transaction)
- Trigger: C-4 thấy pending.
- Pre-condition: region đã validate.
- Call: pg_advisory_xact_lock(hash của 'temples:'+region) rồi SELECT max(n) của id dạng temple_<region>_<n> → id mới, dạng `temple_<region>_<NNN>` (đã chốt).
    - Success → id mới, vd temple_south_180.
    - Failure (lỗi DB) → rollback, 500.
    - Ghi chú: khoá advisory chỉ để hai lần tạo cùng region không sinh trùng id; không liên quan khoá dòng ở C-4.

Step C-6 — Chèn chùa (this module, cùng transaction)
- Trigger: C-5 có id.
- Pre-condition: dữ liệu đã validate; CHECK của `temples` là lưới an toàn thứ hai.
- Call: INSERT INTO temples (cột tường minh, gồm province_code, ward_code, province, ward từ C-3 — v3.2; heritage_priority từ C-2)
    - Success → hàng chùa mới, hiện ngay với API đọc (AC25).
    - Failure (vi phạm unique/CHECK/DB) → rollback, 500 (AC24); id trùng không thể xảy ra khi C-5 đúng.

Step C-7 — Đánh dấu đề xuất (this module, cùng transaction)
- Trigger: C-6 thành công.
- Pre-condition: dòng đề xuất vẫn đang bị khoá từ C-4.
- Call: UPDATE temple_suggestions SET status='created', temple_id=$2, reviewed_by=$3, reviewed_at=now(), reject_reason=NULL, updated_at=now() WHERE id=$1
    - Success → COMMIT; trả {suggestion, temple} → 201.
    - Failure → rollback cả C-6 (chùa không được tạo), 500.
    - Transaction boundary: C-4 → C-7 chung MỘT transaction; lỗi bất kỳ thì không có chùa và đề xuất vẫn pending.
```

## Known gaps carried by this feature

- **Field `province` dạng chuỗi tự do bị bỏ khỏi hợp đồng gửi (v3.0, N8).** Đề xuất cũ giữ nguyên chuỗi và để `province_code`/`ward_code`/`ward` là `NULL`; backoffice phải hiển thị được cả hai dạng (AC9). Không backfill vì không thể suy ra xã từ chuỗi tự do.
- **~~`create-temple` chưa dùng mã~~ (đã làm, v3.4).** `create-temple` nhận và ghi `province_code`/`ward_code` (Step C-3, C-6); `temples.province` lấy từ dataset chứ không còn từ `domain/province.go` (file đó đã xoá).
- **Bản chép SQL tra mã thứ hai.** Backoffice có `WardResolver` riêng (không import `appService/domain`, như `HeritagePriority` ở N7) nên SELECT `addr_new_*` có hai bản; lệch một bên là lỗi im lặng nên cần test trên Postgres thật ở cả hai service.
- **Kiểm mã và ghi không chung transaction (S-3 → S-4).** Chấp nhận được vì dataset chỉ đổi khi vận hành nạp lại; hậu quả xấu nhất là một đề xuất mang mã đã bị đổi/xoá ở bản dữ liệu sau, vẫn đọc được nhờ tên chụp.

- **Ảnh chỉ là đường dẫn.** Backend không kiểm file tồn tại và không nhận upload; chùa mới có thể trỏ tới ảnh chưa được đặt vào `static/images/`. Upload thật chờ GCS (cùng lý do `memorial/TD-001`).
- **Hành động từ chối chưa có ở bản đầu.** Schema và nhánh xem lại/tạo chùa từ `rejected` (AC30–AC31) đã sẵn, nhưng route `reject` (AC28–AC29) làm ở Phase 6, nên trước đó đề xuất rác vẫn nằm ở `pending`.
- **Lý do từ chối bị mất khi mở lại.** Tạo chùa từ đề xuất `rejected` xoá `reject_reason`; chưa có bảng lịch sử duyệt.
- **Trùng id chùa giữa hai service.** Backoffice sinh id còn `cmd/seedtemples` upsert theo id của file nguồn; nếu file seed sau này chứa id trùng với chùa tạo từ đề xuất thì seed sẽ ghi đè (dạng id đã chốt, rủi ro này vẫn còn).
- **Rate-limit theo IP nằm trong bộ nhớ của từng instance.** Đã wire (`delivery/http/client_ip_rate_limit.go`) nhưng mỗi instance đếm riêng nên hạn mức thật là 5 × số instance; muốn chính xác cần store dùng chung (`pkg/ratelimit/store` redis). `trust_forwarded_for` phải bật ở production sau Cloud Run, nếu không mọi người dùng chung một bucket. IP dùng chung (NAT) có thể bị giới hạn chung — rủi ro đã chấp nhận.
- **Ánh xạ priority có hai bản.** Backoffice chép bảng 8 hạng từ appService; chỉ test đối chiếu giữ chúng khớp. Thêm hạng mới phải sửa cả hai nơi (migration CHECK + hai map), test sẽ đỏ nếu quên một bên.
- **Role chưa thành quyền thật.** 3 role là nhãn (xem `admin/requirements.md`); spec này chỉ thêm kiểm "một trong các role" và kiểm admin cho nhánh mở lại, chưa có RBAC đầy đủ.
