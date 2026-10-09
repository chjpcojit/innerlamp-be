# Changelog — sutra Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v2.13 (2026-10-08)

Người dùng yêu cầu API phải trả `summary` cho client ở danh sách, không chỉ ở chi tiết. Đảo quyết định v2.12 (danh sách giữ dạng thẻ không có `summary`), nên là bản minor.

- `requirements.md`: AC41 mới (`summary` có ở thẻ danh sách và by-deity); AC36 chỉ còn năm trường chi tiết-only; bảng hợp đồng `SutraSummary` thêm `summary`.
- `design.md`: Step 1 của flow Chi tiết — `summary` thuộc cột chung của danh sách.
- `flow.md`, `tasks.md`: đổi số version; `tasks.md` thêm mục v2.13 đã `[x]` kèm bằng chứng.
- Reason: client hiển thị mô tả ngay trên thẻ, không phải gọi chi tiết cho từng kinh. Đổi lại mỗi trang danh sách nặng hơn (`summary` 2–4 câu mỗi văn bản).

## v2.12 (2026-10-07)

Người dùng hỏi vì sao `GET /v1/sutras/{id}` trả quá ít thông tin, rồi giao em tách các trường đang gộp chung và tự soạn tóm tắt. Thêm trường vào hợp đồng nên là bản minor.

- `requirements.md`: AC36–AC40 mới (sáu trường chỉ ở chi tiết; `reference` chỉ còn mã trích dẫn; `license` là mã trong tập đóng + `license_url`/`license_note`; `UNKNOWN` không ẩn văn bản; seed kiểm tập mã và `hanTranslationYear`); mở rộng API contract, Out of scope, Pending; thêm một marker `⚠️ Not confirmed` (cách áp dữ liệu mới lên DB đã seed) và mục resolved cho quyết định tách.
- `design.md`: Layers (domain, parser, migration `0020`, data, DTO), mô hình dữ liệu (8 cột mới), bảng "Dữ liệu cần bổ sung", Step 1–2–4 của flow Chi tiết và Step 2 của flow Nạp dữ liệu, Known gaps.
- `flow.md`: chỉ đổi câu SELECT bản dịch của Chi tiết (thêm `license_url`, `license_note`); không thêm hay bớt Step nào.
- `tasks.md`: thêm Phase 6 (viết spec trước khi có code, toàn `[ ]`); sau khi code và test chạy thật thì tick `[x]` từng mục kèm bằng chứng, trừ mục "Áp lên DB đã seed" còn chờ người dùng. Ngoài phạm vi: `TestSutraRepository_ShippedData` có hai số cũ đã lỗi thời từ trước (liên kết Phật/Bồ tát 151 → 191, Thích Ca 81 → 113 văn bản — `TestParseSutraSeed_ShippedData` đã dùng 191); đã sửa theo dữ liệu hiện có.
- Reason: `reference` chứa mã kinh + người dịch Hán + năm trong một chuỗi và `license` chứa mã + URL + câu giải thích, client không hiển thị riêng được; chưa có mô tả ngắn cho mỗi bài.
- Cross-file follow-ups: Đã làm: `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`; `migrations/0020_*.sql`, `docs/specs/db/erd.*`, `migrations/README.md`, `assets/data/sutras/README.md`. Còn lại: `docs/knowledge/features/sutra.md` (`il-okf-writer` làm sau khi code đã kiểm chứng); `docs/assumptions/sutra/A-010`, `A-011`; `docs/tech-debt/sutra/TD-002` (tóm tắt chưa duyệt).

## v2.11 (2026-10-03)

Đóng mục cuối cùng còn mở của Phase 0. Không đổi yêu cầu, thiết kế hay dữ liệu nên là bản minor.

- `requirements.md`: Status và mục Pending — việc soát `durationMinutes` được ghi là đã đóng (người dùng giữ nguyên giá trị đã sinh); `purposes` vẫn ghi là chưa được soát. Không đổi AC nào (AC26 vẫn đúng: giá trị trả về nguyên như đã lưu).
- `design.md`: chỉ sửa Status (bỏ "việc người dùng soát `durationMinutes`"); Layers, mô hình dữ liệu và flow không đổi. Dòng "Dữ liệu cần bổ sung" của `durationMinutes` vẫn đúng (script gợi ý rồi làm tròn; chỉnh tay là tuỳ chọn), đã kiểm.
- `flow.md`: chỉ tăng version, không đổi (đã kiểm).
- `tasks.md`: tick `[x]` mục "Soát và chỉnh tay `durationMinutes`" kèm lý do (giữ nguyên, không sửa dữ liệu); Status không còn mục `[ ]` nào. Không có mã hay dữ liệu nào thay đổi nên không cần chạy lại build/test.
- Reason: người dùng quyết định giữ nguyên các giá trị `durationMinutes` đã sinh (`ceil(âm tiết / 80)` rồi làm tròn).
- Cross-file follow-ups: `docs/tech-debt/sutra/TD-007` (exit criteria vẫn `[ ]`, cần chuyển sang chấp nhận/đóng) và `docs/assumptions/sutra/A-005` (trạng thái `unvalidated`) — chưa sửa ở lượt này; `docs/knowledge/features/sutra.md` đã được `il-okf-writer` cập nhật cùng ngày.

