<script lang="ts">
	import { base } from '$app/paths';
	import type { PageData } from './$types.js';
	import type { DocPage } from '$lib/types/index.js';
	import SeoHead from '$lib/components/layout/SeoHead.svelte';
	import JsonLd from '$lib/components/layout/JsonLd.svelte';
	import TableOfContents from '$lib/components/layout/TableOfContents.svelte';

	let { data }: { data: PageData } = $props();

	const doc: DocPage = $derived(data.doc);

	const jsonLdSchema = $derived({
		'@context': 'https://schema.org',
		'@type': 'TechArticle',
		headline: doc.title,
		description: doc.description,
		url: `${data.siteConfig.url}${base}/docs/${doc.slug}`
	});
</script>

<SeoHead
	siteConfig={data.siteConfig}
	title="{doc.title} - SimConnect MCP"
	description={doc.description}
	path="/docs/{doc.slug}"
/>

<JsonLd schema={jsonLdSchema} />

<div class="flex">
	<article
		class="flex-1 min-w-0 px-6 py-10 prose max-w-none"
		style="color: var(--color-text-primary);"
	>
		<h1>{doc.title}</h1>

		{@html doc.renderedContent}

		<nav class="mt-12 flex justify-between gap-4 border-t pt-6 not-prose text-sm" style="border-color: var(--color-border);">
			<div>
				{#if data.prev}
					<a
						href="{base}/docs/{data.prev.slug}"
						class="flex items-center gap-2 transition-colors"
						style="color: var(--color-link);"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="16"
							height="16"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
						>
							<polyline points="15 18 9 12 15 6" />
						</svg>
						<span>{data.prev.title}</span>
					</a>
				{/if}
			</div>
			<div>
				{#if data.next}
					<a
						href="{base}/docs/{data.next.slug}"
						class="flex items-center gap-2 transition-colors"
						style="color: var(--color-link);"
					>
						<span>{data.next.title}</span>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							width="16"
							height="16"
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
							stroke-linecap="round"
							stroke-linejoin="round"
							aria-hidden="true"
						>
							<polyline points="9 18 15 12 9 6" />
						</svg>
					</a>
				{/if}
			</div>
		</nav>
	</article>

	{#if doc.headings.length > 0}
		<aside class="hidden xl:block sticky top-16 h-[calc(100vh-4rem)] overflow-y-auto shrink-0">
			<TableOfContents headings={doc.headings} />
		</aside>
	{/if}
</div>
