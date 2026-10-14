---
title: Gia phả
description: "Endpoint cây gia phả và phả ký: tạo, dựng cây, mời người thân vào xem bằng số điện thoại, hồ sơ thành viên do server tính (tuổi, chi, xưng hô, ngày âm)."
---

Gia phả là dữ liệu **riêng tư**. **Mọi endpoint đều cần đăng nhập**: gửi `Authorization: Bearer <access_token>` (xem [Xác thực](/guides/authentication/)). Người thao tác luôn là `user_id` trong token; không có endpoint nào nhận user id từ client.

## Hai vai trò

| Vai trò | Là ai | Làm được |
| --- | --- | --- |
| `owner` | Người tạo gia phả | Xem **và sửa** mọi thứ: tên, phả ký, thành viên, mời và gỡ người xem, xoá gia phả. |
| `viewer` | Người được `owner` mời và đã chấp nhận | **Chỉ xem**: cây, phả ký, hồ sơ thành viên. Tự rời được. |

Vai trò được tính lại từ DB ở **mỗi** request, nên gỡ quyền có hiệu lực ngay. Người **không có vai trò** gọi vào một gia phả nhận `404 GENEALOGY_NOT_FOUND`, **giống hệt** như gia phả không tồn tại (kể cả id không phải số nguyên dương) — không lộ gia phả nào có thật. Người xem gọi route ghi nhận `403 GENEALOGY_FORBIDDEN` và không có gì bị đổi.

Không có vai trò đồng sửa, không chuyển quyền chủ. **Trưởng tộc** chỉ là một cờ ghi nhớ trên thành viên (`is_clan_head`), không cấp quyền gì.

## Endpoint

| Thao tác | Endpoint | Ai gọi |
| --- | --- | --- |
| Tạo gia phả (kèm thuỷ tổ) | `POST /v1/genealogies` | mọi người dùng |
| Gia phả của tôi | `GET /v1/genealogies` | mọi người dùng |
| Tổng quan | `GET /v1/genealogies/{id}` | owner, viewer |
| Đổi tên / quê gốc | `PUT /v1/genealogies/{id}` | owner |
| Xoá gia phả | `DELETE /v1/genealogies/{id}` | owner |
| Ghi phả ký | `PUT /v1/genealogies/{id}/chronicle` | owner |
| Toàn bộ thành viên | `GET /v1/genealogies/{id}/members` | owner, viewer |
| Thêm thành viên | `POST /v1/genealogies/{id}/members` | owner |
| Một thành viên | `GET /v1/genealogies/{id}/members/{memberId}` | owner, viewer |
| Sửa thông tin | `PUT /v1/genealogies/{id}/members/{memberId}` | owner |
| Xoá thành viên | `DELETE /v1/genealogies/{id}/members/{memberId}` | owner |
| Sửa nhầm quan hệ | `PUT /v1/genealogies/{id}/members/{memberId}/relation` | owner |
| Hồ sơ thành viên | `GET /v1/genealogies/{id}/members/{memberId}/profile` | owner, viewer |
| Mời một thành viên vào xem | `POST /v1/genealogies/{id}/invitations` | owner |
| Lời mời của gia phả | `GET /v1/genealogies/{id}/invitations` | owner |
| Huỷ lời mời | `DELETE /v1/genealogies/{id}/invitations/{invitationId}` | owner |
| Người đang xem | `GET /v1/genealogies/{id}/viewers` | owner |
| Gỡ người xem / tự rời | `DELETE /v1/genealogies/{id}/viewers/{userId}` | owner; hoặc viewer với `userId` của chính mình |
| Hộp lời mời của tôi | `GET /v1/invitations` | mọi người dùng |
| Chấp nhận / từ chối | `POST /v1/invitations/{id}/accept` · `/decline` | người được mời |

Giới hạn hiện hành: 10 gia phả mỗi người, 5000 thành viên và 60 đời mỗi gia phả, 5 vợ/chồng mỗi người, 20 người xem (tính cả lời mời đang chờ), lời mời hết hạn sau 30 ngày. Các con số này do cấu hình server quyết định, đừng hardcode ở client.

## Dựng cây

Thành viên có hai **loại** (`kind`):

- `blood`: thuộc dòng họ, có đời (`generation`, thuỷ tổ = 1) và có thể có con. Chỉ **nam** thuộc dòng họ mới có con trong cây; con của nữ dòng họ **không** vào cây (`400`, `details.field = related_member_id`).
- `in_law`: dâu hoặc rể, là vợ/chồng của một người dòng họ, cùng đời với người đó. `in_law_label` là `DAU` (vợ của nam) hoặc `RE` (chồng của nữ), do server tính.