## v2.10 (2026-10-03)

Spec `practice` đã bị xoá vì cũ; bỏ các câu nhắc tới nó. Không đổi yêu cầu nào nên là bản minor.

- `requirements.md`, `tasks.md`: dòng Out of scope "thuộc module `journal` / `practice`" thành "thuộc module `journal`".
- `design.md`, `flow.md`: chỉ tăng version, không đổi (đã kiểm).

## v2.9 (2026-10-03)

Thêm target Makefile để seed dữ liệu, tạo OKF và ghi nợ kỹ thuật/giả định. Không đổi yêu cầu nào nên là bản minor.

- `requirements.md`, `flow.md`: chỉ tăng version, không đổi nội dung (đã kiểm: hành vi của API và luồng seed không đổi).
- `design.md`: dòng Known gaps "Dev local cần `cmd/seeddeities`" viết lại — `Makefile` đã có `ensure-deities` (nối vào `run`), `seed-deities`, `seed-sutras`, `seed`.
- `tasks.md`: thêm các mục Makefile và OKF/nợ kỹ thuật (đã `[x]`, đã kiểm chứng); bỏ câu "không sửa `Makefile`".
- Cross-file follow-ups: đã tạo `docs/knowledge/features/sutra.md` (draft); nợ kỹ thuật ở `docs/tech-debt/sutra/`, giả định ở `docs/assumptions/sutra/`.

## v2.8 (2026-10-03)

Phase 5 xong: E2E riêng cho sutra, các số đo, ERD và tài liệu. Chỉ cập nhật trạng thái, bằng chứng và các dòng không còn đúng; không đổi yêu cầu nào nên là bản minor.

- `requirements.md`: Status nói cả 35 tiêu chí đã có test chứng minh (kể cả E2E chạy binary thật); mục Pending chỉ còn việc người dùng soát `durationMinutes`; số đo khởi động và kích thước binary thay cho câu "chưa đo". Không đổi AC nào.
- `design.md`: Status; hàng "DB spec" đã cập nhật; Known gaps viết lại cho đúng thực tế — kích thước binary và thời gian khởi động có số đo (+10,25 MB, +17%; 1,82 s lần đầu, 0,13 s các lần sau), "Chưa có E2E" thay bằng các giới hạn còn lại của E2E, và "Dev local cần `cmd/seeddeities`" ghi là đã có tài liệu. Layers, mô hình dữ liệu và flow không đổi.
- `flow.md`: no change needed to any step, verified (cả năm flow giờ cũng chạy thật qua `e2e/sutra_e2e_test.go`); chỉ đổi Status.
- `tasks.md`: tick `[x]` bảy mục Phase 5, mỗi mục kèm bằng chứng (số đo, tên test, phạm vi đã chạy). Còn đúng 1 mục `[ ]` trên toàn spec: soát và chỉnh tay `durationMinutes` (Phase 0, cần người dùng).
- `test-coverage-report.md` (mới, theo skill `golang-e2e-verification`): bảng 35 AC với test chứng minh từng cái, bốn giới hạn của E2E, kết quả 6 đột biến.
- Reason: hoàn tất Phase 5. Bằng chứng: toàn repo `go build ./...`, `go vet ./...`, `go test -short ./...` thoát 0; bộ test **không** `-short` trên một Postgres mới + Docker: 764 test pass, 0 lỗi (gồm 12 test repository sutra và 7 hàm E2E sutra mới, chạy khoảng 30 s); 6 đột biến cố ý trên code thật đều làm đúng test E2E tương ứng đỏ rồi đã hoàn nguyên; số đo thời gian/kích thước trên Postgres cục bộ (loopback, chưa đo dưới tải hay qua mạng thật); ERD tái tạo bằng `@mermaid-js/mermaid-cli`, công cụ này tái tạo bản ERD cũ gần như y hệt `erd.svg` sẵn có.
- Một phát hiện **ngoài sutra** được sửa luôn vì nó chặn việc chạy sạch bộ test trên DB mới: `applyMigration` trong `repository/postgres_registration_repository_test.go` thiếu `0011_temple_content.sql`, nên 5 test `TestTempleRepository_*` lỗi (`column "introduction" does not exist`) trên mọi DB mới — từ trước khi có sutra (helper ở `HEAD` cũng không có `0011`); từng không lộ vì DB dev của người dùng đã áp đủ migration. Đã thêm đúng một dòng vào danh sách.

### Việc theo dõi ngoài thư mục này

