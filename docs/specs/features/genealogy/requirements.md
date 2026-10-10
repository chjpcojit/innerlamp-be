# Requirements — genealogy

Version: 5.5
Last updated: 2026-10-10
Status: not started — spec viết prospectively, chưa có code/migration. Còn marker `⚠️ Not confirmed` (xem "Pending"); chỉ marker ghi rõ "chặn" mới chặn task tương ứng.
Module: `genealogy`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code) cho tính năng **Gia phả**: người dùng đã đăng nhập tạo một gia phả (cây dòng họ + phả ký), thêm/sửa các thành viên, và **mời người trong họ vào xem** (người được mời chấp nhận hoặc từ chối). Khác `memorial` (riêng tư tuyệt đối theo user), gia phả **có nhiều người xem nhưng chỉ chủ sửa**, và **không công khai**: dữ liệu chứa số điện thoại, nơi ở của người còn sống.

## Quyết định nền tảng (đã chốt với anh Đức, 2026-10-07)

- **Chỉ có hai vai trò: `owner` và `viewer`** (chốt 2026-10-10, thay cho ba vai trò owner/clan_head/editor của v1–v4). `owner` (người tạo) xem **và sửa** mọi thứ. `viewer` (người được owner mời) **chỉ xem**: cây, phả ký, hồ sơ thành viên; không sửa, không xoá, không mời ai. Người không thuộc hai vai trò **không xem được** gì (kể cả sự tồn tại của gia phả). Không có vai trò `editor` hay trưởng tộc nữa.
- **Mời một thành viên cụ thể vào xem, bằng số điện thoại** (chốt 2026-10-10): owner **chọn một thành viên trong cây** rồi mời họ bằng số điện thoại (số tự lấy từ thành viên nếu đã ghi). Lời mời ở trạng thái **chờ**; người được mời thấy lời mời ở hộp thông báo của họ, **chấp nhận** (lời mời thành `accepted`, họ thành `viewer` và đứng đúng ở nút đó) hoặc **từ chối**. Dù số đó **đã hoặc chưa** có tài khoản, hệ thống đều tạo lời mời và trả cùng một kết quả. Không có mã mời/link.
- **Trưởng tộc chỉ là một cờ ghi nhớ trên thành viên** (chốt 2026-10-10): không phải vai trò, không cấp quyền gì. Trưởng tộc phải là **nam, còn sống, thuộc dòng họ**. Mỗi gia phả có **tối đa một** trưởng tộc; đặt người mới thì người cũ tự mất cờ (AC82). Không còn route chỉ định trưởng tộc, không còn `clan_head` là vai trò của tài khoản.
- **Con của ai = chọn đúng một cha/mẹ ruột** trong cây (người đó có thể có vợ/chồng kèm theo). Cây hiển thị theo dòng họ.
- **Nhiều vợ/chồng.** Một thành viên có thể có nhiều người phối ngẫu (tái hôn, đa thê thời xưa). Người phối ngẫu không thuộc dòng họ được ghi là **dâu** (vợ của nam) hoặc **rể** (chồng của nữ) và không có đời riêng.
- **Năm sinh/năm mất chỉ lưu năm, dương lịch**; năm mất là tuỳ chọn khi đã mất. Ngày giỗ (tháng/ngày) lưu ngay trên thành viên theo AC55 (dương lịch + cờ nhập theo âm); chỉ việc **nhắc giỗ hằng năm** thuộc về `memorial`.
- **Chỉ `owner` mới ghi được** (thêm/sửa/xoá thành viên, phả ký, tên gia phả, mời/thu quyền xem); `viewer` gọi route ghi nhận `403 GENEALOGY_FORBIDDEN` (AC81).
- **"Tôi ở đâu trong cây" tính theo số điện thoại, không phải ai tự chọn** (chốt 2026-10-10, thay cho con trỏ riêng tư v1–v4): mỗi tài khoản có đúng một số điện thoại, mỗi thành viên có tối đa một `phone_number`; thành viên nào trong gia phả có `phone_number` trùng số điện thoại của tài khoản người gọi thì đó là "tôi" (AC59). Một số điện thoại chỉ thuộc một thành viên trong cùng gia phả (AC60) nên một nút gắn tối đa một tài khoản; cùng một tài khoản có thể là một nút khác ở gia phả thứ hai. **Người được mời không tự đổi được** — chỉ owner đổi bằng cách sửa `phone_number` của thành viên (AC30). Lời mời luôn gắn một thành viên nên người được mời chấp nhận là có vị trí ngay (AC44, AC71); chỉ khi owner sau đó xoá/đổi số của nút thì `my_member_id = null` và hồ sơ không có xưng hô.
- **Thông tin hiển thị khi chọn một người được tính ở server** (chốt 2026-10-07): tuổi, cha/mẹ/vợ chồng/anh em/con, đường từ thuỷ tổ, chi, xưng hô với "tôi" — một route `profile` trả sẵn, client không tự tính (nhiều client, tránh lệch quy tắc xưng hô).
- **"Chi" bắt đầu từ chỗ cây rẽ nhánh và lồng nhau** (chốt 2026-10-07, thay cho cách hiểu "đời 2" ở bản đầu): mỗi lần một người có từ 2 con trai trở lên thì mỗi người con trai là đầu một chi. Ví dụ thuỷ tổ có hai con là cụ Lộc và cụ Nghĩa (chi Lộc, chi Nghĩa); chi Lộc đi tiếp rồi cụ Lộc có hai con A và B thì tách thành chi A và chi B. Người thuộc chi A có đường chi `[Lộc, A]`, chi hiện tại là A.
- **Các giới hạn số đã chốt** (2026-10-10): tối đa 10 gia phả mỗi người tạo, 5000 thành viên và 60 đời mỗi gia phả, 20 người xem (`viewer`) mỗi gia phả (lời mời chờ tính vào trần; giữ con số 20 đã chốt cho editor), lời mời hết hạn sau 30 ngày.
- **Chỉ thành viên nam thuộc dòng họ mới có con trong cây** (chốt 2026-10-10): con của thành viên nữ dòng họ (con gái) không được thêm vào cây; con gái vẫn là thành viên, là lá của cây, và có thể có chồng (rể).
- **Xoá thành viên phải đi từ đời dưới lên** (chốt 2026-10-10): người còn hậu duệ bị chặn xoá, không xoá cả nhánh.
- **Sửa nhầm quan hệ được** (chốt 2026-10-10): đổi cha/mẹ của một người dòng họ hoặc đổi người mà dâu/rể được gắn, qua một route riêng (AC76–AC80); route `PUT` thành viên vẫn không đổi quan hệ.
- **Lệch năm sinh cha/con chỉ cảnh báo** (chốt 2026-10-10): server vẫn ghi và trả `warnings`, không từ chối.
- **Gia huấn có phiên âm và nghĩa cho cả hoành phi lẫn hai vế câu đối** (chốt 2026-10-10).
- **Ngày chạp họ và ngày giỗ lưu dương lịch kèm cờ "nhập theo âm"; server quy đổi và trả luôn ngày âm** (chốt 2026-10-10): chỉ lưu một loại lịch; ngày âm là field tính ra, không lưu, để client không phải tự quy đổi.
- **Không có chuyển quyền `owner`** (chốt 2026-10-10).
- Mã user story `US-GEN-01..10` do spec tự chọn (theo tiền lệ `US-MEM-*`).

---

## US-GEN-01: Tạo gia phả (kèm thuỷ tổ)

## User Story

> Là một **người dùng đã đăng nhập**,
> Tôi muốn **tạo gia phả của dòng họ với tên gia phả, quê gốc và thuỷ tổ đời 1 (cùng các dâu/rể của thuỷ tổ)**,
> Để **bắt đầu dựng cây gia phả từ người đầu tiên**.

## Acceptance Criteria (EARS notation)

1. WHEN a `POST /v1/genealogies` request arrives with a valid access token and a valid body, THE SYSTEM SHALL — trong **một transaction** — create the genealogy owned by the token's user, an empty chronicle (phả ký), the founder as a blood member of generation 1 with no parent, and every listed spouse as an in-law member attached to the founder, then return `201 Created` with the full `GenealogyDetail`.
2. WHEN the request carries no valid access token, THE SYSTEM SHALL return `401` and SHALL NOT create anything.
3. WHEN `name` is blank after trim or longer than 100 characters, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "name"`; the same for `ancestral_home` (quê gốc, bắt buộc, ≤ 200 ký tự) with `details.field = "ancestral_home"`.
4. WHEN `founder.full_name` is blank or longer than 100 characters, or `founder.gender` is not `male` (thuỷ tổ phải là nam, chốt 2026-10-10 — vì con của thành viên nữ không vào cây, AC19), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field` = `founder.full_name` / `founder.gender`.
5. WHEN `founder.is_alive` is absent, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "founder.is_alive"`.
6. WHEN `is_alive` is `true` AND `death_year` or `tablet_location` (linh vị đặt tại) is present, THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming that field — chỉ người đã mất mới có năm mất và nơi đặt linh vị.
7. WHEN `birth_year` or `death_year` is present, THE SYSTEM SHALL require an integer in `1000..<current year in Asia/Ho_Chi_Minh>`, and `death_year >= birth_year` when both are present; otherwise it SHALL return `400 VALIDATION_ERROR` naming the field. Cả hai năm đều tuỳ chọn.
8. WHEN `residence` > 200, `dharma_name` > 100, `tablet_location` > 200, `note` > 1000 characters, or `phone_number` is present but not a valid phone number (cùng quy tắc chuẩn hoá số điện thoại của register), THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming the field. Áp dụng cho **mọi** nơi nhận thông tin một thành viên (founder, spouses, thêm/sửa thành viên).
9. WHEN `spouses` has more than 5 items, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "spouses"`. Mỗi spouse có cùng các field như thành viên (AC4–AC8, trừ việc `gender` là tuỳ chọn): WHEN `spouses[i].gender` is absent, THE SYSTEM SHALL set it to the opposite of the founder's. WHEN present it is stored as given.
10. WHEN the user already owns 10 genealogies, THE SYSTEM SHALL return `409 GENEALOGY_LIMIT_REACHED` and SHALL NOT insert — kể cả khi hai request tạo chạy đồng thời. (Đã chốt 2026-10-10.)
11. WHEN the body has unknown fields, is empty, or is not valid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (`DecodeJSONBodyStrict`). WHEN the body sets a server-owned field (`id`, `owner_user_id`, `generation`, `kind`, `parent_id`, `spouse_of_id`), THE SYSTEM SHALL reject it as unknown.
12. WHEN any step of AC1 fails, THE SYSTEM SHALL leave no genealogy, chronicle or member row behind.

