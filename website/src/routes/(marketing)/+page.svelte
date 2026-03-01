<script lang="ts">
	import { base } from '$app/paths';
</script>

<svelte:head>
	<title>SimConnect MCP — Model Context Protocol for Microsoft Flight Simulator</title>
	<meta
		name="description"
		content="Model Context Protocol server for Microsoft Flight Simulator — access SimConnect SDK docs and live simulator data from any MCP-compatible AI."
	/>
</svelte:head>

<!-- Hero -->
<section
	class="flex flex-col items-center justify-center px-6 py-24 text-center"
	style="background-color: var(--color-bg-primary);"
>
	<h1
		class="mb-6 text-5xl font-bold tracking-tight sm:text-6xl"
		style="color: var(--color-text-primary);"
	>
		SimConnect MCP
	</h1>
	<p
		class="mb-10 max-w-2xl text-lg leading-relaxed"
		style="color: var(--color-text-muted);"
	>
		Model Context Protocol server for Microsoft Flight Simulator — access SimConnect SDK docs
		and live simulator data from any MCP-compatible AI.
	</p>
	<div class="flex flex-wrap items-center justify-center gap-4">
		<a
			href="{base}/docs/getting-started"
			class="rounded-md px-6 py-3 text-sm font-semibold transition-opacity hover:opacity-90"
			style="background-color: var(--color-link); color: #fff;"
		>
			Get Started
		</a>
		<a
			href="https://github.com/mrlm-net/simconnect-mcp"
			target="_blank"
			rel="noopener noreferrer"
			class="rounded-md border px-6 py-3 text-sm font-semibold transition-colors hover:border-current"
			style="border-color: var(--color-border); color: var(--color-text-primary); background-color: var(--color-bg-secondary);"
		>
			GitHub
		</a>
	</div>
</section>

<!-- Feature grid -->
<section
	class="px-6 py-20"
	style="background-color: var(--color-bg-secondary);"
>
	<div class="mx-auto max-w-5xl">
		<div class="grid gap-6 sm:grid-cols-3">
			<!-- Feature 1: Documentation Mode -->
			<div
				class="rounded-lg border p-6"
				style="background-color: var(--color-bg-tertiary); border-color: var(--color-border);"
			>
				<div
					class="mb-4 inline-flex h-10 w-10 items-center justify-center rounded-md"
					style="background-color: var(--color-bg-code); color: var(--color-link);"
					aria-hidden="true"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" />
						<path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" />
					</svg>
				</div>
				<h3
					class="mb-2 text-base font-semibold"
					style="color: var(--color-text-primary);"
				>
					Documentation Mode
				</h3>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Query the full SimConnect SDK reference: simulation variables, events, functions,
					data structures, and error codes — without leaving your AI chat.
				</p>
			</div>

			<!-- Feature 2: Live Sim Data -->
			<div
				class="rounded-lg border p-6"
				style="background-color: var(--color-bg-tertiary); border-color: var(--color-border);"
			>
				<div
					class="mb-4 inline-flex h-10 w-10 items-center justify-center rounded-md"
					style="background-color: var(--color-bg-code); color: var(--color-link);"
					aria-hidden="true"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
					</svg>
				</div>
				<h3
					class="mb-2 text-base font-semibold"
					style="color: var(--color-text-primary);"
				>
					Live Sim Data
				</h3>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Read real-time simulation variables from a running MSFS session. Altitude,
					heading, speed — anything SimConnect exposes.
				</p>
			</div>

			<!-- Feature 3: Event Transmission -->
			<div
				class="rounded-lg border p-6"
				style="background-color: var(--color-bg-tertiary); border-color: var(--color-border);"
			>
				<div
					class="mb-4 inline-flex h-10 w-10 items-center justify-center rounded-md"
					style="background-color: var(--color-bg-code); color: var(--color-link);"
					aria-hidden="true"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						width="20"
						height="20"
						viewBox="0 0 24 24"
						fill="none"
						stroke="currentColor"
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					>
						<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
					</svg>
				</div>
				<h3
					class="mb-2 text-base font-semibold"
					style="color: var(--color-text-primary);"
				>
					Event Transmission
				</h3>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Send SimConnect events directly from your AI workflow. Toggle lights, set
					transponder, trigger autopilot — full event bus control.
				</p>
			</div>
		</div>
	</div>
</section>

<!-- Quick-start snippet -->
<section
	class="px-6 py-20"
	style="background-color: var(--color-bg-primary);"
>
	<div class="mx-auto max-w-3xl">
		<h2
			class="mb-2 text-2xl font-bold tracking-tight"
			style="color: var(--color-text-primary);"
		>
			Quick Start
		</h2>
		<p class="mb-6 text-sm" style="color: var(--color-text-muted);">
			Install and run in documentation mode — no flight simulator required.
		</p>
		<pre
			class="language-bash overflow-x-auto rounded-lg border p-5 text-sm leading-relaxed"
			style="background-color: var(--color-bg-code); border-color: var(--color-border);"
		><code class="language-bash"><span class="token comment"># Install and run in docs mode (cross-platform)</span>
<span class="token function">go</span> install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest
<span class="token assign-left variable">MCP_MODE</span><span class="token operator">=</span>docs simconnect-mcp</code></pre>
	</div>
</section>