- **Đã làm ngày 2026-10-03**: `docs/specs/db/erd.mmd` + `erd.svg`; `migrations/README.md`; `docs/development/local-stack.md`; `services/appService/e2e/harness.go`; `repository/postgres_registration_repository_test.go` (thêm `0011`).
- **`docs/knowledge/features/sutra.md`**: do `il-okf-writer` tạo. Hiện **đã đủ điều kiện** (feature chạy được từ đầu đến cuối, có E2E chạy binary thật); chưa tạo — chờ người dùng yêu cầu.
- **Cần người dùng quyết**: `sutraSeedTimeout` (2 phút, hằng số trong `cmd/main.go` và `cmd/seedsutras/main.go`) có nên là field trong `config/`; `Makefile` có nên có target seed `deities` (chưa nối vào `make run` vì `seeddeities` là upsert và sẽ ghi đè chỉnh sửa tay mỗi lần chạy).
- **Không nằm trong git** (xem `git status`): `assets/` (11 MB dữ liệu kinh và `embed.go` mà binary cần để build) là `??`; hai file `openapi.yaml` bị `.gitignore` dòng 64 (`*.yaml`) bỏ qua, nên các thay đổi API (kể cả của các feature trước) không hiện trong `git status` trừ khi `git add -f`.
- Các việc cũ vẫn còn như v2.4/v2.7: `practice/requirements.md`, `AGENTS.md`, `README.md` dòng 159/350, marker mã user story trong spec `article`/`deity`/`temple`.

## v2.7 (2026-10-03)

Phase 1–4 đã được implement và kiểm chứng. Spec chỉ được cập nhật cho khớp với code, không đổi yêu cầu nào nên là bản minor.

- `requirements.md`: Status chuyển sang implemented (Phase 1–4); đoạn nói "chưa có code" và "hợp đồng đề xuất" được viết lại vì `openapi.yaml` đã có 4 route; mục Pending còn: E2E riêng cho sutra, ERD, `migrations/README.md`, tài liệu dev local, soát `durationMinutes`; số đo thời gian nạp lần đầu (~1,2 giây) được ghi vào Ghi chú kỹ thuật. Không đổi AC nào (35 AC, 6 US).
- `design.md`: Status; bỏ các nhãn "not yet created"/"need modification" khỏi bảng Layers vì các file đã có; **bốn chỗ lệch so với thiết kế v2.6 được sửa cho khớp code**: (1) `reference` là `TEXT` thay vì `VARCHAR(128)`; (2) `SutraConfig.Validate()` gọi từ `cmd/main.go` thay vì `Load` tự kiểm tra; (3) `cmd/seedsutras` chỉ có `main.go` (parser nằm ở `repository/sutra_seed.go`); (4) thêm hàng E2E harness vào bảng Layers. Known gaps thêm: E2E riêng cho sutra còn thiếu, `sutraSeedTimeout`, dev local cần `cmd/seeddeities`, `TRUNCATE deities` xoá luôn liên kết sutra. Mô hình dữ liệu và 5 Step-by-step Flow không đổi.
- `flow.md`: no change needed to any step, verified against `usecase/sutra_usecase.go`, `repository/postgres_sutra_repository.go` and `delivery/http/sutra_handler.go`; chỉ đổi Status và đoạn mở đầu.
- `tasks.md`: tick `[x]` 28 mục Phase 1–4 và thêm 1 mục mới `[x]` (harness E2E); sửa lời mục `SutraConfig` (`Validate` thay vì "`Load` báo lỗi"); Phase 5 vẫn `[ ]` nhưng 4 mục có ghi bằng chứng đã thu được; Status viết lại. Còn 8 mục `[ ]` (1 ở Phase 0, 7 ở Phase 5).
- Reason: code đã chạy thật. Bằng chứng: `go build ./...`, `go vet` và `go test -short ./...` sạch; `domain` không import lớp ngoài (`check_dependency_rule.sh` pass); 12 test repository pass trên Postgres 16 thật, gồm nạp cả thư viện (153/391/64.009/151, ~1,2 giây); bộ E2E hiện có pass lại; chạy tay binary thật qua migrate → `cmd/seeddeities` → service lần đầu (log `seeded sutras on first run`) → lần hai (không nạp lại) → `cmd/seedsutras` (đã seed: thoát 0 không ghi; trống: nạp đủ; thiếu deity: thoát 1), 4 route đúng shape và đúng mã lỗi, `/v1/deities/{id}` không đổi, thiếu block `sutra:` trong config thì service thoát 1.
- Hai phát hiện lúc implement đáng ghi lại: nạp dữ liệu thật báo `value too long` ở cột `reference` của `lang-nghiem` (135 ký tự) — test dùng dữ liệu giả không bắt được; và bộ E2E hiện có (register/login) **hỏng** ngay khi service seed sutra lúc khởi động vì harness không seed `deities` — sửa ở harness, không đổi hành vi đã chốt.

### Việc theo dõi ngoài thư mục này

