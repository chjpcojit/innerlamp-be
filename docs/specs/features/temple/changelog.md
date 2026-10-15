# Changelog — temple Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first. (`api-contract.md` là tài liệu client có version riêng, không thuộc bộ này.)

## v2.0 (2026-10-15)

Viết lại để trỏ lên product spec `docs/product/temple/product.md` @ v1.1 (bước 3 của quy trình tài liệu) và thêm phần mới product yêu cầu. MAJOR vì hành vi `GET /v1/temples` đổi (trang 12, tham số mới, lỗi mới).

- `requirements.md` (1.14 → 2.0): header `Product spec` / `Contract` / `Architecture` / `Implements`; mục "Ràng buộc từ product spec"; gắn ID (`FR-TPL-*`/`BR-TPL-*`/`(platform)`) cho AC 1–38; AC 5 đổi `page_size` thành 12; thêm US-TMP-09 (AC 39–44, `q` tên + tên khác, `region`), US-TMP-10 (AC 45–49, giờ lễ, liên hệ, giáo hội, tháng cập nhật, kiểm ở seed, migration `0037`), US-TMP-11 (AC 50, `temple.json` vào git). Đóng 4 marker cũ (mã user story, quy trình nạp, tiêu chí nổi bật, phạm vi backoffice); còn 1 marker product (ghi nguồn + quyền ảnh) và 1 marker dữ liệu.
- `design.md` (1.14 → 2.0): hàng mới Domain / Detail fields, Domain / Filter, Migration `0037`; Repository có `templeFilterWhere` (khớp `aliases` bằng `unnest`); handler/usecase/error map/DTO/config/seed/data/API spec/ERD cập nhật; Step-by-step Flow danh sách viết lại (Step 1–2 thêm `q`, `region`); flow seed thêm kiểm AC 47 ở Step 2 và 35 cột ở Step 5; Known gaps cập nhật.
- `flow.md` (1.14 → 2.0): sequence danh sách vẽ lại theo flow mới (ba nhánh lỗi ở Step 2, cùng `WHERE` ở Step 3–4). Sơ đồ `options` và seed: no change needed, verified (kiểm định dạng mới nằm trong Step 2 của seed, không thêm bước).
- `tasks.md` (1.14 → 2.0): Phase 11–13 mới, mọi mục `[ ]`; OKF Phase 5 đánh `[x]` (file đã có); Deferred viết lại theo product.
- Theo sau (không sửa ở đây): `docs/specs/api/openapi.yaml` + bản web, `migrations/0037`, `docs/specs/db/erd.mmd`, `.gitignore`, OKF `docs/knowledge/features/temple.md` — đều là mục của Phase 11–13.

## v1.14 (2026-10-11)

`create-temple` đã có code nên AC 38 được thoả. Không đổi yêu cầu hay thiết kế.

- `requirements.md` (1.13 → 1.14): Status và Pending ghi AC 38 xong. `design.md`, `flow.md`, `tasks.md` (1.13 → 1.14): no change needed, verified — chỉ header.
- Chứng cứ: `services/backofficeService/e2e/` (`TestTempleSuggestion_E2E_CreateTemple`, `..._LegacySuggestionTakesCodesFromRequest`) và test repository `CreateTemple` — chùa tạo từ đề xuất có `province_code`, `ward_code`, `province`, `ward` đều set, tên lấy từ dataset. Spec đầu mối: `temple-suggestion` v3.4.

## v1.13 (2026-10-11)

Áp `0032` và nạp lại lên DB dev. Không đổi yêu cầu hay thiết kế.

- `tasks.md` (1.12 → 1.13): đánh `[x]` mục DB dev kèm kết quả kiểm (93/93 đủ bộ ba, 20 `address` rỗng, 0 xã lệch tỉnh). `requirements.md` (1.12 → 1.13): Status/Pending theo đó. `design.md`, `flow.md`: no change needed, verified — chỉ header.
- Việc đã thực hiện trên DB dev `innerlamp`: `0032` (chỉ thêm cột và CHECK, idempotent) và `seedtemples` (upsert cùng 93 id, ghi đè các cột từ file). Không xoá dữ liệu nào.

## v1.12 (2026-10-11)

