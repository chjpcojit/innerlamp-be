# InnerLamp trên GCP — hạ tầng bằng Terraform

Stack dev chạy **appService** trên Cloud Run, database là **PostgreSQL trên
Neon (free tier, region Singapore)**, secret ở **Secret Manager**, ảnh đại diện
ở **GCS**, image ở **Artifact Registry**. Một service account `innerlamp-run`
là identity duy nhất mà workload dùng.

| Thành phần | Giá trị | Ghi chú |
| --- | --- | --- |
| Project | `innerlamp-dev` | `var.project_id` |
| Region | `asia-southeast1` | Cloud Run, bucket, registry |
| Cloud Run service | `innerlamp-app` | scale-to-zero, tối đa 2 instance, CPU chỉ cấp khi có request |
| Cloud Run job | `innerlamp-migrate` | chạy `migrations/*.sql` một lần mỗi file |
| Cloud Run job | `innerlamp-seed` | nạp toàn bộ catalogue một lần (idempotent) |
| Postgres | Neon, `neondb` / `neondb_owner` | host ở `var.neon_host`, TLS bắt buộc (`sslmode=require`) |
| GCS bucket | `<project_id>-avatars` | private, service tự ký URL |
| Artifact Registry | `innerlamp` (DOCKER) | chứa image `app` và `migrate` |
| Secrets | `innerlamp-db-password`, `innerlamp-jwt-secret-key`, `innerlamp-web-api-key`, `innerlamp-app-config` | `-app-config` là file YAML đã render, mount vào `/app/config` |
| Runtime SA | `innerlamp-run@<project>.iam.gserviceaccount.com` | `cloudtrace.agent`, `logging.logWriter`, `monitoring.metricWriter`, đọc 4 secret ở trên, `objectAdmin` trên bucket avatars, `serviceAccountTokenCreator` trên chính nó (để ký URL) |

Không còn Cloud SQL: instance `innerlamp-pg` đã bị xoá ngày 2026-11-16 sau khi
Neon chạy ổn (xem `docs/adr/0005-neon-postgres-for-dev.md`).

## 1. Chuẩn bị

```bash
gcloud auth login                      # tài khoản sở hữu project
gcloud config set project innerlamp-dev
gcloud auth application-default login  # hoặc dùng GOOGLE_OAUTH_ACCESS_TOKEN
```

Project phải đã bật billing. Terraform tự bật các API cần thiết.

```bash
cd infra/terraform
terraform init
```

Tạo `terraform.tfvars` (đã git-ignore, không commit):

```hcl
project_id    = "innerlamp-dev"
region        = "asia-southeast1"
max_instances = 2

neon_host     = "ep-...ap-southeast-1.aws.neon.tech"
neon_port     = 5432
neon_user     = "neondb_owner"
neon_database = "neondb"
neon_password = "<mật khẩu Neon>"
neon_ssl_mode = "require"
```

`neon_password` được đưa vào Secret Manager (`innerlamp-db-password`) và service
đọc lúc khởi động; không có mật khẩu nào trong repo hay image.

## 2. Apply lần đầu (project trống)

Cloud Run chỉ tạo được khi image đã tồn tại, mà image lại cần Artifact Registry
trước. Vì vậy lần đầu apply phần nền, đẩy image, rồi apply phần workload:

```bash
terraform apply \
  -target=google_project_service.enabled \
  -target=google_artifact_registry_repository.app \
  -target=random_password.jwt -target=random_password.web_api_key \
  -target=google_secret_manager_secret.db_password -target=google_secret_manager_secret_version.db_password \
  -target=google_secret_manager_secret.jwt -target=google_secret_manager_secret_version.jwt \
  -target=google_secret_manager_secret.web_api_key -target=google_secret_manager_secret_version.web_api_key \
  -target=google_secret_manager_secret.app_config -target=google_secret_manager_secret_version.app_config \
  -target=google_service_account.run -target=google_project_iam_member.run \
  -target=google_service_account_iam_member.run_signs_itself \
  -target=google_storage_bucket.avatars -target=google_storage_bucket_iam_member.run_avatars
```

## 3. Build và đẩy image

Cloud Run chạy **linux/amd64**; trên Apple Silicon phải truyền `--platform`,
nếu không container chết với `exec format error`.

