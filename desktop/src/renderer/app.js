"use strict";

const CATEGORIES = [
  ["productivity", "Productivity"], ["developer-tools", "Developer tools"], ["games", "Games"],
  ["social", "Social"], ["utilities", "Utilities"], ["education", "Education"], ["entertainment", "Entertainment"],
  ["photo-video", "Photo & video"], ["music-audio", "Music & audio"], ["health-fitness", "Health & fitness"],
  ["business", "Business"], ["finance", "Finance"], ["lifestyle", "Lifestyle"], ["news", "News"],
  ["travel-local", "Travel & local"], ["other", "Other"],
];
const CONTENT_RATINGS = [["everyone", "Everyone"], ["teen", "Teen"], ["mature", "Mature 17+"], ["adults", "Adults only"]];
const PLATFORMS = [["windows", "Windows"], ["macos", "macOS"], ["linux", "Linux"], ["web", "Web"], ["android", "Android"], ["ios", "iOS"]];
const PERMISSIONS = [
  ["camera", "Camera"], ["microphone", "Microphone"], ["location", "Location"], ["contacts", "Contacts"],
  ["storage", "Storage"], ["network", "Network"], ["notifications", "Notifications"], ["calendar", "Calendar"],
  ["sms", "SMS"], ["bluetooth", "Bluetooth"],
];

function $(id) { return document.getElementById(id); }
function escapeHtml(s) {
  const div = document.createElement("div");
  div.textContent = s == null ? "" : String(s);
  return div.innerHTML;
}

function showView(name) {
  document.querySelectorAll(".view").forEach((el) => el.classList.toggle("active", el.dataset.view === name));
}
function showPanel(name) {
  document.querySelectorAll(".panel").forEach((el) => el.classList.toggle("active", el.dataset.panel === name));
  document.querySelectorAll(".side-link[data-nav]").forEach((el) => el.classList.toggle("active", el.dataset.nav === name));
}

// ---------------------------------------------------------------------
// Titlebar
// ---------------------------------------------------------------------

$("tb-min").addEventListener("click", () => window.poly.win.minimize());
$("tb-max").addEventListener("click", () => window.poly.win.toggleMaximize());
$("tb-close").addEventListener("click", () => window.poly.win.close());
window.poly.win.onState(({ maximized }) => {
  $("tb-max").title = maximized ? "Restore" : "Maximize";
});

// ---------------------------------------------------------------------
// Onboarding
// ---------------------------------------------------------------------

let onboardSlide = 0;
const ONBOARD_SLIDES = 3;

function goToOnboardSlide(n) {
  onboardSlide = n;
  document.querySelectorAll(".onboard-slide").forEach((el) => el.classList.toggle("active", Number(el.dataset.slide) === n));
  document.querySelectorAll(".dot").forEach((el) => el.classList.toggle("active", Number(el.dataset.dot) === n));
  $("onboard-next").textContent = n === ONBOARD_SLIDES - 1 ? "Sign in →" : "Continue →";
}

$("onboard-next").addEventListener("click", async () => {
  if (onboardSlide < ONBOARD_SLIDES - 1) {
    goToOnboardSlide(onboardSlide + 1);
  } else {
    await window.poly.onboarding.markSeen();
    showView("login");
  }
});

// ---------------------------------------------------------------------
// Boot / auth
// ---------------------------------------------------------------------

async function boot() {
  const [user, hasSeenOnboarding] = await Promise.all([
    window.poly.auth.resume(),
    window.poly.onboarding.hasSeen(),
  ]);
  if (user) {
    await enterShell();
    return;
  }
  if (!hasSeenOnboarding) {
    goToOnboardSlide(0);
    showView("onboarding");
  } else {
    showView("login");
  }
}

async function enterShell() {
  showView("shell");
  showPanel("apps");
  await loadProfile();
  await loadApps();
}

