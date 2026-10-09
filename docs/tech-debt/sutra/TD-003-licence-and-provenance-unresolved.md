---
id: TD-003
feature: sutra
title: "Licences of the texts are unresolved / Giấy phép của các văn bản chưa được giải quyết"
severity: medium
status: open
created: 2026-10-03
owner: ducnguyen-wee
code: assets/data/sutras/*.json (translations[].license, sourceUrl)
---

# TD-003 — Licences of the texts are unresolved / Giấy phép của các văn bản chưa được giải quyết

## 1. Debt / Nợ gì

**EN** — Most Vietnamese translations have `license: unknown` or only a "free to copy" notice (phatgiao.org.vn); Thích Minh Châu's SuttaCentral text states no licence; Indacanda's is "used by kind permission of the translator"; the ten Vajrayana texts from Lotsawa House are CC BY-NC 4.0 (non-commercial); Sujato's English asks not to be used for AI training. The data is embedded in every binary and returned by the API.

**VI** — Phần lớn bản dịch tiếng Việt có `license: unknown` hoặc chỉ có thông báo "được sao chép" (phatgiao.org.vn); bản Thích Minh Châu trên SuttaCentral không nêu giấy phép; bản Indacanda là "dùng với sự cho phép của dịch giả"; mười văn bản Kim Cang Thừa từ Lotsawa House là CC BY-NC 4.0 (phi thương mại); bản tiếng Anh của Sujato đề nghị không dùng cho huấn luyện AI. Dữ liệu được nhúng trong mọi binary và API trả ra.

## 2. Why accepted / Vì sao chấp nhận

**EN** — The app is free, so the user chose to set licensing aside for now (2026-10-03).

**VI** — Ứng dụng miễn phí nên người dùng chọn tạm gác vấn đề giấy phép (2026-10-03).

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Charging, ads or in-app purchases would put non-commercial and unknown-licence texts in commercial use; translators or publishers can ask for a takedown.

**VI** — Thu phí, quảng cáo hay mua trong ứng dụng sẽ đưa các văn bản phi thương mại và chưa rõ giấy phép vào mục đích thương mại; dịch giả hoặc nhà xuất bản có thể yêu cầu gỡ.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] Ask each translator/publisher for permission (Minh Châu, Trí Tịnh, Trí Quang, Nhất Hạnh, …) or replace the text with a CC0/public-domain one / Xin phép từng dịch giả/nhà xuất bản (Minh Châu, Trí Tịnh, Trí Quang, Nhất Hạnh, …) hoặc thay bằng bản CC0/public domain
- [ ] Or add a per-text visibility flag so non-commercial texts can be hidden if the app is monetized / Hoặc thêm cờ hiển thị theo từng văn bản để ẩn văn bản phi thương mại nếu ứng dụng thu phí
- [ ] Show `translator`, `source_name`, `source_url` in the client (the API already returns them) / Hiển thị `translator`, `source_name`, `source_url` ở client (API đã trả sẵn)

## 5. Estimate / Ước lượng

**EN** — Unknown: depends on replies from third parties.

**VI** — Chưa rõ: phụ thuộc phản hồi của bên thứ ba.