- **Cần người dùng quyết**: `sutraSeedTimeout` (2 phút, hằng số trong `cmd/main.go` và `cmd/seedsutras/main.go`) có nên là một field trong `config/` hay không — xem Known gaps trong `design.md`.
- **Chưa làm (Phase 5)**: `docs/specs/db/erd.mmd` + `erd.svg`, `migrations/README.md` (liệt kê `0013`), `docs/development/local-stack.md` và/hoặc `Makefile` (phải chạy `cmd/seeddeities` trước lần chạy đầu của service, nếu không service dừng), `docs/specs/features/deity/*` (ghi chú trỏ sang `GET /v1/deities/{id}/sutras`).
- **`docs/knowledge/features/sutra.md`**: do `il-okf-writer` tạo sau khi có E2E riêng cho sutra (Phase 5); chưa tạo vì chưa có test tự động chạy binary thật cho feature này.
- Các việc cũ vẫn còn như v2.4: `practice/requirements.md`, `AGENTS.md`, `README.md` dòng 159/350, marker mã user story trong spec `article`/`deity`/`temple`.

## v2.6 (2026-10-03)

Người dùng đồng ý để em xử lý `purposes` và `durationMinutes`. Đã ghi cả hai vào 153 file JSON và `index.json`; chỉ cập nhật trạng thái và ghi nhận giá trị thực tế, không đổi yêu cầu hay cấu trúc flow nên là bản minor.

- `requirements.md`: dòng Status ghi đã có đủ bốn trường (chưa được người dùng soát `purposes`/`durationMinutes`); số liệu thời lượng ở Ghi chú kỹ thuật đổi từ "gợi ý" sang "đã ghi vào dữ liệu". AC và API contract không đổi.
- `design.md`: mục "Dữ liệu cần bổ sung" ghi đủ bốn trường đã thêm, bổ sung quy tắc làm tròn thời lượng, thêm ghi chú liệt kê các chỗ `purposes` lệch bảng đề xuất (theo tên bài), bỏ mục Known gaps "Dữ liệu JSON chưa đủ trường" vì không còn đúng. Layers, mô hình dữ liệu và flow không đổi.
- `flow.md`: no change needed, verified.
- `tasks.md`: Phase 0 tick `[x]` mục `purposes`, mục `durationMinutes` (script gợi ý và làm tròn) và mục cập nhật `index.json`/README; tách mục "soát và chỉnh tay `durationMinutes`" thành việc `[ ]` riêng; Status cập nhật. Phase 1–5 vẫn `[ ]`.
- Reason: hoàn thành phần dữ liệu của Phase 0 để seed lần đầu có dữ liệu hợp lệ. Kiểm chứng: 153/153 file có `purposes` thuộc bảng đã chốt (99 `HOC_GIAO_LY`, 15 `TRI_CHU`, 9 `THIEN_DINH`, 8 `CAU_AN`, 7 `CAU_SIEU`, 6 `SAM_HOI`, 5 `PHAT_NGUYEN`, 2 `BAO_HIEU`, 2 `HOI_HUONG`, 2 `QUY_Y_THO_GIOI`, 6 văn bản rỗng); `durationMinutes` là số nguyên ≥ 1 hoặc `null` (10 văn bản không có bản `vi`); `index.json` khớp từng file. Thời lượng theo công thức đã chốt (`ceil(âm tiết / 80)` rồi làm tròn: ≤10 giữ nguyên, 11–60 bội của 5, 61–180 bội của 10, lớn hơn bội của 30); phép làm tròn không làm đổi số văn bản ở từng khoảng.

### Việc theo dõi ngoài thư mục này

- `assets/data/sutras/README.md` đã mô tả `purposes` và `durationMinutes`. Thư mục dữ liệu vẫn **chưa được commit vào git**.

## v2.5 (2026-10-03)

Người dùng yêu cầu bổ sung `key` và `dailyChant` vào các file JSON trước khi chạy. Đã làm trên 153 file; chỉ cập nhật trạng thái spec, không đổi yêu cầu hay thiết kế nên là bản minor.

- `requirements.md`: dòng Status ghi `key` và `dailyChant` đã có, còn thiếu `purposes` và `durationMinutes`. AC và API contract không đổi.
- `design.md`: đoạn mở đầu "Dữ liệu cần bổ sung" và Known gaps ghi `key`, `dailyChant` đã thêm (34 văn bản có `dailyChant = true`), còn `purposes`, `durationMinutes`. Layers, mô hình dữ liệu và flow không đổi.
- `flow.md`: no change needed, verified.
- `tasks.md`: Phase 0 tick `[x]` mục `key` và mục `dailyChant` (đã kiểm chứng: 153/153 file có `dailyChant` kiểu bool, `key` duy nhất trong từng văn bản, `index.json` khớp từng file); mục `dailyChant` + `purposes` được tách đôi, `purposes` vẫn `[ ]`; Status và đoạn mở đầu cập nhật. Phase 1–5 vẫn `[ ]`.
- Reason: bắt đầu Phase 0. `key` sinh theo `<lang>-<variant>` (hậu tố `-2` khi trùng, chỉ `tam-kinh` trùng); `dailyChant` gán đúng bảng trong `design.md`. Hai điểm áp dụng theo bảng mà em muốn người dùng biết: `chu-lang-nghiem` là `false` (bảng chỉ cho Chú Đại Bi và Thập Chú là `true`), và các paritta Nam Tông khác ngoài Kinh Từ Bi, Kinh Hạnh Phúc (Khandha, Dhajagga, Angulimala) là `false`.