## US-GEN-02: Xem, đổi thông tin chung, xoá gia phả

## User Story

> Là **chủ gia phả (`owner`)** hoặc **người được mời xem (`viewer`)**,
> Tôi muốn **xem danh sách gia phả của mình, xem tổng quan một gia phả; chủ thì có thể đổi tên/quê gốc và xoá cả gia phả**,
> Để **quản lý gia phả của dòng họ**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/genealogies` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding every genealogy where the token's user is `owner` or `viewer` — each with `id`, `name`, `ancestral_home`, `my_role` (`owner` | `viewer`) — ordered by `created_at` DESC then `id` DESC, and `data.total`. WHEN there are none, `items` SHALL be `[]`.
14. WHEN a `GET /v1/genealogies/{id}` request arrives AND the user has a role in it, THE SYSTEM SHALL return `200` with `GenealogyDetail`: thông tin chung, `my_role`, `my_member_id` (AC62), `chronicle` (US-GEN-05) và `stats` (`generations_count`, `members_count`) tính ra từ cây tại thời điểm đọc, không lưu.
15. WHEN `{id}` does not exist, is not a positive integer, OR the user has no role in it, THE SYSTEM SHALL return the **same** `404 GENEALOGY_NOT_FOUND` for all three — không lộ sự tồn tại của gia phả. Quy tắc này áp cho **mọi** route con của `/v1/genealogies/{id}/...` (members, chronicle, viewers, invitations).
16. WHEN a `PUT /v1/genealogies/{id}` request arrives from the `owner` AND `name`/`ancestral_home` are valid (AC3), THE SYSTEM SHALL update them and return `200` with `GenealogyDetail`. Chỉ `owner` sửa được; `viewer` nhận `403 GENEALOGY_FORBIDDEN` (AC81).
17. WHEN a `DELETE /v1/genealogies/{id}` request arrives from the `owner`, THE SYSTEM SHALL delete the genealogy and, by cascade, its chronicle, members viewer grants and invitations (xoá cứng), and return `200` `{"success": true}`. WHEN the caller is a `viewer`, THE SYSTEM SHALL return `403 GENEALOGY_FORBIDDEN` and delete nothing.

## US-GEN-03: Thêm thành viên vào gia phả

## User Story

> Là **chủ gia phả (`owner`)**,
> Tôi muốn **thêm một người mới là con của ai hoặc vợ/chồng của ai, cùng họ tên, giới tính, tình trạng còn sống/mất, nơi ở, số điện thoại, ghi chú**,
> Để **cây gia phả đầy đủ dần**.

## Acceptance Criteria (EARS notation)

18. WHEN a `POST /v1/genealogies/{id}/members` request arrives from the `owner`, with `relation` (`child` | `spouse`), `related_member_id` and a valid member body, THE SYSTEM SHALL create the member and return `201` with the full `Member`.
19. WHEN `relation = "child"`, THE SYSTEM SHALL create a **blood** member whose `parent_id = related_member_id` and `generation = parent.generation + 1`. WHEN the related member is an in-law (dâu/rể), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` — con luôn gắn vào người thuộc dòng họ. WHEN the related member is a blood member with `gender = "female"`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` — con của thành viên nữ dòng họ không được thêm vào cây (chốt 2026-10-10).
20. WHEN `relation = "spouse"`, THE SYSTEM SHALL create an **in-law** member with `spouse_of_id = related_member_id` and the same `generation` as that member. WHEN the related member is itself an in-law, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"`. WHEN `gender` is absent, THE SYSTEM SHALL default it to the opposite of the related member's. Một thành viên có tối đa 5 người phối ngẫu; vượt quá → `400 VALIDATION_ERROR` với `details.field = "related_member_id"`.
21. WHEN `relation` is neither `child` nor `spouse`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "relation"`.
22. WHEN `related_member_id` does not exist in this genealogy (kể cả khi tồn tại ở gia phả khác), THE SYSTEM SHALL return `404 MEMBER_NOT_FOUND` — không lộ thành viên của gia phả khác.
23. WHEN the member body violates AC4–AC8, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with the same `details.field` (không có prefix `founder.`).
24. WHEN the genealogy already has 5000 members, THE SYSTEM SHALL return `409 MEMBER_LIMIT_REACHED` and SHALL NOT insert — kể cả khi hai request thêm chạy đồng thời. (Đã chốt 2026-10-10.)
25. WHEN the child would be generation greater than 60, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` (chặn cây sâu bất thường). (Đã chốt 2026-10-10.)
26. WHEN the body has unknown fields, is empty or invalid JSON, or sets a server-owned field (AC11), THE SYSTEM SHALL return `400 VALIDATION_ERROR`.
27. THE SYSTEM SHALL NOT reject a write because years are inconsistent between different people — chỉ validate từng người riêng lẻ (AC7). WHEN, after a `POST` (AC18) / `PUT` (AC30) of a member or a relation change (AC76) has been committed, a blood member's `birth_year` is less than the `birth_year` of its `parent_id` member (kể cả khi người vừa ghi là cha/mẹ và một người con có `birth_year` nhỏ hơn), THE SYSTEM SHALL still return success and include in the response `warnings`: `[{"code": "CHILD_BORN_BEFORE_PARENT", "member_id": <con>, "related_member_id": <cha/mẹ>}]`; otherwise `warnings` is `[]`. Chỉ so khi cả hai có `birth_year`; bằng nhau không cảnh báo; không so vợ/chồng. `warnings` được tính lúc ghi, không lưu, không có trong `GET` (chốt 2026-10-10: có cảnh báo, không chặn).

## US-GEN-04: Xem, sửa, xoá thành viên

## User Story

> Là **chủ gia phả (`owner`)**,
> Tôi muốn **xem danh sách thành viên và sửa lại thông tin bất kỳ ai (tên, giới tính, năm sinh, còn sống/mất, nơi ở, pháp danh, linh vị đặt tại...)**, và **xoá người ghi nhầm**,
> Để **gia phả luôn chính xác**.

## Acceptance Criteria (EARS notation)

28. WHEN a `GET /v1/genealogies/{id}/members` request arrives from a user with a role, THE SYSTEM SHALL return `200` with `data.items` holding **every** member of that genealogy (dạng phẳng, client tự dựng cây từ `parent_id`/`spouse_of_id`) ordered by `generation` ASC then `id` ASC, and `data.total`. Không phân trang (tối đa 5000).
29. WHEN a `GET /v1/genealogies/{id}/members/{memberId}` request arrives AND the member belongs to that genealogy, THE SYSTEM SHALL return `200` with the full `Member`; otherwise `404 MEMBER_NOT_FOUND` (cùng mã cho "không tồn tại" và "thuộc gia phả khác").
30. WHEN a `PUT /v1/genealogies/{id}/members/{memberId}` request arrives from the `owner` AND the body is valid, THE SYSTEM SHALL replace the editable fields — `full_name`, `gender`, `birth_year`, `is_alive`, `death_year`, `residence`, `dharma_name`, `tablet_location`, `burial_place`, `death_month`, `death_day`, `death_entered_as_lunar`, `phone_number`, `note`, và `birth_order` (chỉ với thành viên dòng họ không phải thuỷ tổ, AC56) — and return `200` with the full `Member`. Validation theo AC4–AC8 và AC55–AC56. `phone_number` của một thành viên cũng là cách gắn tài khoản với nút trong cây (AC59) nên phải duy nhất trong gia phả (AC60). WHEN a member's `phone_number` is changed or cleared AND the member has a `pending` invitation, THE SYSTEM SHALL mark that invitation `cancelled` in the same transaction (lời mời luôn khớp số trên nút). `is_clan_head` ghi theo AC82 (vắng mặt trong body `PUT` nghĩa là **giữ nguyên**, không xoá cờ). WHEN `gender` is changed to `female` on a blood member that already has children, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "gender"` (giữ bất biến của AC19); WHEN `gender` of the founder is changed to `female`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "gender"` (AC4); the response also carries `warnings` (AC27).
31. WHEN the body of `PUT` carries `relation`, `related_member_id`, `parent_id`, `other_parent_id`, `spouse_of_id`, `generation`, `kind` or any server-owned field, THE SYSTEM SHALL reject it as an unknown field — quan hệ **không** đổi qua route sửa thông tin; muốn sửa nhầm "con của ai"/"vợ chồng của ai" dùng route riêng ở US-GEN-10 (AC76–AC80).
32. WHEN `gender` of a blood member is changed, THE SYSTEM SHALL keep the in-law members attached to it unchanged; nhãn dâu/rể của họ là **giá trị tính ra** từ giới tính của người được gắn (AC33), không lưu.
33. THE SYSTEM SHALL return `in_law_label` on every in-law member: `DAU` when the attached blood member is male, `RE` when female; and `null` for blood members.
34. WHEN a `DELETE /v1/genealogies/{id}/members/{memberId}` request arrives from the `owner` AND the member is a blood member with **no children**, THE SYSTEM SHALL delete the member **and every in-law attached to it** in one transaction and return `200` `{"success": true}`.
35. WHEN the target is a blood member that has at least one child, THE SYSTEM SHALL return `409 MEMBER_HAS_DESCENDANTS` and SHALL NOT delete anything — phải xoá từ đời dưới lên (chốt 2026-10-10: chặn, không xoá cả nhánh). Muốn bỏ nhánh thì xoá từng người từ lá lên, hoặc chuyển người con sang cha/mẹ khác (AC76) trước.
36. WHEN the target is the founder (đời 1, không có cha mẹ), THE SYSTEM SHALL return `409 FOUNDER_NOT_DELETABLE` — muốn bỏ cả cây thì xoá gia phả (AC17).
37. WHEN the target is an in-law, THE SYSTEM SHALL delete just that member and return `200`; WHEN `{memberId}` does not belong to the genealogy, `404 MEMBER_NOT_FOUND`.

