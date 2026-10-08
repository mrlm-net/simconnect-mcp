<script lang="ts">
	import { base } from '$app/paths';
	import { siteConfig } from '$lib/config/site.js';

	let copied = $state('');

	const installCommand = 'go install github.com/mrlm-net/simconnect-mcp/cmd/simconnect-mcp@latest';
	const sdkInstallCommand = 'go get github.com/mrlm-net/simconnect';

	async function copy(text: string) {
		try {
			await navigator.clipboard.writeText(text);
			copied = text;
			setTimeout(() => {
				if (copied === text) copied = '';
			}, 2000);
		} catch {
			// clipboard not available
		}
	}

	const modes = [
		{ name: 'docs', tools: 15, where: 'any OS' },
		{ name: 'simconnect', tools: 63, where: 'Windows' },
		{ name: 'both', tools: 78, where: 'docs anywhere, live on Windows' }
	];

	// Icon paths: 24×24, stroke 1.75
	const features = [
		{
			title: 'Documentation Mode',
			icon: '<path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/>',
			body: 'Query the full SimConnect SDK reference — simulation variables, events, API functions, and data structures — and the 46 guides of the Go library it is built on, from any MCP client.'
		},
		{
			title: 'Live Simulator Data',
			icon: '<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>',
			body: 'Read real-time simulation variables from a running MSFS session. Altitude, heading, speed — anything SimConnect exposes.'
		},
		{
			title: 'Event Transmission',
			icon: '<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>',
			body: 'Send SimConnect events from your AI workflow. Toggle landing gear, set autopilot altitude, trigger any key event.'
		},
		{
			title: 'MSFS 2020 & 2024',
			icon: '<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>',
			body: 'Full corpus coverage for both simulator generations with per-item version tagging.'
		},
		{
			title: '1,800+ SimVars',
			icon: '<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>',
			body: 'Aircraft, helicopter, GPS, environment, services — the complete SimConnect simulation variable reference.'
		},
		{
			title: 'Cross-Platform',
			icon: '<circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>',
			body: 'Docs mode runs on Linux, macOS, and Windows. No simulator required to query the SDK reference.'
		},
		{
			title: 'AI Traffic',
			icon: '<path d="M17.8 19.2 16 11l3.5-3.5C21 6 21.5 4 21 3c-1-.5-3 0-4.5 1.5L13 8 4.8 6.2c-.5-.1-.9.1-1.1.5l-.3.5c-.2.5-.1 1 .3 1.3L9 12l-2 3H4l-1 1 3 2 2 3 1-1v-3l3-2 3.5 5.3c.3.4.8.5 1.3.3l.5-.2c.4-.3.6-.7.5-1.2z"/>',
			body: 'Departures and arrivals of your own on real stands, SIDs and STARs, with fuel trucks, stairs, GPUs and tugs. Clear them yourself, or let the tower and ground do it.'
		},
		{
			title: 'Airborne ATC',
			icon: '<circle cx="12" cy="12" r="2"/><path d="M16.24 7.76a6 6 0 0 1 0 8.49M7.76 16.24a6 6 0 0 1 0-8.49M19.07 4.93a10 10 0 0 1 0 14.14M4.93 19.07a10 10 0 0 1 0-14.14"/>',
			body: 'A tower per runway and landing sequences with wake spacing: speed control, longer downwinds, holds and go-arounds, with every call on the radio.'
		},
		{
			title: 'Scheduled Traffic',
			icon: '<rect x="3" y="4" width="18" height="18" rx="2"/><path d="M16 2v4M8 2v4M3 10h18"/>',
			body: 'Run a realistic airline schedule at your airport: flights board, push and depart on time, arrivals come in from en route and turn around on their stands, overflights cross overhead — with live departure and arrival boards.'
		},
		{
			title: 'Your Aircraft',
			icon: '<rect x="4" y="4" width="16" height="16" rx="2"/><path d="M9 9h6M9 13h6M9 17h3"/>',
			body: "Power, lights, doors by name, chocks and GPU through each aircraft's systems profile (the Fenix on its own variables). Set radios and squawk, call ground services, list your add-ons."
		},
		{
			title: 'Real-World Traffic',
			icon: '<circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><path d="M12 12l7-7"/>',
			body: 'Fly the aircraft a feed such as ADS-B sees instead of the timetable — parked, departing and arriving, each with stand services, ATC and radio.'
		},
		{
			title: 'Traffic Around You',
			icon: '<path d="M2 12h20"/><polyline points="16 6 22 12 16 18"/><path d="M12 2v20"/>',
			body: 'In cruise, airliners ahead the same way, coming the other way and crossing your route, 1,000 or 2,000 ft above or below — replaced as you fly on.'
		}
	];

	const docs = [
		{ title: 'Getting Started', href: '/docs/getting-started', body: 'Install the binary and connect to your MCP client in minutes.' },
		{ title: 'Claude Code Setup', href: '/docs/claude-code', body: 'Add SimConnect MCP to Claude Code for in-editor SDK lookups and live simulator access.' },
		{ title: 'MCP Tools Reference', href: '/docs/mcp-tools-docs', body: 'All 15 docs-mode MCP tools with parameters, examples, and error codes.' },
		{ title: 'AI Traffic & ATC', href: '/docs/ai-traffic', body: 'The guide: an airline schedule at your airport, a tower and approach controller, and what to ask.' }
	];
