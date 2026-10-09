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
| **US-001: Đăng ký tài khoản** | `RegisterUsecase` | `register_handler.go` | `POST /v1/auth/register/request-otp`<br>`POST /v1/auth/resend-otp`<br>`POST /v1/auth/register/verify-otp` | No |
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
   - Trong `cmd/main.go`, cấu hình dev dùng `appConfig.NewMockSecretGetterFromYAML` đọc `dev_secrets` từ file YAML (`⚠️ DEV ONLY`) và `pkg/sms/logsms` (`⚠️ DEV CONFIG`, chỉ log SMS không gửi). OTP đang hardcode `111111`. Khi bổ sung code, luôn tôn trọng các vị trí đánh dấu `⚠️ DEV ONLY`/`⚠️ DEV CONFIG`.
5. **Ưu tiên chạy kiểm thử hẹp và nhanh**:
   - Sử dụng cờ `-short` khi chạy test toàn bộ để bỏ qua các bài test phụ thuộc vào cloud credentials thực tế.

### 5.1. Bốn Nguyên Tắc Làm Việc (Behavioral Guidelines)

Bốn nguyên tắc giảm các lỗi phổ biến của LLM khi code: giả định sai, over-engineering, sửa lan man, và tiêu chí hoàn thành mơ hồ.

**1. Think Before Coding — Đừng giả định, đừng giấu băn khoăn, nêu rõ trade-off.**
- Nêu giả định một cách tường minh; nếu không chắc thì hỏi thay vì đoán.
- Khi yêu cầu có nhiều cách hiểu, trình bày các cách hiểu đó — không âm thầm chọn một.
- Nếu có cách đơn giản hơn thì nói ra, phản biện khi cần.
- Gặp điểm không rõ thì dừng, gọi tên chỗ chưa rõ và hỏi lại.

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

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
