"use strict";

const I18N = {
  en: {
    onboard_badge1: "DESKTOP", onboard_h1: "Your apps.", onboard_h1_2: "Off the browser.",
    onboard_p1: "Poly Desktop is where you build, ship, and maintain what you publish to the Poly Apps store — without a browser tab open.",
    tour1_h: "Publish in minutes", tour1_p: "Icon, screenshots, pricing and a release — one guided flow, with a live preview of your store listing as you type.",
    tour2_h: "Real documentation", tour2_p: "Give every app a proper docs page — setup, usage, configuration — right next to the listing, not bolted on after.",
    tour3_h: "One account, everywhere", tour3_p: "Signs in with the same account as poly login on the CLI. No separate password to manage.",
    onboard_badge2: "MAKE IT YOURS", onboard_h2: "Pick a look.",
    personalize_theme: "Theme", personalize_accent: "Accent color",
    theme_dark: "Dark", theme_light: "Light", theme_system: "System",
    settings_theme: "Theme", settings_accent: "Accent color",
    onboard_badge3: "READY", onboard_h3: "Let's sign you in.", onboard_p3: "Use your Poly account — the one you already have from poly.candygate.eu or the CLI.",
    onboard_continue: "Continue →", onboard_signin: "Sign in →",

    login_h: "Sign in", login_note: "Continue in a secure Poly sign-in window.", login_submit: "Continue with Poly",
    login_no_account: "No account yet?", login_create: "Create one on poly.candygate.eu →",

    nav_apps: "My apps", nav_packages: "Packages", nav_publish: "Publish new app", nav_settings: "Settings",
    nav_site_h: "Poly site", nav_home: "poly>", nav_install: "install", nav_how: "how it works", nav_commands: "commands",
    nav_dependencies: "dependencies", nav_docs: "docs", nav_pro: "pro", nav_community: "community", nav_apps_store: "apps store",
    nav_integrations: "integrations", nav_search: "search", nav_account: "account", nav_github: "github",

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

    account_profile_h: "Profile", account_username: "Username", account_bio: "Bio", account_save: "Save profile", account_saved: "Profile saved.",
    account_notifications_h: "Notifications",
    notif_follow: "New followers", notif_download: "Downloads of my packages/apps",
    notif_friend_request: "Friend requests", notif_friend_accept: "Accepted friend requests",

    community_h: "Community", community_sub: "Find and follow other Poly members.",
    community_search_placeholder: "Search by username…",
    follow_btn: "Follow", unfollow_btn: "Unfollow",

    store_h: "Apps store", store_sub: "Browse what the community has published — the public store, not just your own apps.",
    store_search_placeholder: "Search apps…", store_sort_new: "Newest", store_sort_rating: "Top rated", store_sort_downloads: "Most installed",
    store_open_btn: "View on site",

    docs_h: "Docs",
    doc_toc_install: "Get poly", doc_toc_how: "How it works", doc_toc_commands: "Commands", doc_toc_adapters: "Adapters",
    doc_toc_lockfile: "poly.json", doc_toc_taps: "Tap formulas", doc_toc_registry: "Publishing", doc_toc_pro: "Pro",
    doc_install_h: "Get poly",
    doc_install_p: "You're already running Poly Desktop, but the CLI is a separate install (the Packages tab uses it). Native installers for macOS and Windows; both unsigned for now, so your OS will warn on first run.",
    doc_install_note: "Cross-compiling only needs GOOS/GOARCH, e.g. GOOS=windows GOARCH=amd64 go build -o poly.exe .",
    doc_how_h: "How a name resolves to an adapter",
    doc_how_p: "Without a prefix, poly install name tries each adapter's search in this order and installs through the first hit:",
    doc_th_command: "Command", doc_th_resolves: "Resolves to", doc_th_does: "Does", doc_th_adapter: "Adapter", doc_th_install: "Install", doc_th_search: "Search source",
    doc_how_row1: "first match: tap → brew → pip → npm → cargo → go", doc_how_row2: "forced binary download from the tap formula",
    doc_how_row3: "forced pip install requests", doc_how_row4: "forced npm install -g lodash",
    doc_commands_h: "Commands",
    doc_cmd_install: "Install one or more packages, auto-routed or forced. No args installs from poly.json.",
    doc_cmd_remove: "Uninstall, via whichever adapter installed it.", doc_cmd_list: "Show everything poly has installed.",
    doc_cmd_search: "Check existence and latest version.", doc_cmd_init: "Write poly.json from what's currently installed.",
    doc_cmd_upgrade: "Update every installed package to its latest version.", doc_cmd_selfupdate: "Update poly itself to the latest release.",
    doc_cmd_login: "Sign in/out. Credentials live in ~/.poly/credentials.json.", doc_cmd_send: "Publish to the community registry.",
    doc_cmd_doctor: "Check for common setup problems.",
    doc_commands_note: "State lives in ~/.poly/manifest.json. Tap binaries land in ~/.poly/bin — make sure it's on your PATH.",
    doc_adapters_h: "Adapters", doc_ad_tap: "Direct binary download, SHA-256 verified, extracted to ~/.poly/bin",
    doc_ad_tap_s: "Embedded/local YAML formulas", doc_ad_go_s: "Go module proxy",
    doc_lockfile_h: "Reproducible installs: poly.json",
    doc_lockfile_p: "poly init writes poly.json listing every installed package, pinned to its exact version:",
    doc_lockfile_p2: "Commit it. Anyone reproduces the same environment with a plain poly install, no arguments needed.",
    doc_taps_h: "Writing a tap formula",
    doc_taps_p: "Drop a YAML file into ~/.poly/taps/<name>.yaml, overriding any built-in formula of the same name:",
    doc_taps_p2: "Built-in formulas today, all free: ripgrep, fd, jq.",
    doc_registry_h: "Publishing your own package",
    doc_registry_p: "poly send name version path uploads a file or directory to the community registry under your account.",
    doc_registry_note: "No malware scanning — only install community packages you trust. Names are first-come-first-served.",
    doc_pro_h: "Poly Pro", doc_pro_p: "Poly is and stays 100% free and open source. Pro is a small optional pack.",
    doc_pro1_h: "Parallel installs", doc_pro1_p: "Multi-package installs run concurrently instead of one at a time.",
    doc_pro2_h: "Automatic upgrades", doc_pro2_p: "poly upgrade runs automatically in the background, throttled to ~once a day.",
    doc_pro3_h: "You fund development", doc_pro3_p: "A supporter badge next to your name, and you directly fund what gets built next.",
    doc_pro4_h: "Priority support", doc_pro4_p: "Pro accounts get answered first at open.corp.2030@gmail.com.",
    doc_pro_cta: "Get Poly Pro (€0.99) →",
  },
  fr: {
    onboard_badge1: "BUREAU", onboard_h1: "Tes applications.", onboard_h1_2: "Sans navigateur.",
    onboard_p1: "Poly Desktop, c'est là où tu construis, publies et maintiens ce que tu mets sur le store Poly Apps — sans avoir un onglet de navigateur ouvert.",
    tour1_h: "Publie en quelques minutes", tour1_p: "Icône, captures, prix et release — un seul parcours guidé, avec un aperçu live de ta fiche pendant que tu remplis.",
    tour2_h: "Une vraie documentation", tour2_p: "Donne à chaque app une vraie page de doc — installation, usage, configuration — juste à côté de la fiche, pas ajoutée après coup.",
    tour3_h: "Un seul compte, partout", tour3_p: "Connexion avec le même compte que poly login en CLI. Pas de mot de passe séparé à gérer.",
    onboard_badge2: "PERSONNALISE", onboard_h2: "Choisis ton style.",
    personalize_theme: "Thème", personalize_accent: "Couleur d'accent",
    theme_dark: "Sombre", theme_light: "Clair", theme_system: "Système",
    settings_theme: "Thème", settings_accent: "Couleur d'accent",
    onboard_badge3: "PRÊT", onboard_h3: "On te connecte.", onboard_p3: "Utilise ton compte Poly — celui que tu as déjà sur poly.candygate.eu ou en CLI.",
    onboard_continue: "Continuer →", onboard_signin: "Se connecter →",

    login_h: "Connexion", login_note: "Continue dans une fenêtre de connexion Poly sécurisée.", login_submit: "Continuer avec Poly",
    login_no_account: "Pas encore de compte ?", login_create: "Créer un compte sur poly.candygate.eu →",

    nav_apps: "Mes apps", nav_packages: "Packages", nav_publish: "Publier une app", nav_settings: "Paramètres",
    nav_site_h: "Site Poly", nav_home: "poly>", nav_install: "installer", nav_how: "fonctionnement", nav_commands: "commandes",
    nav_dependencies: "dépendances", nav_docs: "doc", nav_pro: "pro", nav_community: "communauté", nav_apps_store: "boutique d'apps",
    nav_integrations: "intégrations", nav_search: "recherche", nav_account: "compte", nav_github: "github",

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

    account_profile_h: "Profil", account_username: "Nom d'utilisateur", account_bio: "Bio", account_save: "Enregistrer le profil", account_saved: "Profil enregistré.",
    account_notifications_h: "Notifications",
    notif_follow: "Nouveaux abonnés", notif_download: "Téléchargements de mes packages/apps",
    notif_friend_request: "Demandes d'ami", notif_friend_accept: "Demandes d'ami acceptées",

    community_h: "Communauté", community_sub: "Trouve et suis d'autres membres Poly.",
    community_search_placeholder: "Rechercher par nom d'utilisateur…",
    follow_btn: "Suivre", unfollow_btn: "Ne plus suivre",

    store_h: "Boutique d'apps", store_sub: "Parcours ce que la communauté a publié — la vraie boutique publique, pas juste tes apps.",
    store_search_placeholder: "Rechercher des apps…", store_sort_new: "Plus récentes", store_sort_rating: "Mieux notées", store_sort_downloads: "Plus installées",
    store_open_btn: "Voir sur le site",

    docs_h: "Doc",
    doc_toc_install: "Installer", doc_toc_how: "Fonctionnement", doc_toc_commands: "Commandes", doc_toc_adapters: "Adaptateurs",
    doc_toc_lockfile: "poly.json", doc_toc_taps: "Formules tap", doc_toc_registry: "Publication", doc_toc_pro: "Pro",
    doc_install_h: "Installer poly",
    doc_install_p: "Tu utilises déjà Poly Desktop, mais la CLI s'installe séparément (l'onglet Packages en a besoin). Installeurs natifs macOS/Windows ; non signés pour l'instant, ton OS avertira au premier lancement.",
    doc_install_note: "Pour compiler pour une autre plateforme, il suffit de GOOS/GOARCH, ex. GOOS=windows GOARCH=amd64 go build -o poly.exe .",
    doc_how_h: "Comment un nom se résout en adaptateur",
    doc_how_p: "Sans préfixe, poly install nom essaie la recherche de chaque adaptateur dans cet ordre et installe via le premier trouvé :",
    doc_th_command: "Commande", doc_th_resolves: "Se résout en", doc_th_does: "Fait", doc_th_adapter: "Adaptateur", doc_th_install: "Installation", doc_th_search: "Source de recherche",
    doc_how_row1: "premier trouvé : tap → brew → pip → npm → cargo → go", doc_how_row2: "téléchargement binaire forcé via la formule tap",
    doc_how_row3: "force pip install requests", doc_how_row4: "force npm install -g lodash",
    doc_commands_h: "Commandes",
    doc_cmd_install: "Installe un ou plusieurs packages, auto-routé ou forcé. Sans argument, installe depuis poly.json.",
    doc_cmd_remove: "Désinstalle, via l'adaptateur qui l'a installé.", doc_cmd_list: "Affiche tout ce que poly a installé.",
    doc_cmd_search: "Vérifie l'existence et la dernière version.", doc_cmd_init: "Écrit poly.json à partir de ce qui est installé.",
    doc_cmd_upgrade: "Met à jour tous les packages installés vers leur dernière version.", doc_cmd_selfupdate: "Met à jour poly lui-même.",
    doc_cmd_login: "Connexion/déconnexion. Identifiants dans ~/.poly/credentials.json.", doc_cmd_send: "Publie sur le registre communautaire.",
    doc_cmd_doctor: "Vérifie les problèmes de configuration courants.",
    doc_commands_note: "L'état vit dans ~/.poly/manifest.json. Les binaires tap atterrissent dans ~/.poly/bin — vérifie qu'il est dans ton PATH.",
    doc_adapters_h: "Adaptateurs", doc_ad_tap: "Téléchargement binaire direct, vérifié SHA-256, extrait dans ~/.poly/bin",
    doc_ad_tap_s: "Formules YAML intégrées/locales", doc_ad_go_s: "Proxy des modules Go",
    doc_lockfile_h: "Installations reproductibles : poly.json",
    doc_lockfile_p: "poly init écrit poly.json listant chaque package installé, épinglé à sa version exacte :",
    doc_lockfile_p2: "Commite-le. N'importe qui reproduit le même environnement avec un simple poly install, sans argument.",
    doc_taps_h: "Écrire une formule tap",
    doc_taps_p: "Dépose un fichier YAML dans ~/.poly/taps/<nom>.yaml, qui remplace toute formule intégrée du même nom :",
    doc_taps_p2: "Formules intégrées aujourd'hui, toutes gratuites : ripgrep, fd, jq.",
    doc_registry_h: "Publier ton propre package",
    doc_registry_p: "poly send nom version chemin envoie un fichier ou dossier sur le registre communautaire sous ton compte.",
    doc_registry_note: "Aucun scan antimalware — n'installe que des packages communautaires en qui tu as confiance. Premier arrivé, premier servi.",
    doc_pro_h: "Poly Pro", doc_pro_p: "Poly reste 100% gratuit et open source. Pro est un petit pack optionnel.",
    doc_pro1_h: "Installations parallèles", doc_pro1_p: "Les installations multi-packages tournent en parallèle au lieu d'une par une.",
    doc_pro2_h: "Mises à jour automatiques", doc_pro2_p: "poly upgrade tourne automatiquement en arrière-plan, environ une fois par jour.",
    doc_pro3_h: "Tu finances le développement", doc_pro3_p: "Un badge de soutien à côté de ton nom, et tu finances directement la prochaine fonctionnalité.",
    doc_pro4_h: "Support prioritaire", doc_pro4_p: "Les comptes Pro sont répondus en premier à open.corp.2030@gmail.com.",
    doc_pro_cta: "Passer Poly Pro (0,99€) →",
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