</script>

<svelte:head>
	<title>SimConnect MCP — Model Context Protocol for Microsoft Flight Simulator</title>
	<meta
		name="description"
		content="Model Context Protocol server for Microsoft Flight Simulator — query SimConnect SDK docs, read live simulator data, and run AI traffic with a tower and approach controller from Claude, Copilot, or any MCP-compatible AI."
	/>
</svelte:head>

{#snippet arrow()}
	<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14M12 5l7 7-7 7" /></svg>
{/snippet}

{#snippet command(text: string, label: string)}
	<div class="cmd">
		<code><span style="color: var(--text-3);">$&nbsp;</span>{text}</code>
		<button onclick={() => copy(text)} aria-label={label} title="Copy to clipboard" class:ok={copied === text}>
			<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				{#if copied === text}
					<polyline points="20 6 9 17 4 12" />
				{:else}
					<rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
				{/if}
			</svg>
		</button>
	</div>
{/snippet}

<!-- Hero -->
<section class="hero px-6 pt-16 pb-16 sm:pt-24 sm:pb-20">
	<div class="mx-auto grid max-w-6xl items-center gap-12 lg:grid-cols-[1.25fr_1fr] [&>*]:min-w-0">
		<div>
			<div class="mb-6 flex flex-wrap gap-2">
				<span class="pill"><span class="dot"></span>Go 1.27+ · MCP · MSFS 2020 &amp; 2024</span>
			</div>
			<h1 class="mb-5 text-4xl font-semibold tracking-tight sm:text-6xl" style="color: var(--text);">
				SimConnect <span class="brand-word">MCP</span>
			</h1>
			<p class="mb-8 max-w-xl text-base leading-relaxed sm:text-lg" style="color: var(--text-2);">
				Model Context Protocol server for Microsoft Flight Simulator &mdash; query SimConnect SDK docs,
				read live simulator data, and run AI traffic with a tower and approach controller &mdash; from
				Claude, Copilot, or any MCP-compatible AI.
			</p>
			<div class="mb-8 flex flex-wrap gap-3">
				<a href="{base}/docs/getting-started" class="btn btn-primary">Get started {@render arrow()}</a>
				<a href={siteConfig.repoUrl} target="_blank" rel="noopener noreferrer" class="btn">View on GitHub</a>
			</div>
			<div class="max-w-xl">{@render command(installCommand, 'Copy install command')}</div>
			<div class="mt-4 flex flex-wrap gap-2">
				<a href="{base}/docs/mcp-tools-docs" class="pill">docs mode</a>
				<a href="{base}/docs/mcp-tools-simconnect" class="pill">simconnect mode</a>
				<a href="{siteConfig.repoUrl}/blob/main/LICENSE" target="_blank" rel="noopener noreferrer" class="pill">{siteConfig.licenseLabel}</a>
			</div>
		</div>

		<div class="card overflow-hidden">
			<div class="flex items-center justify-between px-5 py-3" style="border-bottom: 1px solid var(--border);">
				<span class="eyebrow">MCP_MODE</span>
				<span class="eyebrow">tools</span>
			</div>
			{#each modes as m (m.name)}
				<div class="flex items-center justify-between gap-4 px-5 py-4" style="border-bottom: 1px solid var(--border);">
					<div>
						<div class="mono text-sm font-medium" style="color: var(--text);">{m.name}</div>
						<div class="text-xs" style="color: var(--text-3);">{m.where}</div>
					</div>
					<div class="mono text-2xl font-medium" style="color: var(--text);">{m.tools}</div>
				</div>
			{/each}
			<div class="px-5 py-3 text-xs" style="color: var(--text-3); background: var(--surface-2);">
				stdio, streamable HTTP and SSE transports
			</div>
		</div>
	</div>
</section>

<!-- Features -->
<section class="px-6 py-20" style="border-top: 1px solid var(--border);">
	<div class="mx-auto max-w-6xl">
		<p class="eyebrow mb-3">Why SimConnect MCP</p>
		<h2 class="mb-10 max-w-2xl text-2xl font-semibold tracking-tight sm:text-3xl" style="color: var(--text);">
			Everything you need to work with the MSFS SDK from your AI
		</h2>
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each features as f (f.title)}
				<div class="card p-5">
					<div class="mb-3 flex items-center gap-3">
						<span class="icon">
							<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">{@html f.icon}</svg>
						</span>
						<h3 class="text-[0.95rem] font-semibold" style="color: var(--text);">{f.title}</h3>
					</div>
					<p class="text-sm leading-relaxed" style="color: var(--text-2);">{f.body}</p>
				</div>
			{/each}
		</div>
	</div>
</section>

<!-- Docs -->
<section class="px-6 py-20" style="border-top: 1px solid var(--border);">
	<div class="mx-auto max-w-6xl">
		<p class="eyebrow mb-3">Explore the docs</p>
		<h2 class="mb-10 text-2xl font-semibold tracking-tight sm:text-3xl" style="color: var(--text);">Get up and running in minutes</h2>
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			{#each docs as d (d.href)}
				<a href="{base}{d.href}" class="card group block p-5">
					<div class="mb-2 flex items-center justify-between text-sm font-semibold" style="color: var(--text);">
						{d.title}
						<span class="go" style="color: var(--text-3);">{@render arrow()}</span>
					</div>
					<p class="text-sm leading-relaxed" style="color: var(--text-2);">{d.body}</p>
				</a>
			{/each}
		</div>
	</div>
</section>

<!-- SDK -->
<section class="px-6 py-20" style="border-top: 1px solid var(--border);">
	<div class="card mx-auto flex max-w-6xl flex-col gap-8 p-8 lg:flex-row lg:items-center lg:gap-12 lg:p-10">
		<div class="flex-1">
			<p class="eyebrow mb-3">Built on</p>
			<h2 class="mb-3 text-2xl font-semibold tracking-tight" style="color: var(--text);">SimConnect Go SDK</h2>
			<p class="mb-6 max-w-md text-sm leading-relaxed" style="color: var(--text-2);">
				Build Microsoft Flight Simulator add-ons with Go. Lightweight, typed, zero-dependency wrapper
				over SimConnect.dll for MSFS 2020 &amp; 2024.
			</p>
			<div class="flex flex-wrap gap-3">
				<a href="https://simconnect.mrlm.net/getting-started" target="_blank" rel="noopener noreferrer" class="btn btn-primary">Get started {@render arrow()}</a>
				<a href="https://simconnect.mrlm.net/docs" target="_blank" rel="noopener noreferrer" class="btn">View documentation</a>
			</div>
		</div>
		<div class="lg:w-96">
			<p class="eyebrow mb-2">Install</p>
			{@render command(sdkInstallCommand, 'Copy SDK install command')}
			<div class="mt-3 flex flex-wrap items-center gap-2 text-xs" style="color: var(--text-3);">
				<a href="https://github.com/mrlm-net/simconnect" target="_blank" rel="noopener noreferrer" class="link">github.com/mrlm-net/simconnect</a>
				<a href="https://github.com/mrlm-net/simconnect/blob/main/LICENSE" target="_blank" rel="noopener noreferrer" class="pill">BSL 1.1 · non-commercial</a>
			</div>
		</div>
	</div>
</section>

<!-- Commercial use + sponsor -->
<section class="px-6 pb-20">
	<div class="mx-auto grid max-w-6xl gap-4 md:grid-cols-2">
		<div class="card p-8">
			<p class="eyebrow mb-3">Business Source License 1.1</p>
			<h2 class="mb-3 text-xl font-semibold tracking-tight" style="color: var(--text);">Commercial use</h2>
			<p class="mb-6 text-sm leading-relaxed" style="color: var(--text-2);">
				Want to use SimConnect MCP for commercial stuff — a paid add-on or product, a paid service, or
				inside a business? Contact me and we'll sort out a licence.
			</p>
			<a href="mailto:support@mrlm.net?subject=SimConnect%20MCP%20commercial%20use" class="btn btn-primary">support@mrlm.net</a>
		</div>
		<div class="card p-8">
			<p class="eyebrow mb-3">Back open-source MSFS tooling</p>
			<h2 class="mb-3 flex items-center gap-2 text-xl font-semibold tracking-tight" style="color: var(--text);">
				<svg width="18" height="18" viewBox="0 0 24 24" fill="var(--danger)" aria-hidden="true"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" /></svg>
				Support the project
			</h2>
			<p class="mb-6 text-sm leading-relaxed" style="color: var(--text-2);">
				Sponsoring covers infrastructure costs, development time, and MSFS 2020 &amp; 2024 licences
				required to test against real simulator versions.
			</p>
			<a href="https://revolut.me/mrlm?currency=EUR" target="_blank" rel="noopener noreferrer" class="btn">Sponsor via Revolut</a>
		</div>
	</div>
</section>

<style>
	.hero {
		background:
			radial-gradient(ellipse 70% 60% at 85% 0%, var(--brand-soft) 0%, transparent 70%),
			var(--bg);
	}
	.icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		flex-shrink: 0;
		border-radius: 8px;
		background: var(--brand-soft);
		color: var(--brand);
	}
	.cmd {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.6rem 0.6rem 0.6rem 1rem;
		border: 1px solid var(--border);
		border-radius: 8px;
		background: var(--surface);
		box-shadow: var(--shadow-1);
	}
	.cmd code {
		flex: 1;
		min-width: 0;
		overflow-x: auto;
		white-space: nowrap;
		font-family: var(--font-mono);
		font-size: 0.8125rem;
		color: var(--text);
		scrollbar-width: none;
	}
	.cmd code::-webkit-scrollbar {
		display: none;
	}
	.cmd button {
		display: inline-flex;
		padding: 0.4rem;
		border-radius: 6px;
		color: var(--text-3);
		cursor: pointer;
	}
	.cmd button:hover {
		color: var(--text);
		background: var(--surface-2);
	}
	.cmd button.ok {
		color: var(--ok);
	}
	.link {
		color: var(--text-2);
		text-decoration: underline;
		text-decoration-color: var(--border-strong);
		text-underline-offset: 3px;
	}
	.link:hover {
		color: var(--text);
	}
	.group:hover .go {
		color: var(--text) !important;
	}
</style>