Hoàn tất Phase 10: dữ liệu, seed, E2E, tài liệu API. Theo yêu cầu của anh Đức: "em tự xử hết cho anh".

- `requirements.md` (1.11 → 1.12): Status và "Pending" ghi lại trạng thái (đã implement; còn áp DB dev, `erd.svg`, OKF, xác minh 6 chùa, AC 38 chờ `create-temple`). AC không đổi.
- `design.md` (1.11 → 1.12): hàng "Domain / Province" ghi `domain/province.go` đã gỡ; Known gaps thêm: chùa chưa nạp lại vẫn đúng cho DB chưa seed, 6 chùa xã chưa chắc chắn (A-005), `temple.json` không nằm trong git. `flow.md`: no change needed, verified.
- `tasks.md` (1.11 → 1.12): đánh `[x]` các mục duyệt, áp dụng dữ liệu, `parse.go`, `resolve.go`/`main.go`, build/vet/test, E2E, openapi + web docs, `api-contract.md`, gỡ `province.go`; còn `[ ]` áp DB dev, `erd.svg`, OKF.
- **Quyết định thay mặt anh** (ghi ở `docs/assumptions/temple/A-005`, `unvalidated`): 25 chùa có xã cũ tách hoặc không có trong bảng cũ được gán xã mới bằng tra nguồn (19 `high`, 5 `medium`, 1 `unverified`); 20 chùa giữ `address` rỗng (A-004).
- Kiểm chứng: `go build ./...`, `go vet ./services/appService/...`, `go test -short ./services/appService/...` sạch; `TEST_DB_NAME=innerlamp_test` test repository temple pass; `go test ./services/appService/e2e/ -run TestTemple_E2E` pass (binary thật + Postgres container); seed thật trên `innerlamp_test` (dataset rỗng → thất bại; sau `seedadminunits` → 93 chùa; chạy lại → vẫn 93).
- Chưa làm: áp `0032` và nạp lại lên DB dev `innerlamp` (**code đọc `province_code` nên DB dev sẽ lỗi cột cho tới khi áp**), `docs/specs/db/erd.svg`, OKF `docs/knowledge/features/temple.md`, và AC 38 (cần `create-temple`).

## v1.11 (2026-10-11)

Làm phần code không phụ thuộc dữ liệu của Phase 10 (domain, repository, DTO). Không đổi yêu cầu hay thiết kế.

- `tasks.md` (1.10 → 1.11): đánh `[x]` ba mục — domain (`Temple` + `FullAddress()` + `TempleOption.ProvinceCode`), repository (đọc/ghi ba cột), handler DTO — kèm bằng chứng kiểm chứng; Status ghi lại.
- `requirements.md`, `design.md`, `flow.md` (1.10 → 1.11): no change needed, verified — chỉ cập nhật header. Code khớp thiết kế: `FullAddress()` ở domain, đọc bằng `COALESCE`, ghi bằng `NULLIF`, DTO `null` khi chưa nạp lại.
- Kiểm chứng: `go build ./...`, `go vet` (domain, repository, delivery/http), `go test -short ./services/appService/...` sạch; `TEST_DB_NAME=innerlamp_test go test ./services/appService/repository/ -run Temple` pass trên Postgres thật. Chưa làm: seed (`parse.go` + bước đối chiếu mã), E2E, openapi — chờ dữ liệu đã duyệt.
- Ghi nhận: `appService/e2e` không có E2E temple nào (chỉ có `static_images`); E2E `options`/chi tiết vẫn là việc của Phase 5/10.

## v1.10 (2026-10-11)

Làm script đề xuất mã (Phase 10, mục dữ liệu). Kết quả chạy cho thấy 20/93 chùa không có phần dưới cấp xã nên AC 34 được nới.