$("login-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const email = $("login-email").value.trim();
  const password = $("login-password").value;
  $("login-error").textContent = "";
  $("login-submit").disabled = true;
  try {
    await window.poly.auth.login(email, password);
    await window.poly.onboarding.markSeen();
    await enterShell();
  } catch (err) {
    $("login-error").textContent = err.message || "Sign-in failed.";
  } finally {
    $("login-submit").disabled = false;
  }
});

$("create-account-link").addEventListener("click", (e) => {
  e.preventDefault();
  window.poly.shell.openExternal("https://poly.candygate.eu/account");
});
$("settings-account-link").addEventListener("click", (e) => {
  e.preventDefault();
  window.poly.shell.openExternal("https://poly.candygate.eu/account");
});

$("signout-btn").addEventListener("click", async () => {
  await window.poly.auth.logout();
  showView("login");
  $("login-email").value = "";
  $("login-password").value = "";
});

document.querySelectorAll(".side-link[data-nav]").forEach((btn) => {
  btn.addEventListener("click", () => {
    const panel = btn.dataset.nav;
    if (panel === "publish") openWizard(null);
    showPanel(panel);
  });
});

// ---------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------

async function loadProfile() {
  try {
    const p = await window.poly.profile.get();
    $("account-email").textContent = p.email;
    $("account-initial").textContent = (p.username || p.email || "?")[0].toUpperCase();
    $("settings-email").textContent = p.email;
    $("settings-plan").textContent = p.plan === "pro" ? "Pro" : "Free";
  } catch {
    // non-fatal -- app list still works
  }
  try {
    $("settings-version").textContent = "v" + (await window.poly.app.getVersion());
  } catch {
    $("settings-version").textContent = "";
  }
}

// ---------------------------------------------------------------------
// My apps
// ---------------------------------------------------------------------

let appsCache = [];

async function loadApps() {
  const grid = $("apps-grid");
  grid.innerHTML = "";
  try {
    appsCache = await window.poly.apps.list();
  } catch (err) {
    grid.innerHTML = `<p class="error-banner">${escapeHtml(err.message)}</p>`;
    return;
  }
  const published = appsCache.filter((a) => a.status === "published").length;
  $("apps-sub").textContent = appsCache.length
    ? `${appsCache.length} app${appsCache.length === 1 ? "" : "s"} · ${published} published`
    : "";
  $("apps-empty").style.display = appsCache.length ? "none" : "block";
  grid.innerHTML = appsCache.map(renderAppCard).join("");

  grid.querySelectorAll("[data-edit]").forEach((btn) =>
    btn.addEventListener("click", () => openWizard(appsCache.find((a) => a.id === btn.dataset.edit)))
  );
  grid.querySelectorAll("[data-delete]").forEach((btn) =>
    btn.addEventListener("click", () => confirmDeleteApp(btn.dataset.delete, btn.dataset.name))
  );
  grid.querySelectorAll("[data-open]").forEach((btn) =>
    btn.addEventListener("click", () => window.poly.shell.openExternal(`https://poly.candygate.eu/app?slug=${encodeURIComponent(btn.dataset.open)}`))
  );
}

function renderAppCard(a) {
  const icon = a.icon_url
    ? `<img class="app-icon" src="${escapeHtml(a.icon_url)}" alt="" />`
    : `<div class="app-icon-fallback">${escapeHtml((a.name || "?")[0].toUpperCase())}</div>`;
  const price = a.is_paid ? `$${(a.price_cents / 100).toFixed(2)} ${String(a.currency).toUpperCase()}` : "Free";
  return `
    <div class="app-card">
      <div class="app-card-banner"></div>
      <div class="app-card-body">
        ${icon}
        <div class="app-name-row">
          <div class="app-name">${escapeHtml(a.name)}</div>
        </div>
        <div class="status-label"><span class="status-dot ${a.status}"></span>${a.status}</div>
        <div class="app-stats">
          <span>${price}</span>
          <span>v${escapeHtml(a.version || "-")}</span>
          <span>${a.download_count || 0} installs</span>
        </div>
        <div class="app-card-actions">
          <button class="btn" data-edit="${a.id}">Edit</button>
          ${a.status === "published" ? `<button class="btn" data-open="${a.slug}">Open</button>` : ""}
          <button class="btn danger" data-delete="${a.id}" data-name="${escapeHtml(a.name)}">Delete</button>
        </div>
      </div>
    </div>`;
}

