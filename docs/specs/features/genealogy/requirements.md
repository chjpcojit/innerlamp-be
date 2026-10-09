# Requirements — genealogy

Version: 3.0
Last updated: 2026-10-08
Status: not started — spec viết prospectively, chưa có code/migration. Còn nhiều marker `⚠️ Not confirmed` (xem "Pending"); chỉ marker ghi rõ "chặn" mới chặn task tương ứng.
Module: `genealogy`
Code: `services/appService/`

> See `changelog.md` in this folder for the version history of this spec.

Spec này được viết **prospectively** (trước khi có code) cho tính năng **Gia phả**: người dùng đã đăng nhập tạo một gia phả (cây dòng họ + phả ký), thêm/sửa các thành viên, và mời người khác trong họ cùng sửa (người được mời chấp nhận hoặc từ chối). Khác `memorial` (riêng tư tuyệt đối theo user), gia phả **nhiều người cùng sửa** nhưng **không công khai**: dữ liệu chứa số điện thoại, nơi ở của người còn sống.

## Quyết định nền tảng (đã chốt với anh Đức, 2026-10-07)

- **Phân quyền ba vai trò, không có viewer.** `owner` (người tạo), `clan_head` (trưởng tộc — do owner chỉ định, đổi được), `editor` (người được cấp quyền). Cả ba **xem và sửa** cây + phả ký. Người không thuộc ba vai trò **không xem được** gì (kể cả sự tồn tại của gia phả).
- **Cấp quyền sửa bằng lời mời theo số điện thoại** (chốt 2026-10-08, thay cho cấp thẳng của v2.x): owner nhập số điện thoại; dù số đó **đã hoặc chưa** có tài khoản, hệ thống đều tạo một lời mời chờ và trả cùng một kết quả. Người được mời thấy lời mời trong app (hộp lời mời), rồi **chấp nhận** (thành `editor`) hoặc **từ chối**. Không có mã mời/link.
- **Trưởng tộc là một vai trò gắn với user**, không bắt buộc trùng một người trong cây gia phả.
- **Con của ai = chọn đúng một cha/mẹ ruột** trong cây (người đó có thể có vợ/chồng kèm theo). Cây hiển thị theo dòng họ.
- **Nhiều vợ/chồng.** Một thành viên có thể có nhiều người phối ngẫu (tái hôn, đa thê thời xưa). Người phối ngẫu không thuộc dòng họ được ghi là **dâu** (vợ của nam) hoặc **rể** (chồng của nữ) và không có đời riêng.
- **Năm sinh/năm mất chỉ lưu năm, dương lịch**; năm mất là tuỳ chọn khi đã mất. Ngày giỗ chi tiết thuộc về `memorial`, không nhân đôi ở đây.
- **Chỉ `owner` chỉ định/đổi trưởng tộc** (chốt 2026-10-07, bổ sung): `clan_head` và `editor` không đổi được trưởng tộc, cũng không quản lý quyền. Trưởng tộc vẫn là vai trò gắn với user (không phải nút trong cây).
- **"Tôi là ai" do client đặt qua API** (chốt 2026-10-07): mỗi user trong mỗi gia phả có thể chọn **một** thành viên là chính mình; con trỏ này **riêng tư** (chỉ ảnh hưởng cách chính user đó nhìn cây, không cần ai duyệt). Chưa chọn thì hồ sơ thành viên không có xưng hô.
- **Thông tin hiển thị khi chọn một người được tính ở server** (chốt 2026-10-07): tuổi, cha/mẹ/vợ chồng/anh em/con, đường từ thuỷ tổ, chi, xưng hô với "tôi" — một route `profile` trả sẵn, client không tự tính (nhiều client, tránh lệch quy tắc xưng hô).
- **"Chi" bắt đầu từ chỗ cây rẽ nhánh và lồng nhau** (chốt 2026-10-07, thay cho cách hiểu "đời 2" ở bản đầu): mỗi lần một người có từ 2 con trai trở lên thì mỗi người con trai là đầu một chi. Ví dụ thuỷ tổ có hai con là cụ Lộc và cụ Nghĩa (chi Lộc, chi Nghĩa); chi Lộc đi tiếp rồi cụ Lộc có hai con A và B thì tách thành chi A và chi B. Người thuộc chi A có đường chi `[Lộc, A]`, chi hiện tại là A.
- Mã user story `US-GEN-01..09` do spec tự chọn (theo tiền lệ `US-MEM-*`).

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
4. WHEN `founder.full_name` is blank or longer than 100 characters, or `founder.gender` is not `male` or `female`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field` = `founder.full_name` / `founder.gender`.
5. WHEN `founder.is_alive` is absent, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "founder.is_alive"`.
6. WHEN `is_alive` is `true` AND `death_year` or `tablet_location` (linh vị đặt tại) is present, THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming that field — chỉ người đã mất mới có năm mất và nơi đặt linh vị.
7. WHEN `birth_year` or `death_year` is present, THE SYSTEM SHALL require an integer in `1000..<current year in Asia/Ho_Chi_Minh>`, and `death_year >= birth_year` when both are present; otherwise it SHALL return `400 VALIDATION_ERROR` naming the field. Cả hai năm đều tuỳ chọn.
8. WHEN `residence` > 200, `dharma_name` > 100, `tablet_location` > 200, `note` > 1000 characters, or `phone_number` is present but not a valid phone number (cùng quy tắc chuẩn hoá số điện thoại của register), THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming the field. Áp dụng cho **mọi** nơi nhận thông tin một thành viên (founder, spouses, thêm/sửa thành viên).
9. WHEN `spouses` has more than 5 items, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "spouses"`. Mỗi spouse có cùng các field như thành viên (AC4–AC8, trừ việc `gender` là tuỳ chọn): WHEN `spouses[i].gender` is absent, THE SYSTEM SHALL set it to the opposite of the founder's. WHEN present it is stored as given.
10. WHEN the user already owns 10 genealogies, THE SYSTEM SHALL return `409 GENEALOGY_LIMIT_REACHED` and SHALL NOT insert — kể cả khi hai request tạo chạy đồng thời.
    > ⚠️ Not confirmed: giới hạn 10 gia phả/người tạo là con số đề xuất — chưa chốt, không chặn việc làm khác.
11. WHEN the body has unknown fields, is empty, or is not valid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (`DecodeJSONBodyStrict`). WHEN the body sets a server-owned field (`id`, `owner_user_id`, `generation`, `kind`, `parent_id`, `spouse_of_id`, `clan_head`), THE SYSTEM SHALL reject it as unknown.
12. WHEN any step of AC1 fails, THE SYSTEM SHALL leave no genealogy, chronicle or member row behind.

## US-GEN-02: Xem, đổi thông tin chung, xoá gia phả

## User Story

> Là một **người có quyền trong gia phả (chủ, trưởng tộc hoặc người được cấp quyền)**,
> Tôi muốn **xem danh sách gia phả của mình, xem tổng quan một gia phả, đổi tên/quê gốc; chủ thì có thể xoá cả gia phả**,
> Để **quản lý gia phả của dòng họ**.

## Acceptance Criteria (EARS notation)

13. WHEN a `GET /v1/genealogies` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding every genealogy where the token's user is `owner`, `clan_head` or `editor` — each with `id`, `name`, `ancestral_home`, `my_role` — ordered by `created_at` DESC then `id` DESC, and `data.total`. WHEN there are none, `items` SHALL be `[]`.
14. WHEN a `GET /v1/genealogies/{id}` request arrives AND the user has a role in it, THE SYSTEM SHALL return `200` with `GenealogyDetail`: thông tin chung, `my_role`, `clan_head` (hoặc `null`), `chronicle` (US-GEN-05) và `stats` (`generations_count`, `members_count`) tính ra từ cây tại thời điểm đọc, không lưu.
15. WHEN `{id}` does not exist, is not a positive integer, OR the user has no role in it, THE SYSTEM SHALL return the **same** `404 GENEALOGY_NOT_FOUND` for all three — không lộ sự tồn tại của gia phả. Quy tắc này áp cho **mọi** route con của `/v1/genealogies/{id}/...` (members, chronicle, collaborators, clan-head).
16. WHEN a `PUT /v1/genealogies/{id}` request arrives from a user with a role AND `name`/`ancestral_home` are valid (AC3), THE SYSTEM SHALL update them and return `200` with `GenealogyDetail`. Cả ba vai trò đều sửa được.
17. WHEN a `DELETE /v1/genealogies/{id}` request arrives from the `owner`, THE SYSTEM SHALL delete the genealogy and, by cascade, its chronicle, members and collaborator grants (xoá cứng), and return `200` `{"success": true}`. WHEN the caller has a role other than `owner`, THE SYSTEM SHALL return `403 GENEALOGY_FORBIDDEN` and delete nothing.

## US-GEN-03: Thêm thành viên vào gia phả

## User Story

> Là một **người có quyền trong gia phả**,
> Tôi muốn **thêm một người mới là con của ai hoặc vợ/chồng của ai, cùng họ tên, giới tính, tình trạng còn sống/mất, nơi ở, số điện thoại, ghi chú**,
> Để **cây gia phả đầy đủ dần**.

## Acceptance Criteria (EARS notation)

18. WHEN a `POST /v1/genealogies/{id}/members` request arrives from a user with a role, with `relation` (`child` | `spouse`), `related_member_id` and a valid member body, THE SYSTEM SHALL create the member and return `201` with the full `Member`.
19. WHEN `relation = "child"`, THE SYSTEM SHALL create a **blood** member whose `parent_id = related_member_id` and `generation = parent.generation + 1`. WHEN the related member is an in-law (dâu/rể), THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` — con luôn gắn vào người thuộc dòng họ.
    > ⚠️ Not confirmed: con của một thành viên **nữ** thuộc dòng họ (con gái) có được thêm vào cây không, hay chỉ con của nam — chặn quyết định thêm check giới tính ở usecase (hiện spec không giới hạn theo giới tính).
