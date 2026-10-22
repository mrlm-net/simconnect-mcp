import fs from 'node:fs';
import path from 'node:path';

/**
 * Numbers the site states about the server, read from the repository at build
 * time so they cannot drift: the embedded library version and guide count, and
 * the tools each mode registers.
 */
export interface SiteFacts {
    libraryVersion: string; // e.g. "v0.108.9"
    guides: number;
    docsTools: number;
    simconnectTools: number;
    allTools: number;
}

function repoPath(...parts: string[]): string {
    return path.resolve(process.cwd(), '..', ...parts);
}

// countTools counts the mcpadapter.NewTool("…") registrations in a mode's
// tools package.
function countTools(dir: string): number {
    return fs
        .readdirSync(dir)
        .filter((f) => f.endsWith('.go') && !f.endsWith('_test.go'))
        .reduce((n, f) => {
            const src = fs.readFileSync(path.join(dir, f), 'utf-8');
            return n + (src.match(/\bmcpadapter\.NewTool\("/g) ?? []).length;
        }, 0);
}

let cached: SiteFacts | undefined;

export function loadFacts(): SiteFacts {
    if (cached) return cached;
    const assets = repoPath('internal', 'libdocs', 'assets');
    const docsTools = countTools(repoPath('internal', 'modes', 'docs', 'tools'));
    const simconnectTools = countTools(repoPath('internal', 'modes', 'simconnect', 'tools'));
    cached = {
        libraryVersion: fs.readFileSync(path.join(assets, 'VERSION'), 'utf-8').trim(),
        guides: fs.readdirSync(assets).filter((f) => f.endsWith('.md')).length,
        docsTools,
        simconnectTools,
        allTools: docsTools + simconnectTools
    };
    return cached;
}
