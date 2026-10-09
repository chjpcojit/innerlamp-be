# User Stories — Module `journal`

Tài liệu user story riêng cho module `journal` (lịch sử tụng/thiền, thống kê hành trì) của Tâm Đăng (InnerLamp).

Đây là module đầu tiên sau `identity` thực sự cần backend/API để ghi nhớ và sync đa thiết bị các phiên tụng/thiền.

---

## US-013: Lưu lại phiên hành trì vừa hoàn thành

**Module:** `journal`
**Priority:** P0 (điều kiện tiên quyết cho mọi tính năng thống kê/lịch sử sau này)

### User Story

> Là một **người dùng vừa hoàn thành một phiên tụng kinh hoặc thiền**,
> Tôi muốn **phiên hành trì đó được tự động lưu lại**,
> Để **theo dõi được quá trình tu tập của mình theo thời gian, đồng bộ được qua nhiều thiết bị**.

### Acceptance Criteria

1. **Given** tôi vừa hoàn thành một phiên nghe/tụng theo audio (US-011) hoặc một phiên thiền có timer, **when** phiên đó kết thúc (audio phát hết, hoặc tôi dừng timer thiền), **then** hệ thống tự động ghi nhận phiên này (loại hành trì, thời lượng, thời điểm) mà không cần tôi thao tác thủ công.
2. **Given** tôi dừng giữa chừng một phiên (chưa nghe hết audio, tắt timer sớm), **when** tôi thoát, **then** hệ thống vẫn ghi nhận phiên đó với thời lượng thực tế đã hành trì (không phải thời lượng dự kiến ban đầu), đánh dấu là phiên chưa hoàn tất.
3. **Given** tôi đang offline khi hoàn thành một phiên, **when** kết nối mạng trở lại, **then** phiên đã lưu local được đồng bộ lên server tự động, không mất dữ liệu.
4. **Given** tôi hoàn thành nhiều phiên trong cùng một ngày, **when** mỗi phiên kết thúc, **then** mỗi phiên được lưu thành một bản ghi riêng biệt, không bị gộp hay ghi đè lên nhau.
5. **Given** phiên hành trì gắn với một bài kinh/thần chú cụ thể (từ US-011), **when** phiên được lưu, **then** bản ghi có liên kết tới bài kinh/thần chú đó, để sau này có thể biết "đã tụng bài nào bao nhiêu lần".
6. **Given** tôi đăng nhập trên thiết bị thứ hai, **when** tôi mở app, **then** các phiên đã lưu từ thiết bị đầu tiên hiển thị đầy đủ (đã sync qua server), không chỉ thấy dữ liệu local của thiết bị hiện tại.

### Out of scope (giai đoạn này)

- Hiển thị lịch sử/thống kê trực quan (biểu đồ, tổng hợp theo tuần/tháng) — cần user story riêng, xây trên dữ liệu US-013 tạo ra
- Diligence Card (thẻ tinh tấn ghi nhận tính đều đặn) — cần user story riêng, tính toán từ dữ liệu này
- Cho phép sửa/xóa thủ công một phiên đã lưu — cần user story riêng
- Ghi nhận Mala Beads Counter (số lần trì tụng theo chuỗi hạt) như một loại phiên riêng — sẽ bổ sung khi có US cho Mala Counter

### Ghi chú kỹ thuật liên quan

- Đây là điểm đầu tiên client cần "nói chuyện" với backend cho việc ghi lịch sử — client gọi API này ngay sau khi một phiên tụng/thiền kết thúc.
- Schema đề xuất:
  ```sql
  CREATE TABLE practice_sessions (
      id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
      session_type      VARCHAR(30) NOT NULL,   -- 'chanting' | 'meditation' | 'mantra'
      content_type      VARCHAR(20),             -- 'sutra' | 'mantra', nullable (meditation timer không gắn content)
      content_id        UUID,                    -- FK mềm tới sutras.id hoặc mantras.id tùy content_type
      duration_seconds  INT NOT NULL,
      is_completed      BOOLEAN NOT NULL DEFAULT true,  -- false nếu dừng giữa chừng (AC2)
      started_at        TIMESTAMPTZ NOT NULL,
      created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
  );

  CREATE INDEX idx_practice_sessions_user_id ON practice_sessions(user_id);
  CREATE INDEX idx_practice_sessions_started_at ON practice_sessions(started_at);
  ```
  Lưu ý: `content_id` là FK mềm (không đặt `REFERENCES` cứng) vì có thể trỏ tới 1 trong 2 bảng khác nhau (`sutras` hoặc `mantras`) tùy `content_type` — ràng buộc này xử lý ở tầng application, không ở tầng DB.
- Endpoint:
  ```
  POST /v1/journal/sessions   (auth required)
  {
    "session_type": "chanting",
    "content_type": "sutra",
    "content_id": "...",
    "duration_seconds": 420,
    "is_completed": true,
    "started_at": "2026-09-10T08:00:00Z"
  }
  ```
- Client cần queue local (ví dụ bảng SQLite riêng đánh dấu "pending sync") để xử lý AC3 — phiên tạo lúc offline phải giữ lại và gửi lại khi có mạng, không được mất nếu request lần đầu thất bại.
- Thuộc `JournalService` trong `usecase/`, đây là module đầu tiên thực sự cần `RequireAuth` middleware bảo vệ toàn bộ route (khác `content` — public read).