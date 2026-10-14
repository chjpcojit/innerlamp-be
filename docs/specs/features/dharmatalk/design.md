# Design — dharmatalk

Version: 2.0
Last updated: 2026-10-11
Status: implemented (2026-10-03) — 2 route đọc + seed lần đầu chạy thật qua binary + Postgres; có test usecase/handler/seed parser/repository/E2E. Thiếu: không còn mục kỹ thuật nào mở; OKF concept chưa viết.
Module: `content`

> See `changelog.md` in this folder for the version history of this spec.

## Overview

Hai route `GET /v1/dharma-talks` và `GET /v1/dharma-talks/{id}` đều công khai, không đi qua middleware xác thực. Feature chạm đủ bốn layer của `services/appService` (`domain` → `usecase` → `repository` / `delivery/http`, nối dây ở `cmd/main.go`), sở hữu migration `0014_dharma_talks.sql`, và nhúng thêm `assets/data/dharma_talks.json` vào binary. Cùng khuôn với `article` (đã implement): cùng `SeedIfEmpty` khi khởi động, cùng phân trang, cùng bao lỗi; **khác** ở chỗ có `media_type` (ba biến thể), `chapters`, và lọc `needs_review`.

Thiết kế bám `article` có chủ đích (code đã viết theo đó): `usecase/article_usecase.go`, `repository/postgres_article_repository.go`, `repository/article_seed.go` làm mẫu và tái dùng `ErrInvalidPage`.

## Layers & components

| Layer | File | Responsibility |
|---|---|---|
| Domain / Entity | `services/appService/domain/dharma_talk.go` | `DharmaTalk` (mọi cột; `PublishedDate *time.Time`, `DurationMinutes int` 0 = chưa biết, `Season`/`YouTubeID`/`AudioURL string` rỗng = không có, `Chapters []DharmaTalkChapter`, `FollowUpMeditation *FollowUpMeditation`, `NeedsReview bool`), `DharmaTalkChapter{StartSeconds int; Title string}`, `FollowUpMeditation{Title string; Minutes int; MeditationID string}`, `DharmaTalkPage`, `MediaType` (`MediaYouTube`/`MediaAudio`/`MediaText`). Không JSON/db tag. |
| Domain / Errors | `domain/dharma_talk.go` | `ErrDharmaTalkNotFound`; tái dùng `ErrInvalidPage` (định nghĩa ở `domain/temple.go`). |
| Domain / Repository interface | `domain/dharma_talk.go` | `DharmaTalkRepository`: `List`, `FindByID`, `Count`, `InsertMissing`. `List` (kể cả `total` nó trả) và `FindByID` **chỉ thấy** dòng `needs_review = false`; `Count` và `InsertMissing` thấy mọi dòng — đây là một phần của hợp đồng. |
| Domain / Mock | `domain/mock_dharma_talk_repository.go` | Mock viết tay cho test usecase. |
| Usecase | `services/appService/usecase/dharma_talk_usecase.go` | `List` (offset/`total_pages`, chặn `page < 1` hoặc `> MaxInt32`), `Get`, `SeedIfEmpty(load)`. Kích thước trang từ `config.DharmaTalkConfig`. |
| Repository | `services/appService/repository/postgres_dharma_talk_repository.go` | SQL cho `dharma_talks`; `InsertMissing` một transaction, `ON CONFLICT (id) DO NOTHING`; `NULL` ↔ `nil`/`0`/`""`; `pq.Array` cho `key_points`/`tags`; `chapters` và `follow_up_meditation` là `JSONB`. **`SeedIfEmpty` đếm cả dòng `needs_review = true`** — xem Step 1 của flow seed. |
| Repository / Seed parser | `services/appService/repository/dharma_talk_seed.go` | `ParseDharmaTalkSeed`: giải mã `{"talks":[…]}`, kiểm hợp lệ (AC 19), đổi `video` → `youtube` và rút `youtube_id` từ `mediaUrl`, gán `SortOrder` = vị trí. |
| Embedded data | `assets/embed.go`, `assets/data/dharma_talks.json` | Thêm `//go:embed data/dharma_talks.json` → biến `DharmaTalksJSON`. |
| Delivery / Handler | `services/appService/delivery/http/dharma_talk_handler.go` | `List`, `Get`; parse `page`; DTO summary/detail; `published_date` định dạng `2006-01-02`; `0` → `null` cho `duration_minutes`; `nonNil` cho mảng. |
| Delivery / DTO | `delivery/http/dharma_talk_handler.go` | `dharmaTalkSummaryResponse`, `dharmaTalkDetailResponse` (nhúng summary), `dharmaTalkListResponse`, `chapterResponse`, `followUpMeditationResponse` — JSON tag chỉ nằm ở đây. |
| Delivery / Error map | `delivery/http/errors.go` | `ErrDharmaTalkNotFound` → `404 DHARMA_TALK_NOT_FOUND`; `ErrInvalidPage` đã có. |
| Config | `services/appService/config/config.go`, `config.dev.yaml` | `DharmaTalkConfig{PageSize}`, block `dharma_talk:` (dev: 10). |
| Wiring | `services/appService/cmd/main.go` | Dựng repo → usecase → handler; (từ 2026-10-11 không còn seed lúc khởi động; `cmd/seeddharmatalks` gọi `SeedIfEmpty`); đăng ký 2 route. Không dính khối `⚠️ DEV ONLY` nào. |
| Migration | `migrations/0014_dharma_talks.sql` | Bảng `dharma_talks` + index `idx_dharma_talks_order (sort_order, id)`. |
| Data | `assets/data/dharma_talks.json` | 55 bài (sinh bởi phiên nghiên cứu); đã khớp "Seed JSON" bên dưới, không cần sửa file (`sortOrder` thừa bị bỏ qua). |
| API spec | `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` | **Đã có** 2 route (tag DharmaTalks) và schema `DharmaTalkSummary`/`DharmaTalkDetail`; hai file giống hệt nhau. |

