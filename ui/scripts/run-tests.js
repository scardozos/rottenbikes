#!/usr/bin/env node
'use strict';

// Test runner: executes every *.test.js file in scripts/tests/ as a
// subprocess and aggregates pass/fail results. Exits non-zero if any
// test file fails.

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const testsDir = path.join(__dirname, 'tests');

const files = fs
    .readdirSync(testsDir)
    .filter((f) => f.endsWith('.test.js'))
    .sort();

if (files.length === 0) {
    console.error('No test files found in', testsDir);
    process.exit(1);
}

let totalPassed = 0;
let totalFailed = 0;
const failedFiles = [];

for (const file of files) {
    const filePath = path.join(testsDir, file);
    console.log(`\n${'='.repeat(60)}`);
    console.log(`Running ${file}`);
    console.log('='.repeat(60));
    const result = spawnSync(process.execPath, [filePath], { stdio: 'inherit' });

    if (result.status !== 0) {
        totalFailed++;
        failedFiles.push(file);
    } else {
        totalPassed++;
    }
}

console.log(`\n${'='.repeat(60)}`);
console.log('Test Suite Summary');
console.log('='.repeat(60));
console.log(`  Files passed: ${totalPassed}/${files.length}`);
if (failedFiles.length > 0) {
    console.log(`  Files failed: ${failedFiles.length}`);
    for (const f of failedFiles) console.log(`    - ${f}`);
    process.exit(1);
}
console.log('  All test files passed.');
process.exit(0);