- `requirements.md` (1.9 → 1.10): AC 34 — `location.address` phải có mặt nhưng **được rỗng**; chỉ `province_code`/`ward_code` bắt buộc không rỗng. Lý do: 20 chùa có địa chỉ cũ chỉ gồm xã/huyện/tỉnh (vd `Xã Hương Sơn, Huyện Mỹ Đức, Hà Nội`), không có gì để đặt vào `address`; AC 33 vốn đã ghép các phần không rỗng nên `full_address` vẫn đúng. `create-temple` không đổi (vẫn bắt buộc `address`).
- `design.md` (1.9 → 1.10): Step 2 của flow seed và hàng "Seed CLI" theo AC 34 mới. `flow.md`: no change needed, verified — chỉ header (Step 2 chỉ nói "không rỗng" ở mức chung, không liệt kê field).
- `tasks.md` (1.9 → 1.10): đánh `[x]` mục script kèm số liệu (68 `auto`, 25 `review`, 20 `address` trống); mục người duyệt trỏ tới `address-review.md`.
- Quyết định kèm theo (người dùng chưa chỉ định, spec chọn): `docs/assumptions/temple/A-004` — `address` được rỗng ở seed.
- File sinh ra (không thuộc spec): `scripts/build_temple_addresses.py`, `assets/data/temple-address-proposal.json`, `docs/specs/features/temple/address-review.md`. Không có thay đổi nào ở `assets/data/temple.json`.

## v1.9 (2026-10-11)

Làm migration `0032` (Phase 10, mục đầu). Không đổi yêu cầu hay thiết kế.

- `tasks.md` (1.8 → 1.9): đánh `[x]` mục migration kèm bằng chứng kiểm chứng (Postgres tạm: dòng cũ giữ nguyên và `NULL`, idempotent, CHECK từ chối bộ ba đi lẻ); Status ghi lại.
- `requirements.md`, `design.md`, `flow.md` (1.8 → 1.9): no change needed, verified — chỉ cập nhật header. Thiết kế khớp file đã viết (cột, kiểu, tên constraint, không FK, không backfill).
- Việc ngoài thư mục này đã làm: `migrations/0032_temple_address_codes.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd`, danh sách migration trong `postgres_registration_repository_test.go`. Còn lại: `docs/specs/db/erd.svg` (cần `mmdc`).

## v1.8 (2026-10-11)

Sửa số liệu về 93 chùa và chốt tên tỉnh trong `full_address`. Theo yêu cầu của anh Đức: "sửa số liệu lại 93", "giữ Hồ Chí Minh không cần Thành phố". Không đổi hành vi.

- `requirements.md` (1.7 → 1.8): thay "179 chùa" bằng số đếm lại từ `assets/data/temple.json` (93 chùa; 32/32/29 theo vùng; `type` và `heritage_rank` đếm lại; cả 93 có `image_url`/`thumbnail_url`); sửa AC 11 và marker về ảnh cho khớp (file seed có đường dẫn ảnh cho cả 93 chùa, nhưng repo không có thư mục `static/images/`); **đóng marker `⚠️ Not confirmed` về tiền tố tỉnh** — chốt giữ tên ngắn (`Hồ Chí Minh`), AC 33 không đổi.
- `design.md` (1.7 → 1.8): hàng "Data" và Known gaps ("Không có E2E", "Ảnh") theo số liệu mới.
- `flow.md` (1.7 → 1.8): no change needed, verified — chỉ cập nhật header.
- `tasks.md` (1.7 → 1.8): sửa số chùa ở Phase 3.
- `docs/assumptions/temple/A-002`: ghi phần tiền tố đã được xác nhận.
- Không sửa (cố ý): `api-contract.md` và `docs/knowledge/` (OKF) vẫn ghi 179 vì là bản ghi của một lần chạy trước đó; OKF do `il-okf-writer` làm mới sau khi code kiểm chứng.

## v1.7 (2026-10-11)

Địa chỉ chùa theo metadata hành chính: thêm `province_code` + `ward_code` (+ `ward` tên chụp), `address` đổi nghĩa thành phần dưới cấp xã, API thêm `full_address`. Theo yêu cầu của anh Đức: dữ liệu `options` đang seed ít khi chạy, cần bổ sung mã tỉnh/xã chính thức; mọi chỗ có địa chỉ gồm `province_code` + `ward_code` + `address` (số nhà, đường…). Chưa có code — Phase 10.