Ghi chú route: `GET /v1/dharma-talks` và `GET /v1/dharma-talks/{id}` là hai pattern riêng của `ServeMux`; không có route literal nào tranh `{id}` (không có `top`).

## Bảng `dharma_talks`

| Cột | Kiểu | Ghi chú |
|---|---|---|
| `id` | `VARCHAR(128)` PK | slug ổn định |
| `sort_order` | `INTEGER NOT NULL` | vị trí trong file seed |
| `title`, `speaker` | `TEXT NOT NULL` | |
| `speaker_initials` | `VARCHAR(8) NOT NULL DEFAULT ''` | |
| `category` | `VARCHAR(128) NOT NULL DEFAULT ''` | |
| `season` | `VARCHAR(64)` | NULL = không gắn dịp |
| `published_date` | `DATE` | NULL = chưa biết |
| `duration_minutes` | `INTEGER` | NULL = chưa biết; `CHECK (duration_minutes > 0)` |
| `summary` | `TEXT NOT NULL DEFAULT ''` | |
| `key_points`, `tags` | `TEXT[] NOT NULL DEFAULT '{}'` | |
| `quote`, `quote_source` | `TEXT NOT NULL DEFAULT ''` | |
| `language` | `VARCHAR(8) NOT NULL DEFAULT 'vi'` | |
| `media_type` | `VARCHAR(16) NOT NULL` | `CHECK IN ('youtube','audio','text')` |
| `youtube_id` | `VARCHAR(16)` | NULL trừ khi `media_type = 'youtube'` |
| `audio_url` | `TEXT` | NULL trừ khi `media_type = 'audio'` |
| `chapters` | `JSONB NOT NULL DEFAULT '[]'` | `[{start_seconds, title}]` |
| `follow_up_meditation` | `JSONB` | NULL = không có thẻ thiền |
| `needs_review` | `BOOLEAN NOT NULL DEFAULT false` | true = ẩn khỏi API |
| `review_note` | `TEXT NOT NULL DEFAULT ''` | lý do bị cờ, chỉ để người duyệt |
| `source_name`, `source_url`, `license` | `TEXT NOT NULL DEFAULT ''` | |
| `created_at`, `updated_at` | `TIMESTAMPTZ NOT NULL DEFAULT now()` | |

Thêm `CHECK` ràng buộc nhất quán phương tiện (youtube ⇒ `youtube_id` NOT NULL và `audio_url` NULL; audio ⇒ ngược lại; text ⇒ cả hai NULL), để dữ liệu sai bị chặn ngay cả khi ghi tay bằng SQL.

## Seed JSON (`assets/data/dharma_talks.json`)

