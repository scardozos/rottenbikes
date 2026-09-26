'use strict';

const fs = require('fs');
const path = require('path');
const { describe, it, eq, finish } = require('./helpers');
const { UI_ROOT } = require('../load-module');

const SRC_DIR = path.join(UI_ROOT, 'src');

function walk(dir, files = []) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const p = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            if (entry.name === 'node_modules' || entry.name === 'translations') continue;
            walk(p, files);
        } else if (/\.(js|jsx|ts|tsx)$/.test(entry.name)) {
            // Only check files in components or screens
            if (p.includes('/components/') || p.includes('/screens/')) {
                files.push(p);
            }
        }
    }
    return files;
}

const sourceFiles = walk(SRC_DIR);

describe('No hardcoded text in UI', () => {
    it('finds no hardcoded strings in JSX Text components', () => {
        const errors = [];
        // Matches <Text>Something</Text> or <Text ...>Something</Text>.
        // Requires text to start with a letter and not just be whitespace.
        // The negative lookahead (?!\{) ensures we don't accidentally match JSX variables/expressions.
        const reRawText = /<Text[^>]*>\s*(?!\{)([a-zA-Z][^<]*)\s*<\/Text>/g;

        // Matches fallback strings: t('key') || 'Fallback string'
        const reFallback = /t\([^)]+\)\s*\|\|\s*(['"])(.*?)\1/g;

        for (const f of sourceFiles) {
            const content = fs.readFileSync(f, 'utf8');
            const relativePath = path.relative(UI_ROOT, f);
            
            let m;
            while ((m = reRawText.exec(content)) !== null) {
                const text = m[1].trim();
                // Ignore brand names and short placeholders
                if (text === 'RottenBikes' || text === '-') continue;
                errors.push(`${relativePath}: RAW TEXT: "${text}"`);
            }
            
            while ((m = reFallback.exec(content)) !== null) {
                const text = m[2].trim();
                if (text === '-') continue;
                errors.push(`${relativePath}: FALLBACK TEXT: "${text}"`);
            }
        }

        eq(errors, [], `Hardcoded text found:\n${errors.join('\n')}`);
    });
});

finish();
