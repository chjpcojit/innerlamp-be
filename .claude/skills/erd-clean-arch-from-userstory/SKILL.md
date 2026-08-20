---
name: erd-clean-arch-from-userstory
description: Đọc một user story (hoặc một nhóm user story cùng module) và sinh ra (1) ERD dạng Mermaid erDiagram cho các bảng DB liên quan, và (2) mô tả từng bước implement backend Go theo Clean Architecture (entity → repository interface → usecase → repository impl → delivery/handler → wiring). Dùng skill này bất cứ khi nào người dùng đưa một user story/acceptance criteria và muốn thiết kế schema DB + kế hoạch code backend cho tính năng đó, kể cả khi họ chỉ nói "thiết kế DB cho cái này" hoặc "implement cái này theo clean architecture" mà không nói rõ "ERD". Áp dụng cho các dự án Go/Clean-Architecture-DDD của người dùng (ví dụ Bankaool, Tâm Đăng/InnerLamp).
---

# ERD + Clean Architecture Implementation Plan từ User Story

## Mục tiêu

Từ một user story (có Acceptance Criteria rõ ràng), sinh ra 2 artifact:
1. **ERD** — dạng Mermaid `erDiagram` code block (không vẽ hình ảnh/SVG, chỉ code text).
2. **Implementation plan** — các bước implement backend Go theo Clean Architecture, đúng convention đang dùng ở Bankaool/Elcora (xem `references/clean-arch-convention.md`).

Ngôn ngữ trả lời: giải thích bằng tiếng Việt thuần (không parallel structure dịch từ tiếng Anh), thuật ngữ kỹ thuật/code/tên field/tên bảng giữ nguyên tiếng Anh.

## Quy trình

### Bước 1 — Đọc và phân rã user story

Đọc kỹ toàn bộ user story: User Story statement, Acceptance Criteria (AC), phần "Ghi chú kỹ thuật" hoặc "Out of scope" nếu có. Với mỗi AC, xác định:
- Entity/dữ liệu nào cần tồn tại để AC đó thoả mãn được (ví dụ: AC nói "lưu cache local kèm `synced_at`" → cần field `synced_at`/`updated_at`).
- Hành vi nào là read, hành vi nào là write, hành vi nào là sync/delta.
- Có ràng buộc nghiệp vụ nào ẩn trong AC không (unique, required, filter theo field nào...).

Nếu user story tham chiếu đến field/endpoint cụ thể (ví dụ `GET /v1/content/sutras?updated_since=...&sect=...`), coi đó là nguồn sự thật cho tên field/tên bảng — không tự đặt tên khác đi.

Nếu thiếu thông tin quan trọng để thiết kế bảng (ví dụ không rõ quan hệ 1-n hay n-n), hỏi lại người dùng 1 câu ngắn gọn thay vì tự đoán và ghi rõ giả định.

### Bước 2 — Thiết kế ERD

Với mỗi entity xác định ở Bước 1, liệt kê: tên bảng, các cột (tên + kiểu dữ liệu hợp lý cho Postgres/MySQL — theo dự án đang dùng), khóa chính, khóa ngoại, unique constraint, index cần thiết để phục vụ đúng AC (ví dụ AC yêu cầu filter theo `sect` → cần index trên cột đó; AC yêu cầu delta sync theo thời gian → cần index trên `updated_at`).

Output bằng một Mermaid `erDiagram` code block, kèm chú thích ngắn ngay dưới mỗi bảng nếu có quyết định thiết kế cần giải thích (vì sao thêm cột này, vì sao tách bảng này). Không render hình, chỉ đưa code Mermaid — người dùng sẽ tự quyết định có render hay không.

### Bước 3 — Kế hoạch implement theo Clean Architecture (Go)

Đọc `references/clean-arch-convention.md` để nắm đúng cấu trúc thư mục và convention (naming, layer nào phụ thuộc layer nào, cách wiring). Sau đó mô tả các bước implement theo đúng thứ tự dependency injection từ trong ra ngoài:

1. **Domain entity** (`domain/`) — struct đại diện entity, không phụ thuộc framework/DB.
2. **Repository interface** (`domain/` hoặc `usecase/repository/`) — định nghĩa các method cần (map từ AC sang method, ví dụ AC "delta sync" → `FindUpdatedSince(ctx, since time.Time, sect *string) ([]Sutra, error)`).
3. **Usecase** (`usecase/`) — business logic, gọi qua repository interface, xử lý fallback cache/offline nếu AC yêu cầu (như AC4 "không mạng vẫn xem được cache").
4. **Repository implementation** (`repository/` — Postgres/Firestore/Mongo tuỳ dự án) — implement interface ở bước 2.
5. **Delivery/handler** (`delivery/http/` hoặc tương đương) — map HTTP request/response, validate input, gọi usecase.
6. **Wiring/DI** — nơi khởi tạo và inject các layer vào nhau (thường ở `main.go` hoặc `wire/`).
7. **Test** — nêu rõ nên test ở layer nào là chính (thường usecase, với repository mock), test case nào ứng với AC nào.

Với mỗi bước, map ngược lại xem nó phục vụ AC nào trong user story gốc — để đảm bảo không sót AC.

### Bước 4 — Đối chiếu lại AC

Sau khi có ERD + implementation plan, rà lại từng AC một lần cuối, xác nhận: AC này đã có bảng/cột nào lưu trữ dữ liệu cần thiết chưa, đã có usecase method nào xử lý chưa. Nếu AC nào chưa được cover, nêu rõ ra thay vì im lặng bỏ qua.

## Khi user story không đủ rõ

Nếu user story thiếu Acceptance Criteria hoặc chỉ có mô tả chung chung, không tự bịa ra AC — hỏi lại người dùng muốn thiết kế cho phạm vi nào, hoặc đề xuất AC nháp và xin xác nhận trước khi thiết kế ERD.

## Phạm vi (Out of scope cho skill này)

- Không thiết kế Flutter/client-side implementation (chỉ backend Go).
- Không tự vẽ diagram dạng hình ảnh/SVG.
- Không đề xuất hạ tầng GCP/deploy trừ khi người dùng hỏi thêm.
