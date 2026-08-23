const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("poly", {
  auth: {
    resume: () => ipcRenderer.invoke("auth:resume"),
    connectPopup: () => ipcRenderer.invoke("auth:connectPopup"),
    logout: () => ipcRenderer.invoke("auth:logout"),
  },
  profile: {
    get: () => ipcRenderer.invoke("profile:get"),
    update: (fields) => ipcRenderer.invoke("profile:update", fields),
    updateNotificationPrefs: (prefs) => ipcRenderer.invoke("profile:updateNotificationPrefs", prefs),
    uploadAvatar: (bytes, fileName) => ipcRenderer.invoke("profile:uploadAvatar", { bytes, fileName }),
  },
  community: {
    search: (query, page) => ipcRenderer.invoke("community:search", { query, page }),
    followStatus: (targetId) => ipcRenderer.invoke("community:followStatus", targetId),
    setFollow: (targetId, follow) => ipcRenderer.invoke("community:setFollow", { targetId, follow }),
  },
  store: {
    search: (query, sort, page) => ipcRenderer.invoke("store:search", { query, sort, page }),
  },
  apps: {
    list: () => ipcRenderer.invoke("apps:list"),
    get: (id) => ipcRenderer.invoke("apps:get", id),
    upsert: (fields) => ipcRenderer.invoke("apps:upsert", fields),
    publishRelease: (fields) => ipcRenderer.invoke("apps:publishRelease", fields),
    delete: (id) => ipcRenderer.invoke("apps:delete", id),
  },
  files: {
    pickImage: () => ipcRenderer.invoke("files:pickImage"),
    pickBuild: () => ipcRenderer.invoke("files:pickBuild"),
  },
  media: {
    uploadImage: (appId, bytes, label) => ipcRenderer.invoke("media:uploadImage", { appId, bytes, label }),
    uploadBuild: (appId, bytes, fileName) => ipcRenderer.invoke("media:uploadBuild", { appId, bytes, fileName }),
  },
  shell: {
    openExternal: (url) => ipcRenderer.invoke("shell:openExternal", url),
  },
  app: {
    getVersion: () => ipcRenderer.invoke("app:getVersion"),
    newAppId: () => ipcRenderer.invoke("app:newAppId"),
  },
  onboarding: {
    hasSeen: () => ipcRenderer.invoke("onboarding:hasSeen"),
    markSeen: () => ipcRenderer.invoke("onboarding:markSeen"),
  },
  state: {
    getLang: () => ipcRenderer.invoke("state:getLang"),
    setLang: (lang) => ipcRenderer.invoke("state:setLang", lang),
    getTheme: () => ipcRenderer.invoke("state:getTheme"),
    setTheme: (theme) => ipcRenderer.invoke("state:setTheme", theme),
    getAccent: () => ipcRenderer.invoke("state:getAccent"),
    setAccent: (accent) => ipcRenderer.invoke("state:setAccent", accent),
  },
  cli: {
    detect: (force) => ipcRenderer.invoke("cli:detect", force),
    search: (query) => ipcRenderer.invoke("cli:search", query),
    list: () => ipcRenderer.invoke("cli:list"),
    install: (spec) => ipcRenderer.invoke("cli:install", spec),
    remove: (name) => ipcRenderer.invoke("cli:remove", name),
  },
  win: {
    minimize: () => ipcRenderer.invoke("window:minimize"),
    toggleMaximize: () => ipcRenderer.invoke("window:toggleMaximize"),
    close: () => ipcRenderer.invoke("window:close"),
    isMaximized: () => ipcRenderer.invoke("window:isMaximized"),
    onState: (cb) => ipcRenderer.on("window:state", (_e, state) => cb(state)),
  },
});