## US-GEN-05: Phả ký của dòng họ

## User Story

> Là **chủ gia phả (`owner`)**,
> Tôi muốn **ghi và xem phả ký: họ gì, ở đâu, từ năm nào, nguồn gốc, lịch sử, nhà thờ họ, ngày chạp họ và gia huấn**,
> Để **lưu giữ thông tin chung của dòng họ**.

## Acceptance Criteria (EARS notation)

38. WHEN a `PUT /v1/genealogies/{id}/chronicle` request arrives from the `owner` AND the body is valid, THE SYSTEM SHALL replace **all** chronicle fields with the body (field vắng/`null` = xoá) and return `200` with the `Chronicle`. Phả ký được tạo rỗng cùng gia phả (AC1) nên route này chỉ ghi đè, không có bước "tạo phả ký".
39. THE SYSTEM SHALL accept these writable fields, all optional: `surname` (họ, ≤ 50), `current_location` (ở đâu, ≤ 200), `since_year` (từ năm nào, `1000..current year`), `origin` (nguồn gốc, ≤ 5000), `history` (lịch sử dòng họ, ≤ 20000), `hall_name` (nhà thờ họ, ≤ 200), `hall_address` (địa chỉ, ≤ 300), `chap_year` + `chap_month` + `chap_day` + `chap_entered_as_lunar` (ngày chạp họ, dương lịch, AC40), and the gia huấn block `hoanh_phi` (≤ 200), `hoanh_phi_reading` (phiên âm, ≤ 200), `hoanh_phi_meaning` (nghĩa, ≤ 1000), `couplet_upper` (câu đối vế trên, ≤ 500), `couplet_upper_reading` (phiên âm, ≤ 500), `couplet_upper_meaning` (nghĩa, ≤ 1000), `couplet_lower` (câu đối vế dưới, ≤ 500), `couplet_lower_reading` (phiên âm, ≤ 500), `couplet_lower_meaning` (nghĩa, ≤ 1000), `teaching` (lời dạy, ≤ 5000). Vi phạm độ dài/range → `400 VALIDATION_ERROR` với `details.field` tương ứng. (Chốt 2026-10-10: phiên âm và nghĩa áp cho cả hoành phi lẫn hai vế câu đối.)
40. THE SYSTEM SHALL store the ngày chạp họ as a **dương lịch** date — `chap_year`, `chap_month`, `chap_day` — whether the user picked it as dương or âm lịch (chốt 2026-10-10: chỉ lưu một loại lịch). WHEN any of the three is present, THE SYSTEM SHALL require all three; `chap_year` in `1000..current year`, `chap_month` in `1..12`, `chap_day` in `1..<số ngày tối đa của tháng đó, tháng 2 tính 29>`; otherwise `400 VALIDATION_ERROR` with `details.field` = the missing/invalid field. THE SYSTEM SHALL accept `chap_entered_as_lunar` (bool, mặc định `false`): `true` nghĩa là người dùng đã **chọn theo âm lịch** và client quy sang dương trước khi gửi — cờ để client biết hiển thị/tính lại theo âm lịch; WHEN it is `true` without the three date fields, `400 VALIDATION_ERROR` with `details.field = "chap_entered_as_lunar"`. Ngày âm lịch tương ứng do server tính (xem dưới), không lưu. Không còn `chap_leap_month` (tháng nhuận là khái niệm âm lịch).
    THE SYSTEM SHALL return trong `chronicle` của `GenealogyDetail` (AC14, AC38) thêm các field **tính ra** `chap_lunar_day`, `chap_lunar_month` (int) và `chap_lunar_leap` (bool): ngày âm lịch Việt Nam (múi giờ UTC+7) tương ứng với ngày dương đã lưu; cả ba là `null` khi chưa có ngày chạp hoặc ngày nằm ngoài khoảng server quy đổi được (năm `1900..2199`). Chúng không ghi được (đưa vào body `PUT` → rejected as unknown field) và được trả cả khi `chap_entered_as_lunar = false`; cờ chỉ cho client biết người dùng đã chọn lịch nào để hiển thị trước. (Chốt 2026-10-10: server quy đổi.)
