// Sinh trang tham chiếu từ nguồn duy nhất trong docs/ (nằm trong git) sang
// src/content/docs/reference/. Thư mục đích là file sinh ra: bị xoá sạch và tạo lại
// mỗi lần chạy, đã .gitignore, không sửa tay.
//   docs/specs/features/<f>/design.md -> reference/nghiep-vu/<f>/design.md
//   docs/specs/features/<f>/flow.md   -> reference/nghiep-vu/<f>/flow.md
//   docs/knowledge/features/*.md      -> reference/okf/features/*.md
//   docs/knowledge/packages/*.md      -> reference/okf/packages/*.md
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, relative, resolve } from 'node:path';

const REPO = resolve('../..');
const OUT = resolve('src/content/docs/reference');

const NOTICE =
	':::caution[Trang sinh tự động]\nNội dung được sinh từ `%SRC%` khi build. Đừng sửa ở đây — hãy sửa file nguồn trong `docs/`.\n:::\n';

// Bảng ánh xạ file nguồn (đường dẫn tương đối repo) -> URL trên web, để viết lại link .md.
const urlMap = new Map();
const pages = [];

function walkBusiness() {
	const base = join(REPO, 'docs/specs/features');
	for (const f of readdirSync(base).sort()) {
		for (const doc of ['design', 'flow']) {
			const src = join(base, f, `${doc}.md`);
			if (existsSync(src)) pages.push({ kind: 'nghiep-vu', feature: f, doc, slug: `${f}/${doc}`, src });
		}
	}
}
function walkOkf(kind) {
	const base = join(REPO, 'docs/knowledge', kind);
	if (!existsSync(base)) return;
	for (const n of readdirSync(base).sort()) {
		if (n.endsWith('.md')) pages.push({ kind, slug: n.slice(0, -3), src: join(base, n) });
	}
}
walkBusiness();
walkOkf('features');
walkOkf('packages');
const dirOf = (p) => (p.kind === 'nghiep-vu' ? 'nghiep-vu' : `okf/${p.kind}`);
for (const p of pages) urlMap.set(p.src, `/reference/${dirOf(p)}/${p.slug}/`);

function splitFrontmatter(text) {
	const m = text.match(/^---\n([\s\S]*?)\n---\n?/);
	return m ? { fm: m[1], body: text.slice(m[0].length) } : { fm: '', body: text };
}
function fmField(fm, key) {
	const m = fm.match(new RegExp(`^${key}:\\s*(.+)$`, 'm'));
	return m ? m[1].trim().replace(/^["']|["']$/g, '') : '';
}
const yamlStr = (s) => JSON.stringify(s);

// Link tương đối tới .md nguồn -> URL web; link tới file khác (code, spec) -> chữ thường dạng code.
function rewriteLinks(body, srcFile) {
	return body.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (all, text, href) => {
		if (/^(https?:|mailto:|#)/.test(href)) return all;
		const [path, hash = ''] = href.split('#');
		const target = resolve(dirname(srcFile), path);
		const url = urlMap.get(target);
		if (url) return `[${text}](${url}${hash ? '#' + hash : ''})`;
		return `${text}`;
	});
}

// Starlight đã render title thành H1 -> bỏ H1 đầu tiên của nguồn để khỏi lặp.
function stripFirstH1(body) {
	return body.replace(/^\s*# .+\n/, '');
}

rmSync(OUT, { recursive: true, force: true });

for (const p of pages) {
	const text = readFileSync(p.src, 'utf8');
	const { fm, body: raw } = splitFrontmatter(text);
	const relSrc = relative(REPO, p.src);

	let title, description, extra = '';
	if (p.kind === 'nghiep-vu') {
		const label = p.doc === 'design' ? 'Design' : 'Flow';
		title = `${label}: ${p.feature}`;
		description = `${label} của feature ${p.feature} (sinh từ spec).`;
	} else {
		title = fmField(fm, 'title') || p.slug;
		description = fmField(fm, 'description');
		const status = fmField(fm, 'status');
		if (status) extra = `**Trạng thái:** \`${status}\` · **Code:** \`${fmField(fm, 'resource')}\`\n\n`;
	}

	const out = join(OUT, dirOf(p), `${p.slug}.md`);
	mkdirSync(dirname(out), { recursive: true });
	const content =
		`---\ntitle: ${yamlStr(title)}\ndescription: ${yamlStr(description)}\n` +
		(p.kind === 'nghiep-vu'
			? `sidebar:\n  label: ${p.doc === 'design' ? 'Design' : 'Flow'}\n  order: ${p.doc === 'design' ? 1 : 2}\n`
			: '') +
		`---\n\n` +
		NOTICE.replace('%SRC%', relSrc) +
		'\n' +
		extra +
		rewriteLinks(stripFirstH1(raw), p.src);
	writeFileSync(out, content);
}

console.log(`sync-docs: ${pages.length} trang -> ${relative(process.cwd(), OUT)}`);
