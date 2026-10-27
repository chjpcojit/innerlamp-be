---
title: Thắp đèn và vườn đèn
description: "Endpoint thắp đèn kèm tâm nguyện, người nhận, lời nguyện, và vườn đèn các ngọn đèn đang sáng của mọi người: lọc theo loại đèn, tải thêm bằng cursor."
---

Người dùng thắp một ngọn đèn kèm tâm nguyện và lời nguyện; đèn sáng trong **vườn đèn** suốt số ngày đã chọn rồi tự tắt. Cả hai endpoint đều **cần đăng nhập**: gửi `Authorization: Bearer <access_token>` (xem [Xác thực](/guides/authentication/)). Khách chưa đăng nhập không thắp được và cũng không xem được vườn đèn (kể cả số đèn). Người thắp luôn là `user_id` trong token; không có endpoint nào nhận user id từ client.

| Thao tác | Endpoint |
| --- | --- |
| Thắp một ngọn đèn | `POST /v1/lamps` |
| Xem vườn đèn | `GET /v1/lamps` |

Miễn phí ở bản này. Mỗi người dùng có tối đa **20 đèn đang sáng** cùng lúc.

:::note
Các con số giới hạn trên trang này (20 đèn, 12 đèn mỗi trang, 5 giây chống bấm đúp, 100 ký tự `recipient`, 300 ký tự `prayer`, các số ngày 1/7/21/49) là giá trị cấu hình của server (`lamp.*`), có thể khác giữa các môi trường. Giá trị ở đây là của môi trường dev.
:::

## Thắp đèn

Field lạ (kể cả `id`, `user_id`, `lit_at`, `end_at`, `lighter_name`) bị từ chối `400 VALIDATION_ERROR`. **Client không gửi thời gian**: thời điểm thắp là giờ máy chủ.

| Field | Kiểu | Bắt buộc | Ghi chú |
| --- | --- | --- | --- |
| `lamp_type` | string | có | `OIL_LAMP` (đèn dầu), `LOTUS_CANDLE` (nến sen), `FLOWER_LANTERN` (hoa đăng). |
| `intention` | string | có | Tâm nguyện: `PEACE` (cầu an), `SALVATION` (cầu siêu), `MERIT_DEDICATION` (hồi hướng), `GRATITUDE` (tri ân). |
| `recipient` | string \| null | không | Dành cho ai, văn bản tự do, tối đa 100 ký tự sau khi cắt khoảng trắng đầu cuối. Đếm theo ký tự, không theo byte. Vắng, `null`, rỗng hoặc toàn khoảng trắng là đèn **không có người nhận** (dành cho tất cả chúng sanh), lưu thành `null`. |
| `prayer` | string | không | Lời nguyện, tối đa 300 ký tự. Rỗng hoặc toàn khoảng trắng được lưu thành `null`. |
| `duration_days` | int | có | Số ngày đèn sáng: `1`, `7`, `21` hoặc `49`. |
| `is_anonymous` | bool | không | Mặc định `false`. `true` = ẩn tên người thắp trong vườn đèn. |

```json title="POST /v1/lamps"
{
  "lamp_type": "OIL_LAMP",
  "intention": "PEACE",
  "recipient": "Bà ngoại",
  "prayer": "Mong bà bình an",
  "duration_days": 7,
  "is_anonymous": false
}
```

### Response (`Lamp`)

Thắp mới trả `201`. `data` chỉ trả cho **chính người thắp**:

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | int | |
| `lamp_type`, `intention` | string | Như đã gửi. |
| `recipient` | string \| null | Như đã gửi, đã cắt khoảng trắng; `null` khi không có người nhận. App hiển thị "Tất cả chúng sanh" cho `null`. |
| `prayer` | string \| null | |
| `duration_days` | int | |
| `is_anonymous` | bool | |
| `lighter_name` | string | Pháp danh **tại thời điểm thắp**, hoặc `Vô danh` khi ẩn danh hoặc chưa có pháp danh. Không đổi theo khi người dùng đổi pháp danh. |
| `lit_at`, `end_at` | string | UTC, RFC 3339. `end_at = lit_at + duration_days × 24 giờ`. |

### Bấm đúp

Gửi lại **đúng nội dung vừa thắp** trong vòng 5 giây không tạo đèn thứ hai: server trả `200` với đèn đã có (không phải `201`). Điều này đúng cả khi người dùng đang ở mức 20 đèn. Hai lần gửi cùng không có người nhận cũng là cùng nội dung. Gửi cùng nội dung sau 5 giây là một ngọn đèn mới. Client có thể coi `200` và `201` đều là thành công.

## Vườn đèn

`GET /v1/lamps` trả các ngọn đèn **đang sáng** của **tất cả** người dùng, đèn mới thắp lên đầu. Người gọi không lọc kết quả; chỉ quyết định `is_mine` của từng đèn.

