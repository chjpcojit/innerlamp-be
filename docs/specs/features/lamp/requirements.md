# Requirements — lamp

Version: 1.2
Last updated: 2026-10-10
Status: v1 (Phase 1–4: domain, config, usecase, repository, migration `0027`, handler, wiring, openapi) đã implement và kiểm chứng 2026-10-10 — unit, repository (Postgres thật) và E2E (binary thật + Postgres) đều pass. Còn 2 marker `⚠️ Not confirmed` (hiệu năng/khối lượng; retention + xoá tài khoản — migration đang dùng `ON DELETE CASCADE` tạm thời). Chưa sinh lại `erd.svg` (thiếu `mmdc`).
Module: `lamp`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code; Phase 1–4 đã được implement theo đúng spec ở v1.2) cho tính năng **Thắp đèn và Vườn đèn**: người dùng đã đăng nhập thắp một ngọn đèn kèm tâm nguyện, người nhận, lời nguyện và thời gian sáng; mọi người dùng đã đăng nhập xem được vườn đèn gồm các ngọn đèn đang sáng của tất cả mọi người. Nguồn nghiệp vụ là [`SRS.md`](SRS.md) v0.3 (IEEE 29148, 26 câu đã chốt với anh Đức ngày 2026-10-10) — các AC dưới đây bám theo `FR-001`…`FR-009` của SRS.

## Quyết định nền tảng (đã chốt, 2026-10-10)

- **Vườn đèn công khai trong phạm vi người đã đăng nhập.** Mọi route đều cần access token; khách chưa đăng nhập không xem được (khác nhóm `temple`/`deity`/`article`). Người xem thấy đèn của **tất cả** người dùng.
- **Chỉ lộ bốn thứ ra ngoài:** tên người thắp, loại đèn, người nhận, ngày tắt. **Tâm nguyện và lời nguyện không bao giờ trả cho người khác** (v1 không có màn "đèn của tôi" nên hai trường này chỉ nằm trong response của chính lần thắp).
- **Tên người thắp = pháp danh, chụp lại lúc thắp.** Không có pháp danh, hoặc người thắp chọn ẩn danh, thì hiển thị `Vô danh`. Đổi pháp danh về sau **không** đổi tên trên đèn đã thắp.
- **Thời gian là mốc tuyệt đối do server tính.** `end_at = lit_at + duration_days × 24h`, lưu UTC; app tự đổi sang múi giờ máy khi hiển thị ngày tắt.
- **Đèn không gia hạn, không tắt sớm.** Hết `end_at` đèn biến mất khỏi vườn đèn và không còn tính vào trần 20. Muốn thắp tiếp thì thắp ngọn mới.
- **Miễn phí ở v1, không kiểm duyệt.** Thắp đèn tại chùa có phí là v2, ngoài phạm vi.
- Mã user story `US-LAMP-01..02` do spec tự chọn (theo tiền lệ `US-MEM-01`).

---

