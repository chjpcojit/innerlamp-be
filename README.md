# Tâm Đăng (InnerLamp)

> Slogan: **"Tự thắp đèn tâm, tự soi lối về"**

Tài liệu tổng hợp các quyết định và thiết kế đã thống nhất cho pet project ứng dụng Phật pháp, định hướng kiến trúc super app.

---

## 1. Tổng quan

- **Tên app:** Tâm Đăng (VI) / InnerLamp (EN)
- **Ý tưởng gốc:** pet project cá nhân về chủ đề Phật pháp
- **Định hướng kiến trúc:** chuẩn bị theo hướng super app ngay từ đầu — không cần làm hết mọi module ngay, nhưng ranh giới domain phải rõ ràng để mở rộng/tách ra sau mà không phải refactor lớn
- **Định hướng dài hạn:** có kế hoạch tích hợp kinh doanh vào app trong tương lai, không chỉ dừng ở mức pet project thuần túy
- **Tech stack dự kiến:**
  - Backend: Go, Clean Architecture / DDD (theo phong cách đang áp dụng ở Bankaool)
  - Mobile: Flutter
  - Hạ tầng: GCP (Cloud Run, Cloud SQL/Postgres, Cloud Storage)

**Định danh kỹ thuật cần chốt sớm** (tránh phải đổi giữa chừng):
- Domain gợi ý: `innerlamp.app` / `tamdang.app` / `innerlamp.io`
- Bundle ID / package name gợi ý: `com.<tên/công ty>.innerlamp`
- Repo name gợi ý: `innerlamp` hoặc `tam-dang`

---

## 2. Nghiên cứu thị trường (app tương tự)

Đã khảo sát các app Phật giáo/thiền hiện có để rút ra tập tính năng tham khảo, đặc biệt là **Om Mani – Global Buddhism** (miễn phí, không quảng cáo, không thu thập dữ liệu, sync qua iCloud không cần tài khoản) và **OmAhHum** (thiên về đếm thời gian hành trì/lễ lạy, có subscription).

### Bóc tách tính năng Om Mani theo nhóm

**Content / Thư viện**
- Kho kinh điển để tụng (Pháp Cú, Địa Tạng, Vu Lan, Chú Đại Bi, Lăng Nghiêm, Tâm Kinh...)
- Filter theo tông phái
- Thư viện Chư Phật & Bồ Tát
- Bài viết/câu chuyện Phật giáo, mục "Most read this week"
- Explore chùa Việt Nam
- Search toàn app
- AI giải thích ý nghĩa trích dẫn hàng ngày

**Practice / Hành trì (lõi)**
- Chanting Player (hỗ trợ background playback)
- Bell & Wooden Fish (chuông + mõ)
- Mala Beads Counter
- Mantra Recitation theo từng vị Phật/Bồ Tát
- Meditation session có màn hình chuẩn bị
- Live Activities (Lock Screen, chỉ iOS)
- Practice Music (nghe offline)

**Personal tracking**
- Chanting & Meditation History
- Practice Statistics
- Diligence Card
- Buddhist Profile
- Cho xóa lịch sử bất cứ lúc nào

**Calendar & Reminder**
- Lịch âm, ngày lễ Phật giáo
- Ngày tốt/xấu, hợp tuổi, giờ hoàng đạo
- Event nhắc nhở (lặp lại theo tuần)
- Widget home screen

**Emotional/Spiritual space**
- Prayer Room
- Merit Dedication Room / Memorial Room

**Kỹ thuật đáng chú ý**
- Không cần tài khoản, sync qua iCloud (CloudKit) — né hẳn backend auth, nhưng không cross-platform và không có user thật để làm community sau này
- Roadmap thực tế của họ: Chanting/Sutra → Bell & Mala → Meditation → History/Statistics → Calendar/Event → Search → AI/Explore (build "practice" trước, "discovery" sau, "AI/personalization" cuối)
- App size 135MB dù ít video — khả năng do audio bundle sẵn trong app thay vì stream

