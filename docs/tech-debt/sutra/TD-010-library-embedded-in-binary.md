---
id: TD-010
feature: sutra
title: "The library is embedded in the binary: +10 MB and a slower first start / Thư viện được nhúng vào binary: +10 MB và lần khởi động đầu chậm hơn"
severity: low
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: assets/embed.go (SutrasFS); services/appService/cmd/main.go (SeedIfEmpty)
---

# TD-010 — The library is embedded in the binary: +10 MB and a slower first start / Thư viện được nhúng vào binary: +10 MB và lần khởi động đầu chậm hơn

## 1. Debt / Nợ gì

**EN** — `go:embed data/sutras/*.json` adds 10.25 MB (+17%) to the binary (70.9 MB vs 60.7 MB) for data that is read once, on the first start; that start takes about 1.8 s instead of 0.13 s. Changing the data needs a rebuild and redeploy. Measured on a local Postgres, on 2026-10-03.

**VI** — `go:embed data/sutras/*.json` làm binary nặng thêm 10,25 MB (+17%) (70,9 MB so với 60,7 MB) cho dữ liệu chỉ đọc một lần ở lần khởi động đầu; lần đó mất khoảng 1,8 giây thay vì 0,13 giây. Đổi dữ liệu cần build và triển khai lại. Đo trên Postgres cục bộ ngày 2026-10-03.

## 2. Why accepted / Vì sao chấp nhận

**EN** — The user asked for a first-run load that works in containers without the source tree, as `article` does.

**VI** — Người dùng yêu cầu nạp lần đầu hoạt động trong container không có mã nguồn, giống `article`.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Image size grows with every text added (the library is 10 MB now); cold-start cost and a 2-minute seed budget matter on a slow or remote database.

**VI** — Kích thước image tăng theo từng văn bản thêm vào (thư viện hiện 10 MB); chi phí cold start và ngân sách seed 2 phút đáng kể trên DB chậm hoặc từ xa.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Revisit if the data passes ~50 MB or cold start becomes a problem / Xem lại nếu dữ liệu vượt ~50 MB hoặc cold start thành vấn đề
- [ ] Alternative: ship the data as a separate artifact loaded by `cmd/seedsutras -dir`, keep the embed only for dev / Phương án: đưa dữ liệu thành artifact riêng do `cmd/seedsutras -dir` nạp, chỉ giữ embed cho dev

## 5. Estimate / Ước lượng

**EN** — 1 day.

**VI** — 1 ngày.
