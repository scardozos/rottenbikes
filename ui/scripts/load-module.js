'use strict';

// Test harness for loading ES-module source files (which use `export`)
// inside Node CommonJS test scripts, without a bundler or babel.
//
// Pure modules (no `import` statements) are loaded by stripping `export`
// keywords and evaluating in the main context so that returned arrays/
// objects share the same prototypes as the test realm (important for
// deepStrictEqual). Modules with `import` statements can provide a `mocks`
// map; mocks are passed in as function arguments to preserve functions.

const fs = require('fs');
const path = require('path');
const vm = require('vm');

const UI_ROOT = path.resolve(__dirname, '..');

// Loads an ES module file and returns an object with its exported bindings.
//
// opts.exportMap: optional array of names to extract (defaults to all
//   top-level const/let/var/function/class bindings).
// opts.mocks: optional map of `specifier -> mockValue` for `import` statements.
// opts.globals: optional map of extra globals to inject (e.g. localStorage).
function loadModule(file, opts = {}) {
    const { exportMap, mocks, globals } = opts;
    const absPath = path.isAbsolute(file) ? file : path.join(UI_ROOT, file);
    let source = fs.readFileSync(absPath, 'utf8');

    // Collect mock values in import order; each becomes a function parameter.
    const mockParams = [];
    const mockValues = [];

    // Replace import statements with variable declarations sourced from params.
    let importIndex = 0;
    source = source.replace(
        /^\s*import\s+(?:([^'"`;]+?)\s+from\s+)?(['"`])([^'""`]+)\2\s*;?\s*$/gm,
        (line, clause, _quote, specifier) => {
            importIndex++;
            const ref = `__dep_${importIndex}`;
            mockParams.push(ref);
            mockValues.push(
                mocks && mocks[specifier] !== undefined
                    ? mocks[specifier]
                    : {} // empty default so named imports resolve to undefined
            );
            if (!clause) return `/* import '${specifier}' (side-effect) */`;
            // Handle: import X from 'spec'
            //         import * as X from 'spec'
            //         import { a, b as c } from 'spec'
            //         import X, { a } from 'spec'
            const parts = clause.trim();
            if (parts.startsWith('{')) {
                const names = parts
                    .slice(1, -1)
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean);
                return names
                    .map((n) => {
                        const [local, aliased] = n.split(/\s+as\s+/).map((s) => s.trim());
                        return `var ${local} = ${ref} && ${ref}.${aliased || local};`;
                    })
                    .join('\n');
            }
            // import * as X from 'spec'
            const namespaceMatch = parts.match(/^\*\s+as\s+([a-zA-Z_$][\w$]*)$/);
            if (namespaceMatch) {
                return `var ${namespaceMatch[1]} = ${ref};`;
            }
            // Default import, optionally with named.
            const match = parts.match(/^([^,{]+)(?:\s*,\s*(\{[^}]+\}))?$/);
            const defaultLocal = match[1].trim();
            const refs = [`var ${defaultLocal} = (${ref} && ${ref}.default) || ${ref};`];
            if (match[2]) {
                const names = match[2]
                    .slice(1, -1)
                    .split(',')
                    .map((s) => s.trim())
                    .filter(Boolean);
                names.forEach((n) => {
                    const [local, aliased] = n.split(/\s+as\s+/).map((s) => s.trim());
                    refs.push(`var ${local} = ${ref} && ${ref}.${aliased || local};`);
                });
            }
            return refs.join('\n');
        }
    );

    // export { a, b as c };
    const reExportList = /export\s*\{([^}]+)\}\s*;?/g;
    source = source.replace(reExportList, '');

    // export default <expr>;
    const reExportDefault = /export\s+default\s+/g;
    let hasDefault = false;
    source = source.replace(reExportDefault, () => {
        hasDefault = true;
        return 'var __default__ = ';
    });

    // export const X = ... / export function X() / export class X
    const reExportDecl = /export\s+(const|let|var|function|class)\s+/g;
    source = source.replace(reExportDecl, (_line, kind) => `${kind} `);

    // Harvest all top-level binding names to return them.
    const bindingNames = [];
    const reBinding = /^(?:const|let|var|function|class)\s+([a-zA-Z_$][\w$]*)/gm;
    let m;
    while ((m = reBinding.exec(source)) !== null) {
        bindingNames.push(m[1]);
    }

    // Build the return object.
    const returnNames = exportMap && exportMap.length ? exportMap : bindingNames;
    const returnLines = [];
    for (const n of [...new Set(returnNames)]) {
        returnLines.push(`"${n}": typeof ${n} !== 'undefined' ? ${n} : undefined,`);
    }
    if (hasDefault) {
        returnLines.push(`default: typeof __default__ !== 'undefined' ? __default__ : undefined,`);
    }

    // Wrap in a function that receives mocks as params, plus any extra globals.
    // Running in the main context ensures shared prototypes (Array, Object, etc.).
    const globalKeys = globals ? Object.keys(globals) : [];
    const globalVals = globals ? Object.values(globals) : [];
    const globalSetup = globalKeys
        .map((k) => `var ${k} = __globals__.${k};`)
        .join('\n');
    const allParams = [...mockParams, '__globals__'];

    const wrapped = `(function(${allParams.join(', ')}) {\n${globalSetup}\n${source}\nreturn { ${returnLines.join('')} };\n})`;

    // Compile in the main context so prototypes match the test realm.
    const fn = vm.runInThisContext(wrapped, { filename: absPath });
    return fn(...mockValues, globals || {});
}

module.exports = { loadModule, UI_ROOT };
