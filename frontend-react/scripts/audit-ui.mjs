import { readdir, readFile } from 'node:fs/promises';
import { extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../src/', import.meta.url));
const violations = [];

async function walk(directory) {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
        const path = join(directory, entry.name);
        if (entry.isDirectory()) await walk(path);
        else if (['.tsx', '.css'].includes(extname(entry.name)) && !entry.name.endsWith('.spec.tsx')) await inspect(path);
    }
}

async function inspect(path) {
    const source = await readFile(path, 'utf8');
    const checks = [
        [/<(button|input|select|textarea)\b/, 'Native interactive control; use a PrimeReact component.'],
        [/(?:bg|text|border|outline|ring|fill|stroke|from|via|to)-(?:black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose|\[[^\]]+\])/i, 'Hardcoded Tailwind color; use PrimeReact theme tokens.'],
        [/(?:color|background(?:-color)?|border-color)\s*:\s*(?:#[0-9a-f]{3,8}|rgba?\(|hsla?\()/i, 'Hardcoded CSS color; use a PrimeReact theme variable.']
    ];
    for (const [pattern, message] of checks) {
        if (extname(path) === '.css' && message.startsWith('Native interactive')) continue;
        if (pattern.test(source)) violations.push(`${relative(root, path)}: ${message}`);
    }
}

await walk(root);
if (violations.length) {
    console.error(violations.join('\n'));
    process.exitCode = 1;
} else {
    console.log('UI policy passed: PrimeReact controls and theme-token color policy enforced.');
}