## US-LAMP-01: Thắp một ngọn đèn

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **thắp một ngọn đèn với loại đèn, tâm nguyện, người nhận, lời nguyện và thời gian sáng**,
> Để **gửi gắm lời nguyện của mình trong vườn đèn suốt thời gian đèn sáng**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/lamps` request arrives with a valid access token and a valid body, THE SYSTEM SHALL create a lit lamp owned by the token's user and return `201 Created` with the full `Lamp` (shape ở "API contract"), where `lit_at` is the server time of the request and `end_at = lit_at + duration_days × 24 hours`. (FR-001, FR-004)
2. WHEN the request carries no valid access token (thiếu, hết hạn, sai loại token), THE SYSTEM SHALL return `401` and SHALL NOT create anything. (FR-009)
3. WHEN `lamp_type` is not one of `OIL_LAMP` (đèn dầu), `LOTUS_CANDLE` (nến sen), `FLOWER_LANTERN` (hoa đăng), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "lamp_type"`. (FR-002)
4. WHEN `intention` is not one of `PEACE` (cầu an), `SALVATION` (cầu siêu), `MERIT_DEDICATION` (hồi hướng), `GRATITUDE` (tri ân), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "intention"`. (FR-002)
5. WHEN `duration_days` is not one of the configured durations (`lamp.durations_days`, dev: `1, 7, 21, 49`), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "duration_days"`. (FR-002)
6. WHEN `recipient` is empty or blank after trimming leading/trailing whitespace, or longer than 100 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "recipient"`; the stored value is the trimmed text. Độ dài đếm theo ký tự (code point sau chuẩn hoá NFC), không theo byte. (FR-002)
7. WHEN `prayer` is longer than 300 characters after trimming, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "prayer"`; WHEN `prayer` is absent, `null` or blank after trimming, THE SYSTEM SHALL store `NULL` and return `prayer: null`. (FR-002)
8. WHEN `is_anonymous` is absent, THE SYSTEM SHALL treat it as `false`; WHEN it is present and not a boolean, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "is_anonymous"`. (FR-001)
9. WHEN the user already has `lamp.max_active_per_user` (dev: 20) lit lamps — a lit lamp being one whose `end_at` is later than now — THE SYSTEM SHALL return `409 LAMP_LIMIT_REACHED` and SHALL NOT insert; lamps whose `end_at` has passed SHALL NOT count, and two requests of the same user running concurrently SHALL NOT push the user above the limit. (FR-003)
10. WHEN the same user submits a lamp whose content (`lamp_type`, `intention`, `recipient`, `prayer`, `duration_days`, `is_anonymous`) is identical to a lamp they lit within the last `lamp.dedupe_window` (dev: 5s), THE SYSTEM SHALL NOT insert a second lamp and SHALL return `200 OK` with that existing lamp; WHEN the previous identical lamp is older than the window, THE SYSTEM SHALL create a new lamp. Việc chống trùng được kiểm **trước** trần (một lần bấm đúp ở đúng 20 đèn vẫn nhận lại đèn vừa thắp, không nhận 409). (FR-001)
11. WHEN the body has unknown fields, is empty, or is not valid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (cùng cơ chế `DecodeJSONBodyStrict` như các route khác). WHEN the body sets `id`, `user_id`, `lit_at`, `end_at`, `lighter_name` or any server-owned field, THE SYSTEM SHALL reject it as an unknown field.
12. WHEN a lamp is created with `is_anonymous = false` AND the user has a dharma name, THE SYSTEM SHALL snapshot that dharma name onto the lamp; WHEN `is_anonymous = true` or the user has no dharma name, THE SYSTEM SHALL store no name on the lamp. WHEN the user later changes their dharma name, THE SYSTEM SHALL leave existing lamps unchanged. (FR-005)

## US-LAMP-02: Xem vườn đèn

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem các ngọn đèn đang sáng của mọi người, lọc theo loại đèn và tải thêm**,
> Để **thấy cộng đồng đang thắp đèn và biết hiện có bao nhiêu ngọn đang sáng**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/lamps` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding the lit lamps of **all** users (không chỉ của người gọi), ordered by `lit_at` DESC then `id` DESC (đèn mới thắp lên đầu), each item carrying only `id`, `lamp_type`, `recipient`, `lighter_name` and `end_at`. (FR-005)
14. WHEN the request carries no valid access token, THE SYSTEM SHALL return `401` and SHALL NOT return any lamp or count. (FR-009)
15. THE SYSTEM SHALL NOT return `intention`, `prayer`, `user_id`, `is_anonymous` or `lit_at` in any garden item. (FR-005)
16. WHEN `lighter_name` is returned for a lamp with no stored name (người thắp ẩn danh, hoặc không có pháp danh), THE SYSTEM SHALL return the literal `Vô danh`; THE SYSTEM SHALL NOT return the dharma name of an anonymous lamp in any field. (FR-005)
17. WHEN `lamp_type` is absent, THE SYSTEM SHALL list lamps of every type; WHEN it is one of the three types, THE SYSTEM SHALL list only that type; WHEN it is present but not one of the three (kể cả rỗng), THE SYSTEM SHALL return `400 INVALID_FILTER` with `details.field = "lamp_type"`. (FR-006)
18. THE SYSTEM SHALL return in the same response `data.total` (số đèn đang sáng của tất cả người dùng, mọi loại) and `data.counts` with one key per lamp type (`OIL_LAMP`, `LOTUS_CANDLE`, `FLOWER_LANTERN`, `0` khi không có đèn), both independent of the `lamp_type` filter and read from the same snapshot as the page. (FR-008)
19. WHEN `limit` is absent, THE SYSTEM SHALL return at most `lamp.page_size` (dev: 12) items; WHEN `limit` is greater than `lamp.page_size`, THE SYSTEM SHALL return at most `lamp.page_size` items (không lỗi); WHEN `limit` is not a positive integer, THE SYSTEM SHALL return `400 INVALID_LIMIT` with `details.field = "limit"`. (FR-007)
20. WHEN more lit lamps match than the page holds, THE SYSTEM SHALL return `has_more: true` and a `next_cursor`; WHEN the client sends that `cursor`, THE SYSTEM SHALL return the lamps strictly after the last one of the previous page in the same order, so that no lamp is repeated or skipped because other lamps expired or were lit in between. WHEN there is no further lamp, THE SYSTEM SHALL return `has_more: false` and `next_cursor: null`. (FR-007)
21. WHEN `cursor` is present but not a value previously issued by this endpoint, THE SYSTEM SHALL return `400 INVALID_CURSOR` with `details.field = "cursor"`. (FR-007)
22. WHEN no lamp is lit, THE SYSTEM SHALL return `200` with `items: []` (không phải `null`), `total: 0`, every count `0`, `has_more: false`, `next_cursor: null`. (FR-005, FR-008)
23. WHEN a lamp's `end_at` is not later than the server time of the request, THE SYSTEM SHALL NOT list it, SHALL NOT count it in `total` or `counts`, and SHALL NOT count it toward the user's limit — kiểm tại thời điểm request, không phụ thuộc job dọn. (FR-004)
24. THE SYSTEM SHALL write `end_at` (and `lit_at` in the `Lamp` response) as RFC 3339 in UTC (`2026-10-17T18:00:00Z`); THE SYSTEM SHALL NOT take the current time from the client. (FR-004)

