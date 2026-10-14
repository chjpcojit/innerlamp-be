# Changelog — article Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v2.0 (2026-10-11)

appService **không còn seed bài viết khi khởi động**: việc nạp chuyển sang lệnh seed, `make run` chạy lệnh đó khi bảng rỗng. Hợp đồng HTTP không đổi.

- `requirements.md`, `design.md`, `flow.md`, `tasks.md` (→ 2.0): AC15–AC18 nay mô tả `cmd/seedarticles` (mặc định dùng dữ liệu nhúng, `-file` để đọc file khác); flow seed vẽ lại theo lệnh; `cmd/main.go` bỏ `SeedIfEmpty`. Ghi chú: lệnh chèn các `id` còn thiếu (không sửa dòng có sẵn); `make ensure-articles` chỉ gọi nó khi bảng rỗng.
- Đổi hành vi: khởi động service trên DB rỗng **thành công** và phục vụ danh sách rỗng (trước đây service log.Fatalf nếu seed lỗi).
- Kiểm chứng: E2E chạy lệnh seed (rồi khởi động service) thay vì dựa vào seed lúc start; `go test -short ./...` pass.
- Theo dõi ngoài thư mục này: `Makefile` (`ensure-*`, `seed-*`), `docs/development/local-stack.md`; concept OKF liên quan cần `il-okf-writer` làm mới khi được yêu cầu.

## v1.6 (2026-10-06)

Thêm `thumbnail_url` (ảnh nhỏ cho thẻ danh sách) vào API bài viết theo yêu cầu của client iOS; ảnh nhỏ do app lưu và phục vụ.

- `requirements.md`: AC 2 thêm `thumbnail_url` vào summary; thêm AC 13 (`thumbnail_url` là chuỗi hoặc `null`, luôn có khoá); API contract và Out of scope cập nhật (app lưu ảnh nhỏ, không lưu ảnh gốc).
- `design.md`: migration `0016`, `FillMissingImages` điền độc lập từng cột, Known gap về `thumbnail_url` và về DB dev đã đủ 60/60.
- `tasks.md`: đánh `[x]` việc chạy `-fill-images` trên DB dev; thêm hai mục `thumbnail_url` (cài đặt và test).
- `flow.md`: chỉ tăng version, không đổi (flow đọc và seed không đổi bước).
- Reason: ảnh `image_url` là hotlink 1200x630, quá nặng cho thẻ 84x84; temple và deity đã có cặp `thumbnail_url` + `image_url`.
- Giả định: `docs/assumptions/article/A-002-hosted-article-thumbnails.md`.
- Code: `migrations/0016_article_thumbnail.sql`, `domain/article.go`, `repository/{postgres_article_repository,article_seed}.go`, `delivery/http/article_handler.go`.

## v1.5 (2026-10-06)

Điền nốt `imageURL` cho 22 bài còn thiếu: dữ liệu seed nay có ảnh cho 60/60 bài.

- `requirements.md`: ghi chú kỹ thuật và marker về quyền dùng ảnh cập nhật nguồn ảnh (15 bài lấy từ ảnh trong trang nguồn giacngo.vn/langmai.org, 7 bài từ Wikimedia Commons); Out of scope ghi hotlink gồm cả Commons. Không đổi AC nào (AC 12 vẫn đúng: có ảnh thì trả chuỗi, không thì `null`).
- `design.md`: Known gap về hotlink cập nhật số liệu 60/60 và nhắc cách đưa ảnh mới vào DB đã seed (`seedarticles -fill-images`).
- `tasks.md`: mục điền `imageURL` thành 60/60; bỏ việc hoãn "tìm ảnh cho 22 bài"; thêm `[ ]` chạy `-fill-images` trên DB dev.
- `flow.md`: chỉ tăng version, không đổi.
- Reason: 7 bài (so-hai-chap-nhan-va-tha-thu, an-com-chanh-niem, dao-phat-va-chu-hieu, gioi-thu-tu-ai-ngu-va-lang-nghe, thien-tra-lang-mai, an-chay-de-nuoi-duong-long-tu-bi, chanh-niem-hanh-phuc-ngay-bay-gio) không có ảnh dùng được ở trang nguồn; người dùng yêu cầu gắn ảnh trên mạng cho đủ nên dùng ảnh minh hoạ chủ đề từ Commons — giả định ghi ở `docs/assumptions/article/A-001-commons-illustrative-images.md`.
- Code: `repository/article_seed_test.go` đổi từ "bài đầu không có ảnh" sang "mọi bài có URL https".

## v1.4 (2026-10-03)

Bỏ tham chiếu tới spec `content` cũ (đã xoá); chỉ sửa một dòng chú thích, không đổi yêu cầu nào.

- `requirements.md`: marker `⚠️ Not confirmed` về mã user story không còn nhắc `content/requirements.md`; nội dung câu hỏi (product chưa chốt số US) giữ nguyên.
- `design.md`, `flow.md`, `tasks.md`: chỉ tăng version, không đổi (đã kiểm).

## v1.3 (2026-09-30)

Thêm bảng `articles` vào ERD; spec cập nhật cho khớp.

- `requirements.md`: bỏ mục Pending về ERD; Status cập nhật.
- `design.md`: Known gaps về ERD được thay bằng ghi nhận tài liệu schema/API đã đủ. Flow không đổi.
- `flow.md`: chỉ tăng version, không đổi.
- `tasks.md`: đánh `[x]` mục ERD; Phase 1–4 nay đã xong hết.
- Reason: `docs/specs/db/erd.mmd` và `erd.svg` đã có bảng `articles`.
- Follow-up ngoài phạm vi spec: `docs/knowledge/features/article.md` và `index.md` vẫn ghi "not yet in openapi.yaml" (`bdd-okf-docs` làm mới).