20. WHEN `relation = "spouse"`, THE SYSTEM SHALL create an **in-law** member with `spouse_of_id = related_member_id` and the same `generation` as that member. WHEN the related member is itself an in-law, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"`. WHEN `gender` is absent, THE SYSTEM SHALL default it to the opposite of the related member's. Một thành viên có tối đa 5 người phối ngẫu; vượt quá → `400 VALIDATION_ERROR` với `details.field = "related_member_id"`.
21. WHEN `relation` is neither `child` nor `spouse`, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "relation"`.
22. WHEN `related_member_id` does not exist in this genealogy (kể cả khi tồn tại ở gia phả khác), THE SYSTEM SHALL return `404 MEMBER_NOT_FOUND` — không lộ thành viên của gia phả khác.
23. WHEN the member body violates AC4–AC8, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with the same `details.field` (không có prefix `founder.`).
24. WHEN the genealogy already has 5000 members, THE SYSTEM SHALL return `409 MEMBER_LIMIT_REACHED` and SHALL NOT insert — kể cả khi hai request thêm chạy đồng thời.
    > ⚠️ Not confirmed: giới hạn 5000 thành viên/gia phả là con số đề xuất — chưa chốt.
25. WHEN the child would be generation greater than 60, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "related_member_id"` (chặn cây sâu bất thường).
    > ⚠️ Not confirmed: trần 60 đời là con số đề xuất — chưa chốt.
26. WHEN the body has unknown fields, is empty or invalid JSON, or sets a server-owned field (AC11), THE SYSTEM SHALL return `400 VALIDATION_ERROR`.
27. THE SYSTEM SHALL NOT validate consistency of years between different people (con sinh trước cha, vợ chồng chênh tuổi) — chỉ validate từng người riêng lẻ (AC7).
    > ⚠️ Not confirmed: có cần cảnh báo/chặn năm sinh con nhỏ hơn năm sinh cha không — hiện không kiểm, không chặn gì.

