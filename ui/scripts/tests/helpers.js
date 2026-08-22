'use strict';

// Minimal test framework: provides describe/it with assert,
// accumulates pass/fail counts, and exits non-zero on any failure.

const assert = require('assert');

const results = { passed: 0, failed: 0, failures: [] };
let currentSuite = '';

function describe(name, fn) {
    currentSuite = name;
    console.log(`\n${name}`);
    fn();
}

async function it(name, fn) {
    const label = `${currentSuite} > ${name}`;
    try {
        await fn();
        results.passed++;
        console.log(`  \u2713 ${name}`);
    } catch (e) {
        results.failed++;
        results.failures.push({ label, error: e });
        console.error(`  \u2717 ${name}`);
        console.error(`      ${e.message}`);
    }
}

function eq(actual, expected, msg) {
    assert.deepStrictEqual(actual, expected, msg);
}

function ok(value, msg) {
    assert.ok(value, msg);
}

function throws(fn, msg) {
    assert.throws(fn, msg);
}

// Called at the end of every test file to report and exit.
function finish() {
    if (results.failed > 0) {
        console.error(`\n${results.failed} test(s) failed, ${results.passed} passed`);
        process.exit(1);
    }
    console.log(`\n${results.passed} test(s) passed`);
}

module.exports = { describe, it, eq, ok, throws, finish, assert };