41. WHEN `GenealogyDetail` is read (AC14), THE SYSTEM SHALL include the chronicle's **derived** fields: `founder_name` (họ tên thuỷ tổ, lấy từ thành viên đời 1), `generations_count` (đời lớn nhất đang có) and `members_count` (tổng số thành viên, **gồm cả dâu/rể**). Chúng không ghi được, đưa vào body `PUT` → rejected as unknown field.
42. WHEN the body has unknown fields, is empty or invalid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR`. WHEN the genealogy is not visible to the user, `404 GENEALOGY_NOT_FOUND` (AC15).
43. THE SYSTEM SHALL NOT store images (ảnh hoành phi, ảnh thành viên) — nằm ngoài v1 (xem Out of scope).

## US-GEN-06: Mời người xem gia phả

## User Story

> Là **người tạo gia phả**,
> Tôi muốn **mời người thân trong họ xem gia phả bằng số điện thoại**,
> Để **họ thấy cây dòng họ và biết mình đứng ở đâu mà không sửa được gì**.

> Là **người được mời**,
> Tôi muốn **nhận lời mời, rồi chấp nhận hoặc từ chối**,
> Để **chỉ xem gia phả mà tôi đồng ý**.

## Acceptance Criteria (EARS notation)

44. WHEN a `POST /v1/genealogies/{id}/invitations` request arrives from the `owner` with `member_id` (một thành viên của gia phả này) and `phone_number` (chuẩn hoá như register; có thể bỏ nếu thành viên đã có `phone_number`), THE SYSTEM SHALL — trong **một transaction** — create a **pending invitation** for that member and number and return `201` with the `Invitation` (`id`, `member_id`, `phone_number`, `status = "pending"`, `created_at`, `expires_at`, `responded_at = null`). WHEN the member has no `phone_number`, THE SYSTEM SHALL write the invited number onto the member (để vị trí của người được mời khớp theo AC59). Người được mời **chưa** có quyền gì cho tới khi chấp nhận (AC71). WHEN `member_id` is absent or not a positive integer, `400 VALIDATION_ERROR` với `details.field = "member_id"`; WHEN it does not belong to this genealogy, `404 MEMBER_NOT_FOUND`; WHEN the member has `is_alive = false`, `400` với `details.field = "member_id"` (chỉ mời người còn sống); WHEN `phone_number` is absent and the member has none, or differs from the number already on the member, `400` với `details.field = "phone_number"`; WHEN the number already belongs to another member of this genealogy, `409 MEMBER_PHONE_ALREADY_USED` (AC60).
45. THE SYSTEM SHALL create the invitation and return the **same** `201 Invitation` whether or not an account with that phone number exists — route này SHALL NOT return `USER_NOT_FOUND`, và không có khác biệt nào ở status/body dùng để dò số đã đăng ký. WHEN the number has no account yet, the invitation waits and is shown to the person after they register and log in with that number (AC75).
46. WHEN the phone number belongs to the owner or to a user who is already a `viewer`, THE SYSTEM SHALL return `409 VIEWER_ALREADY_EXISTS`; WHEN the genealogy already has an unexpired `pending` invitation for that number, `409 INVITATION_ALREADY_PENDING`. In both cases THE SYSTEM SHALL NOT insert a duplicate row. (Hai mã 409 này chỉ cho owner biết điều owner vốn đã thấy qua AC48/AC73, nên không lộ thêm số nào đã đăng ký.)
47. WHEN the number of viewers plus unexpired `pending` invitations of the genealogy has reached 20, THE SYSTEM SHALL return `409 VIEWER_LIMIT_REACHED` for a new invitation; WHEN accepting an invitation would make the viewers exceed 20, the same `409` (AC71). (Đã chốt 2026-10-10: trần 20, lời mời chờ có tính vào trần.)
48. WHEN a `GET /v1/genealogies/{id}/viewers` request arrives from the `owner`, THE SYSTEM SHALL return `200` with `data.items` holding the viewers — each `{user_id, phone_number, member_id, created_at}` where `member_id` is the member whose `phone_number` equals that viewer's account phone number (AC59), hoặc `null` — and `data.total`; để owner biết ai đang đứng ở nút nào. WHEN a `viewer` asks, `403 GENEALOGY_FORBIDDEN`.
49. WHEN a `DELETE /v1/genealogies/{id}/viewers/{userId}` request arrives from the `owner`, THE SYSTEM SHALL remove that viewer's grant and return `200` `{"success": true}`; WHEN that user is not a viewer, `404 VIEWER_NOT_FOUND`.
50. WHEN a `viewer` calls `DELETE /v1/genealogies/{id}/viewers/{their own user id}`, THE SYSTEM SHALL remove their own grant (tự rời gia phả) and return `200`. `owner` không tự rời bằng route này.
51. (Đã bỏ ở v5.0: route chỉ định trưởng tộc — trưởng tộc nay là cờ trên thành viên, xem AC82.)
52. (Đã bỏ ở v5.0: route bỏ trưởng tộc — đặt `is_clan_head = false` qua `PUT` thành viên, AC82.)
53. WHEN a `viewer` calls any route of AC44, AC48, AC49 (cho người khác), AC73, AC74 or any write route (AC81), THE SYSTEM SHALL return `403 GENEALOGY_FORBIDDEN`; WHEN the caller has no role, `404 GENEALOGY_NOT_FOUND` (AC15).
54. WHEN a viewer's grant is removed, THE SYSTEM SHALL deny that user on their very next request (quyền kiểm ở mỗi request từ DB, không nằm trong token). THE SYSTEM SHALL NOT provide any route to transfer the `owner` role (chốt 2026-10-10: không có chuyển quyền owner); xoá tài khoản `owner` bị chặn bởi FK `RESTRICT`.
70. WHEN a `GET /v1/invitations` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding the `pending`, unexpired invitations whose `phone_number` equals the caller's own phone number (ordered by `created_at` DESC) and `data.total`; each item is `{id, genealogy_name, member_full_name, invited_by_phone, created_at, expires_at}` (`member_full_name` là tên nút mà chủ gia phả mời người này vào). THE SYSTEM SHALL NOT expose any other genealogy data (cây, các thành viên khác, phả ký) to the invitee before acceptance. Đây là "hộp lời mời" của người được mời — client hiển thị badge từ route này. Thông báo đẩy (push) là một **tính năng riêng, làm sau** (Out of scope); không dùng SMS (tốn phí).
71. WHEN a `POST /v1/invitations/{id}/accept` request arrives from the user whose phone number matches the invitation AND the invitation is `pending` and unexpired, THE SYSTEM SHALL in one transaction record the caller as `viewer` of the genealogy (`granted_by` = người mời), mark the invitation `accepted`, and return `200` with `{genealogy_id, member_id, role: "viewer"}` (`member_id` là nút của họ, tức `my_member_id` ở lần đọc kế tiếp); quyền có hiệu lực ở request kế tiếp (AC54). WHEN the invitation does not exist, is not addressed to the caller's phone number, or `{id}` is not a positive integer, THE SYSTEM SHALL return `404 INVITATION_NOT_FOUND` (cùng một mã cho cả ba — không lộ lời mời của người khác). WHEN it is no longer pending (đã trả lời, bị huỷ hoặc hết hạn), `409 INVITATION_NOT_PENDING`. WHEN the caller is already the owner or a viewer of the genealogy, `409 VIEWER_ALREADY_EXISTS`; WHEN the genealogy already has 20 viewers, `409 VIEWER_LIMIT_REACHED`.
72. WHEN a `POST /v1/invitations/{id}/decline` request arrives from the user whose phone number matches a `pending`, unexpired invitation, THE SYSTEM SHALL mark it `declined` without granting any role and return `200` `{"success": true}`. The `404 INVITATION_NOT_FOUND` and `409 INVITATION_NOT_PENDING` rules of AC71 apply.
    > ⚠️ Not confirmed: owner có được mời lại ngay số vừa từ chối hay cần chờ/chặn (tránh làm phiền) — spec hiểu là mời lại được, không giới hạn thêm.
73. WHEN a `GET /v1/genealogies/{id}/invitations` request arrives from the `owner`, THE SYSTEM SHALL return `200` with `data.items` holding the genealogy's invitations whose status is `pending`, `accepted`, `declined` or `expired` (ordered by `created_at` DESC), each `{id, member_id, member_full_name, phone_number, status, created_at, expires_at, responded_at}`, and `data.total`; owner thấy trạng thái cập nhật khi người được mời chấp nhận/từ chối. Lời mời `cancelled` bị ẩn. A `viewer` → AC53.
74. WHEN a `DELETE /v1/genealogies/{id}/invitations/{invitationId}` request arrives from the `owner` AND the invitation belongs to that genealogy AND is `pending`, THE SYSTEM SHALL mark it `cancelled` and return `200` `{"success": true}`; otherwise `404 INVITATION_NOT_FOUND`.
75. THE SYSTEM SHALL match an invitation to a person **by phone number only** (không lưu `user_id` của người được mời): người đăng ký tài khoản sau với số đó (đã xác thực OTP) sẽ thấy lời mời còn hạn ở AC70, không cần đổi luồng register. An invitation expires 30 days after creation; một lời mời hết hạn không còn hiện ở AC70 và không chấp nhận được (AC71), không cần job dọn. (Hạn 30 ngày đã chốt 2026-10-10.)
    > ⚠️ Not confirmed: khi số điện thoại bị thu hồi rồi cấp cho người khác trong lúc lời mời còn hạn thì người mới sẽ thấy lời mời (giảm nhẹ: phải chủ động chấp nhận + hết hạn 30 ngày) — có chấp nhận rủi ro này không; không chặn. Cùng rủi ro với việc gắn tài khoản vào nút theo số điện thoại (AC59).
81. WHEN a `viewer` calls any of `PUT /v1/genealogies/{id}`, `DELETE /v1/genealogies/{id}`, `PUT .../chronicle`, `POST`/`PUT`/`DELETE .../members...`, `PUT .../members/{memberId}/relation`, or the owner-only invitation/viewer routes (AC53), THE SYSTEM SHALL return `403 GENEALOGY_FORBIDDEN` and SHALL NOT change anything. `viewer` chỉ gọi được các route `GET` của gia phả (AC13, AC14, AC28, AC29, AC64), tự rời (AC50) và hộp lời mời của chính mình (AC70–AC72).
82. THE SYSTEM SHALL store `is_clan_head` (bool, mặc định `false`) on a member as a **cờ ghi nhớ** — không phải vai trò, không cấp quyền gì. Chỉ một thành viên **blood, `gender = male`, `is_alive = true`** mới giữ được cờ (chốt 2026-10-10). Có thể gửi ở founder, `POST` và `PUT` thành viên (chỉ `owner`; vắng mặt ở `PUT` = giữ nguyên). WHEN `is_clan_head = true` is written for an in-law, a female, or a member with `is_alive = false`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "is_clan_head"` (`spouses[i].is_clan_head` ở `POST /v1/genealogies`). WHEN a write would leave the current holder female or `is_alive = false` (đổi `gender`/`is_alive` của người đang giữ cờ mà không gửi `is_clan_head = false`), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "gender"` hoặc `"is_alive"` — owner phải bỏ cờ hoặc đặt người khác trước (có thể làm trong cùng một `PUT` bằng `is_clan_head = false`). WHEN `is_clan_head = true` is written for a valid member, THE SYSTEM SHALL — trong cùng transaction — clear the flag of the previous holder, nên một gia phả luôn có **tối đa một** trưởng tộc; WHEN `false`, THE SYSTEM SHALL clear it (không ai là trưởng tộc). `GenealogyDetail.clan_head_member_id` trả id nút đang giữ cờ hoặc `null`. Xoá người đang giữ cờ thì cờ mất theo.

## US-GEN-07: Thông tin mở rộng của thành viên (mẹ, thứ tự con, giỗ, an táng)

## User Story

> Là **chủ gia phả (`owner`)**,
> Tôi muốn **ghi thêm mẹ của một người con, thứ tự anh em, ngày giỗ (lưu dương lịch, có cờ nhập theo âm) và nơi an táng**,
> Để **hồ sơ mỗi người đủ thông tin cho việc xem quan hệ, chi trưởng/chi thứ và ngày giỗ**.

## Acceptance Criteria (EARS notation)

55. THE SYSTEM SHALL accept these optional fields wherever a member's information is submitted (founder, spouses, `POST` và `PUT` thành viên): `burial_place` (nơi an táng, ≤ 200), `death_month` (1..12), `death_day` (1..<số ngày tối đa của tháng đó, tháng 2 tính 29>) và `death_entered_as_lunar` (bool, mặc định `false`). Ngày giỗ lưu **dương lịch** dù người dùng chọn âm hay dương, cùng quy tắc ngày chạp (AC40, chốt 2026-10-10); năm là `death_year` đã có. WHEN `death_day` and `death_month` are not both present or both absent, or either is present without `death_year`, or `death_entered_as_lunar` is `true` without both, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field` = `death_day` / `death_month` / `death_year` / `death_entered_as_lunar`. WHEN `is_alive` is `true` AND any of `burial_place`, `death_day`, `death_month`, `death_entered_as_lunar` is present (`true`), THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming that field (như AC6). THE SYSTEM SHALL return trong `Member` thêm các field **tính ra** `death_lunar_day`, `death_lunar_month`, `death_lunar_leap` (ngày âm lịch tương ứng với `death_year`/`death_month`/`death_day` như AC40; `null` khi không có ngày giỗ hoặc ngoài khoảng `1900..2199`); không ghi được, không nằm trong body `POST`/`PUT` (rejected as unknown field).
    (Đã chốt 2026-10-10: ngày giỗ lưu **ngay trên thành viên gia phả**, không liên kết/ghi vào `memorial`.)
