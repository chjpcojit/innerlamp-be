-- Temples: move province names to the 34 province-level units in force since
-- 1 July 2025. Spec: docs/specs/features/temple/ (v1.6, US-TMP-07, AC 30).
--
-- Nghị quyết 202/2025/QH15 (12 June 2025) merged 63 provinces/cities into 34. Each
-- old unit went whole into exactly one new unit (none was split), so a temple's new
-- province follows from its old province name alone. This file rewrites
-- temples.province from the old name to the new one. It does NOT touch temples.address
-- — that stays the display string it was (it still names the old province and
-- district); updating addresses is a separate data job.
--
-- Only names that CHANGE are listed (old names merged away, plus "Thừa Thiên Huế" →
-- "Huế"). Names that survive (Hà Nội, Ninh Bình, Gia Lai, ...) are not keys, so the
-- statement never rewrites a new-form name and re-applying it changes nothing.
-- A value outside both the old and the new set is left as it is; list such rows with
--   SELECT id, province FROM temples
--    WHERE province NOT IN (<the 34 names in services/appService/domain/province.go>);
--
-- The seeder validates against the same 34 names (domain.IsProvince) and overwrites
-- province from assets/data/temple.json, so on a freshly seeded database this file is
-- a no-op. It exists for databases seeded before the 34-unit list.
--
-- Depends on 0024 (temples.province). Idempotent like every other file here.
UPDATE temples t
   SET province = m.new_name
  FROM (VALUES
        ('Kiên Giang', 'An Giang'),
        ('Bắc Giang', 'Bắc Ninh'),
        ('Bạc Liêu', 'Cà Mau'),
        ('Hậu Giang', 'Cần Thơ'),
        ('Sóc Trăng', 'Cần Thơ'),
        ('Bình Định', 'Gia Lai'),
        ('Thừa Thiên Huế', 'Huế'),
        ('Thái Bình', 'Hưng Yên'),
        ('Hải Dương', 'Hải Phòng'),
        ('Bình Dương', 'Hồ Chí Minh'),
        ('Ninh Thuận', 'Khánh Hòa'),
        ('Yên Bái', 'Lào Cai'),
        ('Bình Thuận', 'Lâm Đồng'),
        ('Đắk Nông', 'Lâm Đồng'),
        ('Hà Nam', 'Ninh Bình'),
        ('Nam Định', 'Ninh Bình'),
        ('Hòa Bình', 'Phú Thọ'),
        ('Vĩnh Phúc', 'Phú Thọ'),
        ('Kon Tum', 'Quảng Ngãi'),
        ('Quảng Bình', 'Quảng Trị'),
        ('Bắc Kạn', 'Thái Nguyên'),
        ('Hà Giang', 'Tuyên Quang'),
        ('Long An', 'Tây Ninh'),
        ('Bến Tre', 'Vĩnh Long'),
        ('Trà Vinh', 'Vĩnh Long'),
        ('Quảng Nam', 'Đà Nẵng'),
        ('Phú Yên', 'Đắk Lắk'),
        ('Bình Phước', 'Đồng Nai'),
        ('Tiền Giang', 'Đồng Tháp')
       ) AS m(old_name, new_name)
 WHERE t.province = m.old_name;

-- Bà Rịa - Vũng Tàu is written with a hyphen, en dash or spaces depending on the
-- source, so it is matched by prefix rather than listed above. It became part of
-- Hồ Chí Minh.
UPDATE temples
   SET province = 'Hồ Chí Minh'
 WHERE province ~ '^Bà Rịa';
