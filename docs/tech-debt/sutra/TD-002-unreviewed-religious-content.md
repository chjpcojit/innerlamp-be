---
id: TD-002
feature: sutra
title: "Religious content has not been reviewed by a knowledgeable person / Nội dung tôn giáo chưa được người am hiểu duyệt"
severity: high
status: wontfix
created: 2026-10-03
owner: ducnguyen-wee
code: assets/data/sutras/*.json (reviewStatus, dailyChant, purposes, deities, titles, chu-dai-bi.json)
---

# TD-002 — Religious content has not been reviewed by a knowledgeable person / Nội dung tôn giáo chưa được người am hiểu duyệt

## 1. Debt / Nợ gì

**EN** — All 153 texts carry `reviewStatus: UNREVIEWED`. `dailyChant`, `purposes`, the Phật/Bồ tát links, `titles.vi`, and the Chinese/Sanskrit titles and Taishō numbers (e.g. 1060 for Chú Đại Bi, 945 for Lăng Nghiêm) were assigned or added by the collector (Claude), not taken from the sources. Vietnamese source typos are kept verbatim ("Quán Thế Ám", "nầy", "Thọai", "chúng thôi phải"); Trần Trọng Kim's Tâm Kinh uses 1930s spelling; `chu-dai-bi` follows the phatgiao.org.vn numbered list, which contradicts the prose version on the same page and differs from hoatieu.vn in 8 syllables (e.g. `kiết`/`yết`, `dà`/`già`). About 9 lines exceed 220 characters and a few comma-split lines start in lower case.

**VI** — Cả 153 văn bản mang `reviewStatus: UNREVIEWED`. `dailyChant`, `purposes`, liên kết Phật/Bồ tát, `titles.vi`, tiêu đề Hán/Sanskrit và số Taishō (vd. 1060 cho Chú Đại Bi, 945 cho Lăng Nghiêm) do người thu thập (Claude) gán hoặc thêm, không lấy từ nguồn. Lỗi chính tả của nguồn tiếng Việt được giữ nguyên ("Quán Thế Ám", "nầy", "Thọai", "chúng thôi phải"); Tâm Kinh của Trần Trọng Kim dùng chính tả những năm 1930; `chu-dai-bi` theo danh sách đánh số của phatgiao.org.vn, vốn mâu thuẫn với bản văn xuôi trên cùng trang và khác hoatieu.vn ở 8 âm tiết (vd. `kiết`/`yết`, `dà`/`già`). Khoảng 9 dòng dài hơn 220 ký tự và vài dòng bị cắt ở dấu phẩy bắt đầu bằng chữ thường.

## 2. Why accepted / Vì sao chấp nhận

**EN** — No one with religious knowledge is available to review (user decision 2026-10-03: the AI decides). See assumptions A-004 and A-006.

**VI** — Hiện không có ai am hiểu tôn giáo để duyệt (quyết định của người dùng 2026-10-03: AI tự quyết). Xem giả định A-004 và A-006.

## 3. Risk if unpaid / Rủi ro nếu không trả

**EN** — Wrong syllables of a mantra, a text shown under the wrong purpose (cầu siêu vs cầu an) or a wrong deity link reach practitioners as if authoritative; the damage is cultural and reputational, and it grows with use.

**VI** — Sai âm tiết của một thần chú, văn bản hiển thị sai mục đích (cầu siêu so với cầu an) hay sai liên kết Phật/Bồ tát tới người tu tập như thể là chuẩn; thiệt hại về văn hoá và uy tín, và tăng theo mức độ dùng.

## 4. Exit criteria / Điều kiện trả nợ

- [ ] A knowledgeable reviewer checks `dailyChant`, `purposes`, deity links and every mantra/dharani text / Người am hiểu duyệt `dailyChant`, `purposes`, liên kết Phật/Bồ tát và từng văn bản chú/đà-la-ni
- [ ] Decide the chanted version of Chú Đại Bi and Lăng Nghiêm, syllable by syllable / Chốt bản tụng Chú Đại Bi và Lăng Nghiêm, từng âm tiết
- [ ] Correct source typos (keep a note of each correction against the original) / Sửa lỗi chính tả của nguồn (ghi chú từng chỗ sửa so với bản gốc)
- [ ] Set `reviewStatus` per sutra once reviewed, and consider hiding unreviewed texts in the client / Đổi `reviewStatus` từng kinh khi đã duyệt, và cân nhắc ẩn văn bản chưa duyệt ở client

## 5. Estimate / Ước lượng

**EN** — Reviewer time, days for 153 texts; about half a day of tooling to apply corrections.

**VI** — Thời gian của người duyệt, vài ngày cho 153 văn bản; khoảng nửa ngày làm công cụ áp dụng chỉnh sửa.

## 6. Decision / Quyết định (2026-10-03)

**EN** — The user told the AI to decide the content itself, since no reviewer is available. The AI's labels, versions and durations stand as the working decision; `reviewStatus` stays `UNREVIEWED` on purpose, because nobody has reviewed the content. Re-open when a knowledgeable reviewer is found.

**VI** — Anh giao AI tự chốt nội dung vì không có người duyệt. Nhãn, bản chú và thời lượng do AI chọn được coi là quyết định tạm; `reviewStatus` vẫn là `UNREVIEWED` có chủ ý vì chưa có ai duyệt. Mở lại khi có người am hiểu.

## 7. Update / Cập nhật (2026-10-07)

**EN** — The Vietnamese `summary` of all 153 texts (and the split of `reference` into code, author, Hán translator, year) was added by the AI and is also unreviewed; see `docs/assumptions/sutra/A-011` and `A-010`. Exit criterion: a knowledgeable person reads every `summary` and sets `reviewStatus`.

**VI** — `summary` tiếng Việt của 153 văn bản (và việc tách `reference` thành mã, tác giả, người dịch Hán, năm) do AI thêm và cũng chưa được duyệt; xem `A-011`, `A-010`. Tiêu chí đóng: có người am hiểu đọc từng `summary` rồi đổi `reviewStatus`.
