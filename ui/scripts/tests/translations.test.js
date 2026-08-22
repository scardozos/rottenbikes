'use strict';

const fs = require('fs');
const path = require('path');
const { describe, it, eq, ok, finish } = require('./helpers');
const { loadModule, UI_ROOT } = require('../load-module');

const TRANSLATIONS_DIR = path.join(UI_ROOT, 'src', 'translations');
const SRC_DIR = path.join(UI_ROOT, 'src');
const LANGS = ['en', 'es', 'ca'];

const dicts = {};
const keySets = {};
for (const lang of LANGS) {
    dicts[lang] = loadModule(`src/translations/${lang}.js`).default;
    keySets[lang] = new Set(Object.keys(dicts[lang]));
}

function rawKeys(file) {
    const content = fs.readFileSync(file, 'utf8');
    const keys = [];
    const re = /^[ \t]*([a-zA-Z_][a-zA-Z0-9_]*)\s*:/gm;
    let m;
    while ((m = re.exec(content)) !== null) keys.push(m[1]);
    return keys;
}

function walk(dir, files = []) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
        const p = path.join(dir, entry.name);
        if (entry.isDirectory()) {
            if (entry.name === 'node_modules' || entry.name === 'translations') continue;
            walk(p, files);
        } else if (/\.(js|jsx|ts|tsx)$/.test(entry.name)) {
            files.push(p);
        }
    }
    return files;
}

// Collect every translation key referenced from source.
const sourceFiles = walk(SRC_DIR);
const usedKeys = new Set();
const labelKeys = new Set();
const infoCategories = new Set();
let usesWindowTemplate = false;

const reLiteral = /[^a-zA-Z_]t\(\s*['"]([a-zA-Z_][a-zA-Z0-9_]*)['"]/g;
const reTplLiteral = /[^a-zA-Z_]t\(\s*`([a-zA-Z_][a-zA-Z0-9_]*)`/g;
const reWindowTpl = /[^a-zA-Z_]t\(\s*`window_\$\{[^}]+\}`/g;
const reLabelKey = /labelKey:\s*['"]([a-zA-Z_][a-zA-Z0-9_]*)['"]/g;
const reInfoPress = /handleInfoPress\(\s*['"]([a-zA-Z_][a-zA-Z0-9_]*)['"]\)/g;

for (const f of sourceFiles) {
    const content = fs.readFileSync(f, 'utf8');
    let m;
    while ((m = reLiteral.exec(content)) !== null) usedKeys.add(m[1]);
    while ((m = reTplLiteral.exec(content)) !== null) usedKeys.add(m[1]);
    while ((m = reWindowTpl.exec(content)) !== null) usesWindowTemplate = true;
    while ((m = reLabelKey.exec(content)) !== null) labelKeys.add(m[1]);
    while ((m = reInfoPress.exec(content)) !== null) infoCategories.add(m[1]);
}
for (const k of labelKeys) usedKeys.add(k);
for (const cat of infoCategories) {
    usedKeys.add(cat);
    usedKeys.add(`${cat}_desc`);
    usedKeys.add(`${cat}_5star`);
    usedKeys.add(`${cat}_1star`);
}
if (usesWindowTemplate) {
    for (const w of ['1w', '2w', 'overall']) usedKeys.add(`window_${w}`);
}

const reference = LANGS[0];
const referenceKeys = [...keySets[reference]].sort();

describe('Cross-language key parity', () => {
    for (const lang of LANGS.slice(1)) {
        it(`${lang} has same keys as ${reference}`, () => {
            const missing = referenceKeys.filter((k) => !keySets[lang].has(k));
            const extra = [...keySets[lang]].filter((k) => !keySets[reference].has(k));
            eq(missing, [], `missing: ${missing.join(', ')}`);
            eq(extra, [], `extra: ${extra.join(', ')}`);
        });
    }
});

describe('No empty translation values', () => {
    for (const lang of LANGS) {
        it(`${lang} has no empty values`, () => {
            const empties = Object.entries(dicts[lang])
                .filter(([, v]) => v === undefined || v === null || (typeof v === 'string' && v.trim() === ''))
                .map(([k]) => k);
            eq(empties, [], `empty: ${empties.join(', ')}`);
        });
    }
});

describe('No duplicate keys within a file', () => {
    for (const lang of LANGS) {
        it(`${lang}.js has no duplicate keys`, () => {
            const file = path.join(TRANSLATIONS_DIR, `${lang}.js`);
            const seen = {};
            const dupes = [];
            for (const k of rawKeys(file)) (seen[k] ? dupes.push(k) : (seen[k] = true));
            eq([...new Set(dupes)], [], `duplicates: ${[...new Set(dupes)].join(', ')}`);
        });
    }
});

describe('Every t() key used in source has a translation', () => {
    it(`all ${usedKeys.size} source-referenced keys exist in translations`, () => {
        const missing = [...usedKeys].filter((k) => !keySets[reference].has(k)).sort();
        eq(missing, [], `missing: ${missing.join(', ')}`);
    });
});

describe('Orphan translation keys (informational)', () => {
    it('reports orphan keys without failing', () => {
        const orphanKeys = referenceKeys.filter((k) => !usedKeys.has(k));
        if (orphanKeys.length > 0) {
            console.log(`    (info) ${orphanKeys.length} orphan key(s): ${orphanKeys.join(', ')}`);
        }
        ok(true); // always passes
    });
});

finish();
