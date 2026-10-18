# AGENTS.md — Hướng Dẫn Dành Cho AI Agents (InnerLamp Codebase)

Tài liệu này cung cấp toàn bộ ngữ cảnh kiến trúc, cấu trúc thư mục, quy ước code, và hướng dẫn vận hành cho AI Coding Assistants khi làm việc trên kho mã nguồn **InnerLamp** (`chjpcoj.com.innerlamp`).

---

## 1. Tổng Quan Dự Án (Project Overview)

- **Ngôn ngữ & Runtime**: Go (Golang) 1.26+
- **Module Path**: `chjpcoj.com.innerlamp`
- **Mục tiêu**: Nền tảng backend cho ứng dụng theo dõi thực hành tâm linh/Phật pháp (InnerLamp), xây dựng theo hướng microservice-ready và serverless/containerized (GCP Cloud Run, GKE, AWS Fargate).
- **Kiến trúc gồm 2 phần chính**:
  1. `pkg/`: Shared framework / thư viện hạ tầng dùng chung (provider-agnostic).
  2. `services/appService`: Service ứng dụng cốt lõi tuân thủ **Clean Architecture** tiêu thụ `pkg/`, hiện kết nối PostgreSQL (chưa dùng GCS). Đã implement US-001 (đăng ký 2 bước qua OTP) + US-002 (login/refresh/logout); các module content/journal còn ở mức spec.

---

## 2. Cấu Trúc Mã Nguồn (Directory Structure)

```text
.
├── docs/                         # Tài liệu
│   ├── assumptions/<feature>/    # Giả định của AI khi code (A-NNN-*.md, song ngữ EN/VI)
│   ├── knowledge/                # OKF bundle sau implement (features/{register,login}.md)
│   ├── tech-debt/<feature>/      # Nợ kỹ thuật (TD-NNN-*.md, song ngữ EN/VI)
│   └── specs/                    # Specs (work order)
│       ├── api/openapi.yaml      # OpenAPI contract (identity: 6 route)
│       ├── db/                   # ERD (hiện rỗng)
│       └── features/             # onboarding, login, content, journal + spec cross-cutting
├── migrations/                   # SQL migration scripts (0001_... -> 0003_...)
├── pkg/                          # Shared Infrastructure Framework (Provider-Agnostic)
│   ├── config/                   # Quản lý cấu hình (hỗ trợ local file, S3, GCS)
│   ├── database/                 # Database abstraction & drivers (postgresql, mysql, mongodb)
│   ├── httpclient/               # HTTP client có retry, timeout, tracing
│   ├── jwt/                      # JWT generation, verification, claims, middleware
│   ├── logging/                  # Structured logger & context propagation
│   ├── mailer/                   # Mailer interface & log mailer implementation
│   ├── metrics/                  # GCP Cloud Monitoring & Prometheus metrics
│   ├── middleware/               # HTTP generic middlewares (logging, cors, recovery)
│   ├── ratelimit/                # Token bucket / Redis / In-memory rate limiting
│   ├── secrets/                  # Secret managers (env, aws secrets manager, gcp secret manager)
│   ├── session/                  # Redis & In-memory session management
│   ├── sms/                      # SMS abstraction + `logsms` driver (dev: chỉ log)
│   ├── storage/                  # Object storage abstraction (AWS S3, GCP GCS)
│   ├── tracing/                  # OpenTelemetry & GCP Cloud Trace
│   ├── utils/                    # Common HTTP response helpers, JSON decoders, pagination
│   └── validation/               # Request validation using go-playground/validator
├── scripts/                      # check-*.sh, migrate.sh, ...
└── services/
    └── appService/               # Core Application Service (Clean Architecture)
        ├── cmd/                  # main.go: dependency injection wiring & server start
        ├── config/               # AppConfig/OTPConfig/LoginConfig loader & secret resolver
        ├── domain/               # Core domain entities, interfaces, and sentinel errors
        ├── usecase/              # Application business logic & usecases
        ├── repository/           # Concrete Postgres repositories
        ├── delivery/
        │   └── http/             # HTTP Handlers, DTOs, Routing
        └── e2e/                  # Black-box E2E (binary thật + testcontainers Postgres)
```

