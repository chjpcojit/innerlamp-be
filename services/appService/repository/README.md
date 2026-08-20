# Repository — services/appService/repository

## Mục đích
Tầng "repository implementation" của Clean Architecture cho `appService`: hiện thực cụ thể (PostgreSQL + GCS) cho các interface được `domain`/`usecase` định nghĩa (`domain.UserRepository`, `domain.SutraRepository`, `usecase.AvatarStorage`), dựa trên `pkg/database` và `pkg/storage` chứ không gọi thẳng driver ngoài.

## Khi nào dùng / không dùng
- Dùng khi: wiring ở `cmd/main.go` cần một implementation thật cho `domain.UserRepository`/`domain.SutraRepository`/`usecase.AvatarStorage` chạy trên Postgres + GCS (không phải test).
- Không dùng khi: viết unit test cho usecase — dùng mock trong `domain/mock_user_repository.go`, `domain/mock_sutra_repository.go` (hoặc mock tương ứng cho `AvatarStorage`) thay vì import package này; cũng không dùng khi cần driver khác Postgres/GCS — package này không tự abstract thêm driver, muốn đổi driver thì đổi implementation `pkg/database`/`pkg/storage` được truyền vào constructor.

## API chính

### `NewUserRepository(db database.Database) domain.UserRepository`
Trả về implementation Postgres của `domain.UserRepository`, thao tác trên bảng `users`, `user_devices`, `user_sessions`, `password_reset_tokens`.
```go
userRepo := repository.NewUserRepository(db) // db: chjpcoj.com.innerlamp/pkg/database.Database
```
Các method chính:
- `Register(ctx, user, device, issueTokens)` — insert `users` + `user_devices` + `user_sessions` trong 1 transaction; `issueTokens` (kiểu `domain.TokenIssuer`) được gọi sau khi biết `user.ID` để mint JWT trong cùng transaction.
- `FindByEmail(ctx, email)` — trả `domain.ErrUserNotFound` nếu không có user.
- `CreateSession(ctx, userID, device, issueTokens)` — upsert `user_devices` theo `(user_id, device_id)` rồi tạo session mới; mỗi device giữ session riêng, đăng nhập thiết bị mới không ảnh hưởng thiết bị khác.
- `CreatePasswordResetToken(ctx, userID, tokenHash, expiresAt)` — chỉ lưu `tokenHash`, không lưu token gốc.
- `ResetPassword(ctx, tokenHash, newPasswordHash)` — trong 1 transaction: validate token (`FOR UPDATE`, chưa dùng, chưa hết hạn) → đổi mật khẩu → đánh dấu token đã dùng → revoke toàn bộ session đang active của user. Trả `domain.ErrResetTokenInvalid` nếu token sai/hết hạn/đã dùng.
- `FindByID(ctx, userID)` / `UpdateProfile(ctx, userID, input)` / `UpdateAvatarURL(ctx, userID, avatarURL)` — trả `domain.ErrUserNotFound` nếu không có row khớp `WHERE id = ...` (kể cả `UPDATE ... RETURNING` không match row nào).

### `NewSutraRepository(db database.Database) domain.SutraRepository`
Implementation Postgres của `domain.SutraRepository`, đọc bảng `sutras`.
```go
sutraRepo := repository.NewSutraRepository(db)
sutras, err := sutraRepo.ListSutras(ctx, domain.ListSutrasInput{Sect: "bac-tong"})
sutra, err := sutraRepo.GetSutraBySlug(ctx, "kinh-phap-hoa")
```
- `ListSutras` build query động: filter `updated_at > $n` nếu `UpdatedSince` khác zero-value, filter `sect = $n` nếu `Sect` khác rỗng, cả hai đều optional và có thể kết hợp; luôn `ORDER BY updated_at ASC` để caller dùng row cuối làm cursor cho delta-sync lần sau.
- `GetSutraBySlug` trả `domain.ErrSutraNotFound` nếu không có slug đó.

