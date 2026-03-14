<script lang="ts">
	import { page } from '$app/state';
	import { base } from '$app/paths';
	import Header from '$lib/components/layout/Header.svelte';
	import Footer from '$lib/components/layout/Footer.svelte';
	import { siteConfig } from '$lib/config/site.js';

	const messages404 = [
		{ code: 'TERRAIN PULL UP', body: 'No route found at this altitude. Check your charts.' },
		{ code: 'SQUAWK 7700', body: 'This page is a navigation emergency. Declare intentions.' },
		{ code: 'GO AROUND', body: 'Page not stabilised by 500 ft AGL. Climb and try again.' },
		{ code: 'TCAS ADVISORY', body: 'Descend immediately — this URL does not exist.' },
		{ code: 'NOTAM ACTIVE', body: 'The requested route is closed until further notice.' },
		{ code: 'SIMCONNECT_EXCEPTION_URL_NOT_FOUND', body: 'Probably.' },
	];

	const picked = messages404[Math.floor(Math.random() * messages404.length)];

	const is404 = page.status === 404;
</script>

<svelte:head>
	<title>{page.status} — SimConnect MCP</title>
</svelte:head>

<Header {siteConfig} onToggleSidebar={() => {}} showMenuButton={false} />

<main
	id="main-content"
	tabindex="-1"
	class="flex flex-1 flex-col items-center justify-center px-6 pt-16 pb-24 text-center"
>
	<!-- Status badge -->
	<div
		class="mb-6 inline-flex items-center gap-2 rounded-full border px-3.5 py-1 text-xs font-medium"
		style="background-color: var(--color-bg-secondary); border-color: var(--color-border); color: var(--color-text-muted);"
	>
		<span class="inline-block h-2 w-2 rounded-full" style="background-color: #f85149;"></span>
		HTTP {page.status}
	</div>

	<!-- Big status number -->
	<p
		class="mb-2 font-mono text-8xl font-bold tracking-tight"
		style="color: var(--color-text-primary);"
	>
		{page.status}
	</p>

	{#if is404}
		<!-- Aviation-themed message -->
		<p
			class="mb-3 font-mono text-sm font-semibold uppercase tracking-widest"
			style="color: #f85149;"
		>
			{picked.code}
		</p>
		<p class="mb-8 max-w-md text-base" style="color: var(--color-text-secondary);">
			{picked.body}
		</p>
	{:else}
		<p class="mb-3 text-xl font-semibold" style="color: var(--color-text-primary);">
			{page.error?.message ?? 'Something went wrong'}
		</p>
		<p class="mb-8 max-w-md text-base" style="color: var(--color-text-secondary);">
			An unexpected error occurred. Please try again or file an issue if this persists.
		</p>
	{/if}

	<!-- Actions -->
	<div class="flex flex-wrap justify-center gap-3">
		<a
			href="{base}/"
			class="rounded-md border px-4 py-2 text-sm font-medium transition-colors"
			style="background-color: var(--color-bg-secondary); border-color: var(--color-border); color: var(--color-text-primary);"
		>
			← Back to home
		</a>
		<a
			href="{base}/docs/getting-started/"
			class="rounded-md px-4 py-2 text-sm font-medium transition-colors"
			style="background-color: #1f6feb; color: #ffffff;"
		>
			Getting started
		</a>
	</div>
</main>

<Footer {siteConfig} />
