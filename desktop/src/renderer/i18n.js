"use strict";

const I18N = {
  en: {
    onboard_badge1: "DESKTOP", onboard_h1: "Your apps.", onboard_h1_2: "Off the browser.",
    onboard_p1: "Poly Desktop is where you build, ship, and maintain what you publish to the Poly Apps store — without a browser tab open.",
    tour1_h: "Publish in minutes", tour1_p: "Icon, screenshots, pricing and a release — one guided flow, with a live preview of your store listing as you type.",
    tour2_h: "Real documentation", tour2_p: "Give every app a proper docs page — setup, usage, configuration — right next to the listing, not bolted on after.",
    tour3_h: "One account, everywhere", tour3_p: "Signs in with the same account as poly login on the CLI. No separate password to manage.",
    onboard_badge3: "READY", onboard_h3: "Let's sign you in.", onboard_p3: "Use your Poly account — the one you already have from poly.candygate.eu or the CLI.",
    onboard_continue: "Continue →", onboard_signin: "Sign in →",

    login_h: "Sign in", login_note: "Continue in a secure Poly sign-in window.", login_submit: "Continue with Poly",
    login_no_account: "No account yet?", login_create: "Create one on poly.candygate.eu →",

    nav_apps: "My apps", nav_packages: "Packages", nav_publish: "Publish new app", nav_settings: "Settings",

    apps_h: "My apps", apps_new: "+ New app", apps_empty: "You haven't published anything yet.", apps_empty_cta: "Publish your first app",
    apps_summary: "{n} app{s} · {p} published",
    status_draft: "draft", status_published: "published", free: "Free", installs: "installs",
    btn_edit: "Edit", btn_open: "Open", btn_delete: "Delete",
    confirm_delete_title: "Delete \"{name}\"?", confirm_delete_body: "This removes the listing and its stored files. This can't be undone.",
    confirm_cancel: "Cancel", confirm_delete: "Delete",

    publish_h_new: "Publish an app", publish_h_edit: "Edit \"{name}\"",
    publish_sub: "Fill in the details — the card on the right updates as you go.",
    publish_back: "← Back to my apps",
    step_info: "Info", step_listing: "Listing", step_pricing: "Pricing", step_release: "Release",
    f_name: "App name", f_tagline: "Short tagline", f_tagline_hint: "(max 80 characters)",
    f_category: "Category", f_content_rating: "Content rating", f_platforms: "Platforms",
    f_website: "Website", f_optional: "(optional)", f_support: "Support email", f_privacy: "Privacy policy URL",
    f_icon: "App icon", f_icon_hint: "512×512 square", f_screenshots: "Screenshots", f_screenshots_hint: "(up to 8)",
    f_description: "Full description", f_documentation: "Documentation", f_documentation_hint: "— setup, usage, configuration",
    f_pricing: "Pricing", pricing_free: "Free", pricing_paid: "Paid", f_price: "Price", f_currency: "Currency",
    f_ads: "Contains ads", f_iap: "Has in-app purchases", f_permissions: "Permissions this app requests",
    f_version: "Version", f_release_notes: "What's new", f_release_notes_hint: "(release notes)",
    f_download_method: "How should people get this app?", method_url: "External link", method_upload: "Upload a build",
    f_build_hint: "Choose a build file", download_hint: "Capped at 200 MB. Paid apps gate the download until purchase.",
    btn_back: "← Back", btn_save_draft: "Save draft", btn_next: "Next →", btn_publish: "Publish", btn_save: "Save changes",
    preview_label: "Store listing preview", preview_name: "Your app name", preview_tagline: "Your tagline shows up here",
    err_name_required: "Give your app a name first.", err_generic: "Something went wrong. Try again.",
    draft_saved: "Draft saved.", published_ok: "Published! \"{name}\" is live.",

    pkg_h: "Packages", pkg_sub: "Search, install, and manage packages via the Poly CLI — same as running poly in a terminal.",
    pkg_search_placeholder: "Search a package (e.g. ripgrep, requests, lodash)…", pkg_search_btn: "Search",
    pkg_installed_h: "Installed via Poly", pkg_installed_empty: "No packages installed via Poly yet.",
    pkg_results_empty: "No results.", pkg_install_btn: "Install", pkg_remove_btn: "Remove", pkg_installing: "Installing…", pkg_removing: "Removing…",
    pkg_cli_missing_h: "Poly CLI not found", pkg_cli_missing_p: "Poly Desktop uses the Poly CLI to search, install, and remove packages. Install it to use this tab.",
    pkg_cli_missing_cta: "Get the Poly CLI →", pkg_recheck: "Check again",

    settings_h: "Settings", settings_account: "Signed in as", settings_plan: "Plan", settings_version: "Poly Desktop",
    settings_language: "Language", settings_manage: "Manage account on poly.candygate.eu →", settings_signout: "Sign out",
    plan_pro: "Pro", plan_free: "Free",
  },
  fr: {
    onboard_badge1: "BUREAU", onboard_h1: "Tes applications.", onboard_h1_2: "Sans navigateur.",
    onboard_p1: "Poly Desktop, c'est là où tu construis, publies et maintiens ce que tu mets sur le store Poly Apps — sans avoir un onglet de navigateur ouvert.",
    tour1_h: "Publie en quelques minutes", tour1_p: "Icône, captures, prix et release — un seul parcours guidé, avec un aperçu live de ta fiche pendant que tu remplis.",
    tour2_h: "Une vraie documentation", tour2_p: "Donne à chaque app une vraie page de doc — installation, usage, configuration — juste à côté de la fiche, pas ajoutée après coup.",
    tour3_h: "Un seul compte, partout", tour3_p: "Connexion avec le même compte que poly login en CLI. Pas de mot de passe séparé à gérer.",
    onboard_badge3: "PRÊT", onboard_h3: "On te connecte.", onboard_p3: "Utilise ton compte Poly — celui que tu as déjà sur poly.candygate.eu ou en CLI.",
    onboard_continue: "Continuer →", onboard_signin: "Se connecter →",

    login_h: "Connexion", login_note: "Continue dans une fenêtre de connexion Poly sécurisée.", login_submit: "Continuer avec Poly",
    login_no_account: "Pas encore de compte ?", login_create: "Créer un compte sur poly.candygate.eu →",

    nav_apps: "Mes apps", nav_packages: "Packages", nav_publish: "Publier une app", nav_settings: "Paramètres",

    apps_h: "Mes apps", apps_new: "+ Nouvelle app", apps_empty: "Tu n'as encore rien publié.", apps_empty_cta: "Publier ta première app",
    apps_summary: "{n} app{s} · {p} publiée{s}",
    status_draft: "brouillon", status_published: "publiée", free: "Gratuit", installs: "installs",
    btn_edit: "Modifier", btn_open: "Ouvrir", btn_delete: "Supprimer",
    confirm_delete_title: "Supprimer « {name} » ?", confirm_delete_body: "Ça supprime la fiche et les fichiers associés. Irréversible.",
    confirm_cancel: "Annuler", confirm_delete: "Supprimer",

    publish_h_new: "Publier une app", publish_h_edit: "Modifier « {name} »",
    publish_sub: "Remplis les infos — la fiche à droite se met à jour au fur et à mesure.",
    publish_back: "← Retour à mes apps",
    step_info: "Infos", step_listing: "Fiche", step_pricing: "Prix", step_release: "Release",
    f_name: "Nom de l'app", f_tagline: "Accroche courte", f_tagline_hint: "(80 caractères max)",
    f_category: "Catégorie", f_content_rating: "Classification", f_platforms: "Plateformes",
    f_website: "Site web", f_optional: "(optionnel)", f_support: "Email de support", f_privacy: "URL de confidentialité",
    f_icon: "Icône de l'app", f_icon_hint: "carré 512×512", f_screenshots: "Captures d'écran", f_screenshots_hint: "(8 max)",
    f_description: "Description complète", f_documentation: "Documentation", f_documentation_hint: "— installation, usage, configuration",
    f_pricing: "Tarification", pricing_free: "Gratuit", pricing_paid: "Payant", f_price: "Prix", f_currency: "Devise",
    f_ads: "Contient des publicités", f_iap: "Contient des achats intégrés", f_permissions: "Permissions demandées",
    f_version: "Version", f_release_notes: "Nouveautés", f_release_notes_hint: "(notes de version)",
    f_download_method: "Comment récupère-t-on l'app ?", method_url: "Lien externe", method_upload: "Uploader un fichier",
    f_build_hint: "Choisir un fichier", download_hint: "Limité à 200 Mo. Les apps payantes bloquent le téléchargement avant achat.",
    btn_back: "← Retour", btn_save_draft: "Sauver le brouillon", btn_next: "Suivant →", btn_publish: "Publier", btn_save: "Enregistrer",
    preview_label: "Aperçu de la fiche store", preview_name: "Nom de ton app", preview_tagline: "Ton accroche apparaît ici",
    err_name_required: "Donne d'abord un nom à ton app.", err_generic: "Un problème est survenu. Réessaie.",
    draft_saved: "Brouillon sauvegardé.", published_ok: "Publiée ! « {name} » est en ligne.",

    pkg_h: "Packages", pkg_sub: "Recherche, installe et gère des packages via la CLI Poly — comme en tapant poly dans un terminal.",
    pkg_search_placeholder: "Rechercher un package (ex. ripgrep, requests, lodash)…", pkg_search_btn: "Rechercher",
    pkg_installed_h: "Installés via Poly", pkg_installed_empty: "Aucun package installé via Poly pour l'instant.",
    pkg_results_empty: "Aucun résultat.", pkg_install_btn: "Installer", pkg_remove_btn: "Supprimer", pkg_installing: "Installation…", pkg_removing: "Suppression…",
    pkg_cli_missing_h: "CLI Poly introuvable", pkg_cli_missing_p: "Poly Desktop utilise la CLI Poly pour rechercher, installer et supprimer des packages. Installe-la pour utiliser cet onglet.",
    pkg_cli_missing_cta: "Télécharger la CLI Poly →", pkg_recheck: "Revérifier",

    settings_h: "Paramètres", settings_account: "Connecté en tant que", settings_plan: "Offre", settings_version: "Poly Desktop",
    settings_language: "Langue", settings_manage: "Gérer le compte sur poly.candygate.eu →", settings_signout: "Se déconnecter",
    plan_pro: "Pro", plan_free: "Gratuit",
  },
};

