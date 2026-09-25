# Step-by-step Flow trong design.md

> Mỗi section "Step-by-step Flow" trong `design.md` có một **bản vẽ Mermaid tương ứng trong
> `flow.md`** (sequence diagram là chính, flowchart cho nhánh rẽ logic). Đọc
> `flow_diagram.md` để biết quy ước vẽ và quy tắc "mirror 1:1, không thêm/bớt bước". File này chỉ
> quy định phần text trong `design.md`.

## Vì sao section này tồn tại

Một user story một câu ("người dùng đăng ký bằng email + mật khẩu") không thể mô tả chính xác
một flow backend nhiều bước — nó âm thầm che mất các bước như "kiểm tra email chưa tồn tại trước
khi hash mật khẩu", "mở transaction trước khi ghi 2 bảng", hay "gửi email sau khi commit, không
phải trước". Đây chính là loại chi tiết khiến một AI (hoặc một dev) implement thiếu bước hoặc sai
thứ tự, vì không có gì trong spec gọi tên chúng ra.

Step-by-step Flow sửa việc đó bằng cách bắt buộc mọi lời gọi ra ngoài (DB, external API, storage,
mailer, cache) trong flow phải được viết riêng từng bước, kèm trigger, pre-condition, và cả nhánh
thành công lẫn thất bại — để không có gì bị giả định hay tự bịa ra sau này.

## Khi nào bắt buộc

Theo rule 5 của `SKILL.md`: **bất cứ khi nào flow chính của feature có 2 lời gọi tuần tự trở lên
ra hệ thống ngoài** — DB query/transaction, external HTTP API, object storage, mailer/SMS, cache,
message queue — kể cả khi một phần thuộc service/module khác (ví dụ feature này chỉ gọi một usecase
của module khác không đi qua HTTP).

Nếu flow chỉ có đúng một bước side-effect (một DB query, không transaction, không gọi ngoài), có
thể thay bằng một đoạn "Data flow" ngắn gọn.

Tiêu chí nhận biết nhanh: nếu bạn phải viết chữ "rồi" (then) trong mô tả flow, hoặc có transaction
bao nhiều bảng, hoặc có side effect xảy ra sau khi ghi DB — bạn cần section này.

## Hình dạng bắt buộc mỗi bước

```
Step N — <tên ngắn>
- Trigger: <request/handler nào, hoặc state nào khiến bước này chạy>
- Pre-condition: <điều gì phải đúng trước — ví dụ validate đã qua, token còn hợp lệ, transaction
  đã mở>
- Call: <loại> <chi tiết> (<field chính>)
  <ràng buộc đã biết, hoặc chi tiết confirmed/chưa confirmed của chính bước này>
    - Success → <chuyện gì xảy ra, bước nào tiếp theo>
    - Failure (<case cụ thể>) → <HTTP status / error code / domain error nào, có rollback không>
    - Failure (<case khác, nếu có>) → <...>
```

Với bước thuộc module/service khác, phải nói rõ (ví dụ "Step 3 — Gửi email xác thực (thuộc
`pkg/mailer`, ngoài scope feature này)") và trỏ sang spec của phần đó thay vì lặp lại chi tiết.

## Ví dụ mẫu (dựng theo format US-001 Register)

```
Step 1 — Validate request (feature này)
- Trigger: handler nhận POST /v1/auth/register
- Pre-condition: body parse được, JSON hợp lệ
- Call: go-playground/validator trên RegisterRequest (email, password >= 8, phone_number E.164,
  dharma_name optional, ordination_place optional)
    - Success → Step 2
    - Failure (validation) → 400 kèm field errors, dừng; không chạm DB

Step 2 — Kiểm tra email/phone đã tồn tại (feature này)
- Trigger: Step 1 pass
- Pre-condition: input đã validate
- Call: repository.FindByEmail + repository.FindByPhone (2 query, có thể gộp 1 query OR)
  Confirmed: dedupe dựa vào UNIQUE constraint ở DB, không chỉ dựa vào check trước đó — check này
  chỉ để trả lỗi thân thiện; race condition vẫn bắt bằng unique violation khi INSERT.
    - Success (chưa tồn tại) → Step 3
    - Failure (email đã dùng) → domain.ErrEmailTaken → 409, dừng
    - Failure (phone đã dùng) → domain.ErrPhoneTaken → 409, dừng

Step 3 — Hash password (feature này)
- Trigger: Step 2 pass
- Pre-condition: password plaintext đã qua validate độ dài
- Call: bcrypt.GenerateFromPassword (cost theo config)
    - Success → Step 4
    - Failure (lỗi hash) → 500, log; không ghi gì xuống DB

Step 4 — Tạo user (feature này)
- Trigger: Step 3 pass
- Pre-condition: password_hash đã có
- Call: INSERT INTO users (...) RETURNING id
    - Success → Step 5
    - Failure (unique violation từ race) → map thành 409 như Step 2, dừng
    - Failure (DB error khác) → 500, log; không có gì bị ghi một phần (single INSERT, không cần
      transaction)

Step 5 — Sinh token (feature này)
- Trigger: Step 4 pass
- Pre-condition: user.id và role đã có
- Call: pkg/jwt generate access token (15m) + refresh token (30 ngày), lưu refresh vào bảng
  refresh_tokens (hoặc devices)
    - Success → response 201 {access_token, refresh_token, user}
    - Failure (lỗi sinh/ghi token) → 500. Lưu ý: user ĐÃ tạo ở Step 4 và không rollback — chấp nhận
      được vì user có thể login lại; ghi rõ quyết định này, đừng để ngầm hiểu.
```

Ví dụ trên cho thấy điều quan trọng nhất của section: **nó buộc phải nói ra những chỗ không rõ
ràng** — Step 5 thất bại thì user đã tạo có bị rollback không? Nếu spec không trả lời, người
implement sẽ tự chọn, và hai người sẽ chọn khác nhau.

## Chi tiết chưa chốt hiển thị thế nào

Nếu hợp đồng/behavior của bước chưa được xác nhận, đánh dấu ngay tại bước đó (không chỉ ở
`requirements.md`), vì đây là nơi cụ thể nhất cho chi tiết kỹ thuật:

```
- Call: UPDATE users SET avatar_url = ... sau khi upload GCS xong
  ⚠️ Not confirmed: nếu UPDATE DB thất bại sau khi file đã upload lên GCS, có xóa file GCS
  (compensating action) hay để orphan object? — chặn tasks.md Phase 3 hoàn thành.
```

Khi người dùng trả lời, sửa bước đó thành khẳng định đã xác nhận, và ghi lại vào `changelog.md` —
kèm rõ câu trả lời có làm đổi số bước/cấu trúc flow không, hay chỉ gỡ một điểm chưa chắc chắn (cả
hai đều hợp lệ; phải nói rõ là loại nào).
