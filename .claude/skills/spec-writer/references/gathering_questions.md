# Hỏi người dùng trước khi viết spec

Trước khi sinh hoặc cập nhật bất kỳ file spec nào, hỏi các câu bắt buộc dưới đây bằng công cụ hỏi
người dùng (hoặc tương đương), tuyệt đối không đoán. File này định nghĩa bộ câu hỏi bắt buộc và
quy tắc đặt câu hỏi để người dùng dễ trả lời.

## Quy tắc đặt câu hỏi

- **Một chủ đề mỗi câu.** Không gộp hai quyết định vào một câu.
- **Cụ thể, không trừu tượng.** Thay vì "Thiết kế layer thế nào?", hỏi "Feature này có cần bảng DB
  mới không, hay chỉ đọc/ghi trên bảng đã có?"
- **Đưa ví dụ hoặc mặc định** khi có thể, để người dùng chỉ cần xác nhận thay vì tự soạn câu trả lời:
  - Kém: "Tiêu chí chấp nhận nên là gì?"
  - Tốt: "Khi user gửi request với body sai định dạng — trả 400 kèm danh sách field lỗi (mặc định),
    hay trả 422 với error code riêng?"
- **Ngắn.** Một câu trả lời được trong một dòng tốt hơn một đoạn phải đọc.

## Đọc product spec trước khi hỏi

Đọc `docs/product/<feature>/product.md` trước (xem `upstream_product_spec.md`). Câu nào product spec
đã trả lời (user story, quy tắc nghiệp vụ, phạm vi, edge case) thì **không hỏi lại** — trích ID.
Câu nào là quyết định product mà product spec còn thiếu thì không tự chốt ở đây: ghi
`⚠️ Not confirmed (product)` và chuyển lại cho `ba-writer`. Bộ câu hỏi dưới đây chỉ còn phần kỹ
thuật của backend.

## Bộ câu hỏi bắt buộc (trước khi sinh spec mới)

Hỏi theo thứ tự, từng câu một hoặc gộp thành một form ngắn — tùy định dạng công cụ hỏi:

1. **Tên feature?** (kebab-case, khớp tên thư mục) — ví dụ `register`, `sutra-list`,
   `password-reset`.
2. **Feature này đã được implement trong code chưa?** → quyết định Prospective vs Retroactive
   (`modes.md`).
3. **Một câu: ai là người dùng và họ muốn làm gì?** (user story) — hoặc trỏ tới US-XXX đã có trong
   `requirements.md` hiện tại nếu đang viết tiếp.
4. **Endpoint/route nào?** (`METHOD /v1/...`) — và method có cần auth không (middleware
   `RequireAuth` hay public read)?
5. **Có bảng DB mới không, hay chỉ dùng bảng đã có?** Nếu có: đã có migration chưa, hay cần thêm?
6. **Flow chính có nhiều hơn một lời gọi ra ngoài theo thứ tự không** (nhiều query, transaction
   nhiều bảng, gọi external service, upload storage, gửi mail)? Có/Không — nếu có, bắt buộc có
   section Step-by-step Flow trong `design.md` (xem `step_by_step_flow.md`) **và** sơ đồ Mermaid
   trong `flow.md` (xem `flow_diagram.md`).
7. **Với mỗi lời gọi ở câu 6: contract đã chốt chưa** (schema request/response, mã lỗi, HTTP
   status), hay còn chưa rõ? Cái gì "chưa rõ" trở thành marker `⚠️ Not confirmed`, không được viết
   như thể đã có.
8. **Có bước kiểm tra/validate ngầm nào không hiển nhiên từ hành động chính không** — ví dụ check
   trùng email, rate limit, kiểm tra quyền sở hữu resource, ràng buộc unique ở DB, side effect phụ
   (audit log, counter)? (Đây là câu làm lộ ra những thứ như "email phải unique" trước khi chúng bị
   gộp im lặng vào một dòng user story.)
9. **Có phụ thuộc module/service khác không** — usecase của module `journal` gọi sang `content`,
   hay handler gọi thẳng external API của bên thứ ba?

## Khi cập nhật spec đã có (không tạo mới)

Hỏi thay bằng:

1. **Thay đổi gì?** — một câu: requirement mới, đổi scope (đưa thứ gì đó từ "Out of scope" vào
   scope), hay trả lời một mục `⚠️ Not confirmed`?
2. Nếu là trả lời mục `⚠️ Not confirmed`: **câu trả lời là gì**, nói càng đơn giản càng tốt (ví dụ
   "trả trong cùng response" vs "gọi API riêng" vs "tính ở client").
3. Nếu là behavior mới/đổi: **có ảnh hưởng tới tiêu chí chấp nhận đã có không, hay hoàn toàn mới?**
4. Nếu là chi tiết kỹ thuật mới (schema field, mã lỗi, index): **nó chỉ nằm ở `design.md`, hay có
   làm đổi behavior mà `requirements.md` đang mô tả?**

## Quy tắc "không bao giờ bỏ qua"

Nếu người dùng không trả lời một câu bắt buộc (từ chối, hoặc nói "chưa chắc"), **không được**:
- Bỏ nó khỏi spec hoàn toàn.
- Chọn một mặc định rồi trình bày như thể đã xác nhận.
- Hỏi lại nhiều lần trong cùng một session — hỏi một lần, rồi tiếp tục với marker mở.

Thay vào đó, ghi vào spec đúng format từ rule 3 của `SKILL.md`:

```
> ⚠️ Not confirmed: <câu hỏi, một câu> — <nó chặn gì, ví dụ "chặn tasks.md Phase 3 không
> implement được">.
```

Đặt ở section liên quan nhất (`requirements.md` mục "Pending" cho câu hỏi về behavior/nghiệp vụ,
`design.md` tại đúng bước flow cho câu hỏi kỹ thuật/API) — không dồn hết lên đầu file, nơi chúng
mất ngữ cảnh.