## US-GEN-04: Xem, sửa, xoá thành viên

## User Story

> Là một **người có quyền trong gia phả**,
> Tôi muốn **xem danh sách thành viên và sửa lại thông tin bất kỳ ai (tên, giới tính, năm sinh, còn sống/mất, nơi ở, pháp danh, linh vị đặt tại...)**, và **xoá người ghi nhầm**,
> Để **gia phả luôn chính xác**.

## Acceptance Criteria (EARS notation)

28. WHEN a `GET /v1/genealogies/{id}/members` request arrives from a user with a role, THE SYSTEM SHALL return `200` with `data.items` holding **every** member of that genealogy (dạng phẳng, client tự dựng cây từ `parent_id`/`spouse_of_id`) ordered by `generation` ASC then `id` ASC, and `data.total`. Không phân trang (tối đa 5000).
29. WHEN a `GET /v1/genealogies/{id}/members/{memberId}` request arrives AND the member belongs to that genealogy, THE SYSTEM SHALL return `200` with the full `Member`; otherwise `404 MEMBER_NOT_FOUND` (cùng mã cho "không tồn tại" và "thuộc gia phả khác").
30. WHEN a `PUT /v1/genealogies/{id}/members/{memberId}` request arrives from a user with a role AND the body is valid, THE SYSTEM SHALL replace the editable fields — `full_name`, `gender`, `birth_year`, `is_alive`, `death_year`, `residence`, `dharma_name`, `tablet_location`, `burial_place`, `death_lunar_day`, `death_lunar_month`, `death_lunar_leap`, `phone_number`, `note`, và `birth_order` (chỉ với thành viên dòng họ không phải thuỷ tổ, AC56) — and return `200` with the full `Member`. Validation theo AC4–AC8 và AC55–AC56.
31. WHEN the body of `PUT` carries `relation`, `related_member_id`, `parent_id`, `other_parent_id`, `spouse_of_id`, `generation`, `kind` or any server-owned field, THE SYSTEM SHALL reject it as an unknown field — **không** đổi cha/mẹ hay đổi quan hệ qua route sửa.
    > ⚠️ Not confirmed: có cần chuyển một người sang cha/mẹ khác (sửa nhầm "con của ai") không — hiện chỉ có cách xoá rồi thêm lại; nếu cần thì thêm route riêng ở bản sau.
32. WHEN `gender` of a blood member is changed, THE SYSTEM SHALL keep the in-law members attached to it unchanged; nhãn dâu/rể của họ là **giá trị tính ra** từ giới tính của người được gắn (AC33), không lưu.
33. THE SYSTEM SHALL return `in_law_label` on every in-law member: `DAU` when the attached blood member is male, `RE` when female; and `null` for blood members.
34. WHEN a `DELETE /v1/genealogies/{id}/members/{memberId}` request arrives from a user with a role AND the member is a blood member with **no children**, THE SYSTEM SHALL delete the member **and every in-law attached to it** in one transaction and return `200` `{"success": true}`.
35. WHEN the target is a blood member that has at least one child, THE SYSTEM SHALL return `409 MEMBER_HAS_DESCENDANTS` and SHALL NOT delete anything — phải xoá từ đời dưới lên.
    > ⚠️ Not confirmed: xoá một người có hậu duệ nên bị chặn (như spec này) hay xoá cả nhánh bên dưới — spec đang chặn vì an toàn dữ liệu.
36. WHEN the target is the founder (đời 1, không có cha mẹ), THE SYSTEM SHALL return `409 FOUNDER_NOT_DELETABLE` — muốn bỏ cả cây thì xoá gia phả (AC17).
37. WHEN the target is an in-law, THE SYSTEM SHALL delete just that member and return `200`; WHEN `{memberId}` does not belong to the genealogy, `404 MEMBER_NOT_FOUND`.

## US-GEN-05: Phả ký của dòng họ

## User Story

> Là một **người có quyền trong gia phả**,
> Tôi muốn **ghi và xem phả ký: họ gì, ở đâu, từ năm nào, nguồn gốc, lịch sử, nhà thờ họ, ngày chạp họ và gia huấn**,
> Để **lưu giữ thông tin chung của dòng họ**.

## Acceptance Criteria (EARS notation)

38. WHEN a `PUT /v1/genealogies/{id}/chronicle` request arrives from a user with a role AND the body is valid, THE SYSTEM SHALL replace **all** chronicle fields with the body (field vắng/`null` = xoá) and return `200` with the `Chronicle`. Phả ký được tạo rỗng cùng gia phả (AC1) nên route này chỉ ghi đè, không có bước "tạo phả ký".
39. THE SYSTEM SHALL accept these writable fields, all optional: `surname` (họ, ≤ 50), `current_location` (ở đâu, ≤ 200), `since_year` (từ năm nào, `1000..current year`), `origin` (nguồn gốc, ≤ 5000), `history` (lịch sử dòng họ, ≤ 20000), `hall_name` (nhà thờ họ, ≤ 200), `hall_address` (địa chỉ, ≤ 300), `chap_day` + `chap_month` + `chap_leap_month` (ngày chạp họ), and the gia huấn block `hoanh_phi` (≤ 200), `hoanh_phi_reading` (phiên âm, ≤ 200), `hoanh_phi_meaning` (nghĩa, ≤ 1000), `couplet_upper` (câu đối vế trên, ≤ 500), `couplet_lower` (câu đối vế dưới, ≤ 500), `teaching` (lời dạy, ≤ 5000). Vi phạm độ dài/range → `400 VALIDATION_ERROR` với `details.field` tương ứng.
    > ⚠️ Not confirmed: "phiên âm, nghĩa" của gia huấn chỉ áp cho hoành phi hay cho cả hai vế câu đối — hiện chỉ hoành phi có `*_reading`/`*_meaning`, câu đối là một ô văn bản mỗi vế.