### Việc theo dõi ngoài thư mục này

- `assets/data/sutras/README.md` đã cập nhật mô tả `key` và `dailyChant` (và nêu `purposes`, `durationMinutes` còn thiếu).
- Thư mục `assets/data/sutras/` **chưa được commit vào git** (`??` trong `git status`), nên dữ liệu này hiện không có lịch sử hay bản sao lưu nào ngoài đĩa.

## v2.4 (2026-10-03)

Người dùng quyết định bỏ hẳn spec `content` cũ vì kinh, thần chú, Phật và Bồ Tát đã có feature riêng (`sutra`, `deity`). Chỉ đổi tham chiếu, không đổi hành vi hay thiết kế nên là bản minor.

- `requirements.md`: ghi chú đầu file và mục resolved về `content` đổi từ "đã sửa, US-009/US-010 giữ nguyên" sang "đã xoá hoàn toàn"; dòng Out of scope về đồng bộ delta/`audio_url`/`version` đổi sang "từng có trong spec `content` cũ (đã xoá)". AC và API contract không đổi.
- `design.md`: mục Known gaps về hợp đồng cũ của `content` viết lại thành "spec `content` cũ đã bị xoá" và nêu việc `practice/requirements.md` còn tham chiếu `audio_url`/US-006/US-008 của spec đã xoá. Layers, mô hình dữ liệu và flow không đổi.
- `flow.md`: no change needed, verified (không có bước nào liên quan).
- `tasks.md`: dòng Deferred về đồng bộ delta/`audio_url` đổi lời; không thêm, bớt hay tick mục nào.
- Reason: spec `content` cũ không đầy đủ và đã được thay bằng các feature riêng; xoá để không còn hai nguồn mâu thuẫn. Phạm vi xoá là cả thư mục `docs/specs/features/content/` (gồm US-009/US-010 về Phật & Bồ Tát, vì `deity` đã có spec riêng) — file nằm trong git nên xem lại được bằng `git show HEAD:docs/specs/features/content/requirements.md`.

### Việc theo dõi ngoài thư mục này

- **Đã sửa ngày 2026-10-03 (chỉ đổi đường dẫn bị treo)**: `README.md` dòng 165 (trỏ sang `docs/specs/features/{sutra,deity,temple,article}/`); `CLAUDE.md` dòng 11 (bỏ `content` khỏi danh sách spec prospective); `.claude/skills/il-spec-writer/SKILL.md` dòng 26 (ví dụ mẫu đổi từ `content/` sang `article/`).
- **Chưa sửa, còn tham chiếu file đã xoá**: marker `⚠️ Not confirmed` về mã user story trong `docs/specs/features/{article,deity,temple}/requirements.md` (dòng 15, nói "`content/requirements.md` chưa có US cho …"); `docs/specs/features/practice/requirements.md` (dòng 3, 10, 20, 26, 33, 38); `AGENTS.md` dòng 95 (liệt kê US-003–US-010); `README.md` dòng 159 và 350 (mục tiêu "module `content` (CRUD + delta sync API)"). Tên module `content` trong dòng `Module: \`content\`` của các spec là tên module code, **không** đổi.

## v2.3 (2026-10-03)

Người dùng chốt bốn điểm còn mở của v2.2: thay spec cũ của `content`, không có người duyệt nhãn (giao cho em), không có người kiểm tra bản Chú Đại Bi (em tự quyết), và thêm script nạp song song với tự nạp lúc khởi động. Spec không còn marker `⚠️ Not confirmed` nào. Thêm một tiêu chí và một thành phần nhưng không đổi cấu trúc flow nên là bản minor.

- `requirements.md`: bốn marker còn lại chuyển sang "Open questions (resolved)" (mã user story và hợp đồng cũ của `content` được thay bằng spec này; không có người duyệt nhãn; giữ bản Chú Đại Bi của phatgiao.org.vn; thêm script `cmd/seedsutras`); thêm AC35 cho script (cùng kiểm tra và cùng quy tắc "chỉ nạp khi bảng rỗng" như lúc khởi động, thoát 0 khi bảng đã có dòng); US-SUT-06 đổi tên và lời user story để nói cả hai cửa vào; Out of scope đổi dòng về `cmd/seedsutras` thành "không reseed, không ghi đè"; mục "cách nạp dữ liệu" đã resolved được chú thích rằng script được thêm lại ở v2.3.
- `design.md`: thêm hàng `cmd/seedsutras` vào bảng Layers & components; Overview, Step 1 và Step 5 của flow nạp dữ liệu nêu cả hai cửa vào, marker ở Step 1 đổi thành "đã chốt"; ghi chú bảng nhãn khởi tạo và mục Known gaps về nhãn/Chú Đại Bi viết lại (do em gán, không có người duyệt, `review_status` giữ `UNREVIEWED`). Không đổi mô hình dữ liệu hay số bước của flow.
- `flow.md`: participant `M` của diagram nạp dữ liệu đổi thành `cmd/main.go hoặc cmd/seedsutras`; `Note` ở Step 1 đổi từ `⚠️ Not confirmed` sang "đã chốt". Không đổi bước nào.
- `tasks.md`: bỏ mục xác định người duyệt nhãn; mục `dailyChant`/`purposes` ghi rõ "em gán"; Phase 3 thêm việc tạo `cmd/seedsutras/main.go`; Phase 5 thêm việc chạy script trên DB đã có dữ liệu; Deferred bỏ dòng về script; mọi mục vẫn `[ ]`.
- Reason: người dùng trả lời các câu hỏi còn mở. Về `content`, người dùng cho rằng bản cũ không đầy đủ nên bỏ; em hiểu phạm vi bỏ là US-005 đến US-008 (kinh điển **và** thần chú, vì `kind = DHARANI` của spec này đã bao phủ thần chú) và giữ US-009/US-010 (Chư Phật & Bồ Tát) — sửa lại nếu người dùng muốn phạm vi khác.

