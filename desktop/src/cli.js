// Shells out to the real `poly` CLI for everything package-manager
// related (search/list/install/remove) instead of reimplementing the
// adapters (apt/npm/pip/brew/tap/community/...) in JS -- the CLI is the
// single source of truth for that logic, this just drives it.
const { execFile } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const EXE = process.platform === "win32" ? "poly.exe" : "poly";
let cachedPath = null;
let cacheChecked = false;

function candidatePaths() {
  const candidates = [];
  if (process.platform === "win32") {
    const localAppData = process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local");
    candidates.push(path.join(localAppData, "Poly", EXE));
  } else {
    candidates.push(path.join(os.homedir(), ".local", "bin", EXE));
    candidates.push(path.join("/usr", "local", "bin", EXE));
  }
  return candidates;
}

function findOnPath() {
  return new Promise((resolve) => {
    const finder = process.platform === "win32" ? "where" : "which";
    execFile(finder, [EXE], (err, stdout) => {
      if (err) return resolve(null);
      const first = stdout.split(/\r?\n/).map((s) => s.trim()).find(Boolean);
      resolve(first || null);
    });
  });
}

async function detect(force = false) {
  if (cacheChecked && !force) return cachedPath;
  cacheChecked = true;

  for (const candidate of candidatePaths()) {
    if (fs.existsSync(candidate)) {
      cachedPath = candidate;
      return cachedPath;
    }
  }

  cachedPath = await findOnPath();
  return cachedPath;
}

async function run(args, { timeoutMs = 30000 } = {}) {
  const exe = await detect();
  if (!exe) {
    throw new Error("Poly CLI not found");
  }
  return new Promise((resolve, reject) => {
    execFile(exe, args, { timeout: timeoutMs, maxBuffer: 20 * 1024 * 1024 }, (err, stdout, stderr) => {
      if (err && !stdout) {
        reject(new Error((stderr || err.message || "poly command failed").trim()));
        return;
      }
      resolve({ stdout, stderr, failed: !!err });
    });
  });
}

async function runJSON(args, opts) {
  const { stdout } = await run(args, opts);
  try {
    return JSON.parse(stdout);
  } catch {
    throw new Error("Could not parse poly's output.");
  }
}

module.exports = { detect, run, runJSON };
