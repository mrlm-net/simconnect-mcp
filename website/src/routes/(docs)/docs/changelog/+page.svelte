<script lang="ts">
	import SeoHead from '$lib/components/layout/SeoHead.svelte';
	import type { PageData } from './$types.js';
	import type { ChangelogGroup } from '$lib/content/changelog.server.js';

	let { data }: { data: PageData } = $props();

	const groups: ChangelogGroup[] = $derived(data.groups);

	let activeMinor = $state(groups[0]?.minor ?? '');

	const activeGroup: ChangelogGroup | undefined = $derived(
		groups.find((g) => g.minor === activeMinor)
	);
</script>

<SeoHead
	siteConfig={data.siteConfig}
	title="Changelog - SimConnect MCP"
	description="Release history for SimConnect MCP."
	path="/docs/changelog"
/>

<article class="flex-1 min-w-0 px-6 py-10">
	<h1 class="text-3xl font-bold mb-2" style="color: var(--color-text-primary);">Changelog</h1>
	<p class="mb-8 text-sm" style="color: var(--color-text-secondary);">
		All notable changes to SimConnect MCP.
	</p>

	<!-- Tab bar -->
	<div class="flex gap-1 border-b mb-8 overflow-x-auto" style="border-color: var(--color-border);">
		{#each groups as group (group.minor)}
			<button
				onclick={() => (activeMinor = group.minor)}
				class="px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors border-b-2 -mb-px"
				style="
					color: {activeMinor === group.minor
					? 'var(--color-link)'
					: 'var(--color-text-secondary)'};
					border-color: {activeMinor === group.minor ? 'var(--color-link)' : 'transparent'};
					background: transparent;
				"
				aria-selected={activeMinor === group.minor}
				role="tab"
			>
				{group.label}
			</button>
		{/each}
	</div>

	<!-- Tab panel -->
	{#if activeGroup}
		<div role="tabpanel" class="space-y-10">
			{#each activeGroup.releases as release (release.version)}
				<section>
					<div class="flex items-baseline gap-3 mb-4">
						<h2 class="text-xl font-semibold" style="color: var(--color-text-primary);">
							v{release.version}
						</h2>
						<time
							class="text-xs font-mono"
							style="color: var(--color-text-muted);"
							datetime={release.date}
						>
							{release.date}
						</time>
					</div>

					<div class="prose max-w-none" style="color: var(--color-text-primary);">
						{@html release.renderedBody}
					</div>

					<div class="mt-6 border-b" style="border-color: var(--color-border);"></div>
				</section>
			{/each}
		</div>
	{/if}
</article>
