-- US-004: Cập nhật hồ sơ cá nhân (avatar field, AC3).

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