## API contract (confirmed surface for this feature)

> Đã có trong `docs/specs/api/openapi.yaml` (tag `Lamps`, hai route, schema `Lamp`/`LampRequest`/`GardenLamp`/`GardenPage`) và bản sao `web/docs/public/openapi.yaml`. Envelope `{"success", "data"}` / `{"success": false, "error", "details"?}` như mọi route.

| Method | Path | Auth | Success | Failure codes |
|---|---|---|---|---|
| POST | `/v1/lamps` | `authMiddleware` (access token) | `201` `Lamp` (đèn mới) hoặc `200` `Lamp` (nhận lại đèn trùng trong cửa sổ 5s) | 400 `VALIDATION_ERROR` (`details.field`), 401, 405, 409 `LAMP_LIMIT_REACHED`, 500 |
| GET | `/v1/lamps?lamp_type=&limit=&cursor=` | `authMiddleware` (access token) | `200` `{items: [GardenLamp], total, counts, has_more, next_cursor}` | 400 `INVALID_FILTER`, 400 `INVALID_LIMIT`, 400 `INVALID_CURSOR`, 401, 405, 500 |

**Request `POST /v1/lamps`**

```json
{
  "lamp_type": "OIL_LAMP",
  "intention": "PEACE",
  "recipient": "Bà ngoại",
  "prayer": "Mong bà bình an",
  "duration_days": 7,
  "is_anonymous": false
}
```

`prayer` và `is_anonymous` tuỳ chọn. Các trường còn lại bắt buộc.

**Response `Lamp`** (chỉ trả cho người thắp): `id`, `lamp_type`, `intention`, `recipient`, `prayer` (hoặc `null`), `duration_days`, `is_anonymous`, `lighter_name`, `lit_at`, `end_at`.

