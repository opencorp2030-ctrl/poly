const { app, BrowserWindow, ipcMain, dialog, shell } = require("electron");
const path = require("path");
const fs = require("fs");

// The two windows bleeding into each other (garbled text, JS source
// visible as page content) is a known GPU-compositing glitch on some
// Windows graphics drivers -- not a content bug (the popup's page is
// byte-identical to what's deployed). Belt-and-suspenders: disable GPU
// use as early and as thoroughly as possible, all before app is ready.
app.commandLine.appendSwitch("disable-gpu");
app.commandLine.appendSwitch("disable-gpu-compositing");
app.commandLine.appendSwitch("disable-gpu-rasterization");
app.disableHardwareAcceleration();

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

  // TEMPORARY diagnostic: opens DevTools automatically so we can see
  // the real console/network errors behind the Windows rendering bug
  // instead of guessing. Remove once that's found.
  mainWindow.webContents.openDevTools({ mode: "detach" });
  mainWindow.webContents.on("console-message", (_e, level, message, line, sourceId) => {
    console.log(`[renderer:${level}] ${message} (${sourceId}:${line})`);
  });
  mainWindow.webContents.on("render-process-gone", (_e, details) => {
    console.log("[renderer crashed]", details);
  });
  mainWindow.webContents.on("did-fail-load", (_e, errorCode, errorDescription, validatedURL) => {
    console.log("[did-fail-load]", errorCode, errorDescription, validatedURL);
  });

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
ipcMain.handle("auth:logout", async () => session.logout());

// Opens a small "sign in with Poly" popup pointed at poly.candygate.eu,
// the same OAuth-flavored pattern used to connect an AI assistant
// (see site/mcp-connect.html): the actual credentials form lives on the
// hosted page, never in this app. Its window keeps its own persistent
// session partition, so a returning user sees an account chooser
// (backed by that page's own localStorage) instead of a blank form.
function openConnectWindow() {
  return new Promise((resolve) => {
    const connectWin = new BrowserWindow({
      width: 420,
      height: 620,
      parent: mainWindow,
      modal: true,
      resizable: false,
      minimizable: false,
      backgroundColor: "#0c0e13",
      title: "Sign in to Poly",
      webPreferences: {
        preload: path.join(__dirname, "src", "renderer", "connect-preload.js"),
        contextIsolation: true,
        nodeIntegration: false,
        sandbox: true,
        partition: "persist:poly-connect",
      },
    });
    connectWin.setMenuBarVisibility(false);
    connectWin.loadURL("https://poly.candygate.eu/desktop-connect.html");

    // TEMPORARY diagnostic -- see the note on the main window above.
    connectWin.webContents.openDevTools({ mode: "detach" });
    connectWin.webContents.on("did-fail-load", (_e, errorCode, errorDescription, validatedURL) => {
      console.log("[connect did-fail-load]", errorCode, errorDescription, validatedURL);
    });

    let settled = false;

    const onComplete = async (_e, sessionData) => {
      if (settled) return;
      settled = true;
      ipcMain.removeListener("desktop-connect:complete", onComplete);
      try {
        const user = await session.loginWithTokens(sessionData.access_token, sessionData.refresh_token);
        resolve(user);
      } catch {
        resolve(null);
      } finally {
        if (!connectWin.isDestroyed()) connectWin.close();
      }
    };
    ipcMain.on("desktop-connect:complete", onComplete);

    connectWin.on("closed", () => {
      if (settled) return;
      settled = true;
      ipcMain.removeListener("desktop-connect:complete", onComplete);
      resolve(null);
    });
  });
}

ipcMain.handle("auth:connectPopup", async () => openConnectWindow());

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
