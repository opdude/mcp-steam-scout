#!/usr/bin/env node

const { execFileSync } = require("child_process");
const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const { pipeline } = require("stream");
const { promisify } = require("util");
const zlib = require("zlib");

const pipelineAsync = promisify(pipeline);

const VERSION = require("./package.json").version;
const BIN_DIR = path.join(__dirname, "bin");
const BINARY_NAME = process.platform === "win32" ? "mcp-server.exe" : "mcp-server";
const BINARY_PATH = path.join(BIN_DIR, BINARY_NAME);

function getPlatform() {
  const platform = process.platform;
  if (platform === "darwin") return "darwin";
  if (platform === "linux") return "linux";
  if (platform === "win32") return "windows";
  throw new Error(`Unsupported platform: ${platform}`);
}

function getArch() {
  const arch = process.arch;
  if (arch === "x64") return "amd64";
  if (arch === "arm64") return "arm64";
  throw new Error(`Unsupported architecture: ${arch}`);
}

function getArchiveName() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = platform === "windows" ? "zip" : "tar.gz";
  return `mcp-steam-scout_${VERSION}_${platform}_${arch}.${ext}`;
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const request = (u) => {
      https.get(u, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return request(res.headers.location);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`Download failed: HTTP ${res.statusCode} from ${u}`));
        }
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
      }).on("error", reject);
    };
    request(url);
  });
}

async function extractTarGz(archivePath, binaryName, destPath) {
  const tar = require("tar");
  await tar.extract({
    file: archivePath,
    cwd: path.dirname(destPath),
    filter: (p) => path.basename(p) === binaryName,
    strip: 0,
  });
}

function extractZip(archivePath, binaryName, destPath) {
  // Use system unzip on Windows
  execFileSync("powershell", [
    "-command",
    `Expand-Archive -Path '${archivePath}' -DestinationPath '${path.dirname(destPath)}' -Force`,
  ]);
}

async function install() {
  if (fs.existsSync(BINARY_PATH)) {
    console.log("mcp-steam-scout: binary already installed, skipping download.");
    return;
  }

  const archiveName = getArchiveName();
  const url = `https://github.com/opdude/mcp-steam-scout/releases/download/v${VERSION}/${archiveName}`;
  const tmpArchive = path.join(os.tmpdir(), archiveName);

  console.log(`mcp-steam-scout: downloading ${archiveName}...`);

  if (!fs.existsSync(BIN_DIR)) {
    fs.mkdirSync(BIN_DIR, { recursive: true });
  }

  try {
    await download(url, tmpArchive);

    if (archiveName.endsWith(".tar.gz")) {
      // tar module may not be available; fall back to system tar
      try {
        const tar = require("tar");
        await tar.extract({
          file: tmpArchive,
          cwd: BIN_DIR,
          filter: (p) => path.basename(p) === BINARY_NAME,
        });
      } catch {
        execFileSync("tar", ["-xzf", tmpArchive, "-C", BIN_DIR, BINARY_NAME]);
      }
    } else {
      extractZip(tmpArchive, BINARY_NAME, BINARY_PATH);
    }

    if (process.platform !== "win32") {
      fs.chmodSync(BINARY_PATH, 0o755);
    }

    console.log("mcp-steam-scout: installed successfully.");
  } finally {
    if (fs.existsSync(tmpArchive)) fs.unlinkSync(tmpArchive);
  }
}

install().catch((err) => {
  console.error("mcp-steam-scout: installation failed:", err.message);
  process.exit(1);
});