`{"talks":[…]}`. Mỗi phần tử giữ đúng các trường file hiện có (`id`, `title`, `speaker`, `speakerInitials`, `category`, `publishedDate`, `durationMinutes`, `summary`, `keyPoints`, `quote`, `quoteSource`, `tags`, `language`, `mediaType`, `mediaUrl`, `sourceName`, `sourceUrl`, `license`, `needsReview`, `reviewNote`) và **thêm hai trường tuỳ chọn** `season` (string, vắng = không có) và `chapters` (`[{startSeconds, title}]`, vắng = rỗng). `sortOrder` trong file bị bỏ qua — parser gán theo vị trí.

Ánh xạ trong `ParseDharmaTalkSeed`:

| `mediaType` trong file | `media_type` | Lấy từ `mediaUrl` |
|---|---|---|
| `video` | `youtube` | `youtube_id` rút từ `youtube.com/watch?v=<id>` hoặc `youtu.be/<id>` (đúng 11 ký tự `[A-Za-z0-9_-]`) |
| `audio` | `audio` | `audio_url` = nguyên `mediaUrl` (`http`/`https`) |
| `text` | `text` | bỏ qua `mediaUrl` — cả hai cột để NULL |

`needsReview`/`reviewNote` được lưu vào cột để người duyệt gỡ cờ bằng SQL; không trả ra API.

## Request flow

`Get` chỉ một câu SQL nên giữ dạng ASCII; `List` (đếm rồi lấy trang) và seed nằm ở Step-by-step Flow và `flow.md`.

```
GET /v1/dharma-talks/{id}
  └─ public
       └─ DharmaTalkHandler.Get                           (delivery/http)
            └─ DharmaTalkUsecase.Get(ctx, id)             (usecase)
                 └─ DharmaTalkRepository.FindByID(id)     (repository)
                      └─ SELECT … FROM dharma_talks WHERE id = $1 AND needs_review = false
                           └─ sql.ErrNoRows → ErrDharmaTalkNotFound → 404 DHARMA_TALK_NOT_FOUND
```

> Xem `flow.md` cho sơ đồ sequence Mermaid của hai flow nhiều bước bên dưới (mirror 1:1).

## Step-by-step Flow — Danh sách phân trang (`GET /v1/dharma-talks?page=N`)

```
Step 1 — Đọc và kiểm tra `page` (handler)
- Trigger: request đến route công khai
- Pre-condition: —
- Call: r.URL.Query().Get("page"); rỗng → page = 1, ngược lại strconv.Atoi
    - Success → Step 2
    - Failure (không phải số nguyên, kể cả tràn int) → writeDomainError(ErrInvalidPage) → 400 INVALID_PAGE, details.field = "page"

Step 2 — Kiểm tra khoảng của `page` (usecase)
- Trigger: handler gọi DharmaTalkUsecase.List(ctx, page)
- Pre-condition: page là số nguyên
- Call: page < 1 hoặc page > math.MaxInt32
    - Success → Step 3 (offset = (page-1) * page_size)
    - Failure → domain.ErrInvalidPage → 400 INVALID_PAGE

Step 3 — Đếm số bài hiển thị được (repository)
- Trigger: usecase gọi DharmaTalkRepository.List(offset, limit)
- Pre-condition: offset/limit hợp lệ
- Call: SELECT count(*) FROM dharma_talks WHERE needs_review = false
    - Success → total, sang Step 4
    - Failure (lỗi DB) → error bọc %w → 500 INTERNAL_ERROR

Step 4 — Lấy trang (repository)
- Trigger: Step 3 xong
- Pre-condition: có total
- Call: SELECT <cột summary> FROM dharma_talks WHERE needs_review = false ORDER BY sort_order ASC, id ASC OFFSET $1 LIMIT $2
    - Success → items (rỗng nếu offset vượt cuối), sang Step 5
    - Failure (lỗi DB / scan) → error bọc %w → 500 INTERNAL_ERROR
    - Transaction boundary: không có — Step 3 và Step 4 là hai câu lệnh độc lập, không chung snapshot

Step 5 — Dựng trang và trả response (usecase + handler)
- Trigger: repository trả về
- Pre-condition: items, total
- Call: items nil → []; total_pages = (total + page_size - 1) / page_size; handler map sang DTO summary (duration 0 → null, published_date định dạng YYYY-MM-DD hoặc null)
    - Success → 200 {items, page, page_size, total, total_pages}
    - Failure → không có nhánh lỗi mới
```

## Step-by-step Flow — Seed lần đầu khi khởi động

