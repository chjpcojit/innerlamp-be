# Changelog — dharmatalk Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v1.1 (2026-10-03)

Implement xong Phase 1–4 và E2E của Phase 5 (xem `tasks.md`); spec được làm cho khớp với code, không đổi yêu cầu nào (19 AC giữ nguyên).

- `requirements.md`: Status chuyển sang implemented; mục "API contract" không còn "proposed" (đã có trong `openapi.yaml`); bỏ mục Pending về code/migration/seed; số liệu seed bổ sung 19 bài `needs_review` → 36 bài hiển thị được. Các `⚠️ Not confirmed` giữ nguyên, **chưa** có câu nào được người dùng trả lời.
- `design.md`: hàng Data, API spec và Domain / Repository interface sửa cho khớp (chỉ `List` và `FindByID` lọc `needs_review`, `List` trả `total` đã lọc; `Count` đếm mọi dòng; không cần hai method đếm như `tasks.md` v1.0 dự kiến); Known gaps ghi lại E2E phụ thuộc ngưỡng 10 giây của harness và test repository cần DB scratch. Flow không đổi.
- `flow.md`: chỉ cập nhật Status, hai diagram không đổi (đã kiểm: E2E chạy đúng các nhánh seed `n = 0`, `n > 0`, và bảng toàn bài bị cờ).
- `tasks.md`: đánh `[x]` Phase 1–4 và E2E, `go build`/`go vet`, `openapi.yaml`, ERD; còn `[ ]` duy nhất là khái niệm OKF.
- Kết quả kiểm chứng: `go build ./...`, `go vet ./...`, `go test -short ./...` sạch; `scripts/check_dependency_rule.sh` PASS; 8 test repository qua trên Postgres scratch; 8 test E2E qua khi ngưỡng chờ khởi động được nâng tạm lên 60 giây (ở ngưỡng 10 giây thì thất bại ngẫu nhiên, như `TestSutraAPI_E2E` có sẵn).
- Thay đổi ngoài thư mục spec: `migrations/0014_dharma_talks.sql`, `migrations/README.md`, `assets/embed.go`, `services/appService/{domain,usecase,repository,delivery/http,config,cmd,e2e}`, `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml`, `docs/specs/db/erd.{mmd,svg}`.
- Follow-up ngoài phạm vi spec: `docs/knowledge/features/dharmatalk.md` và dòng của nó trong `docs/knowledge/index.md`/`log.md` (`il-okf-writer`); xác nhận hoặc bác bỏ ba giả định `docs/assumptions/dharmatalk/A-001..A-003`; ngưỡng chờ server 10 giây của `e2e/harness.go`.

## v1.0 (2026-10-03)

Initial spec, written prospectively (Prospective mode) cho màn "Pháp thoại" theo mockup và các điều chỉnh UI của người dùng: hỗ trợ ba loại phương tiện (YouTube nhúng, audio, văn bản), bỏ nút 15s/30s, mục "Các phần" rỗng thì ẩn, thêm thẻ nguồn ở cuối, nhúng YouTube trực tiếp.

- `requirements.md`: 5 US, 19 AC (EARS); bảng API contract đề xuất 2 route (`GET /v1/dharma-talks`, `GET /v1/dharma-talks/{id}`); 11 marker `⚠️ Not confirmed` (mã US, lượt nghe, thẻ thiền, "Các phần", `season`, cờ `needs_review`, audio, bản quyền, số lượng 55/60, tên hiển thị giảng sư, quy trình cập nhật nội dung).
- `design.md`: bảng layers dự kiến, bảng cột `dharma_talks`, đặc tả Seed JSON và ánh xạ `video` → `youtube`, 2 Step-by-step Flow (danh sách phân trang, seed lần đầu), mục stub có chủ đích và Known gaps.
- `flow.md`: 2 sequence diagram mirror 1:1 hai Step-by-step Flow.
- `tasks.md`: 5 Phase, tất cả `[ ]`.
- Quyết định tạm (ghi trong `docs/assumptions/dharmatalk/`): tên feature `dharmatalk`, route `/v1/dharma-talks`, bỏ route `top`, ẩn bài `needs_review` khỏi mọi route, seed đếm cả bài bị cờ.
- Reason: người dùng yêu cầu backend làm theo UI đã chốt; theo quy ước repo, spec đi trước code.
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` (2 route), `docs/specs/db/erd.mmd`/`erd.svg` (bảng `dharma_talks`), `migrations/README.md` (ghi `0014`), `docs/knowledge/features/dharmatalk.md` (sau khi code xong); chuẩn hoá `assets/data/dharma_talks.json` (xem `tasks.md` Phase 3).
