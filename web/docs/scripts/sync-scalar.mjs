// Chép bundle Scalar từ node_modules vào public/ để trang /api-reference/ tải
// từ cùng origin thay vì CDN (không cần SRI, chạy được offline).
import { copyFileSync, mkdirSync } from 'node:fs';

mkdirSync('public/vendor', { recursive: true });
copyFileSync(
	'node_modules/@scalar/api-reference/dist/browser/standalone.js',
	'public/vendor/scalar-api-reference.js',
);