56. THE SYSTEM SHALL store `birth_order` (thứ tự trong các con **cùng `parent_id`**, số nguyên `1..1000`) for blood members other than the founder. WHEN a child is created without `birth_order`, THE SYSTEM SHALL set it to `max(birth_order của các con cùng parent_id) + 1` (hoặc `1` nếu chưa có con). WHEN `birth_order` is present on `POST`/`PUT` of a blood non-founder member, THE SYSTEM SHALL require an integer in `1..1000`; trùng số được phép, khi đó thứ tự theo `id` ASC. WHEN `birth_order` is sent for the founder or an in-law, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "birth_order"`. Thứ tự dùng cho `profile` (AC66, AC68), không suy từ `birth_year` vì nhiều người không có năm sinh.
    > ⚠️ Not confirmed: cách chỉnh thứ tự — gửi `birth_order` qua `PUT` từng người (như spec này) hay có route sắp xếp lại cả nhóm anh em — không chặn.
57. WHEN `POST /v1/genealogies/{id}/members` with `relation = "child"` carries `other_parent_id` (cha/mẹ còn lại của con), THE SYSTEM SHALL require it to be an **in-law** member of the same genealogy with `spouse_of_id = related_member_id`; otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field = "other_parent_id"` (kể cả khi id tồn tại ở gia phả khác). `other_parent_id` is optional, chỉ nhận khi `relation = "child"` (với `spouse` → `400`, `details.field = "other_parent_id"`), và chỉ đổi được qua route đổi quan hệ (AC76), không qua `PUT` thành viên (AC31). `Member` returns it (`null` khi không có).
58. WHEN an in-law member that is the `other_parent_id` of one or more children is deleted (AC37), THE SYSTEM SHALL set those children's `other_parent_id` to `null` and SHALL NOT block the deletion.

## US-GEN-08: Biết mình ở đâu trong cây (theo số điện thoại)

## User Story

> Là **chủ gia phả hoặc người được mời xem**,
> Tôi muốn **app tự biết tôi là nút nào trong cây**,
> Để **hiển thị quan hệ của mọi người với tôi mà không phải tự chọn**.

> Là **chủ gia phả**,
> Tôi muốn **quyết định người được mời là nút nào trong cây**,
> Để **họ không tự gắn nhầm hoặc giả làm người khác**.

## Acceptance Criteria (EARS notation)

59. THE SYSTEM SHALL determine the caller's position in a genealogy **by phone number**: `my_member_id` = the `id` of the member of that genealogy whose `phone_number` equals the caller's own account phone number (cùng chuẩn hoá như register, AC8), hoặc `null` khi không có thành viên nào trùng. Áp dụng cho cả `owner` và `viewer`. Giá trị này **tính ra ở mỗi lần đọc**, không lưu, không có route nào ghi nó.
60. WHEN a member write (founder, spouse, `POST`, `PUT`) would make two members of the same genealogy have the same `phone_number`, THE SYSTEM SHALL return `409 MEMBER_PHONE_ALREADY_USED` with `details.field = "phone_number"` and SHALL NOT write. WHEN the same `phone_number` appears twice inside one `POST /v1/genealogies` body (founder + spouses), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field` = the later item's `phone_number`. Cùng một số có thể xuất hiện ở **gia phả khác** (một tài khoản vào hai gia phả). Hệ quả: một nút gắn tối đa một tài khoản trong một gia phả, và một tài khoản (một số) gắn tối đa một nút.
61. (Đã bỏ ở v5.0: route xoá con trỏ "tôi là ai" — người được mời không tự đổi vị trí.)
62. THE SYSTEM SHALL return `my_member_id` (`null` nếu không trùng số) in `GenealogyDetail` (AC14) and in `MemberProfile` (AC64). Người gọi chỉ thấy vị trí của **chính mình**; chỉ `owner` thấy vị trí của người xem (AC48).
63. WHEN a member is deleted, or its `phone_number` is changed or cleared (AC30), THE SYSTEM SHALL reflect that at the next read — `my_member_id` của tài khoản đó trở thành `null` hoặc nút mới. Không cần dọn dữ liệu riêng vì vị trí không được lưu.
    > ⚠️ Not confirmed: `my_member_id` là `null` khi owner xoá nút hoặc xoá/đổi số của nút sau khi người đó đã là viewer (lời mời chờ thì tự huỷ theo AC30) — viewer vẫn xem được nhưng không có xưng hô; owner điền lại số (AC30) để gắn lại. Spec chấp nhận; không chặn.

## US-GEN-09: Hồ sơ thành viên (thông tin tính ở server)

## User Story

> Là **chủ gia phả (`owner`)**,
> Tôi muốn **chọn một người và xem ngay tuổi, cha mẹ, vợ chồng, anh em, con cái, đường từ thuỷ tổ xuống, thuộc chi nào và người đó là gì của tôi**,
> Để **hiểu vị trí của người đó trong dòng họ mà không phải tự lần cây**.

## Acceptance Criteria (EARS notation)

64. WHEN a `GET /v1/genealogies/{id}/members/{memberId}/profile` request arrives from a user with a role AND the member belongs to that genealogy, THE SYSTEM SHALL return `200` with `MemberProfile`: `member` (full `Member`), `my_member_id`, `age`, `father`, `mother`, `spouses`, `siblings`, `children`, `lineage`, `chi`, `relation_to_me`. Mọi field trên (trừ `member`, `my_member_id`) **tính ra** từ cây tại thời điểm đọc, không lưu. WHEN the member does not belong to the genealogy, `404 MEMBER_NOT_FOUND`; WHEN the caller has no role, `404 GENEALOGY_NOT_FOUND`.
65. THE SYSTEM SHALL compute `age` as: WHEN `is_alive` and `birth_year` present → (năm hiện tại theo `Asia/Ho_Chi_Minh`) − `birth_year`; WHEN not alive and both `birth_year`, `death_year` present → `death_year − birth_year` (hưởng thọ); otherwise `null`. Chỉ có năm nên là số xấp xỉ theo năm dương lịch.
    > ⚠️ Not confirmed: tuổi hiển thị theo tuổi tây (như spec này) hay tuổi mụ (cộng 1) — không chặn.
66. THE SYSTEM SHALL return relatives as references `{id, full_name, gender, generation, is_alive}`: `father` / `mother` = người từ `parent_id` và `other_parent_id` phân theo `gender` (`null` khi không có; dâu/rể không có cha mẹ → cả hai `null`); `spouses` = các in-law gắn với người đó (hoặc, với một in-law, người dòng họ được gắn), ordered by `id`; `siblings` = các thành viên dòng họ **cùng `parent_id`** (trừ chính người đó) ordered by `birth_order` then `id`, mỗi mục kèm `is_half_sibling` (`true` khi hai người đều có `other_parent_id` và khác nhau); `children` = các thành viên có `parent_id` là người đó (hoặc, với một in-law, có `other_parent_id` là người đó) ordered by `birth_order` then `id`. Danh sách rỗng là `[]`.
    > ⚠️ Not confirmed: con mà `other_parent_id` là `null` không được liệt kê cho bất kỳ vợ/chồng nào của cha — chấp nhận hay yêu cầu chọn mẹ khi có nhiều vợ — không chặn.
67. THE SYSTEM SHALL return `lineage` as the chain from generation 1 down to the member: `[{id, full_name, generation}, ...]`, ordered by `generation` ASC, **gồm cả thuỷ tổ và chính người đó**. WHEN the member is an in-law, `lineage` is the lineage of the blood member it is attached to (không thêm chính in-law vào chuỗi).
68. THE SYSTEM SHALL return `chi` (chi hiện tại, trong cùng) and `chi_path` (từ ngoài vào trong) of the member, computed by walking from the founder down to the member (đường `parent_id`): a blood member `N` with parent `P` is a **chi head** WHEN `N.gender = male` AND `P` has **two or more male blood children**. `chi_path` = every chi head on that path, ordered by `generation` ASC, **gồm cả chính người đó nếu họ là đầu chi**; each item is `{ancestor, rank, is_eldest}` where `ancestor` = the chi head (`MemberRef`), `rank` = vị trí (1-based) của `ancestor` trong các con trai của `P` ordered by `birth_order` then `id`, `is_eldest = (rank = 1)` (chi trưởng; các chi còn lại là chi thứ `rank`). `chi` = phần tử cuối của `chi_path`; WHEN the path has no chi head (thuỷ tổ, vợ/chồng của thuỷ tổ, hoặc mọi tổ tiên chỉ có một con trai), `chi` is `null` and `chi_path` is `[]`. Con gái không phải đầu chi và thuộc chi hiện tại của cha. Với một in-law, `chi`/`chi_path` là của người dòng họ được gắn. Client hiển thị "chi <ancestor.full_name>" (kèm "thuộc chi <phần tử trước>" nếu `chi_path` có từ 2 phần tử). Ví dụ: thuỷ tổ có hai con trai Lộc, Nghĩa; Lộc có hai con trai A, B; người thuộc nhánh A có `chi_path = [Lộc (rank 1), A (rank 1)]`, `chi = A`.
    (Đã chốt 2026-10-07: rẽ nhánh chỉ đếm **con trai**, vì con gái đi lấy chồng nên con cháu thuộc họ khác.) Chi được tính từ dữ liệu hiện có: khi một người có thêm con trai thứ hai thì hậu duệ của người đó chuyển sang các chi con; client không nên cache nhãn chi lâu.
69. THE SYSTEM SHALL return `relation_to_me` = `{code, label}` describing **người được chọn là gì của "tôi"** (ví dụ ông nội, chú, anh, vợ) WHEN `my_member_id` is not `null`, and `null` WHEN `my_member_id` is `null`. `code` là enum ổn định (`SELF`, `FATHER`, `MOTHER`, `SPOUSE`, `CHILD`, `PATERNAL_GRANDFATHER`, ...); `label` là xưng hô tiếng Việt (null khi `code = UNKNOWN`). v1 hỗ trợ **bên nội** (theo `parent_id`) cộng quan hệ vợ/chồng/dâu/rể và mẹ qua `other_parent_id`; quan hệ không tính được (đặc biệt bên ngoại: cậu, dì, ông bà ngoại) trả `code = UNKNOWN`, `label = null`, không phải lỗi.
    Vai vế tính theo **vị trí trong cây (đời so với tổ tiên chung), không theo năm sinh**; `birth_order` chỉ dùng để so lớn/nhỏ giữa hai nhánh con trực tiếp của tổ tiên chung (xem `design.md` "Quy tắc vai vế"). Người nhỏ tuổi nhưng thuộc đời trên vẫn được gọi đúng vai. (Bảng xưng hô v1 đã được anh Đức duyệt 2026-10-10.)

## US-GEN-10: Sửa nhầm quan hệ của thành viên

## User Story

> Là **chủ gia phả (`owner`)**,
> Tôi muốn **chuyển một người sang cha/mẹ khác, hoặc đổi người mà dâu/rể được gắn**,
> Để **sửa lại khi nhập sai "con của ai" / "vợ chồng của ai" mà không phải xoá rồi thêm lại cả nhánh**.

## Acceptance Criteria (EARS notation)

76. WHEN a `PUT /v1/genealogies/{id}/members/{memberId}/relation` request arrives from the `owner`, with `relation` (`child` | `spouse`), `related_member_id` and optionally `other_parent_id` / `birth_order`, THE SYSTEM SHALL — trong **một transaction** — re-attach the member and return `200` with the full `Member` (cộng `warnings`, AC27). Với thành viên **dòng họ** (`relation = "child"`): `parent_id = related_member_id`, `generation = related.generation + 1`, **mọi hậu duệ và dâu/rể của họ** dời `generation` cùng độ lệch; `other_parent_id` theo AC57 với cha/mẹ mới (không gửi thì `null`); `birth_order` theo AC56 (không gửi và cùng cha/mẹ cũ thì giữ nguyên, khác cha/mẹ thì `max + 1`). Với **dâu/rể** (`relation = "spouse"`): `spouse_of_id = related_member_id`, `generation = related.generation`; các con có `other_parent_id` là người này được đặt về `null` (như AC58).
77. WHEN `relation` does not match the member's kind (dòng họ mà gửi `spouse`, dâu/rể mà gửi `child`), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "relation"` — không đổi loại thành viên. WHEN `related_member_id` is the member itself or one of its descendants, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` (chống vòng). Các quy tắc AC19 (người được gắn phải là thành viên **nam** dòng họ), AC20 (≤ 5 phối ngẫu, người được gắn không phải in-law) áp dụng cho người được gắn mới.
78. WHEN the move would make any descendant's generation greater than 60, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` (AC25) and SHALL change nothing.
79. WHEN the target is the founder, THE SYSTEM SHALL return `409 FOUNDER_NOT_MOVABLE`. WHEN `{memberId}` or `related_member_id` does not belong to the genealogy, `404 MEMBER_NOT_FOUND` (AC22, AC29).
80. WHEN the body has unknown fields, is empty or invalid JSON, or sets a server-owned field (AC11), THE SYSTEM SHALL return `400 VALIDATION_ERROR`. WHEN any step fails, THE SYSTEM SHALL leave every member unchanged. `chi`, `lineage`, `relation_to_me` chỉ tính lại ở lần đọc `profile` sau (AC64).
    > ⚠️ Not confirmed: có cần đổi **loại** thành viên (dòng họ ↔ dâu/rể) khi nhập nhầm không — hiện không, chỉ xoá rồi thêm lại (AC77) — không chặn.

