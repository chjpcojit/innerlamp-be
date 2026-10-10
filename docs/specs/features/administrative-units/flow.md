# Flow — administrative-units

Version: 1.4
Last updated: 2026-10-10
Status: mirror `design.md` v1.4 §Step-by-step Flow; Step 3 (`Dataset.Verify`) và Step 4–6 (`ReplaceDataset`, đã chạy trên Postgres thật) đã implement; Step 1–2 và 7 thuộc CLI chưa làm.
Module: `administrative-units`

> See `changelog.md` in this folder for the version history of this spec.

File này là bản vẽ của đúng một section "Step-by-step Flow" trong `design.md` (nạp dữ liệu bằng `cmd/seedadminunits`). Các route đọc chỉ một câu SELECT nên không có sơ đồ ở đây; trace ASCII của chúng nằm ở `design.md` §Request flow.

## Sequence — Nạp bộ dữ liệu hành chính (`cmd/seedadminunits`)

```mermaid
sequenceDiagram
    autonumber
    participant O as Operator
    participant S as seedadminunits<br/>(cmd, main.go + parse.go)
    participant D as Dataset<br/>(domain, Verify)
    participant R as AdministrativeUnitRepository
    participant DB as Postgres

    O->>S: chạy seedadminunits -file administrative-units.json

    %% Step 1
    S->>S: đọc file JSON duy nhất (os.ReadFile)
    alt không có file / không đọc được
        S--xO: in đường dẫn, exit ≠ 0 (chưa chạm DB)
    end

    %% Step 2
    S->>S: giải mã JSON, chuẩn hoá NFC, dựng Dataset
    Note over S: parse không sửa, không bỏ dòng nào.<br/>district_mappings được đọc và kiểm nhưng không nạp (hoãn)
    alt JSON hỏng / thiếu khoá bắt buộc
        S--xO: in vị trí, exit ≠ 0 (chưa chạm DB)
    end

    %% Step 3
    S->>D: Verify()
    Note over S,D: file hiện tại đã qua cả bảy quy tắc toàn vẹn
    alt không có vi phạm
        D-->>S: danh sách vi phạm rỗng
    else có vi phạm toàn vẹn
        D--xS: IntegrityIssue[]
        S--xO: in báo cáo từng dòng, exit ≠ 0 (chưa mở transaction)
    end

    S->>R: ReplaceDataset(ctx, dataset)

    rect rgb(235, 240, 255)
        Note over R,DB: MỘT transaction
        %% Step 4
        R->>DB: BEGIN
        R->>DB: UPSERT addr_new_provinces, addr_new_wards,<br/>addr_old_provinces, addr_old_districts, addr_old_wards
        Note over R,DB: chỉ upsert, không DELETE:<br/>đơn vị vắng trong file được giữ nguyên
        %% Step 5
        R->>DB: UPSERT addr_ward_mappings
        %% Step 6
        R->>DB: INSERT addr_dataset_versions ON CONFLICT DO NOTHING
        alt tất cả thành công
            R->>DB: COMMIT
            R-->>S: số dòng mỗi bảng
        else lỗi DB / vi phạm FK / COMMIT lỗi
            R--xDB: ROLLBACK
            R--xS: lỗi
            S--xO: in lỗi, exit ≠ 0 (DB giữ nguyên dữ liệu trước)
        end
    end

    %% Step 7
    S-->>O: in số dòng đã nạp + phiên bản, exit 0
```