**Thuỷ tổ phải là nam.** `GET .../members` trả toàn bộ cây dạng phẳng (tối đa 5000, không phân trang) theo đời tăng dần; client tự dựng cây từ `parent_id` (con của ai) và `spouse_of_id` (dâu/rể của ai). `other_parent_id` cho biết con của người vợ nào khi cha có nhiều vợ.

### Tạo gia phả

```json title="POST /v1/genealogies"
{
  "name": "Họ Nguyễn",
  "ancestral_home": "Nam Định",
  "founder": { "full_name": "Nguyễn Văn A", "gender": "male", "is_alive": true, "birth_year": 1930 },
  "spouses": [{ "full_name": "Trần Thị B", "is_alive": true }]
}
```

`spouses[i].gender` không gửi thì lấy giới tính ngược với thuỷ tổ. Trả `201` với `GenealogyDetail`. Tất cả hoặc không gì cả: lỗi ở bước nào cũng không để lại cây dở.

### Thêm thành viên

```json title="POST /v1/genealogies/{id}/members — thêm con"
{
  "relation": "child",
  "related_member_id": 3,
  "other_parent_id": 6,
  "full_name": "Nguyễn Văn C",
  "gender": "male",
  "is_alive": true,
  "birth_year": 1962
}
```

```json title="POST /v1/genealogies/{id}/members — thêm vợ/chồng"
{ "relation": "spouse", "related_member_id": 3, "full_name": "Lê Thị D", "is_alive": true }
```

