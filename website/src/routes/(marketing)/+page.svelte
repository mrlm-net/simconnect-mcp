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

	let sdkCopied = $state(false);
	const sdkInstallCommand = 'go get github.com/mrlm-net/simconnect';

	async function copySdkInstall() {
		try {
			await navigator.clipboard.writeText(sdkInstallCommand);
			sdkCopied = true;
			setTimeout(() => {
				sdkCopied = false;
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
		content="Model Context Protocol server for Microsoft Flight Simulator — query SimConnect SDK docs, read live simulator data, and run AI traffic with a tower and approach controller from Claude, Copilot, or any MCP-compatible AI."
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
		Go 1.27+ &middot; MCP Protocol &middot; MSFS 2020 &amp; 2024
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
		Model Context Protocol server for Microsoft Flight Simulator &mdash; query SimConnect SDK docs,
		read live simulator data, and run AI traffic with a tower and approach controller &mdash; from
		Claude, Copilot, or any MCP-compatible AI.
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
		<code class="install-cmd flex-1 overflow-x-auto whitespace-nowrap text-left text-sm" style="font-family: var(--font-mono); color: var(--color-text-secondary);">
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
			BSL 1.1 · non-commercial
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
			Everything you need to work with MSFS SDK from your AI
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
					and data structures — and the 46 guides of the Go library it is built on, from any MCP client.
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

			<!-- 7. AI Traffic -->
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
						<path d="M17.8 19.2 16 11l3.5-3.5C21 6 21.5 4 21 3c-1-.5-3 0-4.5 1.5L13 8 4.8 6.2c-.5-.1-.9.1-1.1.5l-.3.5c-.2.5-.1 1 .3 1.3L9 12l-2 3H4l-1 1 3 2 2 3 1-1v-3l3-2 3.5 5.3c.3.4.8.5 1.3.3l.5-.2c.4-.3.6-.7.5-1.2z" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">AI Traffic</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Departures and arrivals of your own on real stands, SIDs and STARs, with fuel trucks,
					stairs, GPUs and tugs. Clear them yourself, or let the tower and ground do it.
				</p>
			</div>

			<!-- 8. Airborne ATC -->
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
						<circle cx="12" cy="12" r="2" />
						<path d="M16.24 7.76a6 6 0 0 1 0 8.49" />
						<path d="M7.76 16.24a6 6 0 0 1 0-8.49" />
						<path d="M19.07 4.93a10 10 0 0 1 0 14.14" />
						<path d="M4.93 19.07a10 10 0 0 1 0-14.14" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Airborne ATC</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					A tower per runway and landing sequences with wake spacing: speed control, longer
					downwinds, holds and go-arounds, with every call on the radio.
				</p>
			</div>

			<!-- 9. Scheduled Traffic -->
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
						<rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
						<line x1="16" y1="2" x2="16" y2="6" />
						<line x1="8" y1="2" x2="8" y2="6" />
						<line x1="3" y1="10" x2="21" y2="10" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Scheduled Traffic</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Run a realistic airline schedule at your airport: flights board, push and depart on
					time, arrivals come in from en route and turn around on their stands, overflights
					cross overhead — with live departure and arrival boards.
				</p>
			</div>

			<!-- 10. Your Aircraft -->
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
						<rect x="4" y="4" width="16" height="16" rx="2" />
						<line x1="9" y1="9" x2="15" y2="9" />
						<line x1="9" y1="13" x2="15" y2="13" />
						<line x1="9" y1="17" x2="12" y2="17" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Your Aircraft</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Power, lights, doors by name, chocks and GPU through each aircraft's systems profile
					(the Fenix on its own variables). Set radios and squawk, call ground services, list
					your add-ons.
				</p>
			</div>

			<!-- 11. Real-World Traffic -->
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
						<circle cx="12" cy="12" r="6" />
						<line x1="12" y1="12" x2="19" y2="5" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Real-World Traffic</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					Fly the aircraft a feed such as ADS-B sees instead of the timetable — parked, departing
					and arriving, each with stand services, ATC and radio.
				</p>
			</div>

			<!-- 12. Traffic Around You -->
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
						<line x1="2" y1="12" x2="22" y2="12" />
						<polyline points="16 6 22 12 16 18" />
						<line x1="12" y1="2" x2="12" y2="22" />
					</svg>
					<h3 class="text-sm font-semibold" style="color: var(--color-text-primary);">Traffic Around You</h3>
				</div>
				<p class="text-sm leading-relaxed" style="color: var(--color-text-muted);">
					In cruise, airliners ahead the same way, coming the other way and crossing your route,
					1,000 or 2,000 ft above or below — replaced as you fly on.
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

		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">

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

			<!-- Claude Code Setup -->
			<a
				href="{base}/docs/claude-code"
				class="group block rounded-lg p-5 transition-colors"
				style="background-color: var(--color-bg-secondary);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-tertiary)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-secondary)'; }}
			>
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-semibold" style="color: var(--color-text-primary);">Claude Code Setup</span>
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
					Add SimConnect MCP to Claude Code for in-editor SDK lookups and live simulator access.
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
					All 15 docs-mode MCP tools with parameters, examples, and error codes.
				</p>
			</a>

			<!-- AI Traffic & ATC guide -->
			<a
				href="{base}/docs/ai-traffic"
				class="group block rounded-lg p-5 transition-colors"
				style="background-color: var(--color-bg-secondary);"
				onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-tertiary)'; }}
				onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.backgroundColor = 'var(--color-bg-secondary)'; }}
			>
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-semibold" style="color: var(--color-text-primary);">AI Traffic &amp; ATC</span>
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
					The guide: an airline schedule at your airport, a tower and approach controller, and what to ask.
				</p>
			</a>

		</div>
	</div>
</section>

<!-- SDK CTA -->
<section
	class="border-t px-6 py-16"
	style="background-color: var(--color-bg-primary); border-color: var(--color-border);"
>
	<div class="mx-auto max-w-5xl">
		<div
			class="rounded-xl border p-8 lg:p-10"
			style="background-color: var(--color-bg-secondary); border-color: var(--color-border);"
		>
			<div class="flex flex-col gap-8 lg:flex-row lg:items-center lg:gap-12">

				<!-- Left: icon + text + buttons -->
				<div class="flex-1">
					<div class="mb-4 flex items-center gap-3">
						<div
							class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
							style="background-color: rgba(88,166,255,0.12); color: var(--color-link);"
						>
							<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" />
							</svg>
						</div>
						<h2 class="text-xl font-bold tracking-tight" style="color: var(--color-text-primary);">
							SimConnect Go SDK
						</h2>
					</div>
					<p class="mb-6 max-w-md text-sm leading-relaxed" style="color: var(--color-text-secondary);">
						Build Microsoft Flight Simulator add-ons with Go. Lightweight, typed,
						zero-dependency wrapper over SimConnect.dll for MSFS 2020 &amp; 2024.
					</p>
					<div class="flex flex-wrap gap-3">
						<a
							href="https://simconnect.mrlm.net/getting-started"
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-2 rounded-md px-5 py-2 text-sm font-semibold transition-opacity hover:opacity-90"
							style="background-color: var(--color-link); color: #fff;"
						>
							Get Started
							<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
								<line x1="5" y1="12" x2="19" y2="12" />
								<polyline points="12 5 19 12 12 19" />
							</svg>
						</a>
						<a
							href="https://simconnect.mrlm.net/docs"
							target="_blank"
							rel="noopener noreferrer"
							class="inline-flex items-center gap-2 rounded-md border px-5 py-2 text-sm font-semibold transition-colors"
							style="border-color: var(--color-border); color: var(--color-text-primary); background-color: transparent;"
							onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-text-secondary)'; }}
							onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--color-border)'; }}
						>
							View Documentation
						</a>
					</div>
				</div>

				<!-- Right: code block -->
				<div class="lg:w-72 xl:w-80">
					<p class="mb-2 text-xs font-medium" style="color: var(--color-text-muted);">Install</p>
					<div
						class="flex items-center justify-between gap-3 rounded-lg border px-4 py-3"
						style="background-color: var(--color-bg-code); border-color: var(--color-border);"
					>
						<code
							class="install-cmd flex-1 overflow-x-auto whitespace-nowrap text-sm"
							style="font-family: var(--font-mono); color: var(--color-text-secondary);"
						>
							<span style="color: var(--color-text-muted);">$</span>
							<span style="color: var(--color-text-primary);"> {sdkInstallCommand}</span>
						</code>
						<button
							onclick={copySdkInstall}
							class="ml-1 shrink-0 cursor-pointer rounded p-1.5 transition-colors"
							style="color: {sdkCopied ? '#3fb950' : 'var(--color-text-muted)'};"
							aria-label="Copy SDK install command"
							title="Copy to clipboard"
						>
							{#if sdkCopied}
								<svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
									<polyline points="20 6 9 17 4 12" />
								</svg>
							{:else}
								<svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
									<rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
									<path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
								</svg>
							{/if}
						</button>
					</div>
					<p class="mt-3 text-xs" style="color: var(--color-text-muted);">
						<a
							href="https://github.com/mrlm-net/simconnect"
							target="_blank"
							rel="noopener noreferrer"
							style="color: var(--color-link);"
						>github.com/mrlm-net/simconnect</a>
						&middot; BSL 1.1
					</p>
				</div>

			</div>
		</div>
	</div>
</section>

<!-- Commercial use -->
<section
	class="border-t px-6 py-16 text-center"
	style="background-color: var(--color-bg-primary); border-color: var(--color-border);"
>
	<div class="mb-4 flex justify-center">
		<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color: var(--color-link);" aria-hidden="true">
			<rect x="2" y="7" width="20" height="14" rx="2" ry="2" />
			<path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
		</svg>
	</div>
	<h2 class="mb-1 text-xl font-bold tracking-tight" style="color: var(--color-text-primary);">
		Commercial use
	</h2>
	<p class="mb-4 text-xs font-semibold uppercase tracking-widest" style="color: var(--color-text-muted);">
		Business Source License 1.1
	</p>
	<p class="mx-auto mb-7 max-w-sm text-sm leading-relaxed" style="color: var(--color-text-secondary);">
		Want to use SimConnect MCP for commercial stuff — a paid add-on or product, a paid
		service, or inside a business? Contact me and we'll sort out a licence.
	</p>
	<a
		href="mailto:support@mrlm.net?subject=SimConnect%20MCP%20commercial%20use"
		class="inline-flex items-center gap-2 rounded-md px-6 py-2.5 text-sm font-semibold transition-opacity hover:opacity-90"
		style="background-color: var(--color-link); color: #fff;"
	>
		support@mrlm.net
		<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
			<path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
			<polyline points="22,6 12,13 2,6" />
		</svg>
	</a>
</section>

<!-- Sponsor / Support -->
<section
	class="border-t px-6 py-16 text-center"
	style="background-color: var(--color-bg-primary); border-color: var(--color-border);"
>
	<div class="mb-4 flex justify-center">
		<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="#f85149" aria-hidden="true">
			<path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
		</svg>
	</div>
	<h2 class="mb-1 text-xl font-bold tracking-tight" style="color: var(--color-text-primary);">
		Support the project
	</h2>
	<p class="mb-4 text-xs font-semibold uppercase tracking-widest" style="color: var(--color-text-muted);">
		Back open-source MSFS tooling
	</p>
	<p class="mx-auto mb-7 max-w-sm text-sm leading-relaxed" style="color: var(--color-text-secondary);">
		Sponsoring covers infrastructure costs, development time, and MSFS 2020 &amp; 2024
		licences required to test against real simulator versions.
	</p>
	<a
		href="https://revolut.me/mrlm?currency=EUR"
		target="_blank"
		rel="noopener noreferrer"
		class="inline-flex items-center gap-2 rounded-md px-6 py-2.5 text-sm font-semibold transition-opacity hover:opacity-90"
		style="background-color: #f85149; color: #fff;"
	>
		Sponsor via Revolut
		<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
			<path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
			<polyline points="15 3 21 3 21 9" />
			<line x1="10" y1="14" x2="21" y2="3" />
		</svg>
	</a>
</section>

<style>
	.install-cmd {
		scrollbar-width: none;
		-ms-overflow-style: none;
	}
	.install-cmd::-webkit-scrollbar {
		display: none;
	}
</style>
