# User Stories — Module `practice`

Tài liệu user story riêng cho module `practice` (hành trì: chanting player, chuông/mõ, đếm chuỗi hạt, timer thiền) của Tâm Đăng (InnerLamp). Đánh số tiếp nối từ `tam-dang-user-stories-content.md` (US-005 → US-010).

---

## US-011: Phát audio tụng kinh (Chanting Player)

**Module:** `practice`
**Priority:** P0 (tính năng lõi đầu tiên của `practice`, kết nối trực tiếp với dữ liệu đã có ở `content`)

### User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **phát audio tụng của một bài kinh hoặc thần chú đang xem**,
> Để **tụng theo mà không cần tự đọc, kể cả khi đang làm việc khác hoặc tắt màn hình**.

### Acceptance Criteria

1. **Given** tôi đang xem chi tiết một bài kinh/thần chú có `audio_url` (US-006/US-008), **when** tôi nhấn nút phát, **then** audio bắt đầu phát và hiển thị thanh điều khiển (play/pause, tua, thời gian đã phát/tổng thời lượng).
2. **Given** audio đang phát, **when** tôi tắt màn hình hoặc chuyển sang app khác, **then** audio tiếp tục phát ở nền (background playback) — không bị dừng khi app không ở foreground.
3. **Given** audio đang phát ở nền, **when** tôi mở màn hình khóa (lock screen) hoặc trung tâm điều khiển, **then** tôi thấy media control (tên bài, play/pause, tua) ngay tại đó — không cần mở lại app.
4. **Given** tôi đang phát một bài, **when** tôi mở bài khác và nhấn phát, **then** bài đang phát trước đó dừng lại, bài mới bắt đầu — chỉ một audio phát tại một thời điểm.
5. **Given** tôi có cuộc gọi đến hoặc thông báo âm thanh khác trong lúc đang phát, **when** sự kiện đó xảy ra, **then** audio tự tạm dừng và tự tiếp tục phát sau khi sự kiện kết thúc (audio session interruption handling theo chuẩn hệ điều hành).
6. **Given** audio đã tải/cache trước đó, **when** tôi phát lại mà không có mạng, **then** vẫn phát được bình thường từ cache.
7. **Given** bài kinh/thần chú không có `audio_url`, **when** tôi xem chi tiết, **then** không có nút phát nào hiển thị (đã nêu ở US-006 AC2/US-008 AC2, nhắc lại ở đây vì liên quan trực tiếp).

### Out of scope (giai đoạn này)

- Tốc độ phát tùy chỉnh (0.5x, 1.5x...) — có thể thêm sau
- Hẹn giờ tự tắt phát (sleep timer) — cần user story riêng, có thể gộp chung với US Meditation Timer
- Playlist/phát liên tục nhiều bài — để sau
- Auto wooden-fish-tap đồng bộ theo audio — phức tạp, nêu rõ trong US-006, để giai đoạn sau

### Ghi chú kỹ thuật liên quan

- Flutter: dùng package hỗ trợ background audio + media session chuẩn (`just_audio` kết hợp `audio_service`) thay vì `audio_player` cơ bản — vì cần lock-screen control (AC3) và xử lý audio interruption (AC5), hai việc này chỉ package có tích hợp `audio_service`/`MediaSession` (Android) và `MPNowPlayingInfoCenter` (iOS) mới hỗ trợ đúng chuẩn hệ điều hành.
- Không cần thêm bảng DB mới — audio file đã có sẵn `audio_url` từ bảng `sutras`/`mantras` (module `content`), `practice` chỉ tiêu thụ dữ liệu này, không sở hữu.
- Thuộc `PlaybackService` hoặc `PracticeService` trong `usecase/` phía client (phần lớn logic nằm ở Flutter, backend không cần thêm endpoint riêng cho tính năng này — chỉ dùng lại `audio_url` đã có).