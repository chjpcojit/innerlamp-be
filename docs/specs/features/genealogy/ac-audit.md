# AC ↔ test audit — genealogy

Version: 5.12
Last updated: 2026-10-11
Scope: `requirements.md` v5.12 (AC65 đã gồm `traditional_age`), acceptance criteria AC1–AC82 (AC51, AC52, AC61 đã bỏ ở v5.0).

Cách đọc: mỗi AC nêu **một test mà nếu hành vi bị hồi quy thì test đó đỏ**, theo từng lớp.
`U` = usecase (`usecase/genealogy_*_test.go`), `R` = repository trên Postgres thật
(`repository/postgres_genealogy_repository_test.go`), `H` = handler (`delivery/http/genealogy_handler_test.go`),
`E` = E2E binary thật (`e2e/genealogy_e2e_test.go`). `V` = `TestBuildMember_*` (validate từng trường).

| AC | Yêu cầu (rút gọn) | Được chứng minh bởi | Trạng thái |
|---|---|---|---|
| AC1 | Tạo cây + thuỷ tổ + dâu/rể trong một tx | R `TestGenealogyRepository_CreateAndFind`; U `TestGenealogyUsecase_Create`; E `TestGenealogyFlow_E2E/AC1-AC17` | ✅ |
| AC2 | Không token → 401, không tạo gì | E `wiring: all 21 routes…` | ✅ (wiring: chỉ E2E chứng minh được) |
| AC3 | Validate tên/quê gốc | U `TestGenealogyUsecase_Create` (blank/too long); E | ✅ |
| AC4 | Validate thuỷ tổ, phải nam | U `…Create` (founder female); V; E (female founder) | ✅ |
| AC5 | `is_alive` bắt buộc | V `…Invalid/is_alive missing`; U `…Create` | ✅ |
| AC6 | Còn sống thì không có năm mất/linh vị | V `living with death year`, `living with tablet location` | ✅ |
| AC7 | Năm 1000..hiện tại, mất ≥ sinh | V `birth year…`, `death before birth` | ✅ |
| AC8 | Độ dài trường, số điện thoại hợp lệ + chuẩn hoá | V `…too long`, `phone not a number`, `TestBuildMember_Valid` | ✅ |
| AC9 | ≤ 5 vợ/chồng, mặc định giới tính ngược | U `…Create` (too many spouses; spouses default female) | ✅ |
| AC10 | Tối đa 10 gia phả, cả khi đồng thời | R `CreateLimit`, `CreateConcurrentlyNeverExceedsTheLimit`; E `TestGenealogyLimitsComeFromConfig_E2E` | ✅ |
| AC11 | Field lạ/server-owned bị từ chối | H `TreeRoutes/unknown and server-owned…`; E `AC1-AC17` | ✅ |
| AC12 | Lỗi giữa chừng không để lại gì | R `TestGenealogyRepository_CreateIsAtomic` | ✅ |
| AC13 | Danh sách của tôi + `my_role`, mới trước | R `RoleAndList`; H `list is never null`; E | ✅ |
| AC14 | Tổng quan + thống kê tính ra | U `…Get`; R `CreateAndFind`; H; E | ✅ |
| AC15 | 404 đồng nhất, không lộ sự tồn tại | U `…Get` (stranger, non-positive id); E `AC15, AC53: a stranger learns nothing` (so byte với id không tồn tại) | ✅ |
| AC16 | Đổi tên: chỉ owner | U `…UpdateInfo` (viewer forbidden); H; E | ✅ |
| AC17 | Xoá gia phả: chỉ owner, cascade | U `…Delete`; R `ChronicleUpsertAndDelete`; E | ✅ |
| AC18 | Thêm thành viên child/spouse | U `MemberUsecase_Add_*`; R `AddChildAndSpouse`; E | ✅ |
| AC19 | Con chỉ gắn vào nam dòng họ; không vào nữ/in-law | U `Add_Child/rejected`; R `AddRules`; E (cháu ngoại → 400) | ✅ |
| AC20 | Vợ/chồng: cùng đời, ≤ 5, không của in-law | U `Add_Spouse`; R `AddRules/spouse cap` | ✅ |
| AC21 | `relation` không hợp lệ → 400 | U `Add_Child/relation is neither…` | ✅ |
| AC22 | Cha/mẹ ở gia phả khác → 404 | U `…related member does not exist`; R `AddRules/…another tree` | ✅ |
| AC23 | Body thành viên sai → field không có tiền tố | V `TestBuildMember_Invalid`; U `…member information is invalid` | ✅ |
| AC24 | ≤ 5000 thành viên, cả khi đồng thời | R `AddRules/member cap`, `ConcurrentAddNeverExceedsTheMemberCap`; E (config `max_members`) | ✅ |
| AC25 | ≤ 60 đời | U `…would be generation 61`; R `AddRules/generation cap` | ✅ |
| AC26 | Field lạ ở thêm thành viên | H `add: server-owned fields…` | ✅ |
| AC27 | Lệch năm sinh chỉ cảnh báo | U `Add_Child/warns…`, `equal birth years do not warn`, `Move/warns…`; E | ✅ |
| AC28 | Toàn bộ thành viên, đời tăng dần | U `ListAndGet`; R; E | ✅ |
| AC29 | Một thành viên; gia phả khác → 404 | U `ListAndGet/…another tree`; E (profile của nút gia phả khác) | ✅ |
| AC30 | Sửa thông tin; đổi giới tính sang nữ khi đã có con bị chặn; xoá số huỷ lời mời | U `MemberUsecase_Update`; R `GenderChange…`, `Invitations_Rules/cancel…`; E | ✅ |
| AC31 | PUT không mang quan hệ | H `update refuses relationship fields`; E | ✅ |
| AC32 | Đổi giới tính giữ nguyên dâu/rể | R `TestGenealogyMemberRepository_GenderChangeKeepsInLawsAttached`; `TestInLawLabel` | ✅ |
| AC33 | `in_law_label` DAU/RE tính ra | U `Add_Spouse`, `ListAndGet`; `TestInLawLabel`; E | ✅ |
| AC34 | Xoá lá kèm dâu/rể | R `Delete`; E | ✅ |
| AC35 | Còn con → 409 | R `Delete`; E | ✅ |
| AC36 | Thuỷ tổ không xoá được | R `Delete`; E | ✅ |
| AC37 | Xoá dâu/rể chỉ xoá chính họ | R `Delete` (in-law only) | ✅ |
| AC38 | Phả ký thay thế toàn bộ | U `UpdateChronicle`; R `ChronicleUpsertAndDelete`; E | ✅ |
| AC39 | Trường và độ dài của phả ký (cả phiên âm/nghĩa của hai vế) | U `UpdateChronicle/invalid`; E (`couplet_upper_reading`) | ✅ |
| AC40 | Ngày chạp dương, đủ ba phần, cờ âm, server trả âm | U `…chạp…`; `TestSolarToLunar`; E (Tết 2024 → 1/1) | ✅ |
| AC41 | Trường tính ra không ghi được | H `chronicle: …derived fields…refused`; E | ✅ |
| AC42 | Body/field lạ ở phả ký | H; E | ✅ |
| AC43 | Không lưu ảnh | H (decoder strict: không có field ảnh) | ✅ (phủ định, bằng cấu trúc) |
| AC44 | Mời một thành viên còn sống, ghi số lên nút | U `AccessUsecase_Invite`; R `Invitations_FullJourney`, `Rules`; E | ✅ |
| AC45 | Cùng kết quả dù số có/chưa có tài khoản | E `…journey` (số chưa đăng ký → 201 giống nhau); `AccessUsecase` không phụ thuộc tra user | ✅ |
| AC46 | 409 chủ/người xem/đã mời | R `FullJourney`, `Rules`; E (`INVITATION_ALREADY_PENDING`) | ✅ |
| AC47 | Trần 20 gồm lời mời chờ | R `Rules/the cap counts pending…`, `AcceptConcurrently…`; E `…FromConfig` | ✅ |
| AC48 | Danh sách người xem kèm nút | R `FullJourney`; H; E | ✅ |
| AC49 | Owner gỡ người xem | U `RemoveViewer`; R; E | ✅ |
| AC50 | Người xem tự rời | U `RemoveViewer`; E (rời → 404 ngay) | ✅ |
| AC53 | Người xem gọi route owner → 403; không vai trò → 404 | U (mọi use case); E `…journey` (10 route ghi) | ✅ |
| AC54 | Thu quyền có hiệu lực ngay; không chuyển owner | E (owner gỡ rồi 404 ngay); không có route chuyển (cấu trúc) | ✅ |
| AC55 | Ngày giỗ dương + cờ âm, server trả âm | V `…anniversary…`; U `ListAndGet` (lunar); E (Tết 2024) | ✅ |
| AC56 | `birth_order` mặc định max+1, không dùng cho thuỷ tổ/dâu rể | V; R `AddChildAndSpouse`, `ConcurrentAdd…` | ✅ |
| AC57 | `other_parent_id` phải là vợ/chồng của cha | U `Add_Child/other parent…`; R `AddRules` | ✅ |
| AC58 | Xoá dâu/rể → `other_parent_id` của con về NULL | R `Delete` | ✅ |
| AC59 | Vị trí "tôi" theo số điện thoại | U `TestProfileUsecase`; R `FullJourney` (viewer.MemberID); E | ✅ |
| AC60 | Một số điện thoại một nút/gia phả | R `PhoneIsUniquePerTree…`; U `…Create` (trùng trong request); E (409) | ✅ |
| AC62 | `my_member_id` ở overview và profile; không lộ người khác | U; E (`vd.MyMemberID`); viewer không thấy của người khác (cấu trúc DTO) | ✅ |
| AC63 | Xoá/đổi số → mất vị trí ở lần đọc sau | E (xoá số của nút → `my_member_id = null`) | ✅ |
| AC64 | Profile route | U `TestProfileUsecase`; H `profile renders…`; E | ✅ |
| AC65 | Tuổi tây và tuổi mụ (đang sống / lúc mất / null) | `TestAge`, `TestTraditionalAge`; U `TestProfileUsecase` (age 36, traditional_age 37); H `profile renders…` (`traditional_age`) | ✅ |
| AC66 | Cha, mẹ, vợ/chồng, anh em (nửa dòng), con | `TestRelativesOf` | ✅ |
| AC67 | Đường từ thuỷ tổ | `TestLineageOf`; E | ✅ |
| AC68 | Chi, `chi_path` lồng nhau, con gái không đầu chi | `TestChiOf`; E | ✅ |
| AC69 | Xưng hô theo đời, không theo tuổi; "họ"; UNKNOWN | `TestKinshipOf`, `TestKinshipIgnoresBirthYears`, `TestKinshipTieBreaksOnID`; E (Bác) | ✅ |
| AC70 | Hộp lời mời, không lộ cây | U `TestInvitationUsecase`; R `FullJourney`; H (không lộ field); E | ✅ |
| AC71 | Chấp nhận → viewer; 404/409 | U; R `FullJourney`, `AcceptConcurrently…`; E | ✅ |
| AC72 | Từ chối | R `FullJourney`; E `AC72: decline…` | ✅ |
| AC73 | Owner xem trạng thái (kể cả accepted) | R `FullJourney`; E | ✅ |
| AC74 | Huỷ lời mời chờ | U; R `Rules`; E | ✅ |
| AC75 | Khớp theo số; hết hạn 30 ngày từ config | R `Rules/an expired invitation…`; E `TestGenealogyInvitationExpiry_E2E` | ✅ |
| AC76 | Đổi quan hệ, dời đời cả nhánh, một tx | R `Move`; U `Move`; E | ✅ |
| AC77 | `relation` khớp loại; chống vòng; luật cha nam | R `Move/relation must match…`, `a loop is refused` | ✅ |
| AC78 | Vượt 60 đời → không đổi gì | R `Move/a move past the generation cap…` | ✅ |
| AC79 | Thuỷ tổ → 409; không thấy → 404 | R `Move`; E | ✅ |
| AC80 | Field lạ / lỗi → không đổi gì | U `Move/…`; R `Move` (không đổi khi lỗi) | ✅ |
| AC81 | Người xem không ghi được gì | E `…journey` (10 route → 403, dữ liệu không đổi); U | ✅ |
| AC82 | Trưởng tộc là cờ: một/gia phả, nam, còn sống, blood | R `ClanHeadSwap`; V; U `Update`; E | ✅ |

## Khoảng trống còn lại

Không có AC nào chưa có test. Những điểm **chứng minh yếu hơn** và lý do:

| Điểm | Vì sao | Cách xử lý |
|---|---|---|
| AC43 (không lưu ảnh), AC54 (không có route chuyển quyền owner) | Là yêu cầu phủ định: chỉ chứng minh được bằng cấu trúc (không có field/route) | Chấp nhận; test strict-decoder và danh sách 21 route trong E2E là lưới chắn |
| AC62 "không lộ vị trí của người khác" | DTO của viewer không có field nào mang nó; chưa có test riêng "viewer không thấy `member_id` người khác" | Đã có `viewers` chỉ owner gọi được (403 cho viewer, E2E) |
| Hiệu năng NFR-001/002 (SRS) | Mục tiêu do BA đề xuất, chưa duyệt; chưa có load test | Ghi ở SRS Appendix A; làm khi anh duyệt ngưỡng |