```bash
REG=asia-southeast1-docker.pkg.dev/innerlamp-dev/innerlamp
gcloud auth configure-docker asia-southeast1-docker.pkg.dev

# từ repo root
docker build --platform linux/amd64 -t "$REG/app:dev" .
docker build --platform linux/amd64 -f migrations/Dockerfile -t "$REG/migrate:dev" .

docker push "$REG/app:dev"
docker push "$REG/migrate:dev"
```

## 4. Apply phần workload

```bash
terraform apply \
  -var="app_image=$REG/app:dev" \
  -var="migrate_image=$REG/migrate:dev"
```

Lần apply này tạo Cloud Run service, job migration và quyền `run.invoker` cho
`allUsers` (đặt `-var allow_unauthenticated=false` nếu không muốn public).

## 5. Chạy migration

```bash
gcloud run jobs execute innerlamp-migrate --region asia-southeast1 --wait
```

Job ghi mỗi file đã chạy vào bảng `schema_migrations`, nên chạy lại chỉ áp file
mới. Migration lỗi thì job dừng trước khi ghi tên file đó. `PGHOST` của job là
`var.neon_host`, không cần Cloud SQL proxy.

Kiểm tra:

```bash
curl "$(terraform output -raw service_url)/v1/dharma-talks?page=1"
```

## 6. Phát hành bản mới

```bash
docker build --platform linux/amd64 -t "$REG/app:<tag>" .
docker push "$REG/app:<tag>"
terraform apply -var="app_image=$REG/app:<tag>" -var="migrate_image=$REG/migrate:dev"
```

## 7. Dữ liệu seed

Service **không tự seed** khi khởi động. Terraform tạo job `innerlamp-seed`
(image `seed`) chạy đúng thứ tự phụ thuộc: administrative units → deities →
temples → gallery → calendar events → temple schedules → temple events →
articles → dharma talks → sutras.

```bash
gcloud run jobs execute innerlamp-seed --region asia-southeast1 --wait
```

Job đọc secret `innerlamp-seed-config` (config cloud + `dev_secrets`), vì các
lệnh `cmd/seed*` vẫn lấy mật khẩu DB từ file cấu hình. Mọi lệnh là upsert hoặc
no-op khi bảng đã có dữ liệu, nên chạy lại không nhân bản.

Build lại image seed khi dữ liệu nguồn đổi:

```bash
docker build --platform linux/amd64 -f seeds/Dockerfile -t "$REG/seed:dev" .
docker push "$REG/seed:dev"
terraform apply -var="seed_image=$REG/seed:dev"
```

## Chi phí (dev, ước tính)

| Mục | Xấp xỉ |
| --- | --- |
| Neon Free (Postgres) | 0 USD — 1 GB storage, 100 CU-giờ/tháng/project; tự ngủ khi rỗi |
| Cloud Run | 0 khi không có request (free tier 180k vCPU-giây, 360k GiB-giây, 2 triệu request/tháng) |
| Artifact Registry (2 image ≈ 0,35 GB) | 0 — 0,5 GB đầu miễn phí |
| GCS avatars | 0 khi chưa có ảnh |
| Secret Manager | 0 — 6 secret version miễn phí, 10.000 lượt đọc/tháng miễn phí |

Tổng hiện tại: **0 USD/tháng** ở mức dùng dev. Muốn dọn hẳn: `terraform destroy`.

## Gaps của bản dev này

- `app.environment: "dev"` → OTP cố định `111111`, mailer chỉ ghi log,
  attestation là mock. **Không đưa người dùng thật vào.**
- Neon free tự ngủ khi rỗng 5 phút: request đầu sau khi ngủ chậm thêm ~1 giây.
- Neon free không có backup tự động và giới hạn 1 GB; dữ liệu thật phải trả phí.
- Rate limit dùng store `memory` (đúng cho tối đa 2 instance); nhiều instance
  hơn thì phải thêm Redis.
- Audio pháp thoại chưa host trên GCS/CDN nên các URL `/static/audio/...`
  chưa phục vụ được ở môi trường này.
- Chưa có seed job; xem mục 7.
- State Terraform để local. Nhiều người cùng apply phải chuyển sang backend GCS.
