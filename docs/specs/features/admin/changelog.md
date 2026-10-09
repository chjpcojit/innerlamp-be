# Changelog — admin Spec

Tracks version history for `requirements.md`, `design.md`, `flow.md`, and `tasks.md` in this folder.
All four files share one changelog since they change together as one spec unit. Newest version
first.

## v1.0 (2026-09-29)

Initial spec, written retroactively from the already-implemented backoffice admin feature in `services/backofficeService/` (Retroactive mode). Phủ sáu user story: đăng nhập, tài khoản admin mặc định (bootstrap + đổi mật khẩu lần đầu), đổi mật khẩu, thu hồi phiên (`tokens_valid_after`), tạo tài khoản (mật khẩu tạm gửi qua email, hết hạn theo config), reset mật khẩu.

- `requirements.md`: 6 US, 30 AC (EARS), bảng API contract 6 route đọc từ handler thực tế, 4 marker `⚠️ Not confirmed` (cấu hình bootstrap production, hình thức gửi mật khẩu qua email thật, định danh `"admin"` trong cột `email`, đổi role phải thu hồi token), 6 open question đã resolved trong quá trình làm.
- `design.md`: bảng layers gắn với file/symbol thật; 6 Step-by-step Flow (Bootstrap, Login, Đổi mật khẩu, Bảo vệ route, Tạo tài khoản, Reset mật khẩu) kèm transaction boundary; mục Deliberately stubbed (LogMailer, mật khẩu `111111`, mock secrets) và Known gaps (thiếu rate limit, thiếu session riêng lẻ, cửa sổ 1 giây của `iat`, tạo tài khoản không atomic với email, chưa có test repository/E2E/openapi/ERD).
- `flow.md`: 6 sequence diagram mirror 1:1 với 6 flow trên; "Bảo vệ route" được các flow còn lại tham chiếu ở Step 1.
- `tasks.md`: Phase 1–2 `[x]`; Phase 3–4 `[x]` phần code, `[ ]` cho test repository, ERD, `openapi.yaml`, README; Phase 5 `[ ]` (E2E, `go test ./...` toàn repo, OKF); Phase 6 liệt kê các việc hoãn có chủ đích.
- Reason: catch up tài liệu cho phần code backoffice vừa làm (bảng `admins` + 4 migration, thu hồi phiên, tạo/reset tài khoản, mailer invite), vốn chưa có spec.

Follow-ups ngoài phạm vi skill này (chỉ flag, không sửa ở đây):
- `docs/specs/api/openapi.yaml`: thêm 5 route `/v1/admin/*` (hiện chưa có route backoffice nào).
- `docs/specs/db/erd.mmd` (+ `erd.svg`): thêm bảng `admins`.
- `migrations/`: không đổi (0006, 0008, 0009, 0010 đã có).
- `docs/knowledge/features/`: chưa có concept cho backoffice admin — `il-okf-writer` tạo sau khi E2E xong.
- `services/backofficeService/README.md`: bổ sung route tạo/reset tài khoản, TTL mật khẩu tạm, mã lỗi `PASSWORD_EXPIRED`.
- `pkg/mailer/README.md`: đã thêm `SendAccountInviteEmail` ở lần làm trước.