### Việc theo dõi ngoài thư mục này

- **Đã làm ngày 2026-10-03**: `docs/specs/features/content/requirements.md` — US-005 đến US-008 thay bằng bảng trỏ sang spec này; US-009/US-010 giữ nguyên.
- Còn lại: `docs/specs/features/practice/requirements.md` tham chiếu US-006/US-008 cũ cho `audio_url` (cần xem lại khi làm `practice`); `AGENTS.md` dòng 95 vẫn liệt kê "sutras/mantras/bodhisattvas" trong US-003–US-010 chưa có code (đúng, nhưng nên trỏ sang spec `sutra` khi implement); các việc khác như v2.0 (`openapi.yaml`, ERD, `migrations/`, `assets/data/sutras/README.md`, `docs/knowledge/features/sutra.md` sau khi chạy thật).

## v2.2 (2026-10-03)

Người dùng làm rõ hai điều: (1) dữ liệu kéo từ web về nên JSON chỉ là nguồn **nạp một lần** (chạy lần đầu hoặc dùng script), sau đó mọi API đọc từ Postgres; (2) thời lượng chỉ là **ước tính do mình ghi** (nghĩ bài cần 30 phút thì ghi 30). Không đổi phạm vi hay cấu trúc flow nên là bản minor.

- `requirements.md`: phần mở đầu nêu rõ JSON là nguồn nạp một lần và API chỉ đọc Postgres; AC26 viết lại — `duration_minutes` là giá trị đã ghi, trả nguyên không làm tròn hay tính lại, `null` khi chưa có ước tính; mục "cách ước tính `duration_minutes`" ở "Open questions (resolved)" viết lại thành "ước tính biên tập, công thức chỉ gợi ý"; mục "cách nạp dữ liệu" được bổ sung làm rõ rằng script chạy tay là cửa vào thứ hai chấp nhận được; Out of scope đổi `cmd/seedsutras` từ "đã bỏ" thành "chưa chốt"; thêm một marker `⚠️ Not confirmed` về việc có thêm script nạp làm cửa vào thứ hai hay không (không chặn implement); số tham chiếu thời lượng ở Ghi chú kỹ thuật đổi thành "gợi ý ban đầu".
- `design.md`: Overview thêm đoạn về nguồn nạp một lần và việc `ParseSutraSeed(fs.FS)` dùng được cho cả dữ liệu nhúng lẫn thư mục trên đĩa; hàng `durationMinutes` trong "Dữ liệu cần bổ sung" đổi từ "tính bằng script" sang "do mình ghi, script chỉ gợi ý"; ghi chú về tốc độ 80 âm tiết/phút viết lại cho khớp; Step 1 của flow nạp dữ liệu thêm marker về cửa vào thứ hai. Không đổi bảng layer, mô hình dữ liệu hay thứ tự các bước.
- `flow.md`: thêm một `Note` `⚠️ Not confirmed` ở Step 1 của diagram nạp dữ liệu cho khớp `design.md`; không đổi bước nào.
- `tasks.md`: mục `durationMinutes` ở Phase 0 đổi thành "script gợi ý, rồi làm tròn hoặc chỉnh tay"; Deferred thêm script `cmd/seedsutras`; mọi mục vẫn `[ ]`.
- Reason: người dùng làm rõ ý. Về cách nạp, spec **giữ** tự nạp lúc khởi động (đã chốt ở v2.0) và chỉ ghi script là cửa vào thứ hai chưa chốt, vì câu "hoặc dùng script chạy" chưa nói rõ chọn bên nào — nếu chọn script thì có thể bỏ `go:embed` và giảm binary ~11 MB.

### Việc theo dõi ngoài thư mục này

- Không đổi so với v2.1.

## v2.1 (2026-10-03)

Người dùng chốt ba điểm còn mở của v2.0 (cập nhật nội dung sau lần seed đầu, hành vi khi thiếu `deities`, dữ liệu bổ sung và nhãn khởi tạo) và quyết định tạm không quan tâm giấy phép vì ứng dụng miễn phí. Không đổi phạm vi hay cấu trúc flow nên là bản minor.