40. WHEN `chap_day` and `chap_month` are present, THE SYSTEM SHALL require `chap_month` in `1..12` and `chap_day` in `1..30`, and default `chap_leap_month` to `false`; WHEN only one of `chap_day`/`chap_month` is present, or `chap_leap_month` is `true` without both, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "chap_day"` (hoặc `chap_month`/`chap_leap_month`). Giá trị luôn là **âm lịch**; server không quy đổi, client tự hiển thị/nhắc.
    > ⚠️ Not confirmed: ngày chạp họ chỉ cần âm lịch (như spec này) hay cần lựa chọn dương lịch như `memorial.death_calendar`.
41. WHEN `GenealogyDetail` is read (AC14), THE SYSTEM SHALL include the chronicle's **derived** fields: `founder_name` (họ tên thuỷ tổ, lấy từ thành viên đời 1), `generations_count` (đời lớn nhất đang có) and `members_count` (tổng số thành viên, **gồm cả dâu/rể**). Chúng không ghi được, đưa vào body `PUT` → rejected as unknown field.
42. WHEN the body has unknown fields, is empty or invalid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR`. WHEN the genealogy is not visible to the user, `404 GENEALOGY_NOT_FOUND` (AC15).
43. THE SYSTEM SHALL NOT store images (ảnh hoành phi, ảnh thành viên) — nằm ngoài v1 (xem Out of scope).

## US-GEN-06: Mời người cùng sửa và chỉ định trưởng tộc

## User Story

> Là **người tạo gia phả**,
> Tôi muốn **mời người khác trong họ cùng sửa bằng số điện thoại và chỉ định trưởng tộc**,
> Để **cả họ cùng chung tay hoàn thiện gia phả**.

> Là **người được mời**,
> Tôi muốn **nhận lời mời, xem gia phả nào mời mình rồi chấp nhận hoặc từ chối**,
> Để **chỉ tham gia gia phả mà tôi đồng ý**.

## Acceptance Criteria (EARS notation)

44. WHEN a `POST /v1/genealogies/{id}/collaborators` request arrives from the `owner` with a valid `phone_number` (chuẩn hoá như register), THE SYSTEM SHALL create a **pending invitation** for that number — người được mời **chưa** có quyền gì cho tới khi chấp nhận (AC71) — and return `201` with the `Invitation` (`id`, `phone_number`, `status = "pending"`, `created_at`, `expires_at`, `responded_at = null`).
45. THE SYSTEM SHALL create the invitation and return the **same** `201 Invitation` whether or not an account with that phone number exists — route này SHALL NOT return `USER_NOT_FOUND`, và không có khác biệt nào ở status/body dùng để dò số đã đăng ký. WHEN the number has no account yet, the invitation waits and is shown to the person after they register and log in with that number (AC75).
46. WHEN the phone number belongs to the owner, to the current clan head, or to a user who is already an editor, THE SYSTEM SHALL return `409 COLLABORATOR_ALREADY_EXISTS`; WHEN the genealogy already has an unexpired `pending` invitation for that number, `409 INVITATION_ALREADY_PENDING`. In both cases THE SYSTEM SHALL NOT insert a duplicate row. (Hai mã 409 này chỉ cho owner biết điều owner vốn đã thấy qua AC48/AC73, nên không lộ thêm số nào đã đăng ký.)
47. WHEN the number of editors plus unexpired `pending` invitations of the genealogy has reached 20, THE SYSTEM SHALL return `409 COLLABORATOR_LIMIT_REACHED` for a new invitation; WHEN accepting an invitation would make the editors exceed 20, the same `409` (AC71).
    > ⚠️ Not confirmed: tối đa 20 editor/gia phả là con số đề xuất, và lời mời chờ có tính vào trần 20 hay không (spec đề xuất: có, để owner không tạo lời mời vô hạn) — chưa chốt.
48. WHEN a `GET /v1/genealogies/{id}/collaborators` request arrives from the `owner`, THE SYSTEM SHALL return `200` with `data.items` holding the editors and `clan_head` (cùng `role`) and `data.total`. WHEN a caller with another role asks, `403 GENEALOGY_FORBIDDEN`. (Đã chốt 2026-10-07: chỉ `owner` quản lý quyền và chỉ định trưởng tộc; trưởng tộc chỉ sửa nội dung.)
49. WHEN a `DELETE /v1/genealogies/{id}/collaborators/{userId}` request arrives from the `owner`, THE SYSTEM SHALL remove that editor's grant and return `200` `{"success": true}`; WHEN that user is not an editor, `404 COLLABORATOR_NOT_FOUND`.
50. WHEN a user who is an `editor` calls `DELETE /v1/genealogies/{id}/collaborators/{their own user id}`, THE SYSTEM SHALL remove their own grant (tự rời gia phả) and return `200`. Owner và clan head không tự rời bằng route này.
51. WHEN a `PUT /v1/genealogies/{id}/clan-head` request arrives from the `owner` with a valid `phone_number` of an existing user, THE SYSTEM SHALL set that user as `clan_head` (thay người cũ nếu có) and return `200` with `GenealogyDetail`. WHEN the user was an `editor`, THE SYSTEM SHALL remove their editor row in the same transaction (một user một vai trò). Số điện thoại phải thuộc user đã có tài khoản, nếu không `404 USER_NOT_FOUND` (AC45 không áp dụng cho route này); số của chính `owner` → `400 VALIDATION_ERROR` với `details.field = "phone_number"`.
    > ⚠️ Not confirmed: chỉ định trưởng tộc vẫn cấp thẳng cho user có sẵn (owner dò được số đã đăng ký) hay cũng đi qua lời mời + chấp nhận như editor — chưa chốt; spec giữ cấp thẳng, không chặn.
52. WHEN a `DELETE /v1/genealogies/{id}/clan-head` request arrives from the `owner`, THE SYSTEM SHALL clear the clan head and return `200` with `GenealogyDetail`. Người cũ mất quyền (không tự thành editor).
53. WHEN a caller with a role other than `owner` calls any route of AC44, AC48, AC49 (cho người khác), AC51, AC52, AC73, AC74, THE SYSTEM SHALL return `403 GENEALOGY_FORBIDDEN`; WHEN the caller has no role, `404 GENEALOGY_NOT_FOUND` (AC15).
54. WHEN a user's grant (editor hoặc clan head) is removed, THE SYSTEM SHALL deny that user on their very next request (quyền kiểm ở mỗi request từ DB, không nằm trong token).
    > ⚠️ Not confirmed: chuyển quyền `owner` cho người khác, và hành vi khi tài khoản `owner` bị xoá — hiện ngoài scope; xoá user bị chặn bởi FK `RESTRICT`.

