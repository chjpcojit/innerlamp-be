# AGENTS.md — Hướng Dẫn Dành Cho AI Agents (InnerLamp Codebase)

Tài liệu này cung cấp toàn bộ ngữ cảnh kiến trúc, cấu trúc thư mục, quy ước code, và hướng dẫn vận hành cho AI Coding Assistants khi làm việc trên kho mã nguồn **InnerLamp** (`chjpcoj.com.innerlamp`).

---

## 1. Tổng Quan Dự Án (Project Overview)

- **Ngôn ngữ & Runtime**: Go (Golang) 1.26+
- **Module Path**: `chjpcoj.com.innerlamp`
- **Mục tiêu**: Nền tảng backend cho ứng dụng theo dõi thực hành tâm linh/Phật pháp (InnerLamp), xây dựng theo hướng microservice-ready và serverless/containerized (GCP Cloud Run, GKE, AWS Fargate).
- **Kiến trúc gồm 2 phần chính**:
  1. `pkg/`: Shared framework / thư viện hạ tầng dùng chung (provider-agnostic).
  2. `services/appService`: Service ứng dụng cốt lõi tuân thủ **Clean Architecture** tiêu thụ `pkg/`, kết nối PostgreSQL và Google Cloud Storage (GCS).

---

## 2. Cấu Trúc Mã Nguồn (Directory Structure)

```text
.
├── cmd/                          # Entrypoints
├── docs/                         # Tài liệu đặc tả và kiến trúc
│   ├── ARCHITECTURE.md           # Điểm vào chính mô tả quan hệ giữa các layer/package
│   ├── api/                      # OpenAPI spec (openapi.yaml)
│   ├── db/                       # ERD diagrams (register-erd.md, content-erd.md)
│   └── specs/                    # User stories & Functional specs (onboarding, content, journal, practice)
├── migrations/                   # SQL migration scripts (0001_... -> 0007_...)
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
│   ├── storage/                  # Object storage abstraction (AWS S3, GCP GCS)
│   ├── tracing/                  # OpenTelemetry & GCP Cloud Trace
│   ├── utils/                    # Common HTTP response helpers, JSON decoders, pagination
│   └── validation/               # Request validation using go-playground/validator
└── services/
    └── appService/               # Core Application Service (Clean Architecture)
        ├── cmd/                  # main.go: dependency injection wiring & server start
        ├── config/               # AppConfig loader & secret resolver
        ├── domain/               # Core domain entities, interfaces, and sentinel errors
        ├── usecase/              # Application business logic & usecases
        ├── repository/           # Concrete DB & Storage repositories (Postgres, GCS)
        └── delivery/
            └── http/             # HTTP Handlers, DTOs, Routing
```

---

## 3. Quy Ước Kiến Trúc & Thiết Kế (Core Architectural Patterns)

> Chi tiết đầy đủ (sơ đồ phụ thuộc thật từ import, danh sách khoảng hở đã biết): [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Phần dưới đây chỉ tóm tắt.

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
- `repository` ➜ `domain` (và implement interfaces do `usecase`/`domain` yêu cầu).
- `domain` **tuyệt đối không phụ thuộc** vào các tầng bên ngoài hoặc các framework cụ thể.
- **Mỗi feature/story**: gồm 1 domain entity/spec + 1 usecase + 1 repository method + 1 handler + unit test tương ứng.

---

## 4. Danh Sách Tính Năng Đã Triển Khai (Implemented Features)

| User Story | Usecase | Handler | Route / Endpoint | Auth Req |
|---|---|---|---|---|
| **US-001: Đăng ký tài khoản** | `RegisterUsecase` | `register_handler.go` | `POST /v1/auth/register` | No |
| **US-002: Đăng nhập** | `LoginUsecase` | `login_handler.go` | `POST /v1/auth/login` | No |
| **US-003: Quên/Đặt lại mật khẩu** | `PasswordResetUsecase` | `forgot_password_handler.go`, `reset_password_handler.go` | `POST /v1/auth/forgot-password`<br>`POST /v1/auth/reset-password` | No |
| **US-004: Hồ sơ & Avatar** | `ProfileUsecase`, `AvatarUsecase` | `profile_handler.go`, `avatar_handler.go` | `GET /v1/users/me`<br>`PATCH /v1/users/me`<br>`POST /v1/users/me/avatar` | Yes |
| **US-005: Danh sách Kinh điển** | `ListSutrasUsecase` | `sutras_handler.go` | `GET /v1/content/sutras` | No |
| **US-006: Chi tiết Bài kinh** | `GetSutraUsecase` | `sutra_detail_handler.go` | `GET /v1/content/sutras/{slug}` | No |
| **US-007: Danh sách Thần chú** | `ListMantrasUsecase` | `mantras_handler.go` | `GET /v1/content/mantras` | No |
| **US-008: Chi tiết Thần chú** | `GetMantraUsecase` | `mantra_detail_handler.go` | `GET /v1/content/mantras/{slug}` | No |
| **US-009: Danh sách Chư Phật/Bồ Tát** | `ListBodhisattvasUsecase` | `bodhisattvas_handler.go` | `GET /v1/content/bodhisattvas` | No |
| **US-010: Chi tiết Chư Phật/Bồ Tát** | `GetBodhisattvaUsecase` | `bodhisattva_detail_handler.go` | `GET /v1/content/bodhisattvas/{slug}` | No |

> ⚠️ **US-004 "Auth Req: Yes" là thiết kế dự kiến, chưa được thực thi thật.** `pkg/jwt/middleware` tồn tại nhưng `cmd/main.go` chưa wrap nó quanh route `/v1/users/me` và `/v1/users/me/avatar` — 2 route này hiện nhận request không kiểm tra token. Đừng giả định auth đã hoạt động khi sửa/mở rộng 2 handler này; xem [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md#4-khoảng-hở-kiến-trúc-đã-biết-đọc-trước-khi-tưởng-đã-production-ready) mục 4.

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
   - Trong `cmd/main.go`, cấu hình mặc định dùng `config.MockSecretGetter` đọc từ `dev_secrets` trong file YAML và `mailer.NewLogMailer`. Khi bổ sung code, luôn tôn trọng các vị trí đánh dấu `⚠️ DEV ONLY`.
5. **Ưu tiên chạy kiểm thử hẹp và nhanh**:
   - Sử dụng cờ `-short` khi chạy test toàn bộ để bỏ qua các bài test phụ thuộc vào cloud credentials thực tế.

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

---

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
