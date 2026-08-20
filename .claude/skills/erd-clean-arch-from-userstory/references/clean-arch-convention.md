# Clean Architecture (Go) — Convention tham chiếu

Convention này dựa trên phong cách đang dùng ở Bankaool/Elcora. Nếu dự án đích có convention riêng khác (ví dụ tên thư mục khác), ưu tiên theo convention người dùng nêu ra trong hội thoại.

## Cấu trúc thư mục (từ trong ra ngoài, theo hướng phụ thuộc)

```
domain/            # Entity thuần Go struct, không import package nào của layer ngoài.
                    # Cũng định nghĩa repository interface tại đây (hoặc usecase/repository/,
                    # tuỳ dự án — domain interface luôn do domain/usecase định nghĩa,
                    # KHÔNG do repository implementation định nghĩa).

usecase/            # Business logic. Nhận dependency là các repository interface (không phải
                    # implementation cụ thể). Đây là nơi orchestrate nhiều repository nếu cần,
                    # xử lý các rule như cache-first, delta sync, fallback offline.

repository/         # Implementation cụ thể của repository interface: Postgres, MongoDB,
                    # Firestore, HTTP client tới service khác, v.v. Chỉ được implement interface,
                    # không được định nghĩa thêm business logic ở đây.

delivery/http/      # HTTP handler: parse request, validate input format (không validate business
                    # rule — việc đó thuộc usecase), gọi usecase, map kết quả ra response DTO.

main.go / wire/      # Nơi duy nhất được phép "biết" tất cả các implementation cụ thể và wiring
                    # chúng lại (dependency injection thủ công hoặc qua wire).
```

## Nguyên tắc dependency

- Import chỉ được đi từ ngoài vào trong: `delivery` → `usecase` → `domain`; `repository` implement interface của `domain`/`usecase` nhưng domain không biết gì về repository cụ thể.
- `domain/` không import bất kỳ package nào thuộc `repository/`, `delivery/`, hay driver DB (`database/sql`, `gorm`, `firestore`...).
- Việc này thường được enforce bằng `golangci-lint` với `depguard`, mỗi rule prefix `[CA]` (Clean Architecture) chặn import sai chiều.

## Convention đặt tên

- Entity: PascalCase, số ít (`Sutra`, không phải `Sutras`).
- Repository interface: `<Entity>Repository` (ví dụ `SutraRepository`), method dùng verb rõ ràng (`FindByID`, `FindUpdatedSince`, `Create`, `Update`).
- Usecase: `<Entity>Usecase` hoặc theo hành vi nghiệp vụ nếu usecase cross-entity.
- DTO ở delivery layer: hậu tố `Request`/`Response` (ví dụ `ListSutrasResponse`), không dùng trực tiếp domain entity làm response nếu cần ẩn/đổi field.

## Mapping AC → Layer (gợi ý khi viết implementation plan)

| Loại AC | Thường rơi vào layer |
|---|---|
| "Given ... khi tải xong, then hiển thị dữ liệu X" | usecase (lấy dữ liệu) + delivery (trả response) |
| "Given không có mạng, then vẫn xem được cache" | usecase (fallback logic), cần 2 nguồn dữ liệu (local cache repository + remote repository) |
| "Given lần đầu tiên, when gọi API, then lưu cache" / "đồng bộ delta theo `updated_since`" | repository interface cần method nhận tham số thời gian; usecase quyết định khi nào gọi full sync vs delta sync |
| "Given tôi lọc theo X, when ..., then ..." | repository interface cần tham số filter; có thể cần index tương ứng ở ERD |

## Test convention

- Test chính đặt ở layer `usecase`, mock repository interface (không mock DB thật).
- Repository implementation có thể có integration test riêng (thường tách bằng build tag hoặc thư mục `_test` riêng), không bắt buộc trong plan trừ khi người dùng yêu cầu.
