<script lang="ts">
	import { base } from '$app/paths';

	let copied = $state(false);
	const installCommand = 'go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest';

	async function copyInstall() {
		try {
			await navigator.clipboard.writeText(installCommand);
			copied = true;
			setTimeout(() => {
				copied = false;
			}, 2000);
		} catch {
			// clipboard not available
		}
	}
</script>

<svelte:head>
	<title>SimConnect MCP — Model Context Protocol for Microsoft Flight Simulator</title>
	<meta
		name="description"
		content="Model Context Protocol server for Microsoft Flight Simulator — query SimConnect SDK docs and read live simulator data from Claude, Copilot, or any MCP-compatible AI."
	/>
</svelte:head>

<!-- Hero -->
<section
	class="flex flex-col items-center justify-center px-6 pt-28 pb-20 text-center"
	style="background-color: var(--color-bg-primary);"
>
	<!-- Badge pill above title -->
	<div
		class="mb-6 inline-flex items-center gap-2 rounded-full border px-3.5 py-1 text-xs font-medium"
		style="background-color: var(--color-bg-secondary); border-color: var(--color-border); color: var(--color-text-muted);"
	>
		<span
			class="inline-block h-2 w-2 rounded-full"
			style="background-color: #3fb950;"
			aria-hidden="true"
		></span>
		Go 1.25+ &middot; MCP Protocol &middot; MSFS 2020 &amp; 2024
	</div>

	<!-- Title -->
	<h1
		class="mb-5 text-5xl font-bold tracking-tight sm:text-6xl"
		style="color: var(--color-text-primary);"
	>
		SimConnect <span style="color: var(--color-link);">MCP</span>
	</h1>

	<!-- Subtitle -->
	<p
		class="mb-10 max-w-2xl text-base leading-relaxed sm:text-lg"
		style="color: var(--color-text-secondary);"
	>
		Model Context Protocol server for Microsoft Flight Simulator &mdash; query SimConnect SDK docs
		and read live simulator data from Claude, Copilot, or any MCP-compatible AI.
	</p>

	<!-- CTA buttons -->
	<div class="mb-8 flex flex-wrap items-center justify-center gap-3">
		<a
			href="{base}/docs/getting-started"
			class="inline-flex items-center gap-2 rounded-md px-6 py-2.5 text-sm font-semibold transition-opacity hover:opacity-90"
			style="background-color: var(--color-link); color: #fff;"
		>
			Get Started
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="14"
				height="14"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2.5"
				stroke-linecap="round"
				stroke-linejoin="round"
				aria-hidden="true"
			>
				<line x1="5" y1="12" x2="19" y2="12" />
				<polyline points="12 5 19 12 12 19" />
			</svg>
		</a>
		<a
			href="https://github.com/mrlm-net/simconnect-mcp"
			target="_blank"
			rel="noopener noreferrer"
			class="inline-flex items-center gap-2 rounded-md border px-6 py-2.5 text-sm font-semibold transition-colors"
			style="border-color: var(--color-border); color: var(--color-text-primary); background-color: transparent;"
			onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-text-secondary)'; }}
			onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="16"
				height="16"
				viewBox="0 0 24 24"
				fill="currentColor"
				aria-hidden="true"
			>
				<path
					d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"
				/>
			</svg>
			View on GitHub
		</a>
	</div>

	<!-- Install command block -->
	<div
		class="mb-5 flex w-full max-w-xl items-center justify-between gap-3 rounded-lg border px-4 py-3"
		style="background-color: var(--color-bg-code); border-color: var(--color-border);"
	>
		<code class="flex-1 overflow-x-auto text-left text-sm" style="font-family: var(--font-mono); color: var(--color-text-secondary);">
			<span style="color: var(--color-text-muted);">$</span>
			<span style="color: var(--color-text-primary);"> {installCommand}</span>
		</code>
		<button
			onclick={copyInstall}
			class="ml-2 shrink-0 cursor-pointer rounded p-1.5 transition-colors"
			style="color: {copied ? '#3fb950' : 'var(--color-text-muted)'};"
			aria-label="Copy install command"
			title="Copy to clipboard"
		>
			{#if copied}
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
					<polyline points="20 6 9 17 4 12" />
				</svg>
			{:else}
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
					<rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
					<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
				</svg>
			{/if}
		</button>
	</div>

	<!-- Badge links -->
	<div class="flex flex-wrap items-center justify-center gap-2">
		<a
			href="{base}/docs/getting-started"
			class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium transition-opacity hover:opacity-80"
			style="background-color: #1a3a5c; color: #58a6ff; border: 1px solid #1f6feb;"
		>
			docs mode
		</a>
		<a
			href="{base}/docs/getting-started"
			class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium transition-opacity hover:opacity-80"
			style="background-color: #1a3a2a; color: #3fb950; border: 1px solid #2ea043;"
		>
			simconnect mode
		</a>
		<a
			href="https://github.com/mrlm-net/simconnect-mcp/blob/main/LICENSE"
			target="_blank"
			rel="noopener noreferrer"
			class="inline-flex items-center rounded-full px-3 py-0.5 text-xs font-medium transition-opacity hover:opacity-80"
			style="background-color: var(--color-bg-tertiary); color: var(--color-text-muted); border: 1px solid var(--color-border);"
		>
			Apache 2.0
		</a>
	</div>
</section>

<!-- Features section -->
<section
	class="px-6 pb-24 pt-16"
	style="background-color: var(--color-bg-primary);"
>
	<div class="mx-auto max-w-5xl">
		<!-- Section label -->
		<p
			class="mb-3 text-center text-xs font-semibold uppercase tracking-widest"
			style="color: var(--color-text-muted);"
		>
			Why SimConnect MCP
		</p>
		<h2
			class="mb-14 text-center text-2xl font-bold tracking-tight"
			style="color: var(--color-text-primary);"
		>
			Everything you need to work with MSFS from your AI
		</h2>

		<!-- 3-col feature grid -->
		<div class="grid gap-x-10 gap-y-12 sm:grid-cols-2 lg:grid-cols-3">

			<!-- 1. Documentation Mode -->
			<div>
				<div class="mb-3 flex items-center gap-2.5">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-link); flex-shrink: 0;"
						aria-hidden="true"
					>
						<path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
						<path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Documentation Mode</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Query the full SimConnect SDK reference — simulation variables, events, API functions,
					and data structures — from any MCP client.
				</p>
			</div>

			<!-- 2. Live Simulator Data -->
			<div>
				<div class="mb-3 flex items-center gap-2.5">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-link); flex-shrink: 0;"
						aria-hidden="true"
					>
						<polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Live Simulator Data</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Read real-time simulation variables from a running MSFS session. Altitude, heading,
					speed — anything SimConnect exposes.
				</p>
			</div>

			<!-- 3. Event Transmission -->
			<div>
				<div class="mb-3 flex items-center gap-2.5">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-link); flex-shrink: 0;"
						aria-hidden="true"
					>
						<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Event Transmission</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Send SimConnect events from your AI workflow. Toggle landing gear, set autopilot
					altitude, trigger any key event.
				</p>
			</div>

			<!-- 4. MSFS 2020 & 2024 -->
			<div>
				<div class="mb-3 flex items-center gap-2.5">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-link); flex-shrink: 0;"
						aria-hidden="true"
					>
						<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">MSFS 2020 &amp; 2024</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Full corpus coverage for both simulator generations with per-item version tagging.
				</p>
			</div>

			<!-- 5. 1,800+ SimVars -->
			<div>
				<div class="mb-3 flex items-center gap-2.5">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-link); flex-shrink: 0;"
						aria-hidden="true"
					>
						<line x1="8" y1="6" x2="21" y2="6" />
						<line x1="8" y1="12" x2="21" y2="12" />
						<line x1="8" y1="18" x2="21" y2="18" />
						<line x1="3" y1="6" x2="3.01" y2="6" />
						<line x1="3" y1="12" x2="3.01" y2="12" />
						<line x1="3" y1="18" x2="3.01" y2="18" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">1,800+ SimVars</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Aircraft, helicopter, GPS, environment, services — the complete SimConnect simulation
					variable reference.
				</p>
			</div>

			<!-- 6. Cross-Platform -->
			<div>
				<div class="mb-3 flex items-center gap-2.5">
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="18"
						height="18"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-link); flex-shrink: 0;"
						aria-hidden="true"
					>
						<circle cx="12" cy="12" r="10" />
						<line x1="2" y1="12" x2="22" y2="12" />
						<path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Cross-Platform</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Docs mode runs on Linux, macOS, and Windows. No simulator required to query the SDK
					reference.
				</p>
			</div>

		</div>
	</div>
