// Preload for the sign-in popup window only (see openConnectWindow in
// main.js). The popup loads the *remote* desktop-connect.html page from
// poly.candygate.eu; contextIsolation keeps that page's own script from
// touching anything but the two calls exposed here.
const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("polyConnect", {
  complete: (session) => ipcRenderer.send("desktop-connect:complete", session),
});