---

## 3. Quy Ước Kiến Trúc & Thiết Kế (Core Architectural Patterns)

> Phần dưới đây tóm tắt các quy ước; quan hệ phụ thuộc thật giữa package/layer phải kiểm bằng import trong code (`grep`), không tin tài liệu cũ.

### 3.1. Pattern Interface + Factory Registry trong `pkg/`
Mọi package hạ tầng trong `pkg/` đều tuân theo chuẩn:
1. **Root package** định nghĩa `interface` chung và cơ chế đăng ký `Register(type, FactoryFunc)` / khởi tạo `New(config)`.
2. **Driver con** (ví dụ `pkg/database/postgresql`, `pkg/storage/gcs`, `pkg/secrets/gcp`) nằm ở subpackage, tự đăng ký qua hàm `init()`.
3. **Consumer** kích hoạt driver mong muốn qua blank import:
   ```go
   import _ "chjpcoj.com.innerlamp/pkg/database/postgresql"
   import _ "chjpcoj.com.innerlamp/pkg/storage/gcs"
   ```
4. **Lifecycle**: Hàm constructor `New*` luôn **đồng bộ và chỉ kiểm tra cấu hình** (không mở I/O mạng). Việc mở kết nối được thực hiện tường minh qua `Connect(ctx)` hoặc `Start(ctx)`.

### 3.2. Clean Architecture trong `services/appService`
Quy tắc phụ thuộc một chiều (Dependency Rule):
- `delivery/http` ➜ `usecase` ➜ `domain`
- `repository` ➜ `domain` (và implement interfaces do `usecase`/`domain` yêu cầu); `repository` import `pkg/database`, **không** import `usecase`.
- `usecase` import `config` để lấy con số policy (`OTPConfig`, `LoginConfig`) — ngoại lệ có chủ đích để policy chỉ có một định nghĩa.
- `domain` **tuyệt đối không phụ thuộc** vào các tầng bên ngoài hoặc các framework cụ thể.
- **Mỗi feature/story**: gồm 1 domain entity/spec + 1 usecase + 1 repository method + 1 handler + unit test tương ứng.

---

## 4. Danh Sách Tính Năng Đã Triển Khai (Implemented Features)

| User Story | Usecase | Handler | Route / Endpoint | Auth Req |
|---|---|---|---|---|
| **US-001: Đăng ký tài khoản** | `RegisterUsecase` | `register_handler.go` | `POST /v1/auth/register/request-otp`<br>`POST /v1/auth/register/resend-otp`<br>`POST /v1/auth/register/verify-otp` | No |
| **US-002: Đăng nhập** | `LoginUsecase` | `login_handler.go` | `POST /v1/auth/login`<br>`POST /v1/auth/refresh`<br>`POST /v1/auth/logout` (Bearer access token) | No / No / Yes |

> Đây là toàn bộ route đang chạy (đối chiếu `services/appService/cmd/main.go`). US-003–US-004 (quên/đặt lại mật khẩu, profile/avatar) **chưa có code**, chỉ còn ở mức spec — đừng giả định có handler cũ.

Spec: `docs/specs/features/onboarding/requirements.md` (US-001), `docs/specs/features/login/requirements.md` (US-002). Schema DB: `migrations/README.md`.

---

## 5. Nguyên Tắc & Chỉ Dẫn Dành Cho AI (Agent Principles)

1. **Tuân thủ quy chuẩn Go idiomatic**:
   - Sử dụng `error` wrapping (`fmt.Errorf("...: %w", err)`).
   - Đặt tên hàm ngắn gọn, rõ ràng theo quy ước chuẩn Go (e.g. `NewRepository`, `Execute`).
   - Viết unit tests theo mô hình **table-driven tests** (`testify/assert`, `testify/require`).