**Response `GardenLamp`** (vườn đèn): `id`, `lamp_type`, `recipient`, `lighter_name`, `end_at`.

**Response `GET /v1/lamps`**

```json
{
  "success": true,
  "data": {
    "items": [
      { "id": 41, "lamp_type": "LOTUS_CANDLE", "recipient": "Thầy", "lighter_name": "Thích Minh An", "end_at": "2026-10-17T18:00:00Z" }
    ],
    "total": 10,
    "counts": { "OIL_LAMP": 5, "LOTUS_CANDLE": 3, "FLOWER_LANTERN": 2 },
    "has_more": true,
    "next_cursor": "MTc5MDAwMDAwMDAwMDAwMC40MQ"
  }
}
```

## Out of scope (this feature boundary, not tracked here)

- **Thắp đèn tại chùa có phí (v2)**, thanh toán, cúng dường — chưa có spec; v1 không đưa khái niệm tiền/chùa vào bảng `lamps`.
- Kiểm duyệt lời nguyện, báo cáo vi phạm, gỡ đèn qua backoffice (quyết định: không có ở v1; lời nguyện vốn không lộ ra ngoài, người nhận thì có).
- Gia hạn, thắp lại, tắt/xoá đèn sớm.
- Màn "đèn của tôi" và lịch sử đèn đã hết hạn.
- Lọc vườn đèn theo tâm nguyện; thông báo "đèn sắp tắt".
- Giao diện, hình ảnh, âm thanh, hiệu ứng của từng loại đèn (client).

## Pending (not yet implemented)

- Sinh lại `docs/specs/db/erd.svg` từ `erd.mmd` bằng `mmdc` (máy này chưa có `mmdc`; `erd.mmd` đã có bảng `lamps`) — `tasks.md` Phase 3.
- Concept OKF `docs/knowledge/features/lamp.md` (`il-okf-writer`, sau khi flow đã được kiểm chứng) — `tasks.md` Phase 5.

> ⚠️ Not confirmed: mục tiêu thời gian phản hồi của `GET /v1/lamps` và số đèn sáng dự kiến cùng lúc (chủ sản phẩm chưa biết, sẽ đo sau khi ra mắt) — chặn quyết định chỉ mục và cách đếm (thiết kế hiện tại đếm trực tiếp bằng `GROUP BY` trên các dòng còn sáng, dễ thay).

> ⚠️ Not confirmed: thời gian giữ đèn đã hết hạn, và đèn của một tài khoản bị xoá thì bị xoá ngay hay để đến hết hạn (đã hoãn có chủ đích: "lưu trước, tính sau") — chặn mệnh đề `ON DELETE` của khoá ngoại `lamps.user_id` và mọi job dọn. **Tạm thời** migration `0027_lamps.sql` dùng `ON DELETE CASCADE` (giống `memorials`; chưa có route xoá tài khoản nên chưa có gì kích hoạt nó) — đổi được bằng một migration mới khi chốt retention.


## Ghi chú kỹ thuật liên quan

