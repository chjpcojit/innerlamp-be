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
| Auth | Auth thật (JWT, email/password) | Không cần tài khoản (iCloud ẩn danh) |
| Content | Tải từ server, sync theo `updated_since` | Bundle sẵn trong app |
| Lý do | Mở đường cho community/dana và kinh doanh sau này; cập nhật nội dung không cần release app mới | Đơn giản, không cross-platform |

### Module hóa (modular monolith)

Chia theo bounded context, mỗi module giữ nguyên Clean Architecture (domain → usecase → repository), giao tiếp nội bộ qua interface/event bus thay vì network — dễ tách thành service riêng sau này nếu cần.

- `identity` — auth, user profile
- `content` — kinh điển, thần chú, Chư Phật/Bồ Tát, audio
- `practice` — timer thiền, chuông/mõ, đếm chuỗi hạt, chanting player
- `journal` — lịch sử tụng/thiền, thống kê hành trì
- `calendar` — lịch âm, ngày lễ, nhắc nhở
- `community` (giai đoạn sau) — thảo luận, chia sẻ
- `dana` (giai đoạn sau) — cúng dường, donation

### Cấu trúc thư mục Go (backend)

```
/cmd/api
/internal
  /identity
    /domain
    /usecase
    /repository
  /content
  /practice
  /journal
  /calendar
  /platform      // shared: db, pubsub, auth middleware
/pkg             // shared kernel nếu cần expose ra ngoài sau này
```

### Cấu trúc thư mục Flutter (mobile)

Pattern shell + feature module — mỗi tính năng độc lập, giao tiếp với shell qua interface rõ ràng:

```
/lib
  /shell           // navigation, theme, DI container
  /features
    /practice
    /content
    /journal
    /calendar
  /core            // shared widgets, API client
```

### Thứ tự xây dựng đề xuất

1. `identity` trước (mọi module khác cần user context)
2. `content`
3. `practice` (chủ yếu logic client, gọi API `content`)
4. `journal`, `calendar`
5. `community`, `dana` — chỉ khi có nhu cầu thật

---

## 4. Module `identity`

### Schema Postgres

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    dharma_name     VARCHAR(100),
    avatar_url      TEXT,
    email_verified  BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(255) NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
```

Lưu ý: `token_hash` lưu SHA-256 hash của refresh token, không lưu raw token — DB leak không đủ để hijack session.

### API endpoints

```
POST   /v1/auth/register        { email, password, dharma_name? }
POST   /v1/auth/login           { email, password } → { access_token, refresh_token }
POST   /v1/auth/refresh         { refresh_token } → { access_token, refresh_token }
POST   /v1/auth/logout          { refresh_token } → revoke
GET    /v1/users/me             (auth required) → user profile
PATCH  /v1/users/me             { dharma_name?, avatar_url? }
```

- Access token JWT: sống ngắn (15 phút)
- Refresh token: sống dài (30 ngày), **rotate mỗi lần dùng** (refresh token cũ bị revoke ngay khi cấp cái mới, chống replay attack)

---

## 5. Module `content`

### Schema Postgres

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

### API endpoints

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

## 7. Việc cần làm tiếp theo

- [ ] Chốt domain, bundle ID, repo name
- [ ] Setup Postgres, chạy migration cho `identity` + `content`
- [ ] Implement module `identity` (register/login/refresh/logout/me)
- [ ] Implement module `content` (CRUD + delta sync API)
- [ ] Dựng Flutter shell + feature module đầu tiên (`practice` hoặc `content`)
- [ ] Quay lại xử lý vấn đề bản quyền nguồn nội dung trước khi public rộng rãi
- [ ] Soạn văn bản xin phép sử dụng nội dung (nếu chọn hướng xin phép chùa/dịch giả)