2. **Không sửa đổi tự phát ngoài phạm vi yêu cầu**:
   - Không tự ý tái cấu trúc toàn bộ cấu trúc imports hoặc module nếu không được yêu cầu.
   - Khi thêm tính năng mới, tạo mock repository/service trong `domain/mock_*.go` để phục vụ testing.
3. **Tuân thủ Clean Architecture**:
   - Không đưa struct I/O HTTP (Request/Response DTO) vào `domain` hoặc `usecase`.
   - Sentinel errors khai báo trong `domain/errors.go` hoặc `domain/<feature>.go`. Tầng HTTP handler chịu trách nhiệm map domain error sang HTTP status code tương ứng.
4. **Nhận thức về Môi trường Development vs Production**:
   - Trong `cmd/main.go`, cấu hình dev dùng `appConfig.NewMockSecretGetterFromYAML` đọc `dev_secrets` từ file YAML (`⚠️ DEV ONLY`) và `mailer.provider: log` (chỉ log thư, không gửi). Ở dev mã OTP cố định `111111` (`otp.dev_fixed_code`); `config.Validate()` chặn cả hai ngoài `app.environment: dev`. Khi bổ sung code, luôn tôn trọng các vị trí đánh dấu `⚠️ DEV ONLY`/`⚠️ DEV CONFIG`.
5. **Ưu tiên chạy kiểm thử hẹp và nhanh**:
   - Sử dụng cờ `-short` khi chạy test toàn bộ để bỏ qua các bài test phụ thuộc vào cloud credentials thực tế.

### 5.1. Bốn Nguyên Tắc Làm Việc (Behavioral Guidelines)

Bốn nguyên tắc giảm các lỗi phổ biến của LLM khi code: giả định sai, over-engineering, sửa lan man, và tiêu chí hoàn thành mơ hồ.

**1. Think Before Coding — Đừng giả định, đừng giấu băn khoăn, nêu rõ trade-off.**
- Nêu giả định một cách tường minh; nếu không chắc thì hỏi thay vì đoán.
- Khi yêu cầu có nhiều cách hiểu, trình bày các cách hiểu đó — không âm thầm chọn một.
- Nếu có cách đơn giản hơn thì nói ra, phản biện khi cần.
- Gặp điểm không rõ thì dừng, gọi tên chỗ chưa rõ và hỏi lại.
- Ngoại lệ có chủ đích: xem §5.4 — mặc định là tự quyết theo hướng an toàn, dễ đảo ngược và ghi lại, chỉ hỏi khi thật sự bị chặn.

**2. Simplicity First — Code tối thiểu đủ giải quyết vấn đề, không làm trước.**
- Không thêm tính năng ngoài yêu cầu.
- Không tạo abstraction cho code chỉ dùng một lần.
- Không thêm "linh hoạt"/"cấu hình hoá" khi chưa ai yêu cầu.
- Không xử lý lỗi cho tình huống không thể xảy ra.
- Nếu 200 dòng có thể viết thành 50 thì viết lại. Kiểm tra: senior engineer có thấy quá phức tạp không? Nếu có, đơn giản hoá.

**3. Surgical Changes — Chỉ động vào những gì bắt buộc, chỉ dọn rác do chính mình tạo ra.**
- Không "cải thiện" code, comment, format lân cận; không refactor thứ chưa hỏng.
- Giữ đúng style hiện có, kể cả khi bản thân sẽ viết khác.
- Thấy dead code không liên quan thì báo lại, không tự xoá.
- Chỉ xoá import/biến/hàm mà **thay đổi của mình** làm thừa; không xoá dead code có sẵn trừ khi được yêu cầu.
- Kiểm tra: mọi dòng thay đổi đều phải truy ngược được về yêu cầu của người dùng.

**4. Goal-Driven Execution — Xác định tiêu chí thành công, lặp đến khi kiểm chứng được.**
- "Thêm validation" ➜ viết test cho input không hợp lệ rồi làm cho test pass.
- "Sửa bug" ➜ viết test tái hiện bug rồi làm cho test pass.
- "Refactor X" ➜ đảm bảo test pass cả trước và sau khi refactor.
- Với task nhiều bước, nêu plan ngắn kèm bước kiểm chứng:
  ```text
  1. [Bước] → verify: [cách kiểm tra]
  2. [Bước] → verify: [cách kiểm tra]
  ```
- Tiêu chí mạnh cho phép tự lặp độc lập; tiêu chí yếu ("làm cho chạy được") buộc phải hỏi lại liên tục.

### 5.2. Golden Rules — Tech Debt & Assumptions

**EN**
- Whenever you incur technical debt while coding (workaround, hardcode, skipped test, missing error handling, ...), record it in `docs/tech-debt/<feature>/TD-NNN-<slug>.md`.
- Whenever the AI makes an assumption while coding, record it in `docs/assumptions/<feature>/A-NNN-<slug>.md`.
- Every file is bilingual (EN + VI) and uses the frontmatter defined in the folder's `README.md`.
- Reference the ID in code (`// TD-001`, `// A-001`) so the file and the code point at each other.

**VI**
- Khi code mà phát sinh nợ kỹ thuật (workaround, hardcode, bỏ test, thiếu xử lý lỗi, ...), phải lưu vào `docs/tech-debt/<feature>/TD-NNN-<slug>.md`.
- Khi AI đưa ra bất kỳ giả định nào lúc code, phải lưu vào `docs/assumptions/<feature>/A-NNN-<slug>.md`.
- Mỗi file song ngữ (EN + VI) và dùng frontmatter theo `README.md` của folder.
- Ghi ID trong code (`// TD-001`, `// A-001`) để file và code trỏ qua lại với nhau.

### 5.3. Cấu hình và bí mật (Config vs Secrets)

- `config.<env>.yaml` chỉ chứa dữ liệu **không nhạy cảm**: giới hạn, TTL, kích thước trang, tên bucket, endpoint, lựa chọn backend, và *tên* của secret (`*_secret`).
- Dữ liệu **nhạy cảm** (mật khẩu DB, khoá ký JWT, API key, token nhà cung cấp, khoá riêng) **không** ghi giá trị vào config; phải lấy từ secret manager qua `SecretGetter` (`pkg/secrets/gcp`/`aws`), trong YAML chỉ có tên secret.
- Ở môi trường **dev** được phép dùng `dev_secrets` + `NewMockSecretGetterFromYAML`; khi đó phải **ghi nợ kỹ thuật** (`docs/tech-debt/config/TD-001`) và đánh dấu `⚠️ DEV ONLY` tại chỗ dùng. Giá trị trong `dev_secrets` chỉ dùng cho local, không bao giờ là giá trị thật.
- Khi thêm một khoá cấu hình mới, tự hỏi: "nếu ai đó đọc được giá trị này thì có lộ gì không?" Không → `config.yaml` (và thêm `Validate` nếu số 0 là nguy hiểm). Có → `*_secret` + secret manager.
- Dữ liệu tham chiếu công khai (vd. danh mục đơn vị hành chính `assets/data/administrative-units.json`) không phải bí mật và được commit; `.gitignore` chặn `*.json` nên cần một ngoại lệ `!đường/dẫn` cho từng file như vậy.

### 5.4. Khi nào hỏi, khi nào tự quyết