- **Đồng hồ:** `lit_at`, `end_at` và mọi so sánh "đang sáng" (`end_at > now`) dùng cùng một giá trị `now` do usecase lấy từ đồng hồ server (inject được cho test), không dùng `now()` của DB rải rác trong từng câu SQL — để `total`, `counts` và trang cùng một thời điểm.
- **Số migration là `0027`** (không phải `0026`): `0026_administrative_units.sql` của công việc khác đã chiếm số đó.
- **Giới hạn nằm ở config, không là hằng trong domain:** `lamp.max_active_per_user`, `lamp.page_size`, `lamp.dedupe_window`, `lamp.durations_days` (xem `design.md`, Config). Độ dài `recipient` (100) và `prayer` (300) khớp kích thước cột `VARCHAR` nên là hằng gắn với schema, như `full_name`/`note` của `memorial`.
- **Chuẩn hoá chữ:** trim hai đầu rồi chuẩn hoá NFC trước khi đếm, để chữ tiếng Việt gõ dạng tổ hợp (`e` + dấu) và dựng sẵn cùng một độ dài. `golang.org/x/text` đã nằm trong `go.mod` (indirect) — đưa lên direct khi dùng.
- **Quyền riêng tư:** `recipient` có thể là tên người thân đã khuất và hiển thị cho mọi người dùng đã đăng nhập; không ghi `recipient`/`prayer` vào log (chỉ id).
- **Tên ẩn danh không nằm trong DB:** đèn ẩn danh lưu `lighter_name = NULL` và `is_anonymous = true`; server không giữ pháp danh trên dòng đó, nên không thể lộ nhầm qua một query khác.
- **Bảng người dùng hiện chỉ có `dharma_name` tuỳ chọn** (không có trường tên thường) — vì vậy "không có pháp danh" luôn hiển thị `Vô danh`, không cần thêm trường hồ sơ.
- **Quan hệ với `memorial`:** cùng mô hình "dữ liệu theo user + trần số lượng + khoá dòng `users` khi tạo"; khác ở chỗ vườn đèn đọc dữ liệu của người khác (chỉ bốn trường công khai).

## Open questions (resolved during implementation)

- ~~Ai thấy đèn nào trong vườn đèn?~~ → Mọi người dùng đã đăng nhập thấy đèn đang sáng của tất cả. Resolved 2026-10-10.
- ~~Khách chưa đăng nhập có xem được không?~~ → Không. Resolved 2026-10-10.
- ~~Hiển thị gì với đèn của người khác?~~ → Tên người thắp, loại đèn, người nhận, ngày tắt; tâm nguyện và lời nguyện không hiển thị. Resolved 2026-10-10.
- ~~`recipient` / `prayer`?~~ → Văn bản tự do; `recipient` 1–100 ký tự (bắt buộc), `prayer` tối đa 300 ký tự (tuỳ chọn), đếm theo ký tự, trim hai đầu. Resolved 2026-10-10.
- ~~Kiểm duyệt ở v1?~~ → Không. Resolved 2026-10-10.
- ~~Chi phí?~~ → Miễn phí ở v1; v2 thêm thắp tại chùa có phí. Resolved 2026-10-10.
- ~~Trần đèn?~~ → 20 đèn đang sáng cùng lúc mỗi người; đèn hết hạn không tính. Resolved 2026-10-10.
- ~~Hết hạn / gia hạn / tắt sớm?~~ → Biến mất khi hết hạn; không gia hạn, không tắt sớm. Resolved 2026-10-10.
- ~~`total` có phụ thuộc bộ lọc không?~~ → Không: `total` và số từng loại là của toàn bộ đèn đang sáng, cùng một response. Resolved 2026-10-10.
- ~~Thứ tự? Cỡ trang?~~ → Đèn mới thắp lên đầu; tối đa 12 mỗi trang. Resolved 2026-10-10.
- ~~Bấm đúp?~~ → Cùng nội dung trong 5 giây chỉ tạo một đèn. Resolved 2026-10-10.
- ~~"N ngày" nghĩa là gì, múi giờ nào?~~ → N×24 giờ từ lúc thắp; lưu UTC; ngày tắt hiển thị theo múi giờ máy người xem. Resolved 2026-10-10.
- ~~Tên người thắp?~~ → Pháp danh; không có thì `Vô danh`; cho phép ẩn danh; không đổi theo khi đổi pháp danh. Resolved 2026-10-10.
- ~~Trim?~~ → Có, hai đầu, trước khi kiểm và lưu. Resolved 2026-10-10.

- ~~Phân trang vườn đèn dùng `page`/`total_pages` như `temple`, `deity`, `article`, hay cursor?~~ → **Cursor** (`cursor`/`next_cursor`/`has_more`), chấp nhận lệch quy ước vì AC20 đòi không lặp, không sót khi đèn hết hạn hoặc đèn mới được thắp giữa hai lần tải. Client hỗ trợ thêm một kiểu phân trang. Resolved 2026-10-10.

Còn mở: hai marker `⚠️ Not confirmed` ở mục **Pending** ở trên.