let pendingDeleteId = null;
function confirmDeleteApp(id, name) {
  pendingDeleteId = id;
  $("confirm-title").textContent = `Delete "${name}"?`;
  $("confirm-body").textContent = "This removes the listing and its stored files. This can't be undone.";
  $("confirm-modal").classList.add("active");
}
$("confirm-cancel").addEventListener("click", () => $("confirm-modal").classList.remove("active"));
$("confirm-ok").addEventListener("click", async () => {
  if (!pendingDeleteId) return;
  $("confirm-ok").disabled = true;
  try {
    await window.poly.apps.delete(pendingDeleteId);
    $("confirm-modal").classList.remove("active");
    await loadApps();
  } catch (err) {
    $("confirm-body").textContent = err.message || "Delete failed.";
  } finally {
    $("confirm-ok").disabled = false;
  }
});

$("apps-new-btn").addEventListener("click", () => { openWizard(null); showPanel("publish"); });
$("apps-empty-new-btn").addEventListener("click", () => { openWizard(null); showPanel("publish"); });
$("publish-cancel-btn").addEventListener("click", () => showPanel("apps"));

// ---------------------------------------------------------------------
// Publish / edit wizard
// ---------------------------------------------------------------------

const TOTAL_STEPS = 4;
let currentStep = 1;
let wizardAppId = null;
let isEditMode = false;
let isPaidState = false;
let downloadMethod = "url";
let formState = { icon_url: null, screenshots: [], storage_path: null, size_bytes: null };

function populateOptions() {
  $("f-category").innerHTML = CATEGORIES.map((c) => `<option value="${c[0]}">${escapeHtml(c[1])}</option>`).join("");
  $("f-content-rating").innerHTML = CONTENT_RATINGS.map((c) => `<option value="${c[0]}">${escapeHtml(c[1])}</option>`).join("");
  $("f-platforms").innerHTML = PLATFORMS.map((p) => `<label><input type="checkbox" value="${p[0]}" class="platform-check" /> ${escapeHtml(p[1])}</label>`).join("");
  $("f-permissions").innerHTML = PERMISSIONS.map((p) => `<label><input type="checkbox" value="${p[0]}" class="perm-check" /> ${escapeHtml(p[1])}</label>`).join("");
}
populateOptions();

function resetForm() {
  formState = { icon_url: null, screenshots: [], storage_path: null, size_bytes: null };
  isPaidState = false;
  downloadMethod = "url";
  $("f-name").value = "";
  $("f-tagline").value = "";
  $("f-website").value = "";
  $("f-privacy").value = "";
  $("f-support").value = "";
  $("f-description").value = "";
  $("f-documentation").value = "";
  $("f-ads").checked = false;
  $("f-iap").checked = false;
  $("f-version").value = "";
  $("f-release-notes").value = "";
  $("f-download-url").value = "";
  $("f-price").value = "";
  setSegmented("pricing-segmented", "free", "pricing");
  $("price-fields").style.display = "none";
  setSegmented("download-method-segmented", "url", "method");
  $("f-download-url").style.display = "block";
  $("build-tile").style.display = "none";
  $("icon-tile").innerHTML = '<div class="dropzone-empty"><span class="dz-icon">+</span><span>512×512 square</span></div>';
  $("build-tile").innerHTML = '<div class="dropzone-empty"><span class="dz-icon">↥</span><span>Choose a build file</span></div>';
  document.querySelectorAll(".platform-check, .perm-check").forEach((el) => (el.checked = false));
  renderScreenshots();
  updatePreview();
}