70. WHEN a `GET /v1/invitations` request arrives with a valid access token, THE SYSTEM SHALL return `200` with `data.items` holding the `pending`, unexpired invitations whose `phone_number` equals the caller's own phone number (ordered by `created_at` DESC) and `data.total`; each item is `{id, genealogy_name, invited_by_phone, created_at, expires_at}`. THE SYSTEM SHALL NOT expose any other genealogy data (cây, thành viên, phả ký) to the invitee before acceptance. Đây là "hộp thông báo" v1 của người được mời — client hiển thị thông báo/badge từ route này.
    > ⚠️ Not confirmed: "user nhận thông báo" v1 chỉ là hộp lời mời trong app (client gọi `GET /v1/invitations`) hay còn cần push notification / SMS báo cho số được mời (nhất là số chưa có tài khoản) — chưa chốt; push/SMS hiện ngoài scope, không chặn.
71. WHEN a `POST /v1/invitations/{id}/accept` request arrives from the user whose phone number matches the invitation AND the invitation is `pending` and unexpired, THE SYSTEM SHALL in one transaction record the caller as `editor` of the genealogy (`granted_by` = người mời), mark the invitation `accepted`, and return `200` with `{genealogy_id, role: "editor"}`; quyền có hiệu lực ở request kế tiếp (AC54). WHEN the invitation does not exist, is not addressed to the caller's phone number, or `{id}` is not a positive integer, THE SYSTEM SHALL return `404 INVITATION_NOT_FOUND` (cùng một mã cho cả ba — không lộ lời mời của người khác). WHEN it is no longer pending (đã trả lời, bị huỷ hoặc hết hạn), `409 INVITATION_NOT_PENDING`. WHEN the caller already has a role in the genealogy (ví dụ vừa được chỉ định trưởng tộc), `409 COLLABORATOR_ALREADY_EXISTS`; WHEN the genealogy already has 20 editors, `409 COLLABORATOR_LIMIT_REACHED`.
72. WHEN a `POST /v1/invitations/{id}/decline` request arrives from the user whose phone number matches a `pending`, unexpired invitation, THE SYSTEM SHALL mark it `declined` without granting any role and return `200` `{"success": true}`. The `404 INVITATION_NOT_FOUND` and `409 INVITATION_NOT_PENDING` rules of AC71 apply.
    > ⚠️ Not confirmed: owner có được mời lại ngay số vừa từ chối hay cần chờ/chặn (tránh làm phiền) — spec hiểu là mời lại được, không giới hạn thêm.
73. WHEN a `GET /v1/genealogies/{id}/invitations` request arrives from the `owner`, THE SYSTEM SHALL return `200` with `data.items` holding the genealogy's invitations whose status is `pending`, `declined` or `expired` (ordered by `created_at` DESC), each `{id, phone_number, status, created_at, expires_at, responded_at}`, and `data.total`. Lời mời `accepted` hiển thị ở danh sách người cộng tác (AC48); lời mời `cancelled` bị ẩn. Other roles → AC53.
74. WHEN a `DELETE /v1/genealogies/{id}/invitations/{invitationId}` request arrives from the `owner` AND the invitation belongs to that genealogy AND is `pending`, THE SYSTEM SHALL mark it `cancelled` and return `200` `{"success": true}`; otherwise `404 INVITATION_NOT_FOUND`.
75. THE SYSTEM SHALL match an invitation to a person **by phone number only** (không lưu `user_id` của người được mời): người đăng ký tài khoản sau với số đó (đã xác thực OTP) sẽ thấy lời mời còn hạn ở AC70, không cần đổi luồng register. An invitation expires 30 days after creation; một lời mời hết hạn không còn hiện ở AC70 và không chấp nhận được (AC71), không cần job dọn.
    > ⚠️ Not confirmed: thời hạn 30 ngày là đề xuất; và khi số điện thoại bị thu hồi rồi cấp cho người khác trong lúc lời mời còn hạn thì người mới sẽ thấy lời mời (giảm nhẹ: phải chủ động chấp nhận + hết hạn) — có chấp nhận rủi ro này không.

## US-GEN-07: Thông tin mở rộng của thành viên (mẹ, thứ tự con, giỗ, an táng)

## User Story

> Là một **người có quyền trong gia phả**,
> Tôi muốn **ghi thêm mẹ của một người con, thứ tự anh em, ngày giỗ âm lịch và nơi an táng**,
> Để **hồ sơ mỗi người đủ thông tin cho việc xem quan hệ, chi trưởng/chi thứ và ngày giỗ**.

## Acceptance Criteria (EARS notation)

55. THE SYSTEM SHALL accept these optional fields wherever a member's information is submitted (founder, spouses, `POST` và `PUT` thành viên): `burial_place` (nơi an táng, ≤ 200), `death_lunar_day` (1..30), `death_lunar_month` (1..12) và `death_lunar_leap` (bool, mặc định `false`; ngày giỗ **âm lịch**, server không quy đổi). WHEN `death_lunar_day` and `death_lunar_month` are not both present or both absent, or `death_lunar_leap` is `true` without both, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field` = `death_lunar_day` / `death_lunar_month` / `death_lunar_leap`. WHEN `is_alive` is `true` AND any of `burial_place`, `death_lunar_day`, `death_lunar_month`, `death_lunar_leap` is present, THE SYSTEM SHALL return `400 VALIDATION_ERROR` naming that field (như AC6).
    > ⚠️ Not confirmed: ngày giỗ lưu ngay trên thành viên gia phả (như spec này) hay thành viên tham chiếu một mục sổ giỗ của `memorial` (đã có `death_day/death_month/death_leap_month` + `death_calendar`) — chặn việc chốt tên cột `death_lunar_*`; không chặn các phần khác.
