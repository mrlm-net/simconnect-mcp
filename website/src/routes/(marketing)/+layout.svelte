<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { SiteConfig } from '$lib/types/index.js';
	import Header from '$lib/components/layout/Header.svelte';
	import Footer from '$lib/components/layout/Footer.svelte';

	let {
		data,
		children
	}: {
		data: { siteConfig: SiteConfig };
		children: Snippet;
	} = $props();
</script>

<Header siteConfig={data.siteConfig} onToggleSidebar={() => {}} showMenuButton={false} />

<main class="flex-1 pt-16">
	{@render children()}
</main>

<Footer siteConfig={data.siteConfig} />