- `requirements.md`: năm marker `⚠️ Not confirmed` chuyển sang "Open questions (resolved)" — quy trình cập nhật nội dung (SQL tay như `article`), hành vi khi thiếu `deities` (abort startup), bảng `purposes` (chấp nhận danh sách đề xuất), cách ước tính `duration_minutes` (chấp nhận 80 âm tiết/phút), quyền dùng nội dung (tạm không quan tâm). Còn ba marker: mã user story/hợp đồng cũ của `content`, người duyệt nhãn, bản Chú Đại Bi. Out of scope thêm hai mục (reseed/backoffice, xử lý giấy phép).
- `design.md`: ba marker (tốc độ ước tính, bảng nhãn khởi tạo, abort khi thiếu deity ở Step 3) đổi thành ghi chú "đã chốt"; Known gaps về giấy phép, `deities` và sửa JSON viết lại theo quyết định; không đổi cấu trúc flow, bảng layer hay mô hình dữ liệu.
- `flow.md`: Note ở Step 3 của diagram nạp dữ liệu đổi từ `⚠️ Not confirmed` sang "đã chốt"; không đổi bước nào.
- `tasks.md`: mục Phase 0 đầu tiên đổi từ "chốt bảng giá trị và người duyệt, chặn các mục dưới" thành "xác định người duyệt nhãn, không chặn"; Deferred ghi reseed/backoffice là đã chốt hoãn; mọi mục vẫn `[ ]`.
- Reason: người dùng trả lời các câu hỏi mở. Điểm "cập nhật nội dung" được chốt mà chưa nêu phương án cụ thể nên spec ghi theo mặc định của `article`; giấy phép chỉ được hoãn vì ứng dụng miễn phí, cần xem lại nếu đổi sang thu phí.

### Việc theo dõi ngoài thư mục này

- Không đổi so với v2.0. `assets/data/sutras/README.md` mục "Before shipping" vẫn nêu cảnh báo giấy phép (đúng với dữ liệu hiện có); chưa sửa vì người dùng chỉ hoãn, không bác bỏ.

## v2.0 (2026-10-03)

Người dùng trả lời ba câu hỏi thiết kế còn mở của v1.0: nạp dữ liệu **một lần khi khởi động lần đầu**, API **lấy theo page**, và nội dung kinh dài **tải từng trang theo số dòng**. Đổi cách nạp dữ liệu và kiểu phân trang nên là bản major.

- `requirements.md`: US-SUT-01 chuyển từ "trả toàn bộ" sang phân trang `page` / `page_size` / `total` / `total_pages` (AC1, AC5, AC6 mới: `INVALID_PAGE`); US-SUT-03 chuyển từ con trỏ `after`/`limit` sang `page` + `page_size` là số dòng mỗi trang (AC12–16, mã lỗi `INVALID_PAGE_SIZE` thay `INVALID_AFTER`/`INVALID_LIMIT`, AC19 cam kết `n` liên tục để đổi `n` ↔ trang); US-SUT-04 cũng phân trang; US-SUT-06 viết lại thành nạp lần đầu lúc khởi động (AC29–34, thay AC29–35 cũ của lệnh seed): `SeedIfEmpty`, một transaction cho toàn bộ, `ON CONFLICT DO NOTHING`, fatal khi dữ liệu sai hoặc thiếu deity. AC từ 35 còn 34. Ba marker `⚠️ Not confirmed` (cách nạp, phân trang dòng, danh sách có phân trang) chuyển sang "Open questions (resolved)"; thêm hai marker mới (quy trình cập nhật nội dung sau seed đầu; hành vi khi `deities` chưa seed); marker giấy phép bổ sung việc dữ liệu được nhúng vào binary.
- `design.md`: Overview và bảng Layers & components đổi theo — bỏ `cmd/seedsutras`, `SutraSeeder`, `ReplaceSutra`, `PruneExcept`; thêm `SeedIfEmpty`, `Count`/`CountByDeity`, `InsertAll`, `repository/sutra_seed.go` (nhận `fs.FS`), `assets.SutrasFS` (`go:embed`), `ErrInvalidPageSize`, `SutraConfig{PageSize, LinesPageSize, LinesMaxPageSize}`. Flow nạp dữ liệu đổi từ 6 bước (từng văn bản một transaction, `-prune`) sang 5 bước (một transaction chung); flow danh sách thêm bước đếm (6 → 7 bước); flow đọc dòng đổi sang truy vấn khoảng `n`; flow tra ngược thêm bước đếm (7 → 8 bước); flow chi tiết không đổi. Known gaps: bỏ các mục chỉ đúng với lệnh seed và với tra ngược không phân trang; thêm kích thước binary/thời gian khởi động, phụ thuộc `deities`, và việc sửa JSON không cập nhật DB.
- `flow.md`: bốn trong năm sequence diagram được vẽ lại cho khớp `design.md` (nạp dữ liệu, danh sách, đọc dòng, tra ngược); diagram chi tiết kinh không đổi. Hai diagram (nạp dữ liệu, tra ngược theo Phật/Bồ tát) được render kiểm tra cú pháp và hợp lệ; ba diagram còn lại dùng cùng cấu trúc nhưng chưa render riêng.
- `tasks.md`: Phase 0 không đổi; Phase 1–5 sửa theo `design.md` — bỏ các việc của `cmd/seedsutras`, thêm nhúng dữ liệu (`assets/embed.go`), `SeedIfEmpty`, gọi seed trong `cmd/main.go`, `SutraConfig.PageSize`, đo thời gian khởi động và kích thước binary; mọi mục vẫn `[ ]`.
- Reason: các câu hỏi thiết kế đã được người dùng trả lời; cách hiểu của spec cho câu "phân trang theo số dòng" là `page` + `page_size` (số dòng mỗi trang, tuỳ chọn, mặc định `sutra.lines_page_size`) cho nhất quán với các danh mục khác — nếu người dùng muốn con trỏ `after` thì đổi lại.

