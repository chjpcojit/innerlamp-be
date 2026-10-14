---
id: A-003
feature: administrative-units
title: "Seeder has a -dry-run flag; real-file tests skip when the data file is absent / Lệnh seed có cờ -dry-run; test dùng file thật skip khi file vắng"
status: unvalidated
created: 2026-10-11
made-by: AI (Claude), while implementing administrative-units
code: services/appService/cmd/seedadminunits/main.go, services/appService/cmd/seedadminunits/parse_test.go, services/appService/e2e/administrative_unit_e2e_test.go
---

# A-003 — Seeder has a -dry-run flag; real-file tests skip when the data file is absent / Lệnh seed có cờ -dry-run; test dùng file thật skip khi file vắng

## 1. Assumption / Giả định

**EN** — `seedadminunits -dry-run` parses and verifies the file and prints the counts without opening a database connection. The tests that need the real data file (`TestParseDataset_RealDataFile`, the E2E) skip, rather than fail, when it is absent.

**VI** — `seedadminunits -dry-run` đọc và kiểm tra file rồi in số dòng mà không mở kết nối DB. Các test cần file dữ liệu thật (`TestParseDataset_RealDataFile`, E2E) skip thay vì fail khi file vắng.

## 2. Rationale / Lý do

**EN** — A dry run is cheap and lets an operator check a file before loading it. Skipping keeps `go test` green on a checkout without the file, at the cost of the guarantee not running there.

**VI** — Chạy thử rẻ và cho người vận hành kiểm file trước khi nạp. Skip giữ `go test` xanh trên checkout không có file, đổi lại đảm bảo đó không chạy ở đó.

## 3. Impact if wrong / Hệ quả nếu sai

**EN** — A CI job that must prove the dataset is valid would need the file present; make that job fail on a skip, or commit the file (A-002).

**VI** — Job CI cần chứng minh dữ liệu hợp lệ thì phải có file; cho job đó fail khi skip, hoặc commit file (A-002).

## 4. How to validate / Cách xác nhận

**EN** — Owner confirms or rejects; then set `status` and note it in `docs/specs/features/administrative-units/changelog.md`.

**VI** — Người dùng xác nhận hoặc bác bỏ; sau đó đổi `status` và ghi vào `docs/specs/features/administrative-units/changelog.md`.