```
Step 1 — Đếm số bài đang có (usecase → repository)
- Trigger: cmd/main.go gọi DharmaTalkUsecase.SeedIfEmpty(ctx, load) sau khi kết nối DB
- Pre-condition: DB kết nối được; migration 0014 đã apply
- Call: DharmaTalkRepository.Count → SELECT count(*) FROM dharma_talks (KHÔNG lọc needs_review)
    - Success (n = 0) → Step 2
    - Success (n > 0) → dừng: KHÔNG gọi load, KHÔNG ghi gì
    - Failure (vd. bảng chưa tồn tại) → error bọc %w → main log.Fatalf "failed to seed dharma talks"

Step 2 — Nạp và kiểm tra dữ liệu seed (repository)
- Trigger: n = 0
- Pre-condition: —
- Call: load() = repository.ParseDharmaTalkSeed(assets.DharmaTalksJSON)
    - Success → danh sách domain.DharmaTalk (đã ánh xạ media_type, youtube_id, SortOrder), sang Step 3
    - Failure (JSON hỏng / vi phạm AC 19) → error → log.Fatalf, chưa ghi gì

Step 3 — Chèn các bài còn thiếu (repository)
- Trigger: Step 2 xong
- Pre-condition: seed hợp lệ
- Call: DharmaTalkRepository.InsertMissing → INSERT … ON CONFLICT (id) DO NOTHING cho từng bài (kể cả bài needs_review = true)
    - Success → số dòng thực sự được chèn (55 ở lần đầu), sang Step 4
    - Failure (lỗi DB, kể cả vi phạm CHECK) → rollback toàn bộ → error → log.Fatalf
    - Transaction boundary: mọi INSERT của Step 3 chung MỘT transaction

Step 4 — Ghi log (cmd/main.go)
- Trigger: SeedIfEmpty trả về số bài đã chèn
- Pre-condition: —
- Call: nếu số > 0 → logger.Info("seeded dharma talks on first run", "count", n)
    - Success → tiếp tục khởi động, đăng ký route
    - Failure → không có nhánh lỗi
```

Vì sao Step 1 không lọc `needs_review`: nếu đếm chỉ các dòng hiển thị được, một bảng mà mọi bài đều đang bị cờ sẽ bị coi là rỗng và seed lại — ghi đè quyết định duyệt bằng dữ liệu seed.

## Deliberately stubbed for now (if applicable)

- `follow_up_meditation`: cột, entity và DTO có thật nhưng **không có nguồn dữ liệu** (app chưa có bài thiền) — luôn `NULL` trong seed, API trả `null`.
- `chapters`: cột và DTO có thật, seed hiện rỗng — API trả `[]`.
- `season`: cột và DTO có thật, seed hiện không gán — API trả `null`.
- Lượt nghe: **không** có cột, trường hay route.

## Known gaps carried by this feature

- **Chưa có OKF concept** `docs/knowledge/features/dharmatalk.md` (`il-okf-writer`); code, migration, test, E2E, `openapi.yaml` và ERD đã có.
- **E2E phụ thuộc thời gian khởi động server**: `e2e/harness.go` chờ server sẵn sàng tối đa 10 giây, nhưng lần seed đầu của thư viện kinh (~11 MB) có thể vượt mức đó trên máy chậm, nên cả E2E của dharmatalk lẫn `TestSutraAPI_E2E` thất bại ngẫu nhiên với "did not become ready within 10s". Khi nâng ngưỡng lên 60 giây (tạm thời, đã hoàn lại) thì cả 8 test của dharmatalk đều qua. Ngưỡng này thuộc harness dùng chung, chưa sửa trong lượt này.
- **Test repository cần Postgres và xoá dữ liệu**: `postgres_dharma_talk_repository_test.go` gọi `connectTestDB` (bỏ qua khi `-short`), helper này `TRUNCATE` nhiều bảng của DB `innerlamp` mặc định; lượt này đã chạy trên DB scratch (`TEST_DB_PORT`, `TEST_DB_NAME`), không phải DB dev.
- **19/55 bài bị `needs_review`** nên chỉ 36 bài hiện ra khi có code; trong số 36, `audio` chỉ có tối đa 8 bài và nhiều bài không có `duration_minutes`.
- **Đếm và lấy trang không cùng snapshot** (như `article`).
- **Sửa file JSON không cập nhật DB có sẵn** (như `article`).
- **YouTube nhúng**: video có thể bị gỡ hoặc tắt nhúng sau khi seed; backend không kiểm tra (`youtube_id` chỉ được kiểm định dạng, không kiểm tồn tại).
- **Bản quyền** chưa được xác nhận — xem `requirements.md`.