function setSegmented(groupId, value, dataAttr) {
  document.querySelectorAll(`#${groupId} .seg-btn`).forEach((btn) =>
    btn.classList.toggle("active", btn.dataset[dataAttr] === value)
  );
}

document.querySelectorAll("#pricing-segmented .seg-btn").forEach((btn) => {
  btn.addEventListener("click", () => {
    isPaidState = btn.dataset.pricing === "paid";
    setSegmented("pricing-segmented", btn.dataset.pricing, "pricing");
    $("price-fields").style.display = isPaidState ? "flex" : "none";
    updatePreview();
  });
});
document.querySelectorAll("#download-method-segmented .seg-btn").forEach((btn) => {
  btn.addEventListener("click", () => {
    downloadMethod = btn.dataset.method;
    setSegmented("download-method-segmented", downloadMethod, "method");
    const isUpload = downloadMethod === "upload";
    $("f-download-url").style.display = isUpload ? "none" : "block";
    $("build-tile").style.display = isUpload ? "flex" : "none";
  });
});

function updatePreview() {
  const name = $("f-name").value.trim();
  const tagline = $("f-tagline").value.trim();
  $("preview-name").textContent = name || "Your app name";
  $("preview-tagline").textContent = tagline || "Your tagline shows up here";
  const catLabel = (CATEGORIES.find((c) => c[0] === $("f-category").value) || [])[1] || "Category";
  $("preview-category").textContent = catLabel;
  const price = isPaidState ? `$${(parseFloat($("f-price").value) || 0).toFixed(2)}` : "Free";
  $("preview-price").textContent = price;
  const checkedPlatforms = Array.from(document.querySelectorAll(".platform-check:checked")).map((el) => el.value);
  $("preview-platforms").innerHTML = checkedPlatforms.map((p) => `<span class="chip">${escapeHtml((PLATFORMS.find((x) => x[0] === p) || [])[1] || p)}</span>`).join("");
  if (formState.icon_url) {
    $("preview-icon").innerHTML = `<img src="${escapeHtml(formState.icon_url)}" alt="" />`;
  } else {
    $("preview-icon").innerHTML = "";
    $("preview-icon").textContent = (name || "?")[0].toUpperCase();
  }
  $("preview-shots").innerHTML = formState.screenshots.map((s) => `<img src="${escapeHtml(s)}" alt="" />`).join("");
}
["f-name", "f-tagline", "f-category", "f-price"].forEach((id) => {
  $(id).addEventListener("input", updatePreview);
  $(id).addEventListener("change", updatePreview);
});
document.addEventListener("change", (e) => {
  if (e.target.classList.contains("platform-check")) updatePreview();
});

async function openWizard(app) {
  resetForm();
  currentStep = 1;
  isEditMode = !!app;
  $("publish-title").textContent = isEditMode ? `Edit "${app.name}"` : "Publish an app";
  $("publish-error").textContent = "";
  $("publish-ok").textContent = "";

  if (isEditMode) {
    wizardAppId = app.id;
    $("f-name").value = app.name || "";
    $("f-tagline").value = app.tagline || "";
    $("f-description").value = app.description_md || "";
    $("f-documentation").value = app.documentation_md || "";
    $("f-website").value = app.website_url || "";
    $("f-privacy").value = app.privacy_policy_url || "";
    $("f-support").value = app.support_email || "";
    $("f-ads").checked = !!app.contains_ads;
    $("f-iap").checked = !!app.has_iap;
    $("f-version").value = app.version || "";
    $("f-release-notes").value = app.release_notes || "";
    $("f-download-url").value = app.download_url || "";
    $("f-category").value = app.category || "productivity";
    $("f-content-rating").value = app.content_rating || "everyone";
    formState.icon_url = app.icon_url || null;
    formState.screenshots = app.screenshots || [];
    formState.storage_path = app.storage_path || null;
    formState.size_bytes = app.size_bytes || null;
    (app.platforms || []).forEach((v) => { const el = document.querySelector(`.platform-check[value="${v}"]`); if (el) el.checked = true; });
    (app.permissions || []).forEach((v) => { const el = document.querySelector(`.perm-check[value="${v}"]`); if (el) el.checked = true; });
    if (app.icon_url) $("icon-tile").innerHTML = `<img src="${escapeHtml(app.icon_url)}" alt="" />`;
    renderScreenshots();
    if (app.is_paid) {
      isPaidState = true;
      setSegmented("pricing-segmented", "paid", "pricing");
      $("price-fields").style.display = "flex";
      $("f-price").value = (app.price_cents / 100).toFixed(2);
      $("f-currency").value = app.currency || "usd";
    }
  } else {
    wizardAppId = await window.poly.app.newAppId();
    $("f-category").value = "productivity";
    $("f-content-rating").value = "everyone";
  }
  updatePreview();
  updateStepButtons();
}

