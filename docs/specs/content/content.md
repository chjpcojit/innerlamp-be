# User Stories — Module `content`

Tài liệu user story riêng cho module `content` (kinh điển, thần chú, Chư Phật/Bồ Tát) của Tâm Đăng (InnerLamp). Đánh số tiếp nối từ file user story chính (`tam-dang-user-stories.md`, đã có US-001 đến US-004 thuộc module `identity`).

---

## US-005: Xem danh sách kinh điển

**Module:** `content`
**Priority:** P0 (tính năng lõi đầu tiên người dùng thấy sau khi đăng nhập)

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem danh sách các bài kinh có sẵn trong app**,
> Để **chọn ra bài kinh mình muốn đọc hoặc tụng**.

### Acceptance Criteria

1. **Given** tôi mở màn hình "Kinh điển", **when** app tải xong, **then** tôi thấy danh sách các bài kinh với tối thiểu tên bài kinh và tông phái (nếu có).
2. **Given** tôi mở màn hình này lần đầu tiên (chưa từng sync), **when** app gọi API, **then** toàn bộ danh sách được tải về và lưu cache local.
3. **Given** tôi đã từng mở màn hình này trước đó (đã có cache), **when** tôi mở lại, **then** app hiển thị ngay danh sách từ cache local (không chờ network), đồng thời âm thầm gọi API để đồng bộ phần thay đổi (nếu có) ở nền.
4. **Given** tôi không có kết nối mạng, **when** tôi mở màn hình này, **then** tôi vẫn xem được danh sách đã cache trước đó — không hiện lỗi chặn toàn màn hình.
5. **Given** danh sách kinh điển có nhiều tông phái khác nhau, **when** tôi xem danh sách, **then** tôi có thể lọc theo tông phái (Nam Tông/Bắc Tông/Kim Cương Thừa).
6. **Given** server có bài kinh mới hoặc bài kinh đã được cập nhật nội dung, **when** app đồng bộ ở nền (AC3), **then** danh sách/cache local được cập nhật mà không cần tôi thao tác gì thêm.

### Out of scope (giai đoạn này)

- Search theo từ khóa trong danh sách — cần user story riêng
- Đánh dấu yêu thích / bookmark bài kinh — cần user story riêng
- Sắp xếp theo "phổ biến nhất" / "đọc gần đây" — cần dữ liệu tracking từ module `journal`, để sau

### Ghi chú kỹ thuật liên quan

- Endpoint: `GET /v1/content/sutras?updated_since=<RFC3339>&sect=<optional>`
- Không yêu cầu auth (public read), nhưng vẫn đặt trong luồng "đã đăng nhập" vì đây là màn hình sau khi vào app chính
- Client cache bằng SQLite/Hive (Flutter), lưu `synced_at` để lần sau chỉ tải phần thay đổi (delta sync)
- Thuộc `ContentService` (hoặc `SutraService` nếu tách nhỏ hơn) trong `usecase/`

---

## US-006: Xem chi tiết một bài kinh

**Module:** `content`
**Priority:** P0

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem toàn văn một bài kinh cụ thể**,
> Để **đọc hoặc tụng theo nội dung đầy đủ**.

### Acceptance Criteria

1. **Given** tôi chọn một bài kinh từ danh sách (US-005), **when** màn hình chi tiết mở ra, **then** tôi thấy toàn văn nội dung (`body`), tiêu đề, tông phái.
2. **Given** bài kinh có `audio_url`, **when** tôi mở chi tiết, **then** tôi thấy nút phát audio tụng đi kèm; **given** bài kinh không có audio, **then** không hiện nút phát, không báo lỗi.
3. **Given** tôi đã xem bài kinh này trước đó (có trong cache), **when** tôi mở lại mà không có mạng, **then** nội dung vẫn hiển thị đầy đủ từ cache.
4. **Given** nội dung bài kinh trên server đã được cập nhật (version tăng) từ lần cuối tôi xem, **when** tôi mở lại màn hình chi tiết và có mạng, **then** app tải bản mới nhất, không hiển thị bản cache cũ đã lỗi thời.
5. **Given** tôi đang đọc bài kinh, **when** tôi phóng to/thu nhỏ cỡ chữ, **then** tùy chỉnh này được áp dụng, phù hợp cho việc đọc kinh lâu (accessibility, đặc biệt với người lớn tuổi).

### Out of scope (giai đoạn này)

- Auto wooden-fish-tap theo nhịp đọc kinh (đồng bộ text scroll + audio timing) — phức tạp, để giai đoạn sau, thuộc phạm vi module `practice`
- Đánh dấu yêu thích / bookmark — cần user story riêng
- Chia sẻ bài kinh ra ngoài app — cần user story riêng

### Ghi chú kỹ thuật liên quan

- Endpoint: `GET /v1/content/sutras/:slug`
- Không auth, public read
- Client kiểm tra `version`/`updated_at` local so với response để quyết định dùng cache hay tải mới (AC4)

---

## US-007: Xem danh sách thần chú

**Module:** `content`
**Priority:** P0

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem danh sách các thần chú có sẵn trong app**,
> Để **chọn thần chú mình muốn trì tụng**.

### Acceptance Criteria