</section>

<!-- Explore the docs -->
<section
	class="border-t px-6 pb-24 pt-16"
	style="background-color: var(--color-bg-primary); border-color: var(--color-border);"
>
	<div class="mx-auto max-w-5xl">
		<p
			class="mb-3 text-center text-xs font-semibold uppercase tracking-widest"
			style="color: var(--color-text-muted);"
		>
			Explore the docs
		</p>
		<h2
			class="mb-10 text-center text-2xl font-bold tracking-tight"
			style="color: var(--color-text-primary);"
		>
			Get up and running in minutes
		</h2>

		<div class="grid gap-4 sm:grid-cols-3">

			<!-- Getting Started -->
			<a
				href="{base}/docs/getting-started"
				class="group block rounded-lg p-5 transition-colors"
				style="background-color: var(--color-bg-secondary);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-tertiary)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-secondary)'; }}
			>
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-semibold" style="color: var(--color-text-primary);">Getting Started</span>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-text-muted);"
						aria-hidden="true"
					>
						<line x1="5" y1="12" x2="19" y2="12" />
						<polyline points="12 5 19 12 12 19" />
					</svg>
				</div>
				<p class="text-xs leading-relaxed" style="color: var(--color-text-muted);">
					Install the binary and connect to your MCP client in minutes.
				</p>
			</a>

			<!-- Claude Desktop Setup -->
			<a
				href="{base}/docs/claude-desktop"
				class="group block rounded-lg p-5 transition-colors"
				style="background-color: var(--color-bg-secondary);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-tertiary)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-secondary)'; }}
			>
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-semibold" style="color: var(--color-text-primary);">Claude Desktop Setup</span>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-text-muted);"
						aria-hidden="true"
					>
						<line x1="5" y1="12" x2="19" y2="12" />
						<polyline points="12 5 19 12 12 19" />
					</svg>
				</div>
				<p class="text-xs leading-relaxed" style="color: var(--color-text-muted);">
					Configure Claude Desktop to use SimConnect MCP as an MCP server.
				</p>
			</a>

			<!-- MCP Tools Reference -->
			<a
				href="{base}/docs/mcp-tools-docs"
				class="group block rounded-lg p-5 transition-colors"
				style="background-color: var(--color-bg-secondary);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-tertiary)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-secondary)'; }}
			>
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-semibold" style="color: var(--color-text-primary);">MCP Tools Reference</span>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="14"
						height="14"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
						style="color: var(--color-text-muted);"
						aria-hidden="true"
					>
						<line x1="5" y1="12" x2="19" y2="12" />
						<polyline points="12 5 19 12 12 19" />
					</svg>
				</div>
				<p class="text-xs leading-relaxed" style="color: var(--color-text-muted);">
					All 11 MCP tools with parameters, examples, and error codes.
				</p>
			</a>

		</div>
	</div>
</section>
