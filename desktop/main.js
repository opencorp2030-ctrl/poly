const { app, BrowserWindow, ipcMain, dialog, shell } = require("electron");
const path = require("path");
const fs = require("fs");

const session = require("./src/session");
const api = require("./src/api");

let mainWindow = null;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1180,
    height: 780,
    minWidth: 960,
    minHeight: 640,
    backgroundColor: "#101319",
    autoHideMenuBar: true,
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
