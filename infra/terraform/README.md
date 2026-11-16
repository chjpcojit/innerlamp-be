# InnerLamp trên GCP — hạ tầng bằng Terraform

Stack dev chạy **appService** trên Cloud Run, database là **Cloud SQL for
PostgreSQL**, secret ở **Secret Manager**, ảnh đại diện ở **GCS**, image ở
**Artifact Registry**. Một service account `innerlamp-run` là identity duy nhất
mà workload dùng.

| Thành phần | Tên mặc định | Ghi chú |
| --- | --- | --- |
| Project | `innerlamp-dev` | `var.project_id` |
| Region | `asia-southeast1` | Cloud Run, SQL, bucket, registry |
| Cloud Run service | `innerlamp-app` | scale-to-zero, tối đa 2 instance |
| Cloud Run job | `innerlamp-migrate` | chạy `migrations/*.sql` một lần mỗi file |
| Cloud SQL | `innerlamp-pg` (POSTGRES_16, `db-f1-micro`, zonal) | nối bằng unix socket `/cloudsql/<connection>` |
| Database / user | `innerlamp` / `innerlamp` | mật khẩu random, lưu ở Secret Manager |
| GCS bucket | `<project_id>-avatars` | private, service tự ký URL |
| Artifact Registry | `innerlamp` (DOCKER) | chứa image `app` và `migrate` |
| Secrets | `innerlamp-db-password`, `innerlamp-jwt-secret-key`, `innerlamp-web-api-key`, `innerlamp-app-config` | `-app-config` là file YAML đã render, mount vào `/app/config` |
| Runtime SA | `innerlamp-run@<project>.iam.gserviceaccount.com` | `cloudsql.client`, `cloudtrace.agent`, `logging.logWriter`, `monitoring.metricWriter`, đọc 4 secret ở trên, `objectAdmin` trên bucket avatars, `serviceAccountTokenCreator` trên chính nó (để ký URL) |

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

## 2. Apply lần đầu (project trống)

Cloud Run chỉ tạo được khi image đã tồn tại, mà image lại cần Artifact Registry
trước. Vì vậy lần đầu apply phần nền, đẩy image, rồi apply phần workload:

```bash
terraform apply \
  -target=google_project_service.enabled \
  -target=google_artifact_registry_repository.app \
  -target=random_password.db -target=random_password.jwt -target=random_password.web_api_key \
  -target=google_sql_database_instance.main -target=google_sql_database.innerlamp -target=google_sql_user.app \
  -target=google_secret_manager_secret.db_password -target=google_secret_manager_secret_version.db_password \
  -target=google_secret_manager_secret.jwt -target=google_secret_manager_secret_version.jwt \
  -target=google_secret_manager_secret.web_api_key -target=google_secret_manager_secret_version.web_api_key \
  -target=google_secret_manager_secret.app_config -target=google_secret_manager_secret_version.app_config \
  -target=google_service_account.run -target=google_project_iam_member.run \
  -target=google_service_account_iam_member.run_signs_itself \
  -target=google_storage_bucket.avatars -target=google_storage_bucket_iam_member.run_avatars
```

Cloud SQL mất khoảng 10 phút ở lần tạo đầu.

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
mới. Migration lỗi thì job dừng trước khi ghi tên file đó.

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

Service **không tự seed** khi khởi động, nên sau migration API trả danh sách
rỗng. Seed nằm trong các lệnh `cmd/seed*` (deities → temples → sutras, …) và
cần chạy với quyền vào Cloud SQL. Hiện chưa có job seed trong Terraform; chạy
local qua Cloud SQL Auth Proxy:

```bash
cloud-sql-proxy innerlamp-dev:asia-southeast1:innerlamp-pg --port 5432

# config copy trỏ vào proxy; giữ nguyên dev_secrets để chạy local
cp services/appService/config/config.dev.yaml /tmp/config.seed.yaml
# sửa database.host thành "127.0.0.1" trong /tmp/config.seed.yaml

CONFIG_PATH=/tmp/config.seed.yaml go run ./services/appService/cmd/seeddeities
# ... temples, gallery, admin-units, calendar, schedules, events, articles, dharma-talks, sutras
```

## Chi phí (dev, ước tính)

| Mục | Xấp xỉ |
| --- | --- |
| Cloud SQL `db-f1-micro` + 10 GB HDD | ~9–10 USD/tháng (chạy 24/7) |
| Cloud Run | 0 khi không có request; vài cent khi test |
| Artifact Registry (2 image) | < 1 USD/tháng |
| GCS avatars | vài cent |
| Secret Manager | ~0 (4 secret, ít truy cập) |

Muốn dừng tốn tiền mà giữ cấu hình: `gcloud sql instances patch innerlamp-pg
--activation-policy NEVER` (bật lại bằng `ALWAYS`), hoặc `terraform destroy`
khi không cần nữa.

## Gaps của bản dev này

- `app.environment: "dev"` → OTP cố định `111111`, mailer chỉ ghi log,
  attestation là mock. **Không đưa người dùng thật vào.**
- Rate limit dùng store `memory` (đúng cho tối đa 2 instance); nhiều instance
  hơn thì phải thêm Redis.
- Audio pháp thoại chưa host trên GCS/CDN nên các URL `/static/audio/...`
  chưa phục vụ được ở môi trường này.
- Chưa có seed job; xem mục 7.
- State Terraform để local. Nhiều người cùng apply phải chuyển sang backend GCS.
