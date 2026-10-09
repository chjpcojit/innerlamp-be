// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import mermaid from 'astro-mermaid';

// https://astro.build/config
export default defineConfig({
	server: { port: 8081 },
	integrations: [
		// Phải đứng trước starlight để xử lý các khối ```mermaid.
		mermaid({ autoTheme: true }),
		starlight({
			title: 'InnerLamp API',
			description: 'Tài liệu tích hợp API InnerLamp cho client và admin.',
			defaultLocale: 'root',
			locales: { root: { label: 'Tiếng Việt', lang: 'vi' } },
			sidebar: [
				{
					label: 'Bắt đầu',
					items: [
						{ slug: 'guides/getting-started' },
						{ slug: 'guides/authentication' },
						{ slug: 'guides/error-handling' },
					],
				},
				{
					label: 'Hướng dẫn tích hợp',
					items: [
						{ slug: 'guides/register-and-login' },
						{ slug: 'guides/attestation' },
					],
				},
				{
					label: 'Sổ giỗ chạp',
					items: [{ slug: 'guides/catalogues/memorials' }],
				},
				{
					label: 'Danh mục nội dung',
					items: [
						{ slug: 'guides/catalogues/temples' },
						{ slug: 'guides/catalogues/deities' },
						{ slug: 'guides/catalogues/articles' },
						{ slug: 'guides/catalogues/sutras' },
						{ slug: 'guides/catalogues/dharma-talks' },
						{ slug: 'guides/catalogues/gallery' },
						{ slug: 'guides/catalogues/calendar' },
					],
				},
				{
					label: 'Kiến trúc',
					items: [
						{ slug: 'architecture/overview' },
						{ slug: 'architecture/request-pipeline' },
						{ slug: 'architecture/security' },
					],
				},
				{
					label: 'Nghiệp vụ',
					collapsed: true,
					items: [{ autogenerate: { directory: 'reference/nghiep-vu', collapsed: true } }],
				},
				{
					label: 'OKF',
					collapsed: true,
					items: [
						{ label: 'Feature', collapsed: true, items: [{ autogenerate: { directory: 'reference/okf/features' } }] },
						{ label: 'Package', collapsed: true, items: [{ autogenerate: { directory: 'reference/okf/packages' } }] },
					],
				},
				{
					label: 'Tham chiếu',
					items: [
						{ label: 'API Reference (Scalar)', link: '/api-reference/' },
						{ label: 'openapi.yaml', link: '/openapi.yaml', attrs: { target: '_blank' } },
						{ slug: 'changelog' },
					],
				},
			],
		}),
	],
});