---

## 3. Quyết định kiến trúc

Khác với Om Mani, Tâm Đăng chọn:

| Quyết định | Tâm Đăng | Om Mani |
|---|---|---|
| Auth | Auth thật (JWT; định danh **số điện thoại + mật khẩu**, verify qua OTP) | Không cần tài khoản (iCloud ẩn danh) |
| Content | Tải từ server, sync theo `updated_since` | Bundle sẵn trong app |
| Lý do | Mở đường cho community/dana và kinh doanh sau này; cập nhật nội dung không cần release app mới | Đơn giản, không cross-platform |

### Module hóa (modular monolith)

Chia theo bounded context, mỗi module giữ nguyên Clean Architecture (domain → usecase → repository), giao tiếp nội bộ qua interface/event bus thay vì network — dễ tách thành service riêng sau này nếu cần.

- `identity` — auth, user profile
- `content` — kinh điển, thần chú, Chư Phật/Bồ Tát, audio
- `journal` — lịch sử tụng/thiền, thống kê hành trì
- `calendar` — lịch âm, ngày lễ, nhắc nhở
- `community` (giai đoạn sau) — thảo luận, chia sẻ
- `dana` (giai đoạn sau) — cúng dường, donation

### Cấu trúc thư mục Go (backend)

Thực tế hiện tại (khác bản đề xuất `/cmd/api` + `/internal` ban đầu): `pkg/` là shared framework, `services/appService/` là service duy nhất theo Clean Architecture. Cây thư mục thật xem `AGENTS.md` §2.

```text
/pkg                      shared infrastructure (config, database, jwt, sms, utils, validation, ...)
/services/appService/     service duy nhất
  cmd/  config/  domain/  usecase/  repository/  delivery/http/  e2e/
/migrations/              SQL migrations (0001–0003)
```

### Cấu trúc thư mục Flutter (mobile)

Pattern shell + feature module — mỗi tính năng độc lập, giao tiếp với shell qua interface rõ ràng:

```
/lib
  /shell           // navigation, theme, DI container
  /features
    /content
    /journal
    /calendar
  /core            // shared widgets, API client
```

### Thứ tự xây dựng đề xuất

