# Đề xuất địa chỉ có cấu trúc cho chùa (cần duyệt)

> **Đã duyệt và đã áp dụng ngày 2026-10-11** — kết quả ở [address-decisions.md](address-decisions.md). File này giữ lại làm bản ghi của bước đề xuất; script đề xuất không chạy lại được trên `temple.json` đã áp dụng (dùng `temple.json.pre-v1.7.bak` làm đầu vào nếu cần tái tạo).

Sinh bởi `scripts/build_temple_addresses.py` — **chỉ là đề xuất**, chưa ghi vào `assets/data/temple.json`.
Dataset hành chính: `2026-09-25` (ghi `data_provenance`: chưa đối chiếu với file chính thức).
Spec: `docs/specs/features/temple/` v1.7+, `tasks.md` Phase 10.

- Tổng: **93** chùa; `auto`: **68** (đúng một xã mới, đúng tỉnh); `review`: **25**.
- Địa chỉ dưới cấp xã **trống**: **20** chùa (địa chỉ cũ chỉ có xã/huyện/tỉnh).

## Cần duyệt

Với mỗi dòng: chọn `ward_code` trong cột *Ứng viên* (hoặc cho mã khác), và sửa cột *address* nếu cần.
`Gợi ý` có giá trị khi chỉ còn một ứng viên nhưng dòng vẫn cần xác nhận vì lý do ở cột *Lý do*.