### Việc theo dõi ngoài thư mục này (cập nhật so với v1.0)

- Bỏ khỏi danh sách: không còn `cmd/seedsutras`, nên không cần ghi chú về lệnh seed trong `migrations/README.md`; thay bằng ghi chú rằng service tự nạp `sutras` ở lần chạy đầu và phải chạy `cmd/seeddeities` trước.
- Các việc còn lại như v1.0: `openapi.yaml` + `web/docs/public/openapi.yaml`, `migrations/0013_sutras.sql` + ERD, `docs/specs/features/content/requirements.md` (US-005/006), `docs/specs/features/deity/*`, `assets/data/sutras/README.md`, `docs/knowledge/features/sutra.md` (sau khi feature chạy thật).

## v1.0 (initial)

Spec viết trước khi triển khai (Prospective), làm work order cho việc nạp bộ kinh ~11 MB vào Postgres và cung cấp API cho client iOS.

- `requirements.md`: 6 user story (`US-SUT-01..06`) và 35 tiêu chí EARS cho danh mục có bộ lọc (tông phái, loại, nhóm, mục đích, kinh nhật tụng, thời lượng, ngôn ngữ), chi tiết kinh, đọc nội dung theo dòng, tra ngược từ Phật/Bồ tát sang kinh, hình dạng dữ liệu nhất quán và lệnh nạp dữ liệu. Có 8 marker `⚠️ Not confirmed` (cách nạp, kiểu phân trang dòng, danh sách không phân trang, bảng `purposes`, người duyệt nhãn, cách ước tính thời lượng, giấy phép/duyệt nội dung, bản Chú Đại Bi) và 1 marker về mã user story / hợp đồng cũ của `content`.
- `design.md`: bảng Layers & components (mọi hàng "not yet created" hoặc "need modification"), mô hình dữ liệu 4 bảng (`sutras`, `sutra_translations`, `sutra_lines`, `sutra_deities`), bảng "Dữ liệu cần bổ sung" (`key`, `dailyChant`, `purposes`, `durationMinutes`) kèm đề xuất khởi tạo theo nhóm kinh, và 5 Step-by-step Flow (nạp dữ liệu, danh sách, chi tiết, đọc dòng, tra ngược theo Phật/Bồ tát).
- `flow.md`: 5 sequence diagram mirror 1:1 các Step-by-step Flow; mọi `⚠️ Not confirmed` ở bước tương ứng xuất hiện dưới dạng `Note`.
- `tasks.md`: Phase 0 (dữ liệu) đến Phase 5; mọi mục `[ ]`.
- Reason: người dùng yêu cầu spec để nạp `assets/data/sutras/` vào DB và cung cấp API cho client hiển thị, kèm các thuộc tính: liên kết Bồ tát, kinh nhật tụng, mục đích (cầu an, cầu siêu, trì chú…), tông phái và thời lượng. Dữ liệu JSON hiện có sẵn liên kết `deities` và tông phái, **chưa có** kinh nhật tụng, mục đích, thời lượng hay `key` bản dịch — phần này thành Phase 0. Ba quyết định thiết kế (cách nạp, phân trang dòng, danh sách không phân trang) chưa được người dùng trả lời nên spec ghi giả định và đánh dấu `⚠️ Not confirmed`.

### Việc theo dõi ngoài thư mục này (không sửa ở đây)

- `docs/specs/api/openapi.yaml` và `web/docs/public/openapi.yaml`: thêm 4 route và schema khi implement.
- `migrations/0013_sutras.sql`, `migrations/README.md`, `docs/specs/db/erd.mmd` + `erd.svg`: thêm 4 bảng.
- `docs/specs/features/content/requirements.md`: US-005/006 mô tả hợp đồng khác cho kinh điển (`/v1/content/sutras`, `updated_since`, `slug`, `audio_url`, `version`); cần quyết định thay hay giữ song song.
- `docs/specs/features/deity/*`: thêm ghi chú trỏ sang `GET /v1/deities/{id}/sutras` (route thuộc feature này, `deityHandler` không đổi).
- `assets/data/sutras/README.md`: mô tả bốn trường dữ liệu mới sau Phase 0.
- `docs/knowledge/features/sutra.md`: do `il-okf-writer` tạo **sau khi** feature chạy thật; chưa tạo.
