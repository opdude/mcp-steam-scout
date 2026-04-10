#!/usr/bin/env node

const { spawn } = require("child_process");
const path = require("path");

const BINARY_NAME = process.platform === "win32" ? "mcp-server.exe" : "mcp-server";
const BINARY_PATH = path.join(__dirname, "bin", BINARY_NAME);

const child = spawn(BINARY_PATH, process.argv.slice(2), {
  stdio: "inherit",
  env: process.env,
});

child.on("exit", (code) => process.exit(code ?? 0));
child.on("error", (err) => {
  console.error(`mcp-steam-scout: failed to start binary: ${err.message}`);
  console.error(`Expected binary at: ${BINARY_PATH}`);
  console.error("Try reinstalling: npm install @opdude/mcp-steam-scout");
  process.exit(1);
});