### `NewGCSAvatarStorage(client storage.Client, bucket string) usecase.AvatarStorage`
Adapter từ `pkg/storage.Client` (đủ interface đầy đủ, nhiều method) sang `usecase.AvatarStorage` (chỉ 1 method `Upload`) mà `usecase.AvatarUsecase` cần.
```go
avatarStorage := repository.NewGCSAvatarStorage(storageClient, cfg.Storage.DefaultBucket)
```
- `Upload(ctx, objectName, reader, contentType)` — upload lên `bucket` với `storage.ACLPublicRead`, trả về URL public dạng `https://storage.googleapis.com/{bucket}/{objectName}` (không dùng signed URL/CDN — xem comment trong file để biết assumption này).
- Object name do caller (usecase) quyết định — `AvatarUsecase.UpdateAvatar` dùng `avatars/{userID}` (deterministic theo user), nên upload lại sẽ tự ghi đè ảnh cũ trong bucket chứ không tạo file rác.

## Cấu hình / dependency
- Phụ thuộc `chjpcoj.com.innerlamp/pkg/database` (interface `database.Database`: `Query`/`QueryRow`/`Exec`/`Begin`) — implementation Postgres cụ thể (`pkg/database/postgresql`) được chọn và kết nối ở `cmd/main.go`, package `repository` chỉ nhận interface đã kết nối sẵn, không tự gọi `Connect`.
- Phụ thuộc `chjpcoj.com.innerlamp/pkg/storage` (interface `storage.Client`) — driver cụ thể là `pkg/storage/gcs` (blank-import ở `cmd/main.go`), package `repository` cũng chỉ nhận client đã kết nối.
- Phụ thuộc `github.com/lib/pq` (nhận diện lỗi Postgres qua `*pq.Error`) và `github.com/google/uuid` (sinh `session_id`).
- Constructor không đọc biến môi trường — `db`, `client`, `bucket` đều truyền từ ngoài vào (wiring ở `cmd/main.go`, bucket lấy từ `cfg.Storage.DefaultBucket`).
- Các struct trả về (`sutraRepository`, `userRepository`, `gcsAvatarStorage`) đều unexported — package ngoài chỉ thấy qua interface domain/usecase, không import type cụ thể.

## Lưu ý / lỗi hay gặp
- `Register`/`CreateSession`/`ResetPassword` đều dùng transaction với `defer tx.Rollback()` — nếu `Commit()` thành công thì `Rollback()` sau đó là no-op an toàn (driver `database/sql` trả lỗi bị bỏ qua bằng `_ =`), không cần tự lo double-rollback.
- Lỗi trùng email/số điện thoại khi `Register` chỉ được nhận diện qua đúng tên constraint Postgres (`users_email_key`, `users_phone_number_key`) trong `isUniqueViolation` — đổi tên constraint trong migration mà không sửa hằng số này sẽ làm `ErrEmailTaken`/`ErrPhoneTaken` không bao giờ được trả về nữa (rơi vào lỗi generic `fmt.Errorf` thay vì domain error).
- `sql.ErrNoRows` được map thủ công sang domain error tương ứng ở từng method (`ErrUserNotFound`, `ErrSutraNotFound`, `ErrResetTokenInvalid`) — thêm method mới đọc theo `id`/khoá duy nhất mà quên map `sql.ErrNoRows` sẽ làm caller nhận lỗi Postgres thô thay vì domain error, phá vỡ hợp đồng interface.
- `ResetPassword` lock row token bằng `SELECT ... FOR UPDATE` trong transaction — chỉ an toàn nếu code gọi luôn nằm trong cùng transaction đó (đã đúng ở implementation hiện tại); không tách các câu UPDATE ra khỏi transaction khi sửa sau này.
- `nullableString("")` map chuỗi rỗng thành `NULL` — dùng cho `dharma_name`/`ordination_place` ở cả `Register` và `UpdateProfile`; nghĩa là muốn "xoá" field này thì gửi input rỗng, không có cách phân biệt "không đổi" với "xoá" ở tầng repository.
- `gcsAvatarStorage.Upload` không có logic dọn ảnh cũ nếu object name đổi — cơ chế "ghi đè" chỉ đúng vì `usecase.AvatarUsecase` luôn dùng object name cố định `avatars/{userID}`; nếu sau này object name đổi theo mỗi lần upload (ví dụ thêm hash/timestamp) thì ảnh cũ sẽ trở thành rác trong bucket, phải tự thêm bước xoá.
- URL trả về là public object URL trực tiếp, không phải signed URL — nếu bucket không bật public-read ở cấp bucket/IAM phù hợp với ACL `public-read` khi upload, URL trả về sẽ không truy cập được dù upload thành công.
