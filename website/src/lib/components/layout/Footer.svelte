<script lang="ts">
	import type { SiteConfig } from '$lib/types/index.js';

	let { siteConfig }: { siteConfig: SiteConfig } = $props();
</script>

<footer
	class="border-t px-6 py-5"
	style="background-color: var(--color-bg-primary); border-color: var(--color-border);"
>
	<div class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-4 text-sm" style="color: var(--color-text-muted);">
		<span>SimConnect MCP &mdash; MIT License &mdash; &copy; 2024&ndash;2026 mrlm-net</span>
		<nav class="flex items-center gap-4" aria-label="Footer navigation">
			<a
				href="https://github.com/mrlm-net/simconnect-mcp"
				target="_blank"
				rel="noopener noreferrer"
				class="footer-link transition-colors"
				style="color: var(--color-text-muted);"
			>
				GitHub
			</a>
			<a
				href="https://github.com/mrlm-net/simconnect-mcp/releases"
				target="_blank"
				rel="noopener noreferrer"
				class="footer-link transition-colors"
				style="color: var(--color-text-muted);"
			>
				Releases
			</a>
			<a
				href="https://github.com/mrlm-net/simconnect-mcp/issues"
				target="_blank"
				rel="noopener noreferrer"
				class="footer-link transition-colors"
				style="color: var(--color-text-muted);"
			>
				Issues
			</a>
		</nav>
	</div>
</footer>

<style>
	.footer-link:hover {
		color: var(--color-text-primary);
	}
</style>