- `requirements.md` (1.6 → 1.7): thêm **US-TMP-08** (AC 32–38); AC 2 và AC 20 cập nhật (`location` thêm trường; `options` có bốn field); bảng schema `TempleSummary`/`TempleOption`; ghi chú đặt AC 28/30 là bị thay ở bước nạp; thêm một marker `⚠️ Not confirmed` (tiền tố tỉnh trong `full_address`); câu hỏi đã chốt ghi ở "Open questions (resolved)"; "Pending" thêm Phase 10.
- `design.md` (1.6 → 1.7): hàng mới "Domain / Address" và "Migration `0032`"; sửa "Domain / Option", "Domain / Province" (hết vai trò), "Repository", "Seed CLI", "Data", "Delivery / DTO"; Step 2 của flow seed đổi kiểm `province` → kiểm mã không rỗng; **thêm Step 4 "đối chiếu mã"** và đánh số lại Step 5–6; Step 4–5 của flow `options` thêm `province_code`; Known gaps thêm ba mục (chưa nạp lại, seed phụ thuộc dataset, gán mã là suy đoán).
- `flow.md` (1.6 → 1.7): sơ đồ `options` thêm `province_code`; sơ đồ seed thêm participant `WardResolver` và Step 4 mới, mirror 1:1 `design.md`.
- `tasks.md` (1.6 → 1.7): thêm **Phase 10** (mọi mục `[ ]`, gồm bước dữ liệu có người duyệt); ba mục `[ ]` cũ của Phase 8–9 trỏ sang Phase 10 / `temple-suggestion` v3.2, không đánh dấu `[x]`.
- Quyết định kèm theo (người dùng chưa chỉ định, spec chọn — `docs/assumptions/temple/`): (1) **A-001** không FK, tên chụp, ba cột cùng có hoặc cùng `NULL`, không backfill; (2) **A-002** `address` đổi nghĩa + `full_address` ghép tên ngắn của tỉnh; (3) **A-003** `options` chỉ thêm `province_code`, không `ward_*`, không lọc.
- Việc cần làm ngoài thư mục này (chưa làm): `migrations/0032_temple_address_codes.sql`, `docs/specs/api/openapi.yaml` + `web/docs`, `docs/specs/db/erd.*`, `temple/api-contract.md`, OKF `docs/knowledge/features/temple.md`, cascade sang `temple-suggestion` v3.2 (đã làm cùng lượt).
- Ghi nhận (đã sửa ở v1.8): nhiều chỗ trong spec này vẫn nói "179 chùa" trong khi `assets/data/temple.json` và DB dev hiện có 93 chùa (v1.5–v1.6 đã ghi 93); số 179 là từ trước, không đổi trong v1.7.

## v1.6 (2026-10-10)

`temples.province` chỉ nhận **34 đơn vị cấp tỉnh** hiện hành (từ 1/7/2025, Nghị quyết 202/2025/QH15) thay vì chuỗi tự do. Theo yêu cầu của người dùng: "dùng 34 mới".

