// Reads/writes ~/.poly/credentials.json in the exact shape the Go CLI
// writes (internal/account/account.go) so signing in once via the CLI or
// Desktop authenticates both -- single sign-on, same file, same path.
const fs = require("fs");
const os = require("os");
const path = require("path");

function credentialsPath() {
  return path.join(os.homedir(), ".poly", "credentials.json");
}

function load() {
  try {
    const raw = fs.readFileSync(credentialsPath(), "utf8");
    return JSON.parse(raw);
  } catch {
    return null;
  }
}

function save(creds) {
  const dir = path.dirname(credentialsPath());
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(credentialsPath(), JSON.stringify(creds, null, 2), { mode: 0o600 });
}

function clear() {
  try {
    fs.unlinkSync(credentialsPath());
  } catch {
    // already gone
  }
}

function jwtExpiry(token) {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = JSON.parse(Buffer.from(parts[1], "base64url").toString("utf8"));
    return payload.exp ? new Date(payload.exp * 1000) : null;
  } catch {
    return null;
  }
}

module.exports = { credentialsPath, load, save, clear, jwtExpiry };