| id | tên | địa chỉ cũ | lý do | ứng viên (xã mới) | gợi ý | address đề xuất |
|---|---|---|---|---|---|---|
| `temple_north_001` | Chùa Trấn Quốc | Đường Thanh Niên, Phường Yên Phụ, Quận Tây Hồ, Hà Nội | multiple_new_wards | `00097` Phường Hồng Hà (ambiguous); `00103` Phường Tây Hồ (ambiguous) | — | Đường Thanh Niên |
| `temple_north_002` | Chùa Một Cột | Phố Chùa Một Cột, Phường Đội Cấn, Quận Ba Đình, Hà Nội | multiple_new_wards | `00004` Phường Ba Đình (ambiguous); `00008` Phường Ngọc Hà (ambiguous) | — | Phố Chùa Một Cột |
| `temple_north_015` | Chùa Vĩnh Nghiêm (Bắc Giang) | Làng Đức La, Xã Trí Yên, Huyện Yên Dũng, Tỉnh Bắc Giang | old_district_not_found, ward_matched_by_name_in_province | `07682` Phường Tân An (merged) | `07682` | Làng Đức La |
| `temple_central_001` | Chùa Thiên Mụ | Đồi Hà Khê, Phường Kim Long, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found, ward_matched_by_name_in_province | `19774` Phường Kim Long (merged) | `19774` | Đồi Hà Khê |
| `temple_central_002` | Chùa Từ Đàm | Số 1 Sư Liễu Quán, Phường Trường An, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found, ward_matched_by_name_in_province | `19789` Phường Thuận Hóa (merged) | `19789` | Số 1 Sư Liễu Quán |
| `temple_central_003` | Chùa Từ Hiếu | Thôn Dương Xuân Thượng, Phường Thủy Xuân, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found, ward_matched_by_name_in_province | `19813` Phường Thủy Xuân (merged) | `19813` | Thôn Dương Xuân Thượng |
| `temple_central_004` | Chùa Huyền Không Sơn Thượng | Thôn Đồng Chầm, Xã Hương Hồ, Thị xã Hương Trà, Tỉnh Thừa Thiên Huế | old_ward_not_found | — | — | Thôn Đồng Chầm |
| `temple_central_012` | Chùa Long Sơn | 20 Đường 23/10, Phường Phương Sơn, TP. Nha Trang, Tỉnh Khánh Hòa | old_ward_not_found | — | — | 20 Đường 23/10 |
| `temple_central_020` | Chùa Sắc Tứ Khải Đoan | 117 Phan Bội Châu, Phường Thống Nhất, TP. Buôn Ma Thuột, Tỉnh Đắk Lắk | old_ward_not_found | — | — | 117 Phan Bội Châu |
| `temple_south_002` | Chùa Ngọc Hoàng | 73 Mai Thị Lựu, Phường Đa Kao, Quận 1, TP.HCM | multiple_new_wards | `26737` Phường Tân Định (ambiguous); `26740` Phường Sài Gòn (ambiguous) | — | 73 Mai Thị Lựu |
| `temple_north_027` | Thiền xá Yên Phú | Xã Liên Ninh, Huyện Thanh Trì, Hà Nội | multiple_new_wards | `00679` Xã Ngọc Hồi (ambiguous); `00685` Xã Nam Phù (ambiguous) | — | *(trống)* |
| `temple_north_028` | Chùa Côn Sơn | Phường Cộng Hòa, TP. Chí Linh, Tỉnh Hải Dương | multiple_new_wards | `10549` Phường Chu Văn An (ambiguous); `10570` Phường Trần Hưng Đạo (ambiguous) | — | *(trống)* |
| `temple_north_030` | Chùa Dư Hàng | 121 Dư Hàng, Phường Hồ Nam, Quận Lê Chân, TP. Hải Phòng | old_ward_not_found | — | — | 121 Dư Hàng |
| `temple_north_035` | Chùa Bà Đá | Số 3 Phố Nhà Thờ, Phường Hàng Trống, Quận Hoàn Kiếm, Hà Nội | multiple_new_wards | `00070` Phường Hoàn Kiếm (ambiguous); `00082` Phường Cửa Nam (ambiguous) | — | Số 3 Phố Nhà Thờ |
| `temple_central_024` | Chùa Báo Quốc | Đồi Hàm Long, Phường Phường Đúc, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found, ward_matched_by_name_in_province | `19789` Phường Thuận Hóa (merged) | `19789` | Đồi Hàm Long |
| `temple_central_025` | Chùa Diệu Đế | Đường Bạch Đằng, Phường Phú Cát, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found | — | — | Đường Bạch Đằng |
| `temple_central_028` | Chùa Hang (Lý Sơn) | Xã An Hải, Huyện Lý Sơn, Tỉnh Quảng Ngãi | old_ward_not_found | — | — | *(trống)* |
| `temple_south_021` | Pháp viện Minh Đăng Quang | 505 Xa lộ Hà Nội, Phường An Phú, TP. Thủ Đức, TP.HCM | multiple_new_wards | `27094` Phường An Khánh (ambiguous); `27097` Phường Bình Trưng (ambiguous) | — | 505 Xa lộ Hà Nội |
| `temple_north_041` | Chùa Kim Liên | Làng Nghi Tàm, Phường Quảng An, Quận Tây Hồ, Hà Nội | multiple_new_wards | `00097` Phường Hồng Hà (ambiguous); `00103` Phường Tây Hồ (ambiguous) | — | Làng Nghi Tàm |
| `temple_north_042` | Chùa Lý Triều Quốc Sư | 50 Lý Quốc Sư, Phường Hàng Trống, Quận Hoàn Kiếm, Hà Nội | multiple_new_wards | `00070` Phường Hoàn Kiếm (ambiguous); `00082` Phường Cửa Nam (ambiguous) | — | 50 Lý Quốc Sư |
| `temple_north_051` | Thiền xá Tảo Sách | 388 Lạc Long Quân, Phường Nhật Tân, Quận Tây Hồ, Hà Nội | multiple_new_wards | `00097` Phường Hồng Hà (ambiguous); `00103` Phường Tây Hồ (ambiguous) | — | 388 Lạc Long Quân |
| `temple_north_053` | Chùa Bằng | Phố Bằng Liệt, Phường Hoàng Liệt, Quận Hoàng Mai, Hà Nội | multiple_new_wards | `00316` Phường Định Công (ambiguous); `00331` Phường Hoàng Mai (ambiguous); `00337` Phường Hoàng Liệt (ambiguous); `00340` Phường Yên Sở (ambiguous) | — | Phố Bằng Liệt |
| `temple_north_056` | Chùa Hòe Nhai | 19 Phố Hàng Than, Phường Nguyễn Trung Trực, Quận Ba Đình, Hà Nội | old_ward_not_found | — | — | 19 Phố Hàng Than |
| `temple_central_041` | Chùa Quốc Ân | Đồi Phước Quả, Phường Trường An, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found, ward_matched_by_name_in_province | `19789` Phường Thuận Hóa (merged) | `19789` | Đồi Phước Quả |
| `temple_central_043` | Chùa Thiền Lâm | Đồi Quảng Tế, Phường Thủy Xuân, TP. Huế, Tỉnh Thừa Thiên Huế | old_district_not_found, ward_matched_by_name_in_province | `19813` Phường Thủy Xuân (merged) | `19813` | Đồi Quảng Tế |