function updateStepButtons() {
  document.querySelectorAll("[data-step]").forEach((el) => el.classList.toggle("active", Number(el.dataset.step) === currentStep));
  document.querySelectorAll("[data-step-pill]").forEach((el) => {
    const n = Number(el.dataset.stepPill);
    el.classList.toggle("active", n === currentStep);
    el.classList.toggle("done", n < currentStep);
  });
  $("prev-btn").style.visibility = currentStep === 1 ? "hidden" : "visible";
  $("next-btn").textContent = currentStep === TOTAL_STEPS ? (isEditMode ? "Save changes" : "Publish") : "Next →";
}

$("prev-btn").addEventListener("click", () => { if (currentStep > 1) { currentStep--; updateStepButtons(); } });
$("next-btn").addEventListener("click", async () => {
  if (currentStep === 1 && !$("f-name").value.trim()) {
    $("publish-error").textContent = "Give your app a name first.";
    return;
  }
  if (currentStep < TOTAL_STEPS) { currentStep++; updateStepButtons(); $("publish-error").textContent = ""; return; }
  await submitApp("published");
});
$("save-draft-btn").addEventListener("click", () => submitApp("draft"));

// --- File uploads ---
$("icon-tile").addEventListener("click", async () => {
  const file = await window.poly.files.pickImage();
  if (!file) return;
  $("icon-tile").innerHTML = '<div class="dropzone-empty"><span>…</span></div>';
  try {
    const url = await window.poly.media.uploadImage(wizardAppId, file.bytes, "icon");
    formState.icon_url = url;
    $("icon-tile").innerHTML = `<img src="${escapeHtml(url)}" alt="" />`;
    updatePreview();
  } catch (err) {
    $("publish-error").textContent = err.message || "Icon upload failed.";
    $("icon-tile").innerHTML = '<div class="dropzone-empty"><span class="dz-icon">+</span><span>512×512 square</span></div>';
  }
});

$("screenshot-add-tile").addEventListener("click", async () => {
  if (formState.screenshots.length >= 8) return;
  const file = await window.poly.files.pickImage();
  if (!file) return;
  try {
    const url = await window.poly.media.uploadImage(wizardAppId, file.bytes, "screenshot");
    formState.screenshots.push(url);
    renderScreenshots();
    updatePreview();
  } catch (err) {
    $("publish-error").textContent = err.message || "Screenshot upload failed.";
  }
});

function renderScreenshots() {
  const row = $("screenshots-row");
  const addTile = $("screenshot-add-tile");
  row.querySelectorAll(".shot-tile").forEach((n) => n.remove());
  formState.screenshots.forEach((url, i) => {
    const tile = document.createElement("div");
    tile.className = "upload-tile shot-tile";
    tile.innerHTML = `<img src="${escapeHtml(url)}" alt="" />`;
    tile.title = "Click to remove";
    tile.addEventListener("click", () => { formState.screenshots.splice(i, 1); renderScreenshots(); updatePreview(); });
    row.insertBefore(tile, addTile);
  });
  addTile.style.display = formState.screenshots.length >= 8 ? "none" : "flex";
}

