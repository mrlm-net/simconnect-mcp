import type { RequestHandler } from './$types.js';
import { loadDocIndex } from '$lib/content/pipeline.server.js';
import { siteConfig } from '$lib/config/site.js';

export const prerender = true;

export const GET: RequestHandler = () => {
    const baseUrl = `${siteConfig.url}${siteConfig.basePath}`;
    const docs = loadDocIndex();
    docs.sort((a, b) => a.order - b.order);

    const docList = docs
        .map((doc) => `- [${doc.title}](${baseUrl}/docs/${doc.slug}): ${doc.description}`)
        .join('\n');

    const content = `# ${siteConfig.title}

> ${siteConfig.description}

## Key URLs

- Website: ${baseUrl}/
- Documentation: ${baseUrl}/docs/
- Repository: ${siteConfig.repoUrl}

## MCP Tools — Docs Mode (MCP_MODE=docs)

- list_simvars: List SimConnect simulation variables with pagination and category filter
- get_simvar: Fetch a single simulation variable by name
- list_events: List SimConnect client input events (Key Event IDs)
- get_event: Fetch a single client event by name
- list_functions: List SimConnect C API functions
- get_function: Fetch a single SDK function by name
- list_structures: List SimConnect C data structures
- get_structure: Fetch a single data structure by name
- list_error_codes: List SIMCONNECT_EXCEPTION enum values
- get_error_code: Fetch an error code by name or integer value
- search_docs: Full-text search across all corpus types

## MCP Tools — SimConnect Mode (MCP_MODE=simconnect, Windows only)

- get_simvar_value: Read a single live simulation variable from the running simulator
- get_simvar_values: Read up to 20 simulation variables in a single call
- transmit_event: Transmit a named SimConnect client event to the simulator
- get_sim_state: Return a snapshot of current simulator connection state and flight status

## Documentation

${docList || '(documentation coming soon)'}
`;

    return new Response(content, {
        headers: { 'Content-Type': 'text/plain; charset=utf-8' }
    });
};
