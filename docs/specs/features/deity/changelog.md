# Changelog — deity Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. (`api-contract.md` là tài liệu client có version riêng, không thuộc bộ này.)

## v3.0 (2026-10-15)

Spec cite product spec `docs/product/deity/product.md` @ v1.0 (anh Đức xác nhận 2026-10-15) và thiết kế các thay đổi hợp đồng của product Appendix B. Đồng thời sửa phần lệch code của v2.3. **Chỉ là spec; phần v3.0 chưa có code, `openapi.yaml` chưa đổi.**

- `requirements.md`: header `Product spec` / `Contract` / `Implements`; `Module` đổi `content` → `deity` (spec `content` đã xoá); mọi AC gắn `BR-DTY-*` / `FR-DTY-*` / `NFR-DTY-*` / `(platform)`. Sửa theo code: 20 vị (7 Phật gồm Thất Phật Quá Khứ, 12 Bồ tát, 1 Chư Thiên Đại Cát Tường Thiên Nữ), AC 13 viết lại cho ảnh gốc tương đối + `public_base_url` (migration `0021`). Thay đổi v3.0: AC 5 trang 12 (BR-DTY-03); AC 2, 8, 12, 22 thêm `next_via`, `via_events`, `date`; US-DTY-06 mới với AC 23–32 (ngày vía chỉ từ lịch lễ, server tính lần vía kế tiếp trong cửa sổ `Y..Y+1`, `date` = ngày thiết bị, `INVALID_DATE`, bỏ sự kiện inactive, thứ tự `via_events` theo BR-CAL-12, `lunar_memorial_days` deprecated và dẫn xuất, số truy vấn cố định, vector `deity-order` / `deity-next-via`); AC 15, 16, 18 seed liên kết vía và kiểm mã sự kiện; AC 33 file danh mục vào git; AC 34 kinh liên quan qua route sẵn có của `sutra`. Marker cũ: mã US và ảnh không phân giải được → resolved; "nổi bật", duyệt nội dung, seed production → chuyển thành `⚠️ Not confirmed (product)` trỏ product Appendix A. Marker mới: ánh xạ ngày vía ↔ sự kiện lịch (product), thứ tự kinh liên quan (product), thời điểm xoá `lunar_memorial_days`, host ảnh production.
- `design.md`: hàng **(v3.0)** cho Domain / Entity, Domain / Vía (`domain/deity_via.go` mới, hàm thuần `PickNextVia` / `ResolveDeityVias` / `NextVia`), Repository interface (`ViaEventCodes`, `MissingCalendarEvents`), Usecase (phụ thuộc `CalendarEventRepository`, clock, `calendar.timezone` / `default_language`), Repository (mảng mã vía trong câu đọc, bỏ JSONB, `Upsert` thay liên kết), Seed CLI, Data, Handler / DTO, Config, Wiring, Migration mới (`deity_via_events`, bỏ `lunar_memorial_days`), Makefile, E2E harness, API spec, ERD. Flow phân trang thêm Step 5–6 (đọc catalogue lịch, tính vía); **thêm** Step-by-step Flow cho `top` và `detail` (trước là một câu SQL, nay nhiều bước) và bỏ khối ASCII Request flow; flow seed thêm Step 4 (kiểm mã sự kiện) và ghi liên kết trong cùng transaction. Known gaps cập nhật (file danh mục vắng làm `TestParseDeities_RealDataFile` fail, catalogue lịch chưa vào git, cửa sổ hai năm, không cache).
- `flow.md`: sơ đồ phân trang và seed cập nhật; thêm sơ đồ `top` và `detail`; mirror 1:1 bốn Step-by-step Flow.
- `tasks.md`: sửa Phase 1–5 theo code (hằng `DEVA`, `0015`, ảnh `0021` + `imageHost` → `[x]`; file danh mục → `[ ]`; parse test fail); thêm Phase 8 (dữ liệu, schema, seed), Phase 9 (đọc, vector, hợp đồng, E2E), Phase 10 (dọn, tài liệu) — tất cả `[ ]`.
- `api-contract.md`: v1.4, chỉ thêm banner "lỗi thời, xem `openapi.yaml` + spec này"; xoá sau khi spec client được duyệt.
- Quyết định kỹ thuật của spec này (chờ duyệt qua PR): server tính lần vía kế tiếp (BR-DTY-06 để hợp đồng quyết) để client không đổi lịch âm; "hôm nay" = tham số `date` từ thiết bị, thiếu thì `calendar.timezone`; liên kết deity ↔ sự kiện lịch nằm trong file danh mục deity (product §3.5); `lunar_memorial_days` giữ tạm, dẫn xuất từ `via_events`.
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` (Phase 9); `migrations/` + `docs/specs/db/erd.mmd` (Phase 8); `.gitignore` + commit `assets/data/phat_botat.json` (Phase 8); `docs/specs/features/calendar/` E26 (catalogue lịch vào git) là điều kiện trước; OKF `docs/knowledge/features/deity.md` cho `il-okf-writer`; cột Backend của product §4.2 cho `ba-writer`.
- Reason: product spec v1.0 thay nguồn nghiệp vụ; Appendix B giao cho backend trang 12, vía từ lịch lễ, lần vía kế tiếp trên thẻ và trang, kinh liên quan, file danh mục vào git, và đưa spec v2.3 về khớp code.

## v2.3 (2026-10-12)

Implement bộ lọc `type` (AC 20–22) đã spec ở v2.2. Theo yêu cầu "implement" của người dùng.

- Code: `domain/deity.go` (`IsValidDeityType`, `ErrInvalidDeityType`, `List(ctx, deityType, offset, limit)`), `usecase/deity_usecase.go`, `repository/postgres_deity_repository.go` (`WHERE type = $1` cho cả đếm và lấy trang), `delivery/http/deity_handler.go`, `delivery/http/errors.go` (`INVALID_DEITY_TYPE`), `domain/mock_deity_repository.go`. `cmd/main.go` không đổi.
- Test: unit usecase/handler, repository (`TestDeityRepository_List_FiltersByType`), E2E mới `e2e/deity_e2e_test.go` (đây là E2E deity đầu tiên). Test repository nay áp thêm `0015_deities_deva_type.sql` trong `applyMigration` vì `deities.type` cần cho `DEVA`.
- `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: thêm parameter `type`, response `400 INVALID_DEITY_TYPE`, sửa mô tả "không lọc".
- `requirements.md`/`design.md`/`flow.md`: chỉ đổi Status và bỏ chú thích "chưa có code"; không đổi yêu cầu. `tasks.md`: Phase 7 đánh `[x]` sau khi chạy build/vet/test. `api-contract.md`: v1.3.
- Còn lại: E2E cho `top`/`detail` (Phase 5), OKF `docs/knowledge/features/deity.md`, tài liệu web `guides/catalogues/deities.md`.