## Tự động (một xã mới, đúng tỉnh) — xem lướt

| id | tên | địa chỉ cũ | tỉnh | xã mới | address đề xuất |
|---|---|---|---|---|---|
| `temple_north_003` | Chùa Hương | Xã Hương Sơn, Huyện Mỹ Đức, Hà Nội | `01` Hà Nội | `10489` Xã Hương Sơn | *(trống)* |
| `temple_north_004` | Chùa Thầy | Chân núi Sài Sơn, Xã Sài Sơn, Huyện Quốc Oai, Hà Nội | `01` Hà Nội | `09895` Xã Quốc Oai | Chân núi Sài Sơn |
| `temple_north_005` | Chùa Tây Phương | Đồi Câu Lậu, Thôn Yên, Xã Thạch Xá, Huyện Thạch Thất, Hà Nội | `01` Hà Nội | `10003` Xã Tây Phương | Đồi Câu Lậu, Thôn Yên |
| `temple_north_006` | Chùa Quán Sứ | 73 Phố Quán Sứ, Phường Trần Hưng Đạo, Quận Hoàn Kiếm, Hà Nội | `01` Hà Nội | `00082` Phường Cửa Nam | 73 Phố Quán Sứ |
| `temple_north_007` | Chùa Dâu | Xã Thanh Khương, Thị xã Thuận Thành, Tỉnh Bắc Ninh | `24` Bắc Ninh | `09427` Phường Trí Quả | *(trống)* |
| `temple_north_008` | Chùa Phật Tích | Sườn núi Lạn Kha, Xã Phật Tích, Huyện Tiên Du, Tỉnh Bắc Ninh | `24` Bắc Ninh | `09349` Xã Phật Tích | Sườn núi Lạn Kha |
| `temple_north_009` | Chùa Bút Tháp | Thôn Bút Tháp, Xã Đình Tổ, Thị xã Thuận Thành, Tỉnh Bắc Ninh | `24` Bắc Ninh | `09427` Phường Trí Quả | Thôn Bút Tháp |
| `temple_north_010` | Chùa Bái Đính | Xã Gia Sinh, Huyện Gia Viễn, Tỉnh Ninh Bình | `37` Ninh Bình | `14533` Phường Tây Hoa Lư | *(trống)* |
| `temple_north_011` | Chùa Bích Động | Xã Ninh Hải, Huyện Hoa Lư, Tỉnh Ninh Bình | `37` Ninh Bình | `14359` Phường Nam Hoa Lư | *(trống)* |
| `temple_north_012` | Chùa Đồng Yên Tử | Đỉnh Yên Tử, Xã Thượng Yên Công, TP. Uông Bí, Tỉnh Quảng Ninh | `22` Quảng Ninh | `06832` Phường Yên Tử | Đỉnh Yên Tử |
| `temple_north_013` | Chùa Ba Vàng | Phường Quang Trung, TP. Uông Bí, Tỉnh Quảng Ninh | `22` Quảng Ninh | `06811` Phường Uông Bí | *(trống)* |
| `temple_north_014` | Chùa Cái Bầu (Thiền viện Trúc Lâm Giác Tâm) | Thôn 1, Xã Hạ Long, Huyện Vân Đồn, Tỉnh Quảng Ninh | `22` Quảng Ninh | `06994` Đặc khu Vân Đồn | Thôn 1 |
| `temple_north_016` | Chùa Bổ Đà | Xã Tiên Sơn, Thị xã Việt Yên, Tỉnh Bắc Giang | `24` Bắc Ninh | `07798` Phường Vân Hà | *(trống)* |
| `temple_north_019` | Thiền viện Trúc Lâm Tây Thiên | Xã Đại Đình, Huyện Tam Đảo, Tỉnh Vĩnh Phúc | `25` Phú Thọ | `08923` Xã Đại Đình | *(trống)* |
| `temple_central_006` | Chùa Tam Thai | Ngọn Thủy Sơn, Phường Hòa Hải, Quận Ngũ Hành Sơn, TP. Đà Nẵng | `48` Đà Nẵng | `20285` Phường Ngũ Hành Sơn | Ngọn Thủy Sơn |
| `temple_central_007` | Chùa Cầu Hội An | Đường Nguyễn Thị Minh Khai, Phường Minh An, TP. Hội An, Tỉnh Quảng Nam | `48` Đà Nẵng | `20410` Phường Hội An | Đường Nguyễn Thị Minh Khai |
| `temple_central_009` | Chùa Thiên Ấn | Đỉnh núi Thiên Ấn, Xã Tịnh Ấn Đông, TP. Quảng Ngãi, Tỉnh Quảng Ngãi | `51` Quảng Ngãi | `21172` Phường Trương Quang Trọng | Đỉnh núi Thiên Ấn |
| `temple_central_010` | Chùa Thập Tháp Di Đà | Thôn Vạn Thuận, Phường Nhơn Thành, Thị xã An Nhơn, Tỉnh Bình Định | `52` Gia Lai | `21925` Phường An Nhơn Bắc | Thôn Vạn Thuận |
| `temple_central_013` | Chùa Từ Vân | Đường 3/4, Phường Cam Linh, TP. Cam Ranh, Tỉnh Khánh Hòa | `56` Khánh Hòa | `22432` Phường Cam Linh | Đường 3/4 |
| `temple_central_015` | Chùa Núi Tà Cú | Đỉnh núi Tà Cú, Thị trấn Thuận Nam, Huyện Hàm Thuận Nam, Tỉnh Bình Thuận | `68` Lâm Đồng | `23110` Xã Hàm Thuận Nam | Đỉnh núi Tà Cú |
| `temple_central_017` | Chùa Hương Tích (Hà Tĩnh) | Núi Hồng Lĩnh, Xã Thiên Lộc, Huyện Can Lộc, Tỉnh Hà Tĩnh | `42` Hà Tĩnh | `18406` Xã Can Lộc | Núi Hồng Lĩnh |
| `temple_central_018` | Chùa Linh Phước | 120 Tự Phước, Trại Mát, Phường 11, TP. Đà Lạt, Tỉnh Lâm Đồng | `68` Lâm Đồng | `24805` Phường Xuân Trường - Đà Lạt | 120 Tự Phước, Trại Mát |
| `temple_central_019` | Thiền Viện Trúc Lâm Đà Lạt | Đường Hoa Cẩm Tú Cầu, Phường 3, TP. Đà Lạt, Tỉnh Lâm Đồng | `68` Lâm Đồng | `24781` Phường Xuân Hương - Đà Lạt | Đường Hoa Cẩm Tú Cầu |
| `temple_south_003` | Chùa Xá Lợi | 89 Bà Huyện Thanh Quan, Phường Võ Thị Sáu, Quận 3, TP.HCM | `79` Hồ Chí Minh | `27139` Phường Xuân Hòa | 89 Bà Huyện Thanh Quan |
| `temple_south_004` | Chùa Bà Thiên Hậu | 710 Nguyễn Trãi, Phường 11, Quận 5, TP.HCM | `79` Hồ Chí Minh | `27343` Phường Chợ Lớn | 710 Nguyễn Trãi |
| `temple_south_005` | Chùa Bửu Long | 81 Nguyễn Xiển, Phường Long Bình, TP. Thủ Đức, TP.HCM | `79` Hồ Chí Minh | `26833` Phường Long Bình | 81 Nguyễn Xiển |
| `temple_south_006` | Chùa Phổ Quang | 64 Huỳnh Lan Khanh, Phường 2, Quận Tân Bình, TP.HCM | `79` Hồ Chí Minh | `26977` Phường Tân Sơn Hòa | 64 Huỳnh Lan Khanh |
| `temple_south_007` | Chùa Hoằng Pháp | Xã Tân Hiệp, Huyện Hóc Môn, TP.HCM | `79` Hồ Chí Minh | `27559` Xã Hóc Môn | *(trống)* |
| `temple_south_008` | Chùa Vĩnh Tràng | Ấp Mỹ An, Xã Mỹ Phong, TP. Mỹ Tho, Tỉnh Tiền Giang | `82` Đồng Tháp | `28273` Phường Mỹ Phong | Ấp Mỹ An |
| `temple_south_010` | Chùa Dơi | Đường Văn Ngọc Chính, Phường 3, TP. Sóc Trăng, Tỉnh Sóc Trăng | `92` Cần Thơ | `31510` Phường Phú Lợi | Đường Văn Ngọc Chính |
| `temple_south_011` | Chùa Chén Kiểu | Quốc lộ 1A, Xã Đại Tâm, Huyện Mỹ Xuyên, Tỉnh Sóc Trăng | `92` Cần Thơ | `31684` Phường Mỹ Xuyên | Quốc lộ 1A |
| `temple_south_012` | Chùa Xiêm Cán | Ấp Biển Tây B, Xã Vĩnh Trạch Đông, TP. Bạc Liêu, Tỉnh Bạc Liêu | `96` Cà Mau | `31840` Phường Hiệp Thành | Ấp Biển Tây B |
| `temple_south_013` | Chùa Hang (An Giang) | Triền núi Sam, Phường Núi Sam, TP. Châu Đốc, Tỉnh An Giang | `91` An Giang | `30325` Phường Vĩnh Tế | Triền núi Sam |
| `temple_south_014` | Chùa Huệ Nghiêm | 220/110/1 Đỗ Năng Tế, Phường An Lạc A, Quận Bình Tân, TP.HCM | `79` Hồ Chí Minh | `27460` Phường An Lạc | 220/110/1 Đỗ Năng Tế |
| `temple_south_017` | Chùa Phật Lớn | Đỉnh núi Cấm, Xã An Hảo, Thị xã Tịnh Biên, Tỉnh An Giang | `91` An Giang | `30538` Xã Núi Cấm | Đỉnh núi Cấm |
| `temple_south_018` | Chùa Sắc Tứ Tam Bảo (Hà Tiên) | Số 75 Phương Thành, Phường Bình San, TP. Hà Tiên, Tỉnh Kiên Giang | `91` An Giang | `30769` Phường Hà Tiên | Số 75 Phương Thành |
| `temple_south_019` | Chùa Phật Học 2 | Phường 8, TP. Sóc Trăng, Tỉnh Sóc Trăng | `92` Cần Thơ | `31507` Phường Sóc Trăng | *(trống)* |
| `temple_north_024` | Chùa Mía | Làng Đông Sàng, Xã Đường Lâm, Thị xã Sơn Tây, Hà Nội | `01` Hà Nội | `09574` Phường Sơn Tây | Làng Đông Sàng |
| `temple_north_038` | Chùa Phổ Minh | Thôn Tức Mặc, Phường Lộc Vượng, TP. Nam Định, Tỉnh Nam Định | `37` Ninh Bình | `13669` Phường Nam Định | Thôn Tức Mặc |
| `temple_north_040` | Chùa Địa Tạng Phi Lai | Thôn Ninh Trung, Xã Liêm Sơn, Huyện Thanh Liêm, Tỉnh Hà Nam | `37` Ninh Bình | `13483` Xã Thanh Bình | Thôn Ninh Trung |
| `temple_central_023` | Thiền viện Vạn Hạnh (Đà Lạt) | 39 Phù Đổng Thiên Vương, Phường 8, TP. Đà Lạt, Tỉnh Lâm Đồng | `68` Lâm Đồng | `24778` Phường Lâm Viên - Đà Lạt | 39 Phù Đổng Thiên Vương |
| `temple_central_027` | Tịnh xá Ngọc Cẩm | Đường Hùng Vương, Phường Cẩm Phô, TP. Hội An, Tỉnh Quảng Nam | `48` Đà Nẵng | `20410` Phường Hội An | Đường Hùng Vương |
| `temple_central_029` | Chùa Ông Núi | Xã Cát Tiến, Huyện Phù Cát, Tỉnh Bình Định | `52` Gia Lai | `21880` Xã Cát Tiến | *(trống)* |
| `temple_central_034` | Chùa Cổ Thạch | Xã Bình Thạnh, Huyện Tuy Phong, Tỉnh Bình Thuận | `68` Lâm Đồng | `22969` Xã Liên Hương | *(trống)* |
| `temple_central_036` | Chùa Cam Lộ | Thị trấn Cam Lộ, Huyện Cam Lộ, Tỉnh Quảng Trị | `44` Quảng Trị | `19597` Xã Cam Lộ | *(trống)* |
| `temple_central_037` | Chùa Tàu Đà Lạt (Thiên Vương Cổ Sát) | 385 Khe Sanh, Phường 10, TP. Đà Lạt, Tỉnh Lâm Đồng | `68` Lâm Đồng | `24781` Phường Xuân Hương - Đà Lạt | 385 Khe Sanh |
| `temple_central_039` | Chùa Bác Ái | Phường Quyết Thắng, TP. Kon Tum, Tỉnh Kon Tum | `51` Quảng Ngãi | `23293` Phường Kon Tum | *(trống)* |
| `temple_central_040` | Chùa Minh Thành | 348 Nguyễn Viết Xuân, Phường Hội Phú, TP. Pleiku, Tỉnh Gia Lai | `52` Gia Lai | `23586` Phường Hội Phú | 348 Nguyễn Viết Xuân |
| `temple_south_022` | Tịnh xá Trung Tâm (Bình Thạnh) | 21 Nguyễn Trung Trực, Phường 5, Quận Bình Thạnh, TP.HCM | `79` Hồ Chí Minh | `26905` Phường Bình Lợi Trung | 21 Nguyễn Trung Trực |
| `temple_south_026` | Chùa Bà Thiên Hậu (Bình Dương) | Số 4 Nguyễn Du, Phường Phú Cường, TP. Thủ Dầu Một, Tỉnh Bình Dương | `79` Hồ Chí Minh | `25747` Phường Thủ Dầu Một | Số 4 Nguyễn Du |
| `temple_south_028` | Thiền viện Thường Chiếu | Quốc lộ 51, Xã Phước Thái, Huyện Long Thành, Tỉnh Đồng Nai | `75` Đồng Nai | `26422` Xã Phước Thái | Quốc lộ 51 |
| `temple_south_034` | Chùa Tôn Thạnh | Ấp Thanh Ba, Xã Mỹ Lộc, Huyện Cần Giuộc, Tỉnh Long An | `80` Tây Ninh | `28177` Xã Mỹ Lộc | Ấp Thanh Ba |
| `temple_south_035` | Chùa Ang | Phường 8, TP. Trà Vinh, Tỉnh Trà Vinh | `86` Vĩnh Long | `29254` Phường Nguyệt Hóa | *(trống)* |
| `temple_south_036` | Chùa Hang (Trà Vinh) | Khóm 4, Thị trấn Châu Thành, Huyện Châu Thành, Tỉnh Trà Vinh | `86` Vĩnh Long | `29374` Xã Châu Thành | Khóm 4 |
| `temple_south_037` | Thiền viện Trúc Lâm Phương Nam | Ấp Mỹ Nhơn, Xã Mỹ Khánh, Huyện Phong Điền, TP. Cần Thơ | `92` Cần Thơ | `31150` Phường An Bình | Ấp Mỹ Nhơn |
| `temple_north_044` | Chùa Hang (Thái Nguyên) | Phường Chùa Hang, TP. Thái Nguyên, Tỉnh Thái Nguyên | `19` Thái Nguyên | `05710` Phường Linh Sơn | *(trống)* |
| `temple_north_050` | Chùa Hàm Long | Phường Nam Sơn, TP. Bắc Ninh, Tỉnh Bắc Ninh | `24` Bắc Ninh | `09286` Phường Nam Sơn | *(trống)* |
| `temple_north_060` | Chùa Ngũ Xã | 44 Ngũ Xã, Phường Trúc Bạch, Quận Ba Đình, Hà Nội | `01` Hà Nội | `00004` Phường Ba Đình | 44 Ngũ Xã |
| `temple_central_047` | Tịnh thất An Lạc | Xã Cẩm Hà, TP. Hội An, Tỉnh Quảng Nam | `48` Đà Nẵng | `20401` Phường Hội An Tây | *(trống)* |
| `temple_central_054` | Chùa Hoằng Phúc | Thôn Thuận Trạch, Xã Mỹ Thủy, Huyện Lệ Thủy, Tỉnh Quảng Bình | `44` Quảng Trị | `19291` Xã Tân Mỹ | Thôn Thuận Trạch |
| `temple_central_056` | Chùa Linh Sơn (Đà Lạt) | 120 Nguyễn Văn Trỗi, Phường 2, TP. Đà Lạt, Tỉnh Lâm Đồng | `68` Lâm Đồng | `24781` Phường Xuân Hương - Đà Lạt | 120 Nguyễn Văn Trỗi |
| `temple_central_059` | Chùa Linh Phong (Đà Lạt) | Đường Hoàng Hoa Thám, Phường 10, TP. Đà Lạt, Tỉnh Lâm Đồng | `68` Lâm Đồng | `24781` Phường Xuân Hương - Đà Lạt | Đường Hoàng Hoa Thám |
| `temple_south_042` | Chùa Vạn Phật | 66/14 Nghĩa Thục, Phường 5, Quận 5, TP.HCM | `79` Hồ Chí Minh | `27316` Phường An Đông | 66/14 Nghĩa Thục |
| `temple_south_046` | Chùa Nam Nhã | 612 Cách Mạng Tháng 8, Phường An Thới, Quận Bình Thủy, TP. Cần Thơ | `92` Cần Thơ | `31168` Phường Bình Thủy | 612 Cách Mạng Tháng 8 |
| `temple_south_048` | Chùa Giác Sanh | 103 Âu Cơ, Phường 14, Quận 11, TP.HCM | `79` Hồ Chí Minh | `27211` Phường Hòa Bình | 103 Âu Cơ |
| `temple_south_049` | Chùa Hội Khánh | 35 Yersin, Phường Phú Cường, TP. Thủ Dầu Một, Tỉnh Bình Dương | `79` Hồ Chí Minh | `25747` Phường Thủ Dầu Một | 35 Yersin |
| `temple_south_054` | Chùa Tây An | Chân núi Sam, Phường Núi Sam, TP. Châu Đốc, Tỉnh An Giang | `91` An Giang | `30325` Phường Vĩnh Tế | Chân núi Sam |
| `temple_south_055` | Chùa Phước Hưng | 461 Hùng Vương, Phường 1, TP. Sa Đéc, Tỉnh Đồng Tháp | `82` Đồng Tháp | `29905` Phường Sa Đéc | 461 Hùng Vương |