- `requirements.md`: AC 28 siết lại (phải thuộc danh sách); thêm AC 30 (danh sách 34 đơn vị, không tiền tố, seed từ chối tên ngoài danh sách kể cả tên cũ/có tiền tố/viết tắt) và AC 31 (migration `0025` đổi tên cũ sang mới, không đụng `address`, idempotent). Đóng marker `⚠️ Not confirmed` về danh sách tên chuẩn (câu trả lời ghi ở "Open questions (resolved)"). Thêm ghi chú: `address` không được cập nhật theo đơn vị mới.
- `design.md`: thêm hàng "Domain / Province" (`domain/province.go`), `0025` ở hàng Migration, seed kiểm theo danh sách; Known gaps: `address` chưa theo đơn vị mới (cố ý), danh sách nằm trong code và `backofficeService` chưa dùng.
- `flow.md`: chỉ tăng version, không đổi (đã kiểm — không có Step mới ở flow `options`; seed flow Step 2 đã nêu kiểm `province` ở `design.md`).
- `tasks.md`: thêm Phase 9, đã tick các mục đã làm và kiểm chứng; bỏ mục "kiểm tên theo danh sách chuẩn" của Phase 8 vì đã thành AC 30; còn `[ ]` cho cập nhật `address` và form `create-temple`.
- Reason: một cột `province` chỉ có giá trị tra cứu/lọc khi tên nhất quán; chuỗi tự do cho phép `Hà Nam` và `Ninh Bình` cùng tồn tại cho cùng một nơi sau sáp nhập.
- Quyết định kèm theo (người dùng chưa chỉ định, spec chọn): (1) danh sách đặt trong code (`domain/province.go`), không `CHECK` ở DB và không phải config, vì là dữ liệu cố định theo luật và đổi cùng một migration dữ liệu; (2) giá trị lưu không tiền tố (`Huế`, `Hồ Chí Minh`), như v1.5; (3) `Thừa Thiên Huế` → `Huế`; `Bà Rịa - Vũng Tàu` khớp theo tiền tố vì nguồn viết bằng nhiều dấu gạch khác nhau; (4) migration chỉ liệt kê tên **thay đổi** nên không bao giờ ghi đè tên đã ở dạng mới; (5) không sửa `address`.
- Nguồn bảng sáp nhập: Nghị quyết 202/2025/QH15 theo các trang tổng hợp của thuvienphapluat.vn; các nguồn có chỗ không khớp nhau về chi tiết nên bảng đã được đối chiếu giữa nhiều nguồn, **chưa** đối chiếu trực tiếp với văn bản Công báo.
- Kiểm chứng: `go build`, `go vet`, `go test -short ./services/appService/...` pass; migration `0025` kiểm trên DB tạm (66 dòng, 0 lệch, 34 giá trị, idempotent); DB dev `innerlamp` đã áp `0025` và nạp lại `temples` (93 chùa, 24 đơn vị).
- Follow-up ngoài phạm vi spec (không sửa ở đây): spec `temple-suggestion` (form `create-temple` chọn `province` từ 34 đơn vị; `province` thô của người gửi nên được dùng làm gợi ý); `docs/specs/features/temple/api-contract.md`; `docs/specs/db/erd.svg`; OKF `docs/knowledge/features/temple.md`.

## v1.5 (2026-10-10)

Hai thay đổi theo yêu cầu của người dùng: (1) `GET /v1/temples/options` thêm tìm theo tên không dấu `?q=`; (2) thêm cột `temples.province` — migration tự tách từ `address` rồi dữ liệu được nạp lại — thay cho việc suy ra `province` lúc phục vụ request.