1. **Given** tôi mở màn hình "Thần chú", **when** app tải xong, **then** tôi thấy danh sách thần chú, mỗi mục hiển thị tên và vị Phật/Bồ Tát liên quan (nếu có gán `bodhisattva_id`).
2. **Given** một thần chú không gán `bodhisattva_id`, **when** hiển thị trong danh sách, **then** app không báo lỗi, chỉ đơn giản không hiện thông tin vị Bồ Tát liên quan.
3. **Given** tôi muốn xem thần chú gắn với một vị cụ thể, **when** tôi lọc theo vị Phật/Bồ Tát, **then** danh sách chỉ hiển thị các thần chú thuộc vị đó.
4. **Given** tôi không có kết nối mạng nhưng đã từng sync trước đó, **when** tôi mở màn hình này, **then** danh sách vẫn hiển thị từ cache — cùng hành vi offline-first như US-005.
5. **Given** server có thần chú mới hoặc đã cập nhật, **when** app đồng bộ ở nền, **then** danh sách local được cập nhật theo cơ chế delta sync (`updated_since`).

### Out of scope (giai đoạn này)

- Đếm số lần trì tụng (Mala Beads Counter) — thuộc module `practice`, không phải `content`
- Search theo từ khóa — để sau, giống US-005

### Ghi chú kỹ thuật liên quan

- Endpoint: `GET /v1/content/mantras?updated_since=<RFC3339>&bodhisattva_id=<optional>`
- Không auth, public read, cùng pattern delta sync như `sutras`

---

## US-008: Xem chi tiết một thần chú

**Module:** `content`
**Priority:** P0

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem nội dung đầy đủ của một thần chú cụ thể**,
> Để **trì tụng theo đúng nội dung**.

### Acceptance Criteria

1. **Given** tôi chọn một thần chú từ danh sách (US-007), **when** màn hình chi tiết mở ra, **then** tôi thấy toàn văn (`body`), tên thần chú, và thông tin vị Phật/Bồ Tát liên quan (nếu có) kèm link để xem chi tiết vị đó (US-010).
2. **Given** thần chú có `audio_url`, **when** tôi mở chi tiết, **then** tôi thấy nút phát audio; nếu không có thì không hiện nút, không báo lỗi.
3. **Given** tôi mở chi tiết mà không có mạng, **when** nội dung đã có trong cache, **then** vẫn hiển thị đầy đủ.

### Out of scope (giai đoạn này)

- Tích hợp trực tiếp với Mala Beads Counter ngay tại màn hình chi tiết — cần thiết kế phối hợp với module `practice`, để user story riêng khi làm `practice`

### Ghi chú kỹ thuật liên quan

- Endpoint: `GET /v1/content/mantras/:slug` (bổ sung, chưa có trong danh sách endpoint ban đầu — cần thêm route này)

---

## US-009: Xem danh sách Chư Phật & Bồ Tát

**Module:** `content`
**Priority:** P0

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem danh sách các vị Phật và Bồ Tát trong thư viện**,
> Để **tìm hiểu và chọn vị mình muốn hướng tới trong tu tập**.

### Acceptance Criteria

1. **Given** tôi mở màn hình "Chư Phật & Bồ Tát", **when** app tải xong, **then** tôi thấy danh sách với tên và ảnh đại diện (`image_url`) của từng vị.
2. **Given** một vị không có `image_url`, **when** hiển thị trong danh sách, **then** app hiện ảnh placeholder mặc định, không hiện lỗi hay khoảng trống vỡ layout.
3. **Given** tôi không có kết nối mạng nhưng đã từng sync trước đó, **when** tôi mở màn hình này, **then** danh sách vẫn hiển thị từ cache.
4. **Given** server cập nhật danh sách, **when** app đồng bộ ở nền, **then** danh sách local cập nhật theo delta sync, giống US-005/US-007.

### Ghi chú kỹ thuật liên quan

- Endpoint: `GET /v1/content/bodhisattvas?updated_since=<RFC3339>`
- Không auth, public read

---

## US-010: Xem chi tiết một vị Phật/Bồ Tát

**Module:** `content`
**Priority:** P0

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **xem thông tin chi tiết và các thần chú liên quan đến một vị Phật/Bồ Tát cụ thể**,
> Để **hiểu rõ hơn về vị đó và dễ dàng tìm thần chú để trì tụng theo**.

### Acceptance Criteria

1. **Given** tôi chọn một vị từ danh sách (US-009), **when** màn hình chi tiết mở ra, **then** tôi thấy tên, mô tả (`description`), ảnh, và danh sách các thần chú liên quan (nếu có).
2. **Given** vị này không có thần chú nào được gán, **when** tôi xem chi tiết, **then** phần "Thần chú liên quan" ẩn đi hoặc hiện thông báo nhẹ nhàng "Chưa có thần chú liên quan", không phải lỗi.
3. **Given** tôi chọn một thần chú trong danh sách liên quan, **when** tôi nhấn vào, **then** tôi được điều hướng sang màn hình chi tiết thần chú đó (US-008).
4. **Given** tôi mở chi tiết mà không có mạng, **when** thông tin đã có trong cache, **then** vẫn hiển thị đầy đủ.

### Ghi chú kỹ thuật liên quan

- Endpoint chính: `GET /v1/content/bodhisattvas/:slug`
- Danh sách thần chú liên quan: có thể trả kèm trong cùng response (`GET /v1/content/bodhisattvas/:slug` trả thêm field `mantras: [...]`) để giảm số lần gọi API, thay vì bắt client tự gọi `GET /v1/content/mantras?bodhisattva_id=...` riêng — quyết định cụ thể nên chốt khi thiết kế response schema chi tiết