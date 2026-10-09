// Chép nguồn openapi duy nhất (docs/specs/api/openapi.yaml, nằm trong git) vào
// public/ để trang /api-reference/ và link tải file phục vụ được. Bản trong public/
// là file sinh ra, không sửa tay (đã bị .gitignore).
import { copyFileSync } from 'node:fs';

copyFileSync('../../docs/specs/api/openapi.yaml', 'public/openapi.yaml');