56. THE SYSTEM SHALL store `birth_order` (thứ tự trong các con **cùng `parent_id`**, số nguyên `1..1000`) for blood members other than the founder. WHEN a child is created without `birth_order`, THE SYSTEM SHALL set it to `max(birth_order của các con cùng parent_id) + 1` (hoặc `1` nếu chưa có con). WHEN `birth_order` is present on `POST`/`PUT` of a blood non-founder member, THE SYSTEM SHALL require an integer in `1..1000`; trùng số được phép, khi đó thứ tự theo `id` ASC. WHEN `birth_order` is sent for the founder or an in-law, THE SYSTEM SHALL return `400 VALIDATION_ERROR` with `details.field = "birth_order"`. Thứ tự dùng cho `profile` (AC66, AC68), không suy từ `birth_year` vì nhiều người không có năm sinh.
    > ⚠️ Not confirmed: cách chỉnh thứ tự — gửi `birth_order` qua `PUT` từng người (như spec này) hay có route sắp xếp lại cả nhóm anh em — không chặn.
57. WHEN `POST /v1/genealogies/{id}/members` with `relation = "child"` carries `other_parent_id` (cha/mẹ còn lại của con), THE SYSTEM SHALL require it to be an **in-law** member of the same genealogy with `spouse_of_id = related_member_id`; otherwise it SHALL return `400 VALIDATION_ERROR` with `details.field = "other_parent_id"` (kể cả khi id tồn tại ở gia phả khác). `other_parent_id` is optional, chỉ nhận khi `relation = "child"` (với `spouse` → `400`, `details.field = "other_parent_id"`), và **không đổi được** sau khi tạo (AC31). `Member` returns it (`null` khi không có).
58. WHEN an in-law member that is the `other_parent_id` of one or more children is deleted (AC37), THE SYSTEM SHALL set those children's `other_parent_id` to `null` and SHALL NOT block the deletion.

## US-GEN-08: Chọn "tôi là ai" trong cây

## User Story

> Là một **người có quyền trong gia phả**,
> Tôi muốn **chọn thành viên nào là chính mình trong cây**,
> Để **app biết tôi đang ở đâu và hiển thị quan hệ của mọi người với tôi**.

## Acceptance Criteria (EARS notation)

59. WHEN a `PUT /v1/genealogies/{id}/me/member` request arrives from a user with a role (cả ba vai trò) with `{"member_id": <int>}` of a member of that genealogy, THE SYSTEM SHALL set (hoặc thay) the caller's pointer and return `200` with `{"my_member_id": <int>}`. Mỗi (user, gia phả) có tối đa một con trỏ.
60. WHEN `member_id` is absent, not a positive integer, or the body has unknown fields / is empty / is not valid JSON, THE SYSTEM SHALL return `400 VALIDATION_ERROR` (`details.field = "member_id"` khi do `member_id`). WHEN the member does not exist in this genealogy (kể cả tồn tại ở gia phả khác), THE SYSTEM SHALL return `404 MEMBER_NOT_FOUND`. WHEN the caller has no role, `404 GENEALOGY_NOT_FOUND` (AC15).
61. WHEN a `DELETE /v1/genealogies/{id}/me/member` request arrives from a user with a role, THE SYSTEM SHALL clear the caller's pointer and return `200` `{"success": true}` — cả khi chưa có con trỏ nào (idempotent).
62. THE SYSTEM SHALL return the caller's own pointer as `my_member_id` (`null` nếu chưa chọn) in `GenealogyDetail` (AC14) and in `MemberProfile` (AC64). Con trỏ **riêng tư**: không có route hay field nào trả con trỏ của user khác.
63. WHEN the selected member is deleted, THE SYSTEM SHALL clear the pointer (DB cascade) so that `my_member_id` is `null` ở lần đọc sau. WHEN a user loses their role (editor bị thu quyền hoặc tự rời — AC49/AC50; trưởng tộc bị thay hoặc bỏ — AC51/AC52), THE SYSTEM SHALL remove that user's pointer in the same transaction.
    > ⚠️ Not confirmed: nhiều user có được cùng chọn một thành viên là mình không — spec này **cho phép** (không unique, vì con trỏ riêng tư); nếu muốn "một nút một tài khoản" thì phải thêm unique và `409`, đồng thời mở lại bài toán ai chiếm nút trước.

## US-GEN-09: Hồ sơ thành viên (thông tin tính ở server)

## User Story

> Là một **người có quyền trong gia phả**,
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
69. THE SYSTEM SHALL return `relation_to_me` = `{code, label}` describing **người được chọn là gì của "tôi"** (ví dụ ông nội, chú, anh, vợ) WHEN the caller has a pointer, and `null` WHEN `my_member_id` is `null`. `code` là enum ổn định (`SELF`, `FATHER`, `MOTHER`, `SPOUSE`, `CHILD`, `PATERNAL_GRANDFATHER`, ...); `label` là xưng hô tiếng Việt (null khi `code = UNKNOWN`). v1 hỗ trợ **bên nội** (theo `parent_id`) cộng quan hệ vợ/chồng/dâu/rể và mẹ qua `other_parent_id`; quan hệ không tính được (đặc biệt bên ngoại: cậu, dì, ông bà ngoại) trả `code = UNKNOWN`, `label = null`, không phải lỗi.
    > ⚠️ Not confirmed: bảng xưng hô v1 (bác/chú/cô, anh/chị/em, thím/dượng, cháu, cụ/kỵ...; xem `design.md` mục "Bảng xưng hô v1") chưa được duyệt — **chặn** việc cài hàm xưng hô (`Phase 2`), không chặn các field khác của `profile`.

## API contract (spec của feature này — chưa có trong `openapi.yaml`)