| Field | Kiểu | Ghi chú |
| --- | --- | --- |
| `relation` | string | `child` hoặc `spouse`. |
| `related_member_id` | int | Người được gắn vào (cha, hoặc người có vợ/chồng). Phải thuộc gia phả này. |
| `other_parent_id` | int | Chỉ với `child`, **tuỳ chọn**: dâu/rể là vợ/chồng của cha. Con luôn nằm ở nhánh của người cha dù có gửi hay không; chỉ khi có `other_parent_id` thì hồ sơ của con mới có `mother` và xưng hô "Mẹ". |
| `full_name` | string | Bắt buộc, tối đa 100 ký tự. |
| `gender` | string | `male` / `female`. Bắt buộc, trừ `spouse` (mặc định ngược giới tính người được gắn). |
| `is_alive` | bool | Bắt buộc. |
| `birth_year`, `death_year` | int | `1000` đến năm hiện tại; mất không nhỏ hơn sinh. `death_year` chỉ khi đã mất. |
| `residence`, `dharma_name`, `note` | string | Nơi ở (≤ 200), pháp danh (≤ 100), ghi chú (≤ 1000). |
| `tablet_location`, `burial_place` | string | Linh vị đặt tại, nơi an táng (≤ 200). Chỉ khi đã mất. |
| `death_month`, `death_day`, `death_entered_as_lunar` | int, int, bool | Ngày giỗ, xem bên dưới. |
| `phone_number` | string | Điện thoại của người đó; **duy nhất trong gia phả**. Xem [vị trí của tôi](#vị-trí-của-tôi). |
| `is_clan_head` | bool | Cờ trưởng tộc (nam, còn sống, dòng họ). Tối đa một nút mỗi gia phả; đặt `true` cho người mới thì người cũ tự mất cờ. |
| `birth_order` | int | 1–1000, thứ tự trong các con cùng cha. Mặc định: lớn nhất + 1. Không dùng cho thuỷ tổ và dâu/rể. |

Field do server giữ (`id`, `kind`, `parent_id`, `spouse_of_id`, `generation`, `in_law_label`, timestamp) bị từ chối `400 VALIDATION_ERROR`.

### Cảnh báo lệch năm sinh

Năm sinh con nhỏ hơn năm sinh cha **không bị chặn**: server vẫn lưu và trả thêm `warnings` ở `POST`, `PUT` thành viên và `PUT .../relation`:

```json
{ "data": { "id": 50, "warnings": [{ "code": "CHILD_BORN_BEFORE_PARENT", "member_id": 50, "related_member_id": 3 }] } }
```

`GET` không có `warnings`. Không lệch thì `warnings: []`.

### Sửa, xoá và sửa nhầm quan hệ

- `PUT .../members/{memberId}` chỉ đổi **thông tin**. Gửi `relation`, `related_member_id`, `parent_id`, `other_parent_id`... bị từ chối. `is_clan_head` **vắng mặt = giữ nguyên**.
- `PUT .../members/{memberId}/relation` chuyển một người sang cha/mẹ khác (`relation = child`) hoặc đổi người mà dâu/rể được gắn (`relation = spouse`). Cả nhánh dưới tự dời đời. Bị từ chối nếu tạo vòng, vượt 60 đời hoặc di chuyển thuỷ tổ (`409 FOUNDER_NOT_MOVABLE`).
- `DELETE` xoá một người **không có con** (kèm dâu/rể của họ). Người còn con bị chặn (`409 MEMBER_HAS_DESCENDANTS`): xoá từ đời dưới lên. Thuỷ tổ không xoá được (`409 FOUNDER_NOT_DELETABLE`); muốn bỏ cả cây thì xoá gia phả.

## Ngày chạp họ và ngày giỗ: lưu dương lịch, server trả thêm ngày âm

Cả hai lưu **dương lịch** dù người dùng chọn lịch nào, kèm cờ nhớ người dùng đã chọn âm:

| | Ngày dương (ghi được) | Cờ | Ngày âm (server tính, **không ghi**) |
| --- | --- | --- | --- |
| Ngày chạp họ (phả ký) | `chap_year`, `chap_month`, `chap_day` (gửi đủ ba hoặc không gửi) | `chap_entered_as_lunar` | `chap_lunar_day`, `chap_lunar_month`, `chap_lunar_leap` |
| Ngày giỗ (thành viên) | `death_year`, `death_month`, `death_day` (cần `death_year`) | `death_entered_as_lunar` | `death_lunar_day`, `death_lunar_month`, `death_lunar_leap` |

Người dùng nhập âm lịch thì **client quy sang dương trước khi gửi** và đặt cờ `*_entered_as_lunar = true`; lúc hiển thị, client dùng cờ để biết nên hiện âm lịch trước, với ngày âm do server trả. Ngày âm là `null` khi chưa có ngày hoặc ngoài khoảng năm 1900–2199. Server **không** tính ngày dương của năm sau và không nhắc giỗ.

## Phả ký

`PUT /v1/genealogies/{id}/chronicle` **thay thế toàn bộ**: field không gửi hoặc `null` bị xoá. Các field: `surname`, `current_location`, `since_year`, `origin`, `history`, `hall_name`, `hall_address`, ngày chạp (ở trên), `hoanh_phi`, `couplet_upper`, `couplet_lower` — **mỗi mục** hoành phi và hai vế câu đối có thêm `_reading` (phiên âm) và `_meaning` (nghĩa) — và `teaching`. Response có thêm `founder_name`, `generations_count`, `members_count` do server tính; gửi chúng bị từ chối.

## Vị trí của tôi

`my_member_id` (ở `GET /v1/genealogies/{id}` và hồ sơ thành viên) là **nút có `phone_number` trùng số điện thoại tài khoản của người gọi**; `null` khi không có nút nào trùng. Một số điện thoại chỉ thuộc một nút trong cùng gia phả, nhưng cùng tài khoản có thể là một nút khác ở gia phả thứ hai.

Người được mời **không tự đổi** được vị trí của mình: chỉ chủ gia phả đổi bằng cách sửa `phone_number` của nút. Khi `my_member_id` là `null` thì `relation_to_me` cũng `null`.

## Mời người thân vào xem

1. **Owner** chọn một thành viên **còn sống** và gọi `POST /v1/genealogies/{id}/invitations` với `{"member_id": 11, "phone_number": "0912345678"}`. Bỏ `phone_number` được nếu thành viên đã có số. Nếu thành viên chưa có số, server **ghi số này lên thành viên**. Lời mời ở trạng thái `pending`; người được mời **chưa** có quyền gì.
2. **Người được mời** mở app, `GET /v1/invitations` thấy `{id, genealogy_name, member_full_name, invited_by_phone, created_at, expires_at}` — chỉ chừng đó, không lộ cây.
3. Họ gọi `POST /v1/invitations/{id}/accept` (thành `viewer`, đứng đúng nút được mời) hoặc `.../decline`.
4. **Owner** `GET .../invitations` thấy trạng thái đổi (`pending` → `accepted` / `declined` / `expired`; lời mời đã huỷ bị ẩn) và `GET .../viewers` thấy ai đứng ở nút nào (`member_id`).

Kết quả của bước 1 **giống hệt nhau** dù số đã hay chưa có tài khoản — không có `USER_NOT_FOUND`, không dò được số nào đã đăng ký. Người chưa có tài khoản thấy lời mời sau khi đăng ký bằng đúng số đó.

:::note
v1 **không gửi SMS hay push**. Client của người được mời tự gọi `GET /v1/invitations` để hiện badge; chủ gia phả cần báo người thân ngoài app. Thông báo đẩy sẽ là một tính năng riêng.
:::

Sửa hoặc xoá `phone_number` của một thành viên thì lời mời đang chờ của thành viên đó **tự huỷ**. Người đã mất không mời được.

## Hồ sơ thành viên

`GET .../members/{memberId}/profile` trả mọi thứ cần hiển thị khi chạm vào một người, tính ở server để mọi client giống nhau:

| Field | Ghi chú |
| --- | --- |
| `member` | Đầy đủ như `GET .../members/{memberId}`. |
| `my_member_id` | Nút của người gọi (xem trên). |
| `age` | Tuổi tây: năm hiện tại trừ năm sinh; người đã mất thì tuổi thọ; `null` khi thiếu năm. |
| `traditional_age` | Tuổi mụ = `age + 1`; `null` đúng khi `age` là `null`. Hiển thị được cả hai. |
| `father`, `mother`, `spouses`, `siblings`, `children` | `MemberRef`; anh em có thêm `is_half_sibling`. Danh sách rỗng là `[]`. |
| `lineage` | Chuỗi từ thuỷ tổ xuống người đó (kể cả hai đầu). |
| `chi`, `chi_path` | Chi bắt đầu từ chỗ một người có **từ 2 con trai**; chi lồng nhau. `chi_path` từ ngoài vào trong, `chi` là chi trong cùng (`null` khi chưa rẽ nhánh). |
| `relation_to_me` | `{code, label}`: người này là gì của "tôi" (Bác, Chú, Thím, Anh họ, Ông nội…). |

**Xưng hô theo đời trong cây, không theo tuổi**: người nhỏ tuổi nhưng thuộc đời trên vẫn là vai trên. Anh/em và bác/chú so theo `birth_order`. Quan hệ xa hơn phạm vi (bên ngoại, quá xa) trả `code = UNKNOWN`, `label = null` — không phải lỗi.

## Lỗi

| HTTP | code | Khi nào |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Field sai (`details.field`, vd. `founder.gender`, `related_member_id`, `chap_day`), body sai JSON, field lạ hoặc field do server giữ. |
| 401 | `UNAUTHORIZED` | Thiếu token, token sai/hết hạn, hoặc gửi refresh token thay cho access token. |
| 403 | `GENEALOGY_FORBIDDEN` | Người xem gọi route chỉ dành cho chủ gia phả. |
| 404 | `GENEALOGY_NOT_FOUND` | Không tồn tại, id không phải số nguyên dương, **hoặc không có vai trò**. Cả ba trả đúng một phản hồi. |
| 404 | `MEMBER_NOT_FOUND` | Thành viên không có trong gia phả này (kể cả thuộc gia phả khác). |
| 404 | `VIEWER_NOT_FOUND` | Người đó không phải người xem. |
| 404 | `INVITATION_NOT_FOUND` | Lời mời không tồn tại, gửi tới số khác, hoặc (với owner) không còn `pending`. |
| 409 | `GENEALOGY_LIMIT_REACHED` | Đã đủ số gia phả tối đa. |
| 409 | `MEMBER_LIMIT_REACHED` | Gia phả đã đủ thành viên. |
| 409 | `MEMBER_HAS_DESCENDANTS` | Xoá người còn con. |
| 409 | `FOUNDER_NOT_DELETABLE`, `FOUNDER_NOT_MOVABLE` | Xoá hoặc di chuyển thuỷ tổ. |
| 409 | `MEMBER_PHONE_ALREADY_USED` | `phone_number` đã thuộc một thành viên khác của gia phả (`details.field = phone_number`). |
| 409 | `VIEWER_ALREADY_EXISTS` | Số thuộc chủ gia phả hoặc người đã xem. |
| 409 | `VIEWER_LIMIT_REACHED` | Đủ số người xem (kể cả lời mời đang chờ). |
| 409 | `INVITATION_ALREADY_PENDING` | Đã có lời mời chờ cho số này. |
| 409 | `INVITATION_NOT_PENDING` | Lời mời đã trả lời, đã huỷ hoặc hết hạn. |
| 413 | `PAYLOAD_TOO_LARGE` | Body vượt 1 MB. |

Cấu trúc đầy đủ và nút gọi thử: [API Reference](/api-reference/). Envelope và mã lỗi chung: [Xử lý lỗi](/guides/error-handling/).
