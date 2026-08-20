-- US-002: Đăng nhập (login) — see docs/db/register-erd.md and US-002 AC4.
--
-- Login upserts the user_devices row for (user_id, device_id) so repeat
-- logins from the same device refresh its metadata instead of creating
-- duplicate device rows. That requires a uniqueness guarantee to upsert
-- against, which the original migration didn't need (register only ever
-- inserts a device once).

ALTER TABLE user_devices ADD CONSTRAINT user_devices_user_id_device_id_key UNIQUE (user_id, device_id);