## v2.2 (2026-10-12)

Thêm bộ lọc `type` cho danh sách Phật / Bồ tát (`GET /v1/deities?type=`). Theo yêu cầu của người dùng sau khi xác nhận hiện chưa có API lọc theo loại. **Chỉ là spec; chưa có code, test hay thay đổi `openapi.yaml`.**

- `requirements.md`: thêm AC 20–22 (lọc hợp lệ, `INVALID_DEITY_TYPE`, kết hợp `page`) vào US-DTY-02, đánh số nối đuôi để không lệch các tham chiếu AC 1–19; sửa AC 4, 5 (lọc và `total`); AC 16 và `DeitySummary.type` thêm `DEVA` cho khớp code/migration `0015`; bảng route, Out of scope, Pending; ghi quyết định vào "Open questions (resolved)".
- `design.md`: thêm hàng (v2.2) ở Domain / Errors, Repository interface, Usecase, Repository, Handler, Error map; Step 1–4 của flow phân trang thêm `type`; Known gaps cập nhật; `cmd/main.go` không đổi (cùng route).
- `flow.md`: sơ đồ phân trang thêm nhánh `type` lạ và `WHERE type = $1`, mirror `design.md` 1:1; không thêm/bớt Step.
- `tasks.md`: thêm Phase 7 (7 mục, tất cả `[ ]`). Không mục nào đánh `[x]` vì chưa có code.
- `api-contract.md`: v1.2, thêm mục "Filter by type" đánh dấu chưa implement.
- Quyết định tự chốt, chờ người dùng/product duyệt (chi tiết ở "Open questions (resolved)"): tham số trên route sẵn có; khớp chính xác phân biệt hoa/thường; rỗng = không lọc; một giá trị mỗi lần; mã lỗi `INVALID_DEITY_TYPE` theo tiền lệ `INVALID_GALLERY_TYPE`.
- Không breaking: client không gửi `type` nhận đúng kết quả cũ.
- Drift có sẵn, chưa sửa: spec còn ghi "18 mục / hai loại" ở nhiều chỗ trong khi code và dữ liệu đã có `DEVA` (migration `0015`) và 20 mục (theo `openapi.yaml`). Chỉ sửa các chỗ chạm bộ lọc.
- Follow-up ngoài phạm vi spec: `docs/specs/api/openapi.yaml` (+ `web/docs/public/openapi.yaml`) — mô tả `/v1/deities` đang ghi "không lọc"; OKF `docs/knowledge/features/deity.md` (chưa có); tài liệu web `web/docs/src/content/docs/guides/catalogues/deities.md`.
- Không có `SRS.md` cho feature này (chỉ `administrative-units`, `genealogy`, `lamp`… có), nên chưa có SRS để cập nhật.

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
