# Changelog — deity Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. (`api-contract.md` là tài liệu client có version riêng, không thuộc bộ này.)

## v2.1 (2026-10-03)

Bỏ tham chiếu tới spec `content` cũ (đã xoá); chỉ sửa một dòng chú thích, không đổi yêu cầu nào.

- `requirements.md`: marker `⚠️ Not confirmed` về mã user story không còn nhắc `content/requirements.md`; nội dung câu hỏi (product chưa chốt số US) giữ nguyên.
- `design.md`, `flow.md`, `tasks.md`: chỉ tăng version, không đổi (đã kiểm).

## v2.0 (2026-09-30)

Danh sách phân trang gộp cả Phật và Bồ tát (cùng một bảng `deities`); route `GET /v1/bodhisattvas` được thay bằng `GET /v1/deities?page=N`. Theo yêu cầu của người dùng sau khi rà lại API Phật/Bồ tát.

- `requirements.md`: US-DTY-02 đổi thành "Duyệt danh sách Phật và Bồ tát theo trang"; AC 4–5 (route, không lọc loại, `total` là số mục của cả bảng); bảng route; giải quyết câu hỏi "cách xem Phật" (chuyển sang "Open questions (resolved)"); còn 4 marker `⚠️ Not confirmed`.
- `design.md`: bỏ `ListByType`, thay bằng `List` (không lọc loại); Step 3–4 đổi câu SQL (`count(*)` toàn bảng, không `WHERE type`); ghi chú route.
- `flow.md`: sequence phân trang đổi route, tên method và câu SQL cho khớp `design.md` 1:1; không thêm/bớt step.
- `tasks.md`: đổi tên mục usecase/handler/repository/test; E2E cập nhật kỳ vọng. Không mục nào mới được đánh `[x]` ngoài những gì đã có trong code và đã chạy `-short`.
- Reason: cùng một bảng, chỉ khác `type`; trước đây Phật chỉ xem được qua `top`/`detail`.
- **Breaking change**: `/v1/bodhisattvas` bị xoá, không giữ alias; client iOS phải đổi sang `/v1/deities` và tự lọc theo `type` nếu cần chỉ hiện Bồ tát. Repository test cần Postgres và server chưa được chạy lại sau thay đổi.
- Follow-up ngoài phạm vi spec: `api-contract.md` (cập nhật cùng lượt, v1.1), `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml`, OKF `docs/knowledge/features/deity.md` (làm bằng `/il-okf-writer`), tài liệu web `web/docs/src/content/docs/guides/catalogues/deities.md` và `changelog.md`.

## v1.1 (2026-09-30)

Thêm bảng `deities` vào ERD; spec cập nhật cho khớp.

- `requirements.md`: bỏ mục Pending về ERD; Status bỏ "chưa có bảng trong ERD".
- `design.md`: hàng ERD chuyển sang "đã có"; Known gaps chỉ còn OKF. Flow không đổi.
- `flow.md`: chỉ tăng version, không đổi.
- `tasks.md`: đánh `[x]` mục ERD.
- Reason: `docs/specs/db/erd.mmd` và `erd.svg` đã có bảng `deities` (cột lấy từ migration, ràng buộc `CHECK` ghi trong chú thích).
- Follow-up ngoài phạm vi spec: OKF concept `docs/knowledge/features/deity.md` vẫn chưa có.

## v1.0 (2026-09-30)

Initial spec, written retroactively from the already-implemented Phật / Bồ tát catalogue in `services/appService/` (Retroactive mode), theo yêu cầu "làm tài liệu cho phần chùa, Phật/Bồ tát và bài viết Phật pháp". Phủ năm user story: nổi bật (trộn Phật và Bồ tát), danh sách Bồ tát phân trang, chi tiết, hình dạng dữ liệu nhất quán, nạp dữ liệu bằng lệnh seed.

- `requirements.md`: 5 US, 19 AC (EARS), bảng API contract 3 route đọc từ handler thực tế, 5 marker `⚠️ Not confirmed` (tiêu chí "nổi bật", cách xem Phật khi không có route danh sách, nguồn ảnh, quy trình nạp dữ liệu ở môi trường thật, nguồn và độ chính xác nội dung tôn giáo) cùng 1 marker về mã US.
- `design.md`: bảng layers gắn với file/symbol thật; 2 Step-by-step Flow (danh sách Bồ tát phân trang, nạp dữ liệu bằng `cmd/seeddeities`) kèm transaction boundary; mục Known gaps (không có E2E, test repository cần Postgres và xoá dữ liệu dev, ảnh không phân giải được và không theo quy ước `null`, không có route danh sách Phật, seed dùng config dev, upsert ghi đè và không xoá, validation seeder chỉ một phần, JSONB hỏng làm hỏng cả trang, đếm và lấy trang không cùng snapshot, ERD/OKF chưa có).
- `flow.md`: 2 sequence diagram mirror 1:1 hai Step-by-step Flow.
- `tasks.md`: 5 Phase, `[x]` cho những mục đã xác minh tồn tại (build, vet, `go test -short` pass); còn `[ ]` cho ảnh dùng được, seed ngoài dev, ERD, E2E, OKF, rà `api-contract.md`.
- Reason: tính năng đã ship và có `api-contract.md` cho iOS nhưng chưa có spec theo convention; cần spec để `admin/requirements.md` (đang trỏ về `deity/`) có nơi tham chiếu.
- Lưu ý so với `api-contract.md`: hợp đồng HTTP khớp code; spec bổ sung phần seed, hệ quả của việc Phật chỉ xem được qua `top`/`detail`, và ghi rõ quy ước URL ảnh khác `temple`/`article` (chuỗi, không `null`). Số liệu 18 mục (6/12) và giá trị URL ảnh giữ chỗ được đối chiếu lại với `assets/data/phat_botat.json`.
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/db/erd.mmd`/`erd.svg` (thêm bảng `deities`); OKF concept `docs/knowledge/features/deity.md` chưa có, do `bdd-okf-docs` tạo sau khi có E2E; `docs/specs/api/openapi.yaml` đã có route nên không cần sửa.