## API contract (spec của feature này — chưa có trong `openapi.yaml`)

Mọi route đều `RequireAuth` (`authMiddleware.AuthenticateFunc`, `RequireTokenType: "access"`); `user_id` luôn lấy từ token. Vai trò được tính lại từ DB ở **mỗi** request (xem `design.md` Step A).

| Method | Path | Auth | Quyền | Success | Failure codes |
|---|---|---|---|---|---|
| POST | `/v1/genealogies` | RequireAuth | bất kỳ user | `201` `GenealogyDetail` | 400 `VALIDATION_ERROR`, 401, 409 `GENEALOGY_LIMIT_REACHED`/`MEMBER_PHONE_ALREADY_USED`, 405, 500 |
| GET | `/v1/genealogies` | RequireAuth | bất kỳ user | `200` `{items, total}` | 401, 405, 500 |
| GET | `/v1/genealogies/{id}` | RequireAuth | `owner`, `viewer` | `200` `GenealogyDetail` | 401, 404 `GENEALOGY_NOT_FOUND`, 405, 500 |
| PUT | `/v1/genealogies/{id}` | RequireAuth | `owner` | `200` `GenealogyDetail` | 400, 401, 403 `GENEALOGY_FORBIDDEN`, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}` | RequireAuth | `owner` | `200` `{"success": true}` | 401, 403, 404, 405, 500 |
| PUT | `/v1/genealogies/{id}/chronicle` | RequireAuth | `owner` | `200` `Chronicle` | 400, 401, 403, 404, 405, 500 |
| GET | `/v1/genealogies/{id}/members` | RequireAuth | `owner`, `viewer` | `200` `{items, total}` | 401, 404, 405, 500 |
| POST | `/v1/genealogies/{id}/members` | RequireAuth | `owner` | `201` `Member` (+ `warnings`) | 400, 401, 403, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 409 `MEMBER_LIMIT_REACHED`/`MEMBER_PHONE_ALREADY_USED`, 405, 500 |
| GET | `/v1/genealogies/{id}/members/{memberId}` | RequireAuth | `owner`, `viewer` | `200` `Member` | 401, 404, 405, 500 |
| PUT | `/v1/genealogies/{id}/members/{memberId}` | RequireAuth | `owner` | `200` `Member` (+ `warnings`) | 400, 401, 403, 404, 409 `MEMBER_PHONE_ALREADY_USED`, 405, 500 |
| DELETE | `/v1/genealogies/{id}/members/{memberId}` | RequireAuth | `owner` | `200` `{"success": true}` | 401, 403, 404, 409 `MEMBER_HAS_DESCENDANTS`/`FOUNDER_NOT_DELETABLE`, 405, 500 |
| PUT | `/v1/genealogies/{id}/members/{memberId}/relation` | RequireAuth | `owner` | `200` `Member` (+ `warnings`) | 400, 401, 403, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 409 `FOUNDER_NOT_MOVABLE`, 405, 500 |
| GET | `/v1/genealogies/{id}/members/{memberId}/profile` | RequireAuth | `owner`, `viewer` | `200` `MemberProfile` | 401, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 405, 500 |
| GET | `/v1/genealogies/{id}/viewers` | RequireAuth | `owner` | `200` `{items, total}` | 401, 403, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}/viewers/{userId}` | RequireAuth | `owner`, hoặc chính viewer đó | `200` `{"success": true}` | 401, 403, 404 `VIEWER_NOT_FOUND`, 405, 500 |
| POST | `/v1/genealogies/{id}/invitations` | RequireAuth | `owner` | `201` `Invitation` | 400, 401, 403, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 409 `MEMBER_PHONE_ALREADY_USED`/ `VIEWER_ALREADY_EXISTS`/`INVITATION_ALREADY_PENDING`/`VIEWER_LIMIT_REACHED`, 405, 500 |
| GET | `/v1/genealogies/{id}/invitations` | RequireAuth | `owner` | `200` `{items, total}` | 401, 403, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}/invitations/{invitationId}` | RequireAuth | `owner` | `200` `{"success": true}` | 401, 403, 404 `GENEALOGY_NOT_FOUND`/`INVITATION_NOT_FOUND`, 405, 500 |
| GET | `/v1/invitations` | RequireAuth | bất kỳ user | `200` `{items, total}` | 401, 405, 500 |
| POST | `/v1/invitations/{id}/accept` | RequireAuth | người được mời | `200` `{genealogy_id, member_id, role: "viewer"}` | 401, 404 `INVITATION_NOT_FOUND`, 409 `INVITATION_NOT_PENDING`/`VIEWER_ALREADY_EXISTS`/`VIEWER_LIMIT_REACHED`, 405, 500 |
| POST | `/v1/invitations/{id}/decline` | RequireAuth | người được mời | `200` `{"success": true}` | 401, 404 `INVITATION_NOT_FOUND`, 409 `INVITATION_NOT_PENDING`, 405, 500 |

`Member`:

| Field | Kiểu | Ghi chú |
|---|---|---|
| `id` | int | |
| `kind` | `"blood"` \| `"in_law"` | thuộc dòng họ / dâu-rể |
| `full_name` | string | ≤ 100 |
| `gender` | `"male"` \| `"female"` | |
| `birth_year` | int \| `null` | |
| `is_alive` | bool | |
| `death_year` | int \| `null` | chỉ khi `is_alive = false` |
| `residence` | string \| `null` | nơi ở hiện tại, ≤ 200 |
| `dharma_name` | string \| `null` | pháp danh, ≤ 100 |
| `tablet_location` | string \| `null` | linh vị đặt tại, chỉ khi đã mất, ≤ 200 |
| `burial_place` | string \| `null` | nơi an táng, chỉ khi đã mất, ≤ 200 (AC55) |
| `death_month`, `death_day` | int \| `null` | ngày giỗ **dương lịch** (năm là `death_year`), chỉ khi đã mất, cùng có hoặc cùng `null` (AC55) |
| `death_entered_as_lunar` | bool \| `null` | người dùng đã chọn theo âm lịch; `null` khi không có ngày giỗ |
| `death_lunar_day`, `death_lunar_month`, `death_lunar_leap` | int \| int \| bool, `null` | ngày giỗ **âm lịch tính ra** từ ngày dương đã lưu (AC55); không ghi được |
| `phone_number` | string \| `null` | điện thoại của người đó; duy nhất trong gia phả; trùng số tài khoản thì là nút của tài khoản đó (AC59, AC60) |
| `is_clan_head` | bool | cờ trưởng tộc, tối đa một nút/gia phả; chỉ blood, nam, còn sống (AC82) |
| `note` | string \| `null` | ≤ 1000 |
| `parent_id` | int \| `null` | `null` với thuỷ tổ và với dâu/rể |
| `other_parent_id` | int \| `null` | cha/mẹ còn lại của con: một in-law là vợ/chồng của `parent_id` (AC57) |
| `birth_order` | int \| `null` | thứ tự trong các con cùng `parent_id`; `null` với thuỷ tổ và dâu/rể (AC56) |
| `spouse_of_id` | int \| `null` | chỉ với dâu/rể |
| `generation` | int | đời (thuỷ tổ = 1); dâu/rể = đời của người được gắn |
| `in_law_label` | `"DAU"` \| `"RE"` \| `null` | tính ra (AC33) |
| `created_at`, `updated_at` | RFC 3339 | |

`POST`/`PUT` thành viên và `PUT .../relation` trả `Member` kèm `warnings`: `{code: "CHILD_BORN_BEFORE_PARENT", member_id, related_member_id}[]` (`[]` khi không có, AC27); `GET` không có `warnings`.

`GenealogyDetail`: `id`, `name`, `ancestral_home`, `my_role` (`owner`/`viewer`), `my_member_id` (int \| `null`, tính ra theo số điện thoại, AC59/AC62), `clan_head_member_id` (int \| `null`, AC82), `chronicle` (các field AC39 + `chap_lunar_day`, `chap_lunar_month`, `chap_lunar_leap` tính ra (AC40) + `founder_name`, `generations_count`, `members_count`), `created_at`, `updated_at`. Các `null` được trả tường minh, không bỏ key.

`MemberProfile` (AC64–AC69): tất cả tính ở server, không lưu.

| Field | Kiểu | Ghi chú |
|---|---|---|
| `member` | `Member` | đầy đủ như `GET .../members/{memberId}` |
| `my_member_id` | int \| `null` | nút có `phone_number` trùng số điện thoại của người gọi (AC59, AC62) |
| `age` | int \| `null` | AC65 |
| `father`, `mother` | `MemberRef` \| `null` | AC66 |
| `spouses` | `MemberRef[]` | AC66 |
| `siblings` | `(MemberRef + is_half_sibling)[]` | AC66 |
| `children` | `MemberRef[]` | AC66 |
| `lineage` | `{id, full_name, generation}[]` | từ đời 1 xuống người đó (AC67) |
| `chi` | `{ancestor: MemberRef, rank, is_eldest}` \| `null` | chi hiện tại (trong cùng), AC68 |
| `chi_path` | `{ancestor: MemberRef, rank, is_eldest}[]` | các chi từ ngoài vào trong, `[]` khi không có (AC68) |
| `relation_to_me` | `{code, label}` \| `null` | `null` khi chưa chọn "tôi là ai" (AC69) |

`MemberRef`: `{id, full_name, gender, generation, is_alive}`.

`Invitation` (phía owner, AC44/AC73): `{id, member_id, member_full_name, phone_number, status (pending|accepted|declined|expired), created_at, expires_at, responded_at | null}`. `MyInvitation` (phía người được mời, AC70): `{id, genealogy_name, member_full_name, invited_by_phone, created_at, expires_at}`. Body của `POST .../invitations`: `{member_id, phone_number?}`.

## Out of scope (this feature boundary, not tracked here)

- Ảnh (hoành phi, ảnh thành viên) — chặn bởi GCS, cùng nợ với `docs/tech-debt/memorial/TD-001`.
- Nhắc giỗ hằng năm và sổ giỗ chi tiết — thuộc `memorial`; gia phả chỉ lưu ngày giỗ (dương lịch + cờ, AC55) ngay trên thành viên và **không** liên kết/ghi vào `memorial` (chốt 2026-10-10).
- Push notification từ server (xem `memorial`); nhắc ngày chạp/ngày giỗ hằng năm (xem `memorial`) — server chỉ quy đổi ngày đã lưu ra âm lịch (AC40, AC55), không tính ngày dương của năm sau.
- Chuyển quyền `owner` cho người khác (chốt 2026-10-10: không có).
- Thông báo (push notification) báo lời mời và các thông báo khác — **tính năng riêng của ứng dụng, làm sau** (chốt 2026-10-10); không dùng SMS (tốn phí). v1 chỉ có hộp lời mời `GET /v1/invitations` (AC70). Hệ quả: người chưa có tài khoản chỉ thấy lời mời sau khi đăng ký bằng đúng số đó (AC75); owner tự báo họ ngoài ứng dụng.
- Xưng hô **bên ngoại** (ông bà ngoại, cậu, dì): dâu/rể hiện không có cha mẹ trong cây nên chưa tính được (`code = UNKNOWN`, AC69).
- Người được mời tự chọn/đổi "tôi là ai" (chốt 2026-10-10: không cho); gắn một tài khoản vào nhiều nút trong cùng một gia phả.
- Vai trò `editor` (bỏ ở v5.0; trưởng tộc chỉ còn là cờ, AC82), link công khai, xuất PDF/in cây, tìm kiếm/lọc thành viên, lịch sử chỉnh sửa (audit), hoàn tác.
- Tự gộp/nhập khẩu hai gia phả, GEDCOM.

## Pending (not yet implemented)

- Toàn bộ feature chưa implement — xem `tasks.md` Phase 1–5.
- Các marker `⚠️ Not confirmed` còn mở: AC56, AC63 (xoá nút/số sau khi đã là viewer), AC65, AC66, AC72, AC75 (số điện thoại bị cấp lại), AC80 (đổi loại thành viên). Đã chốt 2026-10-10: AC82 (trưởng tộc là cờ), lời mời gắn thành viên (AC44), AC10, 19, 24, 25, 27, 31, 35, 39, 40 (lưu dương + cờ, server quy đổi), 47, 51 (cấp thẳng), 54 (không chuyển owner), 69 (bảng xưng hô v1), 55 (ngày giỗ lưu trên thành viên, không vào `memorial`), 75 (hạn 30 ngày), thuỷ tổ phải là nam. Không còn marker nào chặn task. Các marker còn lại đã có mặc định trong spec nên không chặn.
- Sửa đồng thời: hai người sửa cùng thành viên một lúc thì lần ghi sau thắng (không có kiểm phiên bản).
  > ⚠️ Not confirmed: có cần chống ghi đè đồng thời (`updated_at`/`version` trong body `PUT`, trả `409`) không — vì nhiều người cùng sửa; hiện là last-write-wins.

## Ghi chú kỹ thuật liên quan

- **Quyền tính từ DB mỗi request**, không đưa vào JWT: grant/revoke có hiệu lực ngay (AC54). Một truy vấn duy nhất trả về vai trò (owner, viewer hoặc không có) — xem `design.md` Step A.
- **404 đồng nhất** cho "không tồn tại" và "không có quyền" (AC15) vì id tự tăng dễ đoán; 403 chỉ dùng khi người gọi đã có vai trò nhưng thiếu quyền (owner-only).
- **Dữ liệu cá nhân**: `phone_number`, `residence` của người còn sống, `note` — không log các field này; response chỉ cho người có vai trò.
- **Nhãn dâu/rể** tính ra từ giới tính người được gắn, không lưu — đổi giới tính không làm sai dữ liệu. `generation` thì lưu (phi chuẩn hoá) để list/stat không phải đệ quy; đổi quan hệ (AC76) tính lại `generation` của cả nhánh trong cùng transaction nên không lệch.
- **Cây lưu theo adjacency list** (`parent_id`) vì tối đa 5000 thành viên/60 đời và API trả danh sách phẳng cho client dựng cây; không cần `ltree`/closure table. Kiểm tra "có con" bằng một `EXISTS` trên `parent_id`.
- **Cùng gia phả được ép ở DB**: khoá ngoại ghép `(parent_id, genealogy_id)` → `(id, genealogy_id)` nên không thể gắn cha/mẹ hay vợ/chồng từ gia phả khác dù usecase quên kiểm.
- Hình dạng gần nhất để theo: `memorial` (user-scoped, 404 đồng nhất, trần số lượng kiểm trong transaction có khoá) cộng thêm lớp phân quyền theo vai trò — chưa feature nào có.
- **Vì sao `profile` tính ở server**: có nhiều client (iOS, Android, web), quy tắc xưng hô tiếng Việt phức tạp; để một chỗ (hàm thuần trong usecase, có bảng test) tránh lệch. Cả cây tối đa 5000 thành viên nên một truy vấn tải hết rồi tính trong bộ nhớ là đủ nhanh, không cần đệ quy SQL.
- **`other_parent_id` thay cho "con gắn cả cha lẫn mẹ"**: `parent_id` vẫn là đúng một cha/mẹ thuộc dòng họ; `other_parent_id` chỉ trỏ tới một vợ/chồng của người đó để biết con của bà nào. Nhờ vậy tách được anh em cùng cha khác mẹ và gọi đúng "mẹ".
- **Vị trí "tôi" tính ra theo số điện thoại, không lưu**: một tài khoản có đúng một số điện thoại, một thành viên có tối đa một `phone_number`, nên `my_member_id` chỉ là một phép so khớp lúc đọc; ràng buộc duy nhất `(genealogy_id, phone_number)` ở DB đảm bảo một nút gắn tối đa một tài khoản. Cùng số có thể ở gia phả khác. Dữ liệu thành viên chứa số điện thoại thật nên vẫn chỉ owner/viewer của gia phả đó thấy.
- Cần xác nhận khi implement: repo `users` đã có hàm tìm user theo `phone_number` chưa (`grep -n phone_number services/appService/repository/`); nếu chưa thì thêm một method vào port tối thiểu (`FindIDByPhone`), không mở rộng port hiện có tuỳ tiện.

## Open questions (resolved during implementation)

- ~~Ai sở hữu và ai sửa được gia phả?~~ → Nhiều người: `owner` (người tạo), `clan_head` (trưởng tộc), `editor` (được cấp quyền); cả ba xem + sửa. Resolved 2026-10-07.
- ~~Trưởng tộc xác định thế nào?~~ → Owner chỉ định một user, đổi được; không bắt buộc là node trong cây. Resolved 2026-10-07.
- ~~Cấp quyền bằng cách nào?~~ → Nhập số điện thoại của user đã có tài khoản. Resolved 2026-10-07.
- ~~Người ngoài có xem được không?~~ → Không; chỉ ba vai trò trên, không có viewer. Resolved 2026-10-07.
- ~~"Con của ai" chọn thế nào?~~ → Chọn đúng 1 cha/mẹ ruột trong cây. Resolved 2026-10-07.
- ~~Quy tắc vợ chồng?~~ → Nhiều vợ/chồng; người ngoài họ ghi dâu/rể, không có đời riêng. Resolved 2026-10-07.
- ~~Năm sinh/mất?~~ → Chỉ lưu năm, dương lịch, năm mất tuỳ chọn. Resolved 2026-10-07.
- ~~Ai chỉ định/đổi trưởng tộc, trưởng tộc có quản lý quyền không?~~ → Chỉ `owner` chỉ định/đổi; trưởng tộc không quản lý quyền (AC48, AC51–AC53). Không đổi `design.md`/`tasks.md` (đúng như đã thiết kế). Resolved 2026-10-07.
- ~~Làm sao biết mình ở đâu trong cây?~~ → Con trỏ "tôi là ai" riêng tư, client đặt qua API (AC59–AC63); thêm bảng `genealogy_user_members`. Resolved 2026-10-07.
- ~~Tính thông tin hiển thị ở đâu?~~ → Ở server, route `profile` (AC64–AC69). Resolved 2026-10-07.
- ~~"Chi" định nghĩa thế nào?~~ → Chi bắt đầu từ chỗ cây rẽ nhánh (một người có từ 2 con trai) và **lồng nhau** (chi Lộc tách thành chi A, B); chọn người ở nhánh A là chi A (AC68). Bản đầu trong ngày hiểu nhầm là "tổ tiên đời 2" — đã sửa theo ý anh Đức. Resolved 2026-10-07.
- ~~Rẽ nhánh để chia chi đếm con trai hay cả con gái?~~ → Chỉ con trai (AC68). Không đổi `design.md` (thuật toán đã đếm con trai), `flow.md`, `tasks.md`. Resolved 2026-10-07.
- ~~Con gắn cha hay mẹ khi cặp vợ/chồng có con chung?~~ → Con gắn đúng một cha/mẹ thuộc dòng họ (`parent_id`) và tuỳ chọn ghi người còn lại bằng `other_parent_id` (AC57). Resolved 2026-10-07.
- ~~Cấp quyền sửa có hiệu lực ngay hay phải chấp nhận? Số chưa đăng ký xử lý thế nào (AC44/AC45)?~~ → Mọi lần cấp quyền editor đều qua **lời mời**: số đã hay chưa đăng ký đều tạo lời mời chờ và trả cùng một response (không còn `USER_NOT_FOUND`, không lộ số đã đăng ký); người được mời xem lời mời ở `GET /v1/invitations` rồi chấp nhận hoặc từ chối (AC44–AC47, AC70–AC75). Thay giả định A-001. Đổi `design.md` (Step C-1/C-2 viết lại, thêm Step I-1/I-2, bảng `genealogy_invitations`, thêm 5 route), `flow.md` (sơ đồ C viết lại, thêm sơ đồ chấp nhận), `tasks.md` (mục mới ở Phase 1–5). Resolved 2026-10-08.
- ~~Chỉ cho xem hay cho sửa (editor/trưởng tộc)? Gắn tài khoản vào nút thế nào?~~ → **Chỉ còn `owner` (sửa) và `viewer` (chỉ xem)**; bỏ `editor` và `clan_head` (AC51–AC52 bỏ). "Tôi ở đâu" **tính theo `phone_number`** của thành viên trùng số tài khoản (AC59), duy nhất trong gia phả (AC60), người được mời không đổi được; bỏ route `me/member` (AC59–AC63 viết lại). Đổi tên route `collaborators` → `viewers` và mời qua `POST .../invitations`. Đổi `design.md` (Step A, schema bỏ `clan_head_user_id`/`genealogy_user_members`, thêm unique `(genealogy_id, phone_number)`), `flow.md`, `tasks.md`. Resolved 2026-10-10.
- ~~Trưởng tộc còn không? Mời người vào thế nào?~~ → Trưởng tộc chỉ là cờ `is_clan_head` trên thành viên blood, một gia phả một người (AC82). Mời: owner chọn một thành viên rồi mời theo số điện thoại; lời mời chờ, người được mời chấp nhận/từ chối ở hộp thông báo, trạng thái cập nhật cho owner (AC44, AC70–AC73). Đổi `design.md` (schema `is_clan_head`, `genealogy_invitations.member_id`, Step C-1/C-2, G-2, M-2), `flow.md`, `tasks.md`. Resolved 2026-10-10.
- ~~Trưởng tộc phải thế nào? Không mời người đã mất? Xoá số của nút thì lời mời? Thông báo?~~ → Trưởng tộc là nam, còn sống, blood (AC82, ép cả ở DB). Không mời người đã mất (AC44). Sửa/xoá số của nút thì lời mời chờ tự huỷ (AC30). Thông báo/push là tính năng riêng làm sau, không dùng SMS (marker AC70 gỡ). Đổi `design.md` (CHECK, validation), `tasks.md`. Resolved 2026-10-10.
- ~~Giới hạn số (AC10/24/25/47) và hạn lời mời (AC75)?~~ → 10 gia phả/người, 5000 thành viên, 60 đời, 20 editor (tính cả lời mời chờ), 30 ngày. Bỏ cờ "đề xuất" ở `design.md`; không đổi cấu trúc. Resolved 2026-10-10.
- ~~Con của thành viên nữ dòng họ (AC19)?~~ → Không thêm vào cây; thuỷ tổ phải là nam (AC4, AC30). Đổi `design.md` (check giới tính ở Step M-2), `tasks.md`. Resolved 2026-10-10.
- ~~Xoá người có hậu duệ (AC35)?~~ → Chặn (409), xoá từ đời dưới lên; không xoá cả nhánh. Không đổi `design.md`/`flow.md`/`tasks.md` ngoài việc bỏ marker. Resolved 2026-10-10.
- ~~Năm sinh con nhỏ hơn cha (AC27)?~~ → Cảnh báo, không chặn: `warnings` trong response ghi. Đổi `design.md` Step M-3, `flow.md`, `tasks.md`. Resolved 2026-10-10.
- ~~Đổi quan hệ khi nhập sai (AC31)?~~ → Có, route `PUT .../members/{memberId}/relation` (AC76–AC80), thêm Step R-1..R-3. Resolved 2026-10-10.
- ~~Phiên âm/nghĩa gia huấn (AC39)?~~ → Cho cả hoành phi lẫn hai vế câu đối. Thêm 4 cột `couplet_*_reading/meaning`. Resolved 2026-10-10.
- ~~Ngày chạp họ/ngày giỗ âm hay dương (AC40, AC55)?~~ → Lưu dương lịch một loại, kèm cờ `*_entered_as_lunar`; thêm `chap_year`, bỏ `chap_leap_month`, đổi `death_lunar_*` → `death_month/death_day/death_entered_as_lunar`; **server quy đổi và trả** `chap_lunar_*`/`death_lunar_*` tính ra (AC40, AC55). Thêm hàm thuần quy đổi ở usecase, field tính ra ở response. Resolved 2026-10-10.
- ~~Chỉ định trưởng tộc cấp thẳng hay qua lời mời (AC51)?~~ → Cấp thẳng (như spec). Không đổi `design.md`/`flow.md`/`tasks.md`. Resolved 2026-10-10.
- ~~Ngày giỗ lưu trên thành viên hay tham chiếu `memorial` (AC55)?~~ → Lưu ngay trên thành viên, không ghi vào `memorial`. Không đổi `design.md`/`flow.md`/`tasks.md` ngoài việc bỏ ghi chú "chờ chốt". Resolved 2026-10-10.
- ~~Bảng xưng hô v1 (AC69)?~~ → Đã duyệt bảng đề xuất; vai vế theo đời không theo tuổi, `birth_order` chỉ so giữa hai nhánh con của tổ tiên chung; thêm quy tắc "họ", "ông bác/ông chú", "chắt/chút" cho các ca xa. Đổi `design.md` (bảng + Quy tắc vai vế), `flow.md` (bỏ marker), `tasks.md` (bỏ chặn). Resolved 2026-10-10.
- ~~Chuyển quyền `owner` (AC54)?~~ → Không có. Không đổi `design.md`/`flow.md`/`tasks.md`. Resolved 2026-10-10.