| Query | Ghi chú |
| --- | --- |
| `lamp_type` | Chỉ lấy loại này. **Bỏ hẳn** tham số để lấy mọi loại; để trống hoặc giá trị lạ trả `400 INVALID_FILTER`. |
| `limit` | Số đèn mỗi trang, mặc định và tối đa **12**. Lớn hơn 12 thì bị hạ về 12; không phải số nguyên dương trả `400 INVALID_LIMIT`. |
| `cursor` | `next_cursor` của trang trước. Lần đầu không gửi. |

```json title="Response"
{
  "success": true,
  "data": {
    "items": [
      { "id": 41, "lamp_type": "LOTUS_CANDLE", "recipient": "Thầy", "lighter_name": "Thích Minh An", "is_mine": true, "end_at": "2026-10-17T18:00:00Z" },
      { "id": 40, "lamp_type": "OIL_LAMP", "recipient": null, "lighter_name": "Vô danh", "is_mine": false, "end_at": "2026-10-11T10:00:00Z" }
    ],
    "total": 10,
    "counts": { "OIL_LAMP": 5, "LOTUS_CANDLE": 3, "FLOWER_LANTERN": 2 },
    "has_more": true,
    "next_cursor": "MTc5MDAwMDAwMDAwMDAwMC40MQ"
  }
}
```

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `items[]` | object[] | Mỗi mục chỉ có `id`, `lamp_type`, `recipient` (`null` khi không có người nhận), `lighter_name`, `is_mine`, `end_at`. **Tâm nguyện và lời nguyện không bao giờ được trả cho người khác.** `items: []` khi chưa có đèn nào. |
| `items[].is_mine` | bool | `true` khi người gọi là người thắp đèn này, kể cả đèn ẩn danh của chính mình (app dùng để đánh dấu "Đèn của tôi"). Không có field nào lộ chủ của đèn người khác. |
| `total` | int | Tổng số đèn đang sáng của mọi người dùng và mọi loại. |
| `counts` | object | Số đèn đang sáng theo từng loại, luôn đủ ba khoá, `0` khi không có. |
| `has_more` | bool | Còn trang sau hay không. |
| `next_cursor` | string \| null | Gửi lại làm `cursor` để tải thêm; `null` khi hết. |

:::note
`total` và `counts` là của **toàn bộ** đèn đang sáng và **không đổi theo `lamp_type`**: lọc theo "nến sen" vẫn nhận số đèn dầu và số hoa đăng. App dùng một request để vừa hiện danh sách vừa hiện các con số trên bộ lọc.
:::

### Tải thêm bằng cursor

Vườn đèn thay đổi liên tục (đèn hết hạn, đèn mới được thắp), nên danh sách này **không dùng `page`** như các danh sách khác: số trang sẽ làm lặp hoặc sót đèn giữa hai lần tải. Thay vào đó, mỗi trang trả `next_cursor`, và client gửi lại nó để lấy các đèn nằm ngay sau đèn cuối của trang trước.

- `cursor` là chuỗi **mờ** do server cấp. Đừng tự dựng hay đọc nội dung của nó.
- Giữ `lamp_type` và `limit` giống nhau giữa các lần tải của cùng một danh sách.
- Dừng khi `has_more` là `false`.

```js title="Tải hết vườn đèn (rút gọn)"
let cursor = null;
const all = [];
do {
  const qs = new URLSearchParams({ limit: "12" });
  if (cursor) qs.set("cursor", cursor);
  const res = await fetch(`/v1/lamps?${qs}`, { headers: { Authorization: `Bearer ${accessToken}` } });
  const { data } = await res.json();
  all.push(...data.items);
  cursor = data.has_more ? data.next_cursor : null;
} while (cursor);
```

### Hiển thị ngày tắt

`end_at` là UTC. App đổi sang **múi giờ của máy** khi hiển thị ngày tắt, nên cùng một ngọn đèn có thể hiện hai ngày khác nhau ở hai múi giờ. Đèn tắt đúng một khoảnh khắc, tính bằng đồng hồ máy chủ; đừng tự ẩn đèn theo giờ máy.

## Lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Field sai (`details.field`: `lamp_type`, `intention`, `recipient`, `prayer`, `duration_days`, `is_anonymous`), body sai JSON, field lạ, sai Content-Type. |
| 400 | `INVALID_FILTER` | `lamp_type` của vườn đèn rỗng hoặc không phải một trong ba loại (`details.field = "lamp_type"`). |
| 400 | `INVALID_LIMIT` | `limit` không phải số nguyên dương (`details.field = "limit"`). |
| 400 | `INVALID_CURSOR` | `cursor` không do server cấp (`details.field = "cursor"`). Bỏ `cursor` và tải lại từ đầu. |
| 401 | `UNAUTHORIZED` | Thiếu token, token sai/hết hạn, hoặc gửi refresh token thay cho access token. |
| 409 | `LAMP_LIMIT_REACHED` | Đã có đủ 20 đèn đang sáng. Đèn hết hạn không tính, nên thắp lại được khi một ngọn tắt. |
| 413 | `PAYLOAD_TOO_LARGE` | Body vượt 1 MB. |

Cấu trúc đầy đủ và nút gọi thử: [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
