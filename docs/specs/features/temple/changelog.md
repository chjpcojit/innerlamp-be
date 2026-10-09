# Changelog — temple Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. (`api-contract.md` là tài liệu client có version riêng, không thuộc bộ này.)

## v1.2 (2026-10-03)

Bỏ tham chiếu tới spec `content` cũ (đã xoá); chỉ sửa một dòng chú thích, không đổi yêu cầu nào.

- `requirements.md`: marker `⚠️ Not confirmed` về mã user story không còn nhắc `content/requirements.md`; nội dung câu hỏi (product chưa chốt số US) giữ nguyên.
- `design.md`, `flow.md`, `tasks.md`: chỉ tăng version, không đổi (đã kiểm).

## v1.1 (2026-09-30)

Thêm bảng `temples` vào ERD; spec cập nhật cho khớp.

- `requirements.md`: bỏ mục Pending về ERD; Status bỏ "chưa có bảng trong ERD".
- `design.md`: hàng ERD chuyển sang "đã có"; Known gaps chỉ còn OKF. Flow không đổi.
- `flow.md`: chỉ tăng version, không đổi.
- `tasks.md`: đánh `[x]` mục ERD.
- Reason: `docs/specs/db/erd.mmd` và `erd.svg` đã có bảng `temples` (cột lấy từ migration, ràng buộc `CHECK` ghi trong chú thích).
- Follow-up ngoài phạm vi spec: OKF concept `docs/knowledge/features/temple.md` vẫn chưa có.

## v1.0 (2026-09-30)

Initial spec, written retroactively from the already-implemented temple catalogue in `services/appService/` (Retroactive mode), theo yêu cầu "làm tài liệu cho phần chùa, Phật/Bồ tát và bài viết Phật pháp". Phủ năm user story: chùa nổi bật, danh sách phân trang, chi tiết, hình dạng dữ liệu nhất quán, nạp dữ liệu bằng lệnh seed.

- `requirements.md`: 5 US, 18 AC (EARS), bảng API contract 3 route đọc từ handler thực tế, 6 marker `⚠️ Not confirmed` (quy trình nạp dữ liệu ở môi trường thật, ảnh, quyền dùng/ghi công của nội dung biên tập, độ chính xác dữ liệu, tiêu chí "nổi bật", phạm vi spec backoffice) cùng 1 marker về mã US.
- `design.md`: bảng layers gắn với file/symbol thật; 2 Step-by-step Flow (danh sách phân trang, nạp dữ liệu bằng `cmd/seedtemples`) kèm transaction boundary; mục Known gaps (không có E2E, test repository cần Postgres và xoá dữ liệu dev, seed dùng config dev, upsert ghi đè và không xoá, validation seeder chỉ một phần, đếm và lấy trang không cùng snapshot, ảnh rỗng, ERD/OKF chưa có).
- `flow.md`: 2 sequence diagram mirror 1:1 hai Step-by-step Flow.
- `tasks.md`: 5 Phase, `[x]` cho những mục đã xác minh tồn tại (build, vet, `go test -short` pass); còn `[ ]` cho seed dùng được ngoài dev, ERD, E2E, OKF, rà `api-contract.md`.
- Reason: tính năng đã ship và có `api-contract.md` cho iOS nhưng chưa có spec theo convention; cần spec để `admin/requirements.md` (đang trỏ về `temple/`) có nơi tham chiếu.
- Lưu ý so với `api-contract.md`: hợp đồng HTTP khớp code; spec bổ sung phần seed, các quyết định về thứ tự và các khoảng trống mà tài liệu client không nói (seed chỉ chạy tay với config dev, file JSON không nhúng vào binary).
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/db/erd.mmd`/`erd.svg` (thêm bảng `temples`); OKF concept `docs/knowledge/features/temple.md` chưa có, do `bdd-okf-docs` tạo sau khi có E2E; `docs/specs/api/openapi.yaml` đã có route nên không cần sửa.
