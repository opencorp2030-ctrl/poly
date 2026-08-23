// Tiny local-only preferences file (userData/state.json) -- just whether
// the first-run onboarding has been shown. Unrelated to ~/.poly/credentials.json,
// which is the cross-CLI session file.
const fs = require("fs");
const path = require("path");
const { app } = require("electron");

function statePath() {
  return path.join(app.getPath("userData"), "state.json");
}

function load() {
  try {
    return JSON.parse(fs.readFileSync(statePath(), "utf8"));
  } catch {
    return {};
  }
}

function save(state) {
  fs.mkdirSync(path.dirname(statePath()), { recursive: true });
  fs.writeFileSync(statePath(), JSON.stringify(state, null, 2));
}

function hasOnboarded() {
  return !!load().onboarded;
}

function setOnboarded() {
  save({ ...load(), onboarded: true });
}

function getLang() {
  return load().lang || null;
}

function setLang(lang) {
  save({ ...load(), lang });
}

function getTheme() {
  return load().theme || null; // "dark" | "light" | "system" | null (unset -> system)
}

function setTheme(theme) {
  save({ ...load(), theme });
}

function getAccent() {
  return load().accent || null;
}

function setAccent(accent) {
  save({ ...load(), accent });
}

module.exports = { hasOnboarded, setOnboarded, getLang, setLang, getTheme, setTheme, getAccent, setAccent };
