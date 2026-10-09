-- Catalogue images: store a root-relative path, not an absolute URL.
--
-- Seed data used to carry http://<dev machine IP>:8080/static/images/... , so every
-- change of network (new IP) turned each stored image_url / thumbnail_url into a dead
-- link. The API now prepends server.public_base_url (or the request's own origin when
-- empty) to any value starting with "/", so the stored value is just
-- "/static/images/<kind>/<size>/<file>" and survives a new host, network or CDN.
--
-- Only values that are an http(s) URL of a "/static/images/" path are rewritten; a
-- hotlink to another site (e.g. an article's source image) is left alone, and an
-- already relative value does not match, so this is idempotent like every other file.
-- Down: not reversible (the original host is not recorded) and not needed, since an
-- absolute URL is still returned unchanged by the API.
UPDATE temples
SET image_url     = regexp_replace(image_url,     '^https?://[^/]+(/static/images/.*)$', '\1'),
    thumbnail_url = regexp_replace(thumbnail_url, '^https?://[^/]+(/static/images/.*)$', '\1')
WHERE image_url ~ '^https?://[^/]+/static/images/' OR thumbnail_url ~ '^https?://[^/]+/static/images/';

UPDATE deities
SET image_url     = regexp_replace(image_url,     '^https?://[^/]+(/static/images/.*)$', '\1'),
    thumbnail_url = regexp_replace(thumbnail_url, '^https?://[^/]+(/static/images/.*)$', '\1')
WHERE image_url ~ '^https?://[^/]+/static/images/' OR thumbnail_url ~ '^https?://[^/]+/static/images/';

UPDATE gallery_images
SET image_url     = regexp_replace(image_url,     '^https?://[^/]+(/static/images/.*)$', '\1'),
    thumbnail_url = regexp_replace(thumbnail_url, '^https?://[^/]+(/static/images/.*)$', '\1')
WHERE image_url ~ '^https?://[^/]+/static/images/' OR thumbnail_url ~ '^https?://[^/]+/static/images/';

UPDATE articles
SET image_url     = regexp_replace(image_url,     '^https?://[^/]+(/static/images/.*)$', '\1'),
    thumbnail_url = regexp_replace(thumbnail_url, '^https?://[^/]+(/static/images/.*)$', '\1')
WHERE image_url ~ '^https?://[^/]+/static/images/' OR thumbnail_url ~ '^https?://[^/]+/static/images/';