## v1.2 (2026-09-30)

Đối chiếu lại spec với code và tài liệu hiện có khi viết spec cho `temple` và `deity` (cùng lượt); sửa những chỗ đã lệch, không đổi hành vi.

- `requirements.md`: bỏ mục Pending "cập nhật `openapi.yaml`" và sửa câu ở API contract — `docs/specs/api/openapi.yaml` đã có 3 route (`/v1/articles/top`, `/v1/articles`, `/v1/articles/{id}`) và `ARTICLE_NOT_FOUND`; Status ghi lại là chỉ còn thiếu ERD và E2E.
- `design.md`: hàng API spec chuyển từ "chưa cập nhật" sang "đã có"; hàng Domain / Errors sửa — `ErrInvalidPage` định nghĩa ở `domain/temple.go`, không phải `domain/errors.go`; Step 3 của flow seed sửa "30 ở lần đầu" thành 60 (lỗi sót từ v1.1); Known gaps chỉ còn ERD. Cấu trúc flow không đổi.
- `flow.md`: chỉ tăng version, không đổi (hai flow không đổi).
- `tasks.md`: đánh `[x]` mục `docs/specs/api/openapi.yaml` (đã kiểm bằng `grep` trên file); ghi chú vị trí định nghĩa `ErrInvalidPage`; Status cập nhật. Còn `[ ]`: ERD và E2E.
- Reason: `openapi.yaml` được cập nhật sau v1.1 nên spec ghi nhận nó "chưa có" là sai; hai lỗi nhỏ còn lại phát hiện khi đọc lại code.
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/knowledge/features/article.md` và dòng của nó trong `docs/knowledge/index.md` vẫn ghi "not yet in `openapi.yaml`" — `bdd-okf-docs` cần làm mới; ERD `docs/specs/db/erd.mmd`/`erd.svg` vẫn thiếu bảng `articles`.

## v1.1 (2026-09-30)

Chốt nguồn ảnh `image_url`: lấy theo bài gốc (`og:image` của `source_url`), theo yêu cầu của người dùng.

- `requirements.md`: marker "nguồn ảnh" được thay bằng marker mới về quyền dùng/độ bền ảnh hotlink; số liệu seed sửa từ 30 bài sang 60 bài (58 vi / 2 en, 29 có ngày, 38 có ảnh); mục Out of scope ghi rõ ảnh là hotlink.
- `design.md`: mô tả `FillMissingImages` và flag `-fill-images`; Known gaps cập nhật (60 bài, `image_url` là hotlink, seed vẫn không đổi các cột khác).
- `flow.md`: chỉ tăng version, không đổi (hai flow không đổi).
- `tasks.md`: thêm mục đã làm (`FillMissingImages`, dữ liệu ảnh); Deferred thêm việc chép ảnh về GCS và tìm ảnh cho 22 bài còn `null`.
- Reason: 38/60 bài trong `assets/data/buddhist_articles.json` nay có ảnh; 8 bài giacngo chỉ có logo chung nên bị loại; DB dev đã được điền qua `go run ./services/appService/cmd/seedarticles -fill-images`.

## v1.0 (2026-09-30)

Initial spec, written retroactively from the already-implemented "Bài viết Phật pháp hay" feature in `services/appService/` (Retroactive mode), theo yêu cầu của session [BE-Developer] vừa làm và chạy thật tính năng này. Phủ năm user story: bài nổi bật, danh sách phân trang, chi tiết, hình dạng dữ liệu nhất quán, seed lần đầu khi khởi động.

- `requirements.md`: 5 US, 18 AC (EARS), bảng API contract 3 route đọc từ handler thực tế và xác nhận bằng cách gọi server dev, 4 marker `⚠️ Not confirmed` (quy trình cập nhật nội dung sau seed, nguồn ảnh, điều kiện hiển thị/ghi công của nội dung tóm tắt, tiêu chí "nổi bật") cùng 1 marker về mã US.
- `design.md`: bảng layers gắn với file/symbol thật; 2 Step-by-step Flow (danh sách phân trang, seed lần đầu) kèm transaction boundary; mục Known gaps (không có E2E, test repository cần Postgres, thiếu `openapi.yaml`/ERD, sửa JSON không cập nhật DB có sẵn, bảng bị xoá sạch thì nạp lại, đếm và lấy trang không cùng snapshot, `id = "top"` bị che).
- `flow.md`: 2 sequence diagram mirror 1:1 hai Step-by-step Flow.
- `tasks.md`: 5 Phase, đã đánh `[x]` những mục xác minh tồn tại trong code; còn `[ ]` cho `openapi.yaml`, ERD và E2E.
- Reason: tính năng đã ship nhưng chưa có spec; cần spec để phần còn lại của tài liệu (OKF, `migrations/README.md`, tài liệu API) có nơi tham chiếu.
- Lưu ý sửa so với mô tả bàn giao: `published_date` là `null` ở **một nửa** số bài (15/30 bài có ngày) chứ không phải đa số, còn `image_url` là `null` ở **tất cả** bài (0/30).
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/api/openapi.yaml`, `web/docs/public/openapi.yaml` (thêm 3 route), `docs/specs/db/erd.mmd`/`erd.svg` (thêm bảng `articles`); OKF concept `docs/knowledge/features/article.md` đã được tạo cùng lượt này.