const CATEGORIES = [
  ["productivity", "Productivity", "Productivité"], ["developer-tools", "Developer tools", "Outils développeur"],
  ["games", "Games", "Jeux"], ["social", "Social", "Social"], ["utilities", "Utilities", "Utilitaires"],
  ["education", "Education", "Éducation"], ["entertainment", "Entertainment", "Divertissement"],
  ["photo-video", "Photo & video", "Photo et vidéo"], ["music-audio", "Music & audio", "Musique et audio"],
  ["health-fitness", "Health & fitness", "Santé et fitness"], ["business", "Business", "Business"],
  ["finance", "Finance", "Finance"], ["lifestyle", "Lifestyle", "Style de vie"], ["news", "News", "Actualités"],
  ["travel-local", "Travel & local", "Voyage et local"], ["other", "Other", "Autre"],
];
const CONTENT_RATINGS = [
  ["everyone", "Everyone", "Tout public"], ["teen", "Teen", "Ado"], ["mature", "Mature 17+", "Mature 17+"], ["adults", "Adults only", "Adultes uniquement"],
];
const PLATFORMS = [["windows", "Windows"], ["macos", "macOS"], ["linux", "Linux"], ["web", "Web"], ["android", "Android"], ["ios", "iOS"]];
const PERMISSIONS = [
  ["camera", "Camera", "Caméra"], ["microphone", "Microphone", "Microphone"], ["location", "Location", "Position"],
  ["contacts", "Contacts", "Contacts"], ["storage", "Storage", "Stockage"], ["network", "Network", "Réseau"],
  ["notifications", "Notifications", "Notifications"], ["calendar", "Calendar", "Calendrier"], ["sms", "SMS", "SMS"], ["bluetooth", "Bluetooth", "Bluetooth"],
];

let currentLang = "en";

function setLang(l) { currentLang = I18N[l] ? l : "en"; }
function getLang() { return currentLang; }

function t(key, vars) {
  let s = (I18N[currentLang] && I18N[currentLang][key]) ?? I18N.en[key] ?? key;
  if (vars) {
    for (const [k, v] of Object.entries(vars)) s = s.split(`{${k}}`).join(v);
  }
  return s;
}

function catLabel(key) { const c = CATEGORIES.find((x) => x[0] === key); return c ? (currentLang === "fr" ? c[2] : c[1]) : key; }
function ratingLabel(key) { const c = CONTENT_RATINGS.find((x) => x[0] === key); return c ? (currentLang === "fr" ? c[2] : c[1]) : key; }
function platformLabel(key) { const c = PLATFORMS.find((x) => x[0] === key); return c ? c[1] : key; }
function permissionLabel(key) { const c = PERMISSIONS.find((x) => x[0] === key); return c ? (currentLang === "fr" ? c[2] : c[1]) : key; }