- `requirements.md`: US-TMP-06 thêm AC 23–25 (`q` khớp một phần chuỗi, không phân biệt hoa thường và dấu, `đ`↔`d`, `%`/`_`/`\` hiểu đúng nghĩa đen; `q` rỗng hoặc quá 100 ký tự → `400 INVALID_FILTER`; không khớp → `items: []`); AC 20 đổi nguồn của `province` sang cột. Thêm US-TMP-07 (AC 26–29: cột riêng, `location.province` ở summary/detail, seed bắt buộc `province`, migration backfill idempotent). AC 2 và AC 15 cập nhật. Đóng marker `⚠️ Not confirmed` về `?q=`; thêm marker về danh sách tên tỉnh chuẩn (34 mới hay 63 cũ) — người dùng chưa trả lời. Hai câu trả lời ghi ở "Open questions (resolved)".
- `design.md`: bỏ hàng "Domain / Province" (hàm suy ra bị gỡ khỏi code); `ListOptions(ctx, query, offset, limit)`, `Options(ctx, query, page)`, lỗi `ErrInvalidTempleQuery` → `400 INVALID_FILTER`, `locationResponse.province`, migration `0024`, seed kiểm `province`; Step-by-step Flow của `options` thêm kiểm `q` và điều kiện `unaccent`; Known gaps: `province` và `address` có thể lệch nhau, tên tỉnh chưa theo danh sách chuẩn, tìm không dấu không có index và chỉ khớp `name`.
- `flow.md`: sequence diagram của `options` thêm nhánh `q`, mirror 1:1 Step-by-step Flow.
- `tasks.md`: Phase 7 viết lại theo v1.5 và thêm Phase 8; các mục đã làm được tick sau khi kiểm chứng (xem dưới); còn `[ ]` cho E2E `options`, render lại `erd.svg`, kiểm theo danh sách tên chuẩn, form `create-temple`.
- Reason: màn đăng ký cần tìm chùa theo tên (người dùng hay gõ không dấu) và cần tỉnh/thành tường minh thay vì đoán từ chuỗi địa chỉ tự do.
- Quyết định kèm theo (người dùng chưa chỉ định, spec chọn): (1) `?q=` chỉ khớp `name`, không khớp `aliases`/`province`, không xếp theo độ liên quan; (2) `q` giới hạn 100 ký tự (hằng số, theo tiền lệ `GET /v1/sutras?q=`), `?q=` rỗng là lỗi chứ không phải "không tìm" (như sutra); (3) `province` ≤ 100 ký tự khớp `temple_suggestions.province`; (4) không kiểm chéo `province` với `address`; (5) không validate tên tỉnh theo danh sách chuẩn; (6) hằng số `maxProvinceRunes` và `maxTempleQueryRunes` ở code (độ rộng cột/giới hạn đầu vào), không đưa vào config.
- Kiểm chứng: `go build ./...`, `go vet ./services/appService/...`, `go test -short ./services/appService/...` pass; test repository (gồm `ListOptions` và không dấu) pass trên DB Postgres tạm; `go test ./services/appService/e2e/` pass (binary thật + Postgres container); migration `0024` kiểm trên DB tạm (93/93 khớp JSON, ca biên đúng, áp lại idempotent); gọi API thật qua HTTP trên DB tạm. DB dev `innerlamp` đã được áp `0024` và nạp lại `temples` (93 chùa).
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/db/erd.svg` cần render lại từ `erd.mmd`; spec `temple-suggestion` (form `create-temple` cần field `province` chọn từ danh sách chuẩn); `docs/specs/features/temple/api-contract.md` (tài liệu client iOS) chưa có `options`, `q`, `location.province`; spec onboarding và guide `register-and-login.mdx` vẫn trỏ `GET /v1/temples`; `web/docs/.../guides/catalogues/temples.md`; OKF `docs/knowledge/features/temple.md`.

## v1.4 (2026-10-10)

`GET /v1/temples/options` chuyển từ "toàn bộ, không phân trang, trả `address`" sang **phân trang** và trả **`province`** (tên tỉnh/thành cắt gọn). Theo yêu cầu của người dùng: "trả theo trang (cho tương lai số lượng lớn hơn)" và "địa chỉ cắt gọn, Hà Nội, Hồ Chí Minh…".

- `requirements.md`: US-TMP-06 viết lại (user story + AC 19–22): phân trang với `page_size` = `temple.options_page_size`, `province` thay `address`, `INVALID_PAGE`, trang vượt cuối. Bảng API contract và schema `TempleOption` cập nhật. Hai marker `⚠️ Not confirmed` của v1.3 (tìm kiếm phía server; định dạng `address`) — câu về `address` đã chốt và chuyển sang "Open questions (resolved)", câu về tìm kiếm giữ mở và nâng mức độ (không còn lọc client trên toàn bộ danh sách).
- `design.md`: hàng mới "Domain / Province" (`ProvinceFromAddress`), `TempleOptionPage`, `ListOptions(ctx, offset, limit)`, `TempleConfig.OptionsPageSize` + `Validate`; thêm Step-by-step Flow cho `options` (đếm rồi lấy trang, cùng khung với danh sách phân trang); bỏ cây ASCII một câu SQL của v1.3; Known gaps: `province` suy ra chứ không phải cột (đánh đổi và đường nâng cấp), hai câu lệnh không cùng snapshot.
- `flow.md`: thêm sequence diagram cho `options`, mirror 1:1 Step-by-step Flow mới.
- `tasks.md`: Phase 7 viết lại theo v1.4; `[x]` cho domain (+ test `ProvinceFromAddress`), mock, config + `Validate`, usecase, handler, `cmd/main.go`, openapi; còn `[ ]` cho test repository (chưa chạy, cần Postgres) và E2E. Code v1.4 đã làm cùng lượt: build, vet, `go test -short ./services/appService/...` pass; kiểm hàm cắt trên 93 chùa trong `assets/data/temple.json` ra 37 tỉnh/thành sạch.
- Reason: pick-list đăng ký cần nhẹ khi danh mục lớn lên; địa chỉ đầy đủ (xã, huyện, tỉnh) dài và không dùng ở màn đăng ký. Một giá trị gọn `province` đủ để người dùng phân biệt các chùa trùng tên.
- Quyết định kèm theo (người dùng chưa chỉ định, spec chọn): (1) field đặt tên `province` thay vì giữ `address` vì giá trị không còn là địa chỉ; (2) `province` suy ra từ `address` bằng hàm thuần thay vì thêm cột `province` — không cần migration và chạy ngay với dữ liệu đã seed (kiểm 93/93 chùa trong `assets/data/temple.json` có đoạn cuối là tỉnh/thành); (3) page size riêng `temple.options_page_size` (dev: 100), tách khỏi `temple.page_size` (20) vì pick-list cần trang lớn hơn danh sách duyệt, và bắt buộc `>= 1` lúc khởi động.
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/api/openapi.yaml` + `web/docs/public/openapi.yaml` (cập nhật cùng lượt implement v1.4); `docs/specs/features/temple/api-contract.md` chưa có mục `options`; `docs/specs/features/onboarding/requirements.md` và guide `register-and-login.mdx` vẫn trỏ `GET /v1/temples`; OKF `docs/knowledge/features/temple.md`. Ghi chú: spec nhiều chỗ còn nêu "179 chùa" trong khi `assets/data/temple.json` hiện có 93 chùa và ví dụ openapi ghi `total: 93` — chưa sửa vì ngoài phạm vi, cần rà riêng.

## v1.3 (2026-10-10)

Thêm `GET /v1/temples/options` — danh sách chùa rút gọn (`id`, `name`, `address`), không phân trang, cho màn đăng ký chọn nơi quy y/xuất gia. Spec và code cùng lượt: đã implement domain/mock/usecase/handler/`cmd/main.go` + openapi (build, vet, `go test -short ./services/appService/...` pass); `ListOptions` ở repository đã có code và test nhưng chưa chạy test (cần Postgres, xoá dữ liệu dev); chưa có E2E.

- `requirements.md`: thêm US-TMP-06 với AC 19–21 (EARS), dòng `options` và schema `TempleOption` trong bảng API contract, hai marker `⚠️ Not confirmed` (tìm kiếm phía server; `address` nguyên chuỗi hay chỉ tỉnh/thành — mặc định: không tìm kiếm, nguyên chuỗi). Cập nhật Out of scope, Ghi chú kỹ thuật (kích thước phản hồi không giới hạn).
- `design.md`: thêm hàng `TempleOption`, `ListOptions`, `Options`, DTO, route và OpenAPI vào bảng Layers (đánh dấu chưa tồn tại); thêm cây request flow cho `options`; thêm Known gap (không phân trang, thứ tự theo ranking chứ không theo bảng chữ cái).
- `flow.md`: chỉ tăng version, không đổi — `options` chỉ một câu SQL, không có side effect nên không có Step-by-step Flow mới (đã kiểm).
- `tasks.md`: thêm Phase 7; `[x]` cho domain, mock, usecase, handler, `cmd/main.go`, openapi; còn `[ ]` cho test repository (chưa chạy) và E2E.
- Reason: màn đăng ký chỉ cần tên và địa điểm chùa; `GET /v1/temples` trả 10 field mỗi phần tử (có URL ảnh lớn, toạ độ), phân trang 20/trang nên phải gọi nhiều trang và không có tìm kiếm. Người dùng chọn phương án endpoint riêng thay cho `?fields=` hoặc giữ nguyên.
- Quyết định kèm theo: không thêm migration, không thêm config (dùng lại bảng `temples`, `templeOrder`, route công khai như 3 route cũ).
- Đã cập nhật cùng lượt: `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml` (route, `TempleOption`, `TempleOptionsEnvelope`, mô tả `ordination_temple_id`).
- Follow-up ngoài phạm vi spec (không sửa ở đây): `docs/specs/features/temple/api-contract.md` (tài liệu client iOS, giữ nguyên) chưa có mục `options`; `docs/specs/features/onboarding/requirements.md` dòng `ordination_temple_id` vẫn trỏ `GET /v1/temples` (nên trỏ thêm `options`); `web/docs/.../guides/catalogues/temples.md` và `register-and-login.mdx`; OKF `docs/knowledge/features/temple.md` cần làm mới sau khi verify.

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