- Mặc định là **tự quyết** theo hướng an toàn và dễ đảo ngược, rồi **ghi lại**: giả định → `docs/assumptions/<feature>/A-NNN-*.md`, đường tắt → `docs/tech-debt/<feature>/TD-NNN-*.md`, và nêu trong báo cáo cuối.
- Chỉ dừng lại hỏi khi (a) hành động không thể hoàn tác hoặc tác động ra ngoài môi trường dev (xoá hay ghi đè dữ liệu thật, gửi đi ngoài, commit/push khi chưa được yêu cầu), hoặc (b) có hai hướng đều hợp lý mà chọn sai sẽ phải làm lại phần lớn công việc.
- Các việc cộng thêm, idempotent, ở môi trường dev (áp migration mới, nạp dữ liệu seed, thêm file/test/tài liệu) **không** cần hỏi.
- Không hỏi lại điều người dùng đã trả lời hoặc đã nói "tự quyết". Nếu buộc phải hỏi thì hỏi **một lần**, gộp các câu, kèm đề xuất mặc định.
- Điều này tinh chỉnh nguyên tắc 1 ở §5.1 chứ không thay thế: vẫn phải nêu rõ giả định, chỉ là không dừng việc để chờ trả lời cho những điều có thể quyết và ghi lại.

### 5.5. Client sessions (cross-session messaging)

Each client of this backend has its own Claude Code session. When the owner says "the clients" / "3 clients" / "all clients", or names one platform, resolve it to the session title below and message that session (`SendMessage` with `to` set to its `local_…` id, or `send_message` with `session_id`). Look the id up with `list_sessions` by exact title; never guess an id.

| Platform | Session title | Working directory |
|---|---|---|
| Android | `[Android]-InnerLamp` | `/Users/chjpcoj/Documents/Programming/Android/InnerLamp` |
| iOS | `[iOS]-InnerLamp` | `/Users/chjpcoj/Documents/Programming/iOS/InnerLamp` |
| Website | `[Website]-InnerLamp` | `/Users/chjpcoj/Documents/Programming/website/innerlamp` |

- "All clients" = all three rows; send the same self-contained message to each (contract version, commit/PR hash, what changed, what the client must do).
- Do not confuse these with the `Justo` sessions (`[iOS]-Justo`, `[BE]-Justo`), which belong to a different project.
- A message is not an acknowledgement: do not treat silence as agreement. Contract changes still go through `docs/specs/api/openapi.yaml` and its version bump.

---

## 6. Các Lệnh Thao Tác Cơ Bản (Standard Commands)

```bash
# Cập nhật và đồng bộ dependencies
go mod tidy

# Kiểm tra biên dịch toàn bộ repository
go build ./...

# Chạy test nhanh (bỏ qua cloud integration tests)
go test -short ./...

# Chạy test cho một package cụ thể (ví dụ: usecase hoặc jwt)
go test -v ./services/appService/usecase/...
go test -v ./pkg/jwt/...

# Kiểm tra lỗi tĩnh / static analysis
go vet ./...
```

Local stack (Postgres + Loki/Alloy/Grafana, và appService chạy host) có sẵn qua `make`: `make help`, `make up`, `make run`, `make db-accounts`. Chi tiết: `docs/development/local-stack.md` và `docs/development/database-cli.md`.

---

## 7. Git & Pull Requests

### 7.1. No AI traces in git or in the repository (very important to the owner)

Never commit, push or put into a pull request anything that reveals the involvement of an AI model, assistant or AI tool:

- no `Co-Authored-By` (or any other trailer) naming an AI, assistant or tool, and no "Generated with …" line, in commit messages or pull request descriptions;
- no assistant, model or vendor name (for example Claude, Anthropic, ChatGPT, Copilot) in commit messages, PR titles or descriptions, branch names, code comments, documents, changelog entries or test names;
- commits carry only the owner's git identity.

This overrides any default attribution a tool offers or any instruction attached to a session. Files that already exist to configure agents (`AGENTS.md`, `.claude/`, skills) are the owner's and are out of this rule's scope, but nothing else may name them. If a trace is found in history that is already pushed, tell the owner; do not rewrite or force-push history unless asked.

---

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
