const { app, BrowserWindow, ipcMain, dialog, shell } = require("electron");
const path = require("path");
const fs = require("fs");

const session = require("./src/session");
const api = require("./src/api");
const localState = require("./src/localstate");
const cli = require("./src/cli");

let mainWindow = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1220,
    height: 800,
    minWidth: 980,
    minHeight: 660,
    backgroundColor: "#0c0e13",
    frame: false,
    show: false,
    webPreferences: {
      preload: path.join(__dirname, "preload.js"),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
    },
  });

  mainWindow.loadFile(path.join(__dirname, "src", "renderer", "index.html"));
  mainWindow.once("ready-to-show", () => mainWindow.show());

  mainWindow.on("maximize", () => mainWindow.webContents.send("window:state", { maximized: true }));
  mainWindow.on("unmaximize", () => mainWindow.webContents.send("window:state", { maximized: false }));
}

app.whenReady().then(() => {
  createWindow();
  app.on("activate", () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

// --- Auth ---
ipcMain.handle("auth:resume", async () => session.resume());
ipcMain.handle("auth:login", async (_e, { email, password }) => session.login(email, password));
ipcMain.handle("auth:logout", async () => session.logout());

// --- Profile ---
ipcMain.handle("profile:get", async () => api.getProfile());

// --- Apps ---
ipcMain.handle("apps:list", async () => api.listOwnApps());
ipcMain.handle("apps:get", async (_e, id) => api.getApp(id));
ipcMain.handle("apps:upsert", async (_e, fields) => api.upsertApp(fields));
ipcMain.handle("apps:publishRelease", async (_e, fields) => api.publishRelease(fields));
ipcMain.handle("apps:delete", async (_e, id) => api.deleteApp(id));

// --- Files ---
const IMAGE_FILTERS = [{ name: "Images", extensions: ["png", "jpg", "jpeg", "webp", "gif"] }];
const MAX_BUILD_BYTES = 200 * 1024 * 1024;

ipcMain.handle("files:pickImage", async () => {
  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ["openFile"],
    filters: IMAGE_FILTERS,
  });
  if (result.canceled || !result.filePaths.length) return null;
  const filePath = result.filePaths[0];
  const buffer = fs.readFileSync(filePath);
  return { fileName: path.basename(filePath), bytes: new Uint8Array(buffer) };
});

ipcMain.handle("files:pickBuild", async () => {
  const result = await dialog.showOpenDialog(mainWindow, { properties: ["openFile"] });
  if (result.canceled || !result.filePaths.length) return null;
  const filePath = result.filePaths[0];
  const stat = fs.statSync(filePath);
  if (stat.size > MAX_BUILD_BYTES) {
    throw new Error("File exceeds the 200MB limit.");
  }
  const buffer = fs.readFileSync(filePath);
  return { fileName: path.basename(filePath), bytes: new Uint8Array(buffer) };
});

// --- Uploads ---
ipcMain.handle("media:uploadImage", async (_e, { appId, bytes, label }) =>
  api.uploadAppMedia(appId, Buffer.from(bytes), label)
);
ipcMain.handle("media:uploadBuild", async (_e, { appId, bytes, fileName }) =>
  api.uploadAppBuild(appId, Buffer.from(bytes), fileName)
);

// --- Shell / misc ---
ipcMain.handle("shell:openExternal", async (_e, url) => shell.openExternal(url));
ipcMain.handle("app:getVersion", async () => app.getVersion());
ipcMain.handle("app:newAppId", async () => require("crypto").randomUUID());

// --- Onboarding (local-only, unrelated to the CLI session file) ---
ipcMain.handle("onboarding:hasSeen", async () => localState.hasOnboarded());
ipcMain.handle("onboarding:markSeen", async () => localState.setOnboarded());

// --- Language (local-only display preference) ---
ipcMain.handle("state:getLang", async () => localState.getLang());
ipcMain.handle("state:setLang", async (_e, lang) => localState.setLang(lang));

// --- Poly CLI (search/list/install/remove) ---
ipcMain.handle("cli:detect", async (_e, force) => cli.detect(!!force));
ipcMain.handle("cli:search", async (_e, query) => cli.runJSON(["search", query, "--json"]));
ipcMain.handle("cli:list", async () => cli.runJSON(["list", "--json"]));
ipcMain.handle("cli:install", async (_e, spec) => cli.runJSON(["install", spec, "--json"], { timeoutMs: 180000 }));
ipcMain.handle("cli:remove", async (_e, name) => cli.runJSON(["remove", name, "--json"]));

// --- Custom titlebar window controls (frameless window) ---
ipcMain.handle("window:minimize", () => mainWindow.minimize());
ipcMain.handle("window:toggleMaximize", () => {
  if (mainWindow.isMaximized()) mainWindow.unmaximize();
  else mainWindow.maximize();
});
ipcMain.handle("window:close", () => mainWindow.close());
ipcMain.handle("window:isMaximized", () => mainWindow.isMaximized());