Mọi route đều `RequireAuth` (`authMiddleware.AuthenticateFunc`, `RequireTokenType: "access"`); `user_id` luôn lấy từ token. Vai trò được tính lại từ DB ở **mỗi** request (xem `design.md` Step A).

| Method | Path | Auth | Quyền | Success | Failure codes |
|---|---|---|---|---|---|
| POST | `/v1/genealogies` | RequireAuth | bất kỳ user | `201` `GenealogyDetail` | 400 `VALIDATION_ERROR`, 401, 409 `GENEALOGY_LIMIT_REACHED`, 405, 500 |
| GET | `/v1/genealogies` | RequireAuth | bất kỳ user | `200` `{items, total}` | 401, 405, 500 |
| GET | `/v1/genealogies/{id}` | RequireAuth | có vai trò | `200` `GenealogyDetail` | 401, 404 `GENEALOGY_NOT_FOUND`, 405, 500 |
| PUT | `/v1/genealogies/{id}` | RequireAuth | có vai trò | `200` `GenealogyDetail` | 400, 401, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}` | RequireAuth | `owner` | `200` `{"success": true}` | 401, 403 `GENEALOGY_FORBIDDEN`, 404, 405, 500 |
| PUT | `/v1/genealogies/{id}/chronicle` | RequireAuth | có vai trò | `200` `Chronicle` | 400, 401, 404, 405, 500 |
| GET | `/v1/genealogies/{id}/members` | RequireAuth | có vai trò | `200` `{items, total}` | 401, 404, 405, 500 |
| POST | `/v1/genealogies/{id}/members` | RequireAuth | có vai trò | `201` `Member` | 400, 401, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 409 `MEMBER_LIMIT_REACHED`, 405, 500 |
| GET | `/v1/genealogies/{id}/members/{memberId}` | RequireAuth | có vai trò | `200` `Member` | 401, 404, 405, 500 |
| PUT | `/v1/genealogies/{id}/members/{memberId}` | RequireAuth | có vai trò | `200` `Member` | 400, 401, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}/members/{memberId}` | RequireAuth | có vai trò | `200` `{"success": true}` | 401, 404, 409 `MEMBER_HAS_DESCENDANTS`/`FOUNDER_NOT_DELETABLE`, 405, 500 |
| GET | `/v1/genealogies/{id}/collaborators` | RequireAuth | `owner` | `200` `{items, total}` | 401, 403, 404, 405, 500 |
| POST | `/v1/genealogies/{id}/collaborators` | RequireAuth | `owner` | `201` `Invitation` | 400, 401, 403, 404 `GENEALOGY_NOT_FOUND`, 409 `COLLABORATOR_ALREADY_EXISTS`/`INVITATION_ALREADY_PENDING`/`COLLABORATOR_LIMIT_REACHED`, 405, 500 |
| DELETE | `/v1/genealogies/{id}/collaborators/{userId}` | RequireAuth | `owner`, hoặc chính editor đó | `200` `{"success": true}` | 401, 403, 404 `COLLABORATOR_NOT_FOUND`, 405, 500 |
| GET | `/v1/genealogies/{id}/invitations` | RequireAuth | `owner` | `200` `{items, total}` | 401, 403, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}/invitations/{invitationId}` | RequireAuth | `owner` | `200` `{"success": true}` | 401, 403, 404 `GENEALOGY_NOT_FOUND`/`INVITATION_NOT_FOUND`, 405, 500 |
| GET | `/v1/invitations` | RequireAuth | bất kỳ user | `200` `{items, total}` | 401, 405, 500 |
| POST | `/v1/invitations/{id}/accept` | RequireAuth | người được mời | `200` `{genealogy_id, role}` | 401, 404 `INVITATION_NOT_FOUND`, 409 `INVITATION_NOT_PENDING`/`COLLABORATOR_ALREADY_EXISTS`/`COLLABORATOR_LIMIT_REACHED`, 405, 500 |
| POST | `/v1/invitations/{id}/decline` | RequireAuth | người được mời | `200` `{"success": true}` | 401, 404 `INVITATION_NOT_FOUND`, 409 `INVITATION_NOT_PENDING`, 405, 500 |
| PUT | `/v1/genealogies/{id}/clan-head` | RequireAuth | `owner` | `200` `GenealogyDetail` | 400, 401, 403, 404, 405, 500 |
| DELETE | `/v1/genealogies/{id}/clan-head` | RequireAuth | `owner` | `200` `GenealogyDetail` | 401, 403, 404, 405, 500 |
| PUT | `/v1/genealogies/{id}/me/member` | RequireAuth | có vai trò | `200` `{my_member_id}` | 400, 401, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 405, 500 |
| DELETE | `/v1/genealogies/{id}/me/member` | RequireAuth | có vai trò | `200` `{"success": true}` | 401, 404, 405, 500 |
| GET | `/v1/genealogies/{id}/members/{memberId}/profile` | RequireAuth | có vai trò | `200` `MemberProfile` | 401, 404 `GENEALOGY_NOT_FOUND`/`MEMBER_NOT_FOUND`, 405, 500 |

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
| `death_lunar_day`, `death_lunar_month` | int \| `null` | ngày giỗ **âm lịch**, chỉ khi đã mất, cùng có hoặc cùng `null` (AC55) |
| `death_lunar_leap` | bool \| `null` | tháng nhuận; `null` khi không có ngày giỗ |
| `phone_number` | string \| `null` | điện thoại của người đó (không phải tài khoản user) |
| `note` | string \| `null` | ≤ 1000 |
| `parent_id` | int \| `null` | `null` với thuỷ tổ và với dâu/rể |
| `other_parent_id` | int \| `null` | cha/mẹ còn lại của con: một in-law là vợ/chồng của `parent_id` (AC57) |
| `birth_order` | int \| `null` | thứ tự trong các con cùng `parent_id`; `null` với thuỷ tổ và dâu/rể (AC56) |
| `spouse_of_id` | int \| `null` | chỉ với dâu/rể |
| `generation` | int | đời (thuỷ tổ = 1); dâu/rể = đời của người được gắn |
| `in_law_label` | `"DAU"` \| `"RE"` \| `null` | tính ra (AC33) |
| `created_at`, `updated_at` | RFC 3339 | |

`GenealogyDetail`: `id`, `name`, `ancestral_home`, `my_role` (`owner`/`clan_head`/`editor`), `my_member_id` (int \| `null`, AC62), `clan_head` (`{user_id, phone_number}` \| `null`), `chronicle` (các field AC39 + `founder_name`, `generations_count`, `members_count`), `created_at`, `updated_at`. Các `null` được trả tường minh, không bỏ key.

`MemberProfile` (AC64–AC69): tất cả tính ở server, không lưu.

| Field | Kiểu | Ghi chú |
|---|---|---|
| `member` | `Member` | đầy đủ như `GET .../members/{memberId}` |
| `my_member_id` | int \| `null` | con trỏ riêng của người gọi (AC62) |
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

`Invitation` (phía owner, AC44/AC73): `{id, phone_number, status (pending|declined|expired), created_at, expires_at, responded_at | null}`. `MyInvitation` (phía người được mời, AC70): `{id, genealogy_name, invited_by_phone, created_at, expires_at}`.

## Out of scope (this feature boundary, not tracked here)

- Ảnh (hoành phi, ảnh thành viên) — chặn bởi GCS, cùng nợ với `docs/tech-debt/memorial/TD-001`.
- Ngày giỗ chi tiết và nhắc giỗ từng người — thuộc `memorial`; gia phả chỉ lưu năm. Liên kết một thành viên gia phả với một mục sổ giỗ để sau.
- Quy đổi âm lịch, push notification từ server (xem `memorial`).
- Push notification/SMS báo lời mời — v1 chỉ có hộp lời mời `GET /v1/invitations` (AC70).
- Xưng hô **bên ngoại** (ông bà ngoại, cậu, dì): dâu/rể hiện không có cha mẹ trong cây nên chưa tính được (`code = UNKNOWN`, AC69).
- Đổi/duyệt "ai là tài khoản nào" cho người khác (con trỏ chỉ riêng tư); hiển thị nút nào đã có tài khoản.
- Vai trò `viewer`, link công khai, xuất PDF/in cây, tìm kiếm/lọc thành viên, lịch sử chỉnh sửa (audit), hoàn tác.
- Tự gộp/nhập khẩu hai gia phả, GEDCOM.

## Pending (not yet implemented)

- Toàn bộ feature chưa implement — xem `tasks.md` Phase 1–6.
- Các marker `⚠️ Not confirmed` ở AC10, 19, 24, 25, 27, 31, 35, 39, 40, 47, 51, 54, 55, 56, 63, 65, 66, 69, 70, 72, 75 chờ anh Đức chốt (AC48 đã chốt ở v2.0, AC68 ở v2.2, AC44/AC45 ở v3.0). **Chặn**: AC69 (hàm xưng hô). Các marker mới AC51/70/72/75 đã có mặc định trong spec nên không chặn.
- Sửa đồng thời: hai người sửa cùng thành viên một lúc thì lần ghi sau thắng (không có kiểm phiên bản).
  > ⚠️ Not confirmed: có cần chống ghi đè đồng thời (`updated_at`/`version` trong body `PUT`, trả `409`) không — vì nhiều người cùng sửa; hiện là last-write-wins.

## Ghi chú kỹ thuật liên quan

- **Quyền tính từ DB mỗi request**, không đưa vào JWT: grant/revoke có hiệu lực ngay (AC54). Một truy vấn duy nhất trả về vai trò (owner, clan head, editor hoặc không có) — xem `design.md` Step A.
- **404 đồng nhất** cho "không tồn tại" và "không có quyền" (AC15) vì id tự tăng dễ đoán; 403 chỉ dùng khi người gọi đã có vai trò nhưng thiếu quyền (owner-only).
- **Dữ liệu cá nhân**: `phone_number`, `residence` của người còn sống, `note` — không log các field này; response chỉ cho người có vai trò.
- **Nhãn dâu/rể** tính ra từ giới tính người được gắn, không lưu — đổi giới tính không làm sai dữ liệu. `generation` thì lưu (phi chuẩn hoá) để list/stat không phải đệ quy; vì không có route đổi cha mẹ nên không bao giờ lệch.
- **Cây lưu theo adjacency list** (`parent_id`) vì tối đa 5000 thành viên/60 đời và API trả danh sách phẳng cho client dựng cây; không cần `ltree`/closure table. Kiểm tra "có con" bằng một `EXISTS` trên `parent_id`.
- **Cùng gia phả được ép ở DB**: khoá ngoại ghép `(parent_id, genealogy_id)` → `(id, genealogy_id)` nên không thể gắn cha/mẹ hay vợ/chồng từ gia phả khác dù usecase quên kiểm.
- Hình dạng gần nhất để theo: `memorial` (user-scoped, 404 đồng nhất, trần số lượng kiểm trong transaction có khoá) cộng thêm lớp phân quyền theo vai trò — chưa feature nào có.
- **Vì sao `profile` tính ở server**: có nhiều client (iOS, Android, web), quy tắc xưng hô tiếng Việt phức tạp; để một chỗ (hàm thuần trong usecase, có bảng test) tránh lệch. Cả cây tối đa 5000 thành viên nên một truy vấn tải hết rồi tính trong bộ nhớ là đủ nhanh, không cần đệ quy SQL.
- **`other_parent_id` thay cho "con gắn cả cha lẫn mẹ"**: `parent_id` vẫn là đúng một cha/mẹ thuộc dòng họ; `other_parent_id` chỉ trỏ tới một vợ/chồng của người đó để biết con của bà nào. Nhờ vậy tách được anh em cùng cha khác mẹ và gọi đúng "mẹ".
- **"Tôi là ai" là con trỏ riêng tư theo (user, gia phả)**, không phải cột trên thành viên, nên không cần duyệt, không lộ ai là tài khoản nào, và số điện thoại thành viên (tuỳ chọn, nhiều cụ không có) không liên quan.
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