$("build-tile").addEventListener("click", async () => {
  let file;
  try {
    file = await window.poly.files.pickBuild();
  } catch (err) {
    $("publish-error").textContent = err.message || "Could not read that file.";
    return;
  }
  if (!file) return;
  $("build-tile").innerHTML = '<div class="dropzone-empty"><span>…</span></div>';
  try {
    const result = await window.poly.media.uploadBuild(wizardAppId, file.bytes, file.fileName);
    formState.storage_path = result.storage_path;
    formState.size_bytes = result.size_bytes;
    $("build-tile").innerHTML = `<div class="dropzone-empty"><span>${escapeHtml(file.fileName)}</span></div>`;
  } catch (err) {
    $("publish-error").textContent = err.message || "Build upload failed.";
    $("build-tile").innerHTML = '<div class="dropzone-empty"><span class="dz-icon">↥</span><span>Choose a build file</span></div>';
  }
});

async function submitApp(status) {
  const name = $("f-name").value.trim();
  if (!name) { $("publish-error").textContent = "Give your app a name first."; return; }

  const platforms = Array.from(document.querySelectorAll(".platform-check:checked")).map((el) => el.value);
  const permissions = Array.from(document.querySelectorAll(".perm-check:checked")).map((el) => el.value);
  const priceValue = parseFloat($("f-price").value) || 0;

  $("publish-error").textContent = "";
  $("publish-ok").textContent = "";
  $("next-btn").disabled = true;
  $("save-draft-btn").disabled = true;

  try {
    const appRow = await window.poly.apps.upsert({
      p_id: wizardAppId,
      p_name: name,
      p_tagline: $("f-tagline").value.trim() || null,
      p_description_md: $("f-description").value.trim() || null,
      p_category: $("f-category").value,
      p_content_rating: $("f-content-rating").value,
      p_icon_url: formState.icon_url,
      p_cover_url: null,
      p_screenshots: formState.screenshots,
      p_platforms: platforms,
      p_permissions: permissions,
      p_website_url: $("f-website").value.trim() || null,
      p_privacy_policy_url: $("f-privacy").value.trim() || null,
      p_support_email: $("f-support").value.trim() || null,
      p_contains_ads: $("f-ads").checked,
      p_has_iap: $("f-iap").checked,
      p_is_paid: isPaidState,
      p_price_cents: isPaidState ? Math.round(priceValue * 100) : 0,
      p_currency: $("f-currency").value,
      p_status: status,
      p_documentation_md: $("f-documentation").value.trim() || null,
    });

    const version = $("f-version").value.trim();
    const isUploadMethod = downloadMethod === "upload";
    const downloadUrl = isUploadMethod ? null : ($("f-download-url").value.trim() || null);

    if (status === "published" && version && (downloadUrl || formState.storage_path)) {
      await window.poly.apps.publishRelease({
        p_app_id: wizardAppId,
        p_version: version,
        p_release_notes: $("f-release-notes").value.trim() || null,
        p_download_url: downloadUrl,
        p_storage_path: isUploadMethod ? formState.storage_path : null,
        p_size_bytes: isUploadMethod ? formState.size_bytes : null,
      });
    }

    $("publish-ok").textContent = status === "draft" ? "Draft saved." : `Published! "${appRow.name}" is live.`;
    await loadApps();
    if (status === "published") setTimeout(() => showPanel("apps"), 900);
  } catch (err) {
    $("publish-error").textContent = err.message || "Something went wrong. Try again.";
  } finally {
    $("next-btn").disabled = false;
    $("save-draft-btn").disabled = false;
  }
}

boot();
