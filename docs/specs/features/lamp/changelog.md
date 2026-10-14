# Changelog — lamp

Lịch sử phiên bản của bộ spec `requirements.md` / `design.md` / `flow.md` / `tasks.md` (cùng một version tại mọi thời điểm).

## v1.3 (2026-10-11)

- `requirements.md`: AC6 và AC7 nêu giới hạn độ dài theo khoá config `lamp.recipient_max_chars` (dev: 100) và `lamp.prayer_max_chars` (dev: 300) thay vì số cứng; ghi chú kỹ thuật "mọi giới hạn nằm ở config" kèm quy tắc trần theo cột `VARCHAR`; câu trả lời đã chốt về độ dài giữ nguyên giá trị, chỉ ghi rõ chúng nằm ở config.
- `design.md`: dòng Config thêm `RecipientMaxChars`, `PrayerMaxChars` và hai hằng trần schema; dòng Usecase lấy cả ba giới hạn từ config; ghi chú cột `VARCHAR` là trần cứng. Không đổi flow.
- `flow.md`: no change needed, verified — chỉ đổi nguồn của hai con số, không thêm bước.
- `tasks.md`: thêm bốn mục `[ ]` (config + yaml + usecase + openapi/web docs) cho việc chuyển hai giới hạn vào config.
- Reason: người dùng yêu cầu đưa mọi giới hạn của thắp đèn vào `config.yaml` (theo nguyên tắc giá trị không nhạy cảm, đổi theo môi trường nằm ở config), thay vì hai hằng 100/300 trong usecase. Không đổi hành vi với giá trị dev hiện tại.

## v1.2 (2026-10-10)

- `requirements.md`: Status chuyển sang đã implement Phase 1–4 và kiểm chứng; ghi chú hợp đồng API đổi sang "đã có trong `openapi.yaml`"; Pending liệt kê phần còn lại (sinh lại `erd.svg`, concept OKF); marker retention được bổ sung giá trị tạm `ON DELETE CASCADE` (marker vẫn mở); thêm ghi chú số migration là `0027`. Không đổi AC nào.
- `design.md`: cập nhật bảng Layers cho khớp tên kiểu thật trong code (`LampInput`, `GardenFilter`, `LightLampParams`, `GardenLamp.DisplayName()`, `Garden` trả `Limit+1` dòng rồi usecase cắt); số migration `0026` → `0027`; `ON DELETE CASCADE` tạm thời trong schema; Step G-4: việc đổi `lighter_name` NULL → `Vô danh` nằm ở handler qua `DisplayName()` (không đổi hành vi, chỉ đúng tầng); Known gaps cập nhật theo thực tế.
- `flow.md`: sơ đồ vườn đèn tách bước map response sang handler cho khớp Step G-4; sơ đồ thắp đèn không đổi.
- `tasks.md`: tick Phase 1–4 và hai mục Phase 5 (E2E, vet/test), tất cả đã được chạy thật. Còn mở: sinh lại `erd.svg` (thiếu `mmdc`), concept OKF. Sửa chữ cho khớp: migration `0027`, cursor nằm trong `lamp_usecase.go`, quyết định `ON DELETE`.
- Reason: implement theo TDD. Hai điều khác spec v1.1: (1) số migration phải là `0027` vì `0026_administrative_units.sql` của công việc khác đã chiếm `0026`; (2) mệnh đề `ON DELETE` được chọn tạm là `CASCADE` để không chặn migration, và marker retention vẫn mở. Cấu trúc flow không đổi, chỉ đổi tầng làm việc quy `Vô danh`.

### Việc theo sau (cập nhật 2026-10-10)

- `docs/specs/db/erd.svg`: sinh lại từ `erd.mmd` bằng `mmdc` (`erd.mmd` đã có bảng `lamps`).
- `docs/knowledge/features/lamp.md` + `index.md` + `log.md`: viết bằng `il-okf-writer` khi flow đã được dùng thật end-to-end.
- Chốt retention (marker còn mở): nếu đổi `ON DELETE`, thêm migration mới, không sửa `0027_lamps.sql`.
- Gói `usecase` không biên dịch với `go test` thường do `genealogy_member_usecase_test.go` (công việc khác); không thuộc feature này nhưng chặn `go test ./services/appService/usecase/` cho mọi người cho tới khi file đó có cài đặt.

## v1.1 (2026-10-10)

- `requirements.md`: đóng marker "phân trang cursor" — anh Đức chốt dùng **cursor** cho vườn đèn, chấp nhận lệch quy ước `page`/`total_pages`; chuyển sang "Open questions (resolved)", bỏ khỏi Pending, đổi ghi chú bảng hợp đồng API từ "đề xuất" sang "đã chốt hình dạng". Còn 2 marker.
- `design.md`: không đổi cấu trúc — câu trả lời xác nhận đúng thiết kế sẵn có (keyset, `cursor`/`next_cursor`/`has_more`); chỉ gỡ marker ở Overview và Known gaps.
- `flow.md`: no change needed, verified — hai sequence diagram đã vẽ đúng cursor.
- `tasks.md`: no change needed, verified — các mục cursor ở Phase 2–4 giữ nguyên; chỉ cập nhật dòng Status (2 marker còn lại).
- Reason: người dùng trả lời câu hỏi phân trang (2026-10-10): đồng ý đề xuất cursor. Câu trả lời gỡ một điểm chưa chắc chắn, không đổi số bước hay cấu trúc flow.

## v1.0 (2026-10-10)

- `requirements.md`: tạo mới. Hai user story `US-LAMP-01` (thắp đèn) và `US-LAMP-02` (xem vườn đèn) với 24 AC theo EARS, bám `FR-001`…`FR-009` của `SRS.md` v0.3; bảng hợp đồng API đề xuất cho `POST /v1/lamps` và `GET /v1/lamps`; ba marker `⚠️ Not confirmed` (hiệu năng/khối lượng, retention + xoá tài khoản, phân trang cursor).
- `design.md`: tạo mới. Bảng layers (domain, usecase, repository, delivery, config, wiring, migration `0026_lamps.sql`, openapi, ERD, E2E), schema `lamps` + ba index, hai Step-by-step Flow (`L-1..L-4` thắp đèn, `G-1..G-4` vườn đèn), lý do dùng keyset thay `OFFSET` và lý do chống trùng nằm cùng transaction với trần.
- `flow.md`: tạo mới. Hai sequence diagram mirror 1:1 hai Step-by-step Flow; marker hiệu năng hiện dưới dạng `Note` ở bước đếm.
- `tasks.md`: tạo mới. Năm Phase, toàn bộ `[ ]` (chưa có code); mục chốt `ON DELETE` chặn migration ở Phase 3.
- Reason: người dùng yêu cầu tính năng thắp đèn và vườn đèn; BA đã chốt 26 câu trong `SRS.md` v0.3 (2026-10-10), còn hai điểm hoãn có chủ đích (hiệu năng, retention). Spec phát hiện thêm một lệch quy ước cần xác nhận: phân trang cursor thay vì `page`/`total_pages`.

### Việc theo sau (không thuộc spec này, ghi để không bị bỏ sót)

- `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`): thêm hai route khi làm handler (Phase 4).
- `migrations/0026_lamps.sql` và `migrations/README.md`: Phase 3, sau khi chốt `ON DELETE`.
- `docs/specs/db/erd.mmd` / `erd.svg`: thêm bảng `lamps` (Phase 3).
- `docs/knowledge/features/lamp.md`: chưa có; chỉ viết sau khi code đã kiểm chứng (`il-okf-writer`).