1. `identity` trước (mọi module khác cần user context)
2. `content`
3. `journal`, `calendar`
4. `community, `dana` — chỉ khi có nhu cầu thật

---

## 4. Module `identity`

**Trạng thái: đã implement US-001 (đăng ký) + US-002 (đăng nhập).** Nguồn sự thật:
- `docs/specs/features/onboarding/requirements.md` — US-001: đăng ký 2 bước qua OTP (`request-otp` → `verify-otp`), endpoint dùng chung `resend-otp`
- `docs/specs/features/login/requirements.md` — US-002: `login` / `refresh` / `logout`
- `docs/specs/api/openapi.yaml` — contract 6 route (nguồn cho client)
- `migrations/README.md` + `migrations/0001`–`0003` — schema thật
- `docs/knowledge/features/{register,login}.md` — tài liệu OKF sau implement

Khác bản thiết kế ban đầu: định danh là **số điện thoại + mật khẩu** (đã verify OTP), không phải email/password; session nằm ở `user_sessions`, không phải bảng `refresh_tokens`. Bảng thật: `users`, `pending_registration`, `otp_codes`, `otp_send_limits`, `user_devices`, `user_sessions`, `login_attempts`.

### API endpoints (thật)

```
POST /v1/auth/register/request-otp   { phone_number, email, password, dharma_name?, ordination_place? }
POST /v1/auth/register/verify-otp    { phone_number, otp_code }         → 201 + access/refresh token
POST /v1/auth/resend-otp             { phone_number, purpose }          (dùng chung mọi purpose; hiện chỉ register)
POST /v1/auth/login                  { phone_number, password, device } → access/refresh token
POST /v1/auth/refresh                { refresh_token }                  → cặp token mới (rotation)
POST /v1/auth/logout                 Authorization: Bearer <access_token> → 200 {"success": true}
```

- Access token JWT: 15 phút. Refresh token: **7 ngày** (`JWTConfig.RefreshTokenExpiration` = `168h`), **rotate mỗi lần dùng**; dùng lại token cũ → revoke session của thiết bị đó (chống replay).
- OTP: TTL 90 giây; tối đa 4 lượt gửi rồi khoá-gửi 30 phút; nhập sai 3 lần khoá xác thực 30 phút; payload `pending_registration` sống 24h.
- ⚠️ Dev-only: OTP hardcode `111111`, SMS chỉ log qua `pkg/sms/logsms`, secret đọc từ YAML (`⚠️ DEV ONLY`/`⚠️ DEV CONFIG` trong `cmd/main.go`).

> `GET/PATCH /v1/users/me` (profile/avatar) **chưa được implement**; kinh điển (`sutra`) và Phật/Bồ Tát (`deity`) đã có API đọc riêng, phần đồng bộ delta/audio của module `content` thì chưa — xem mục 5 và mục 7.

---

## 5. Module `content`

> ⚠️ **Chưa implement.** Phần này là thiết kế dự kiến; spec ở `docs/specs/features/{sutra,deity,temple,article}/`. Hiện chỉ module `identity` (US-001/US-002) có code.

### Schema Postgres (dự kiến)

```sql
CREATE TABLE bodhisattvas (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            VARCHAR(150) UNIQUE NOT NULL,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    image_url       TEXT,
    sect            VARCHAR(50),          -- nullable: vị được tôn kính chung nhiều tông phái thì để trống
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sutras (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            VARCHAR(150) UNIQUE NOT NULL,
    title           VARCHAR(255) NOT NULL,
    sect            VARCHAR(50),          -- Nam Tông / Bắc Tông / Kim Cương Thừa (gồm Mật Tông)...
    language        VARCHAR(10) NOT NULL DEFAULT 'vi',
    body            TEXT NOT NULL,        -- nội dung kinh, có thể markdown
    audio_url       TEXT,                 -- Cloud Storage URL, nullable
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE mantras (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            VARCHAR(150) UNIQUE NOT NULL,
    title           VARCHAR(255) NOT NULL,
    sect            VARCHAR(50),          -- nullable, cùng quy ước với sutras/bodhisattvas
    body            TEXT NOT NULL,
    audio_url       TEXT,
    bodhisattva_id  UUID REFERENCES bodhisattvas(id),
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sutras_updated_at ON sutras(updated_at);
CREATE INDEX idx_mantras_updated_at ON mantras(updated_at);
CREATE INDEX idx_bodhisattvas_updated_at ON bodhisattvas(updated_at);
```

Lưu ý thứ tự migration: tạo `bodhisattvas` trước vì `mantras` reference tới nó.

### API endpoints (dự kiến)

```
GET  /v1/content/sutras?updated_since=<RFC3339>&sect=<optional>
GET  /v1/content/sutras/:slug
GET  /v1/content/mantras?updated_since=<RFC3339>
GET  /v1/content/bodhisattvas?updated_since=<RFC3339>
GET  /v1/content/bodhisattvas/:slug
```

Response dạng delta sync:

```json
{
  "items": [ { "id": "...", "slug": "...", "version": 2, "updated_at": "..." } ],
  "synced_at": "2026-08-18T10:00:00Z"
}
```

Client lưu `synced_at` local, gọi lại với `updated_since=<synced_at cũ>` để chỉ nhận phần thay đổi. Content endpoints **không cần auth** (public read).

### Chiến lược lưu trữ & phân phối

- Text content (kinh, thần chú) lưu trong Postgres
- Audio files lưu Cloud Storage, serve qua signed URL (giai đoạn cá nhân) → Cloud CDN khi traffic tăng
- Client cache local (SQLite/Hive trong Flutter), sync incremental theo `updated_since`, hoạt động offline sau lần sync đầu

---

## 6. Nguồn dữ liệu nội dung

### Các nguồn tham khảo đã khảo sát

- **SuttaCentral** — kinh tạng Pali/Sanskrit, có API + data trên GitHub, license mở, nhưng chủ yếu tiếng Anh/Pali, thiếu bản dịch tiếng Việt
- **Access to Insight** — kinh Nam Tông tiếng Anh, cho phép reuse phi thương mại có ghi nguồn
- **Thư Viện Hoa Sen, Rộng Mở Tâm Hồn** — kho tư liệu tiếng Việt lớn, nhưng không có API, phải tự thu thập
- **Làng Mai** — tư liệu theo pháp môn Làng Mai, bản quyền thuộc Unified Buddhist Church

### ⚠️ Vấn đề bản quyền — QUAN TRỌNG, cần xử lý trước khi mở rộng public

**Quyết định hiện tại:** tạm gác vấn đề bản quyền để tập trung hoàn thiện backend trước, sẽ xử lý sau khi có sản phẩm.

**Cần nhớ khi quay lại vấn đề này:**
- Bản quyền áp dụng cho **mọi hình thức phân phối công khai**, không phụ thuộc việc có thu phí hay không — "public nhưng miễn phí" **không** đồng nghĩa với "không dính bản quyền"
- Nguyên tác Pali/Sanskrit/Hán cổ: public domain, an toàn tuyệt đối
- Bản dịch tiếng Việt hiện có (HT. Thích Minh Châu, HT. Thích Trí Tịnh...): là tác phẩm phái sinh, có bản quyền riêng — đây là điểm rủi ro chính
- Audio tụng niệm: tuyệt đối không lấy từ nguồn có sẵn, cần TTS tự tạo hoặc thu âm riêng
- 3 hướng an toàn: (1) xin phép trực tiếp từ chùa/dịch giả/Giáo hội, (2) tự dịch hoặc thuê dịch mới từ nguyên tác public domain, (3) dùng nguồn đã tuyên bố public domain/CC rõ ràng (cần verify kỹ, không tự suy đoán)
- Schema nên có sẵn field tracking provenance khi triển khai thật: `source_type`, `translator`, `source_reference`, `license_note`, `license_document_url`

---

## 7. Chạy local (dev)

Stack local nằm ở `docker-compose.yml`:

- `postgres` khớp nguyên `services/appService/config/config.dev.yaml` + `migrate` one-shot apply `migrations/0001`–`0003` bằng `psql -f`.
- `loki` + `alloy` + `grafana` để gom log (xem mục dưới).

Tài liệu đầy đủ: [`docs/development/local-stack.md`](docs/development/local-stack.md) (mô tả cấu hình compose + hướng dẫn chạy) và [`docs/development/database-cli.md`](docs/development/database-cli.md) (CLI Docker + xem dữ liệu Postgres).

```bash
# Từ repo root (bắt buộc — cmd/main.go đọc config theo đường dẫn tương đối)
docker compose up -d          # Postgres + migrations + Loki/Alloy/Grafana

# Chạy app và tee log ra file để Alloy tail — appService chạy trên host, không
# nằm trong compose (xem "Vì sao không gộp appService vào compose" bên dưới)
go run ./services/appService/cmd 2>&1 | tee -a observability/logs/appservice.log
```

Hoặc dùng `Makefile` (chạy `make help` để xem hết target):

```bash
make up           # dựng stack
make run          # chạy appService + tee log (tự gọi `make up` trước)
make db-accounts  # xem tài khoản đã đăng ký
make reset        # xoá sạch Postgres + Loki + Grafana rồi dựng lại
```

Server nghe ở `http://localhost:8080`. Sau khi thêm migration mới, apply lại không cần tạo lại Postgres:

```bash
docker compose run --rm migrate
```

Xoá sạch Postgres + Loki + Grafana để chạy lại từ đầu:

```bash
docker compose down -v && docker compose up -d
```

Vài điểm cần biết khi tích hợp frontend:

- **CORS**: middleware hardcode `Access-Control-Allow-Origin: *` — FE gọi thẳng, không cần cấu hình thêm.
- **OTP**: đang hardcode `111111`; SMS chỉ ghi log ra stdout của server (`pkg/sms/logsms`), không gửi thật.
- **Contract**: `docs/specs/api/openapi.yaml` (server base `/v1`) là nguồn để generate API client.
- **Token**: access 15 phút, refresh 7 ngày và **rotate mỗi lần dùng**; dùng lại refresh token cũ sẽ revoke session thiết bị đó.
- ⚠️ Các điểm `⚠️ DEV ONLY`/`⚠️ DEV CONFIG` trong `cmd/main.go` (secret đọc từ `dev_secrets` trong YAML) **chỉ dành cho local**, không dùng cho môi trường thật.

### Gom log: Loki + Grafana (+ Alloy)

appService chạy trên host nên Alloy **không** đọc được stdout của nó như log container. Luồng log là:

```
go run (host) --tee--> observability/logs/*.log --tail--> Alloy --> Loki --> Grafana
```

- **Grafana**: http://localhost:3000 — datasource Loki đã provision sẵn; mở **Explore**, query `{service="appservice"}`.
- **Loki API**: http://localhost:3100, **Alloy UI**: http://localhost:12345.
- Query nhanh không cần UI:

```bash
curl -G 'http://localhost:3100/loki/api/v1/query_range' \
  --data-urlencode 'query={service="appservice"}' --data-urlencode 'limit=20'
```

- File config: `observability/loki/config.yml`, `observability/alloy/config.alloy`, `observability/grafana/provisioning/datasources/loki.yml`.
- Log ghi thêm (append) vào `observability/logs/` — đã gitignore qua `*.log`.

Grafana yêu cầu đăng nhập (`admin` / `admin` mặc định, hoặc mật khẩu đã đổi trong volume `grafana-data`). Chỉ dùng local, không expose port 3000 ra ngoài.

### Vì sao không gộp appService vào compose

`postgres` publish `5432:5432`, nên process trên host vẫn kết nối được qua `localhost:5432` — đúng như `config.dev.yaml` khai báo. Việc app nằm trong Docker hay không **không** quyết định nó có chạm được Postgres hay không. Giữ app trên host để vòng lặp sửa–chạy nhanh (`go run` restart vài giây thay vì rebuild image mỗi lần sửa một dòng), và vì repo chưa có Dockerfile nào. Hệ quả duy nhất là log phải `tee` ra file cho Alloy tail, như ở trên.

Kiểm thử nhanh không cần Docker: `go test -short ./...` (bỏ qua e2e dùng testcontainers).

---

## 8. Việc cần làm tiếp theo

- [ ] Chốt domain, bundle ID, repo name
- [x] Setup Postgres, chạy migration cho `identity` (`migrations/0001`–`0003`)
- [x] Implement module `identity` phần register/login/refresh/logout (US-001/US-002)
- [ ] Implement nốt `identity`: `GET/PATCH /v1/users/me` (profile/avatar) + quên/đặt lại mật khẩu
- [ ] Đồng bộ delta + `audio_url` cho nội dung kinh (kinh/Phật/Bồ Tát đã có API đọc riêng: `sutra`, `deity`)
- [ ] Dựng Flutter shell + feature module đầu tiên (`content`)
- [ ] Quay lại xử lý vấn đề bản quyền nguồn nội dung trước khi public rộng rãi
- [ ] Soạn văn bản xin phép sử dụng nội dung (nếu chọn hướng xin phép chùa/dịch giả)