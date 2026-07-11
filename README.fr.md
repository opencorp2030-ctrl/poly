# poly

[English](README.md) · **[Français](README.fr.md)**

Une commande, tous les gestionnaires de paquets. `poly` installe depuis
pip, npm, Homebrew, et des binaires vérifiés par checksum, derrière une
seule commande unifiée, sur macOS, Linux et Windows.

Site + gestion de compte : **[poly.candygate.eu](https://poly.candygate.eu)**

```
$ poly install ripgrep
downloading ripgrep  [############################] 100%  1.7MiB/1.7MiB
installed ripgrep 15.1.0 (via tap)
note: tap binaries are installed to ~/.poly/bin — make sure it's on your PATH

$ poly install requests@2.31.0
installed requests 2.31.0 (via pip)

$ poly install npm:cowsay
installed cowsay 1.6.0 (via npm)

$ poly install brew:jq
installed jq 1.8.2 (via brew)

$ poly list
NAME      VERSION  ADAPTER  INSTALLED
cowsay    1.6.0    npm      2026-07-11 15:52
jq        1.8.2    brew     2026-07-11 17:30
requests  2.31.0   pip      2026-07-11 15:52
ripgrep   15.1.0   tap      2026-07-11 15:52
```

## Installation

**macOS** — télécharge [`poly-macos.pkg`](https://github.com/opencorp2030-ctrl/poly/releases/latest/download/poly-macos.pkg),
ouvre-le, suis l'installeur. Installe un binaire universel
(Intel + Apple Silicon) dans `/usr/local/bin/poly`. Non signé/non
notarié (pas encore de compte Apple Developer), donc Gatekeeper va
avertir à la première ouverture — clic droit → Ouvrir, ou
`Réglages Système → Confidentialité et sécurité → Ouvrir quand même`.

**Windows** — télécharge [`poly-setup.exe`](https://github.com/opencorp2030-ctrl/poly/releases/latest/download/poly-setup.exe),
lance-le. Installe dans `%LOCALAPPDATA%\Poly` et l'ajoute au `PATH` de
ton utilisateur (pas besoin des droits admin). Non signé, donc
SmartScreen va avertir — "Plus d'infos" → "Exécuter quand même".
*(Construit en cross-plateforme via NSIS ; pas encore vérifié sur une
vraie machine Windows — remonte les problèmes si tu en rencontres.)*

**Depuis les sources** (tout OS, Go 1.21+ nécessaire) :

```
git clone https://github.com/opencorp2030-ctrl/poly.git
cd poly
go build -o poly .
./poly version           # vérif rapide : lance le binaire sur place

# mets-le sur le PATH, par exemple :
cp poly /opt/homebrew/bin/poly   # macOS (bin de Homebrew, pas besoin de sudo)
# ou : sudo cp poly /usr/local/bin/poly       # macOS/Linux, besoin de sudo
# ou : ajoute ce dossier à ton PATH toi-même

poly version              # marche maintenant depuis n'importe où
```

Recompile les installeurs toi-même avec `VERSION=x.y.z installers/build.sh`
(nécessite `makensis` pour celui de Windows : `brew install makensis`).

Cross-compile pour un autre OS/architecture avec `GOOS`/`GOARCH`, ou
lance `scripts/build-all.sh` pour builder les cinq cibles
(darwin/linux/windows × amd64/arm64) dans `dist/`.

## Comment la résolution de paquet fonctionne

`poly install <name>` essaie la recherche (`Search`) de chaque adapter
dans l'ordre — **tap → brew → pip → npm → cargo → go** — et installe via
le premier qui signale que le paquet existe. Force un backend précis
avec un préfixe :

| Commande | Résout vers |
|---|---|
| `poly install ripgrep` | première correspondance : tap → brew → pip → npm → cargo → go |
| `poly install tap:ripgrep` | téléchargement binaire forcé depuis la formule tap |
| `poly install brew:wget` | `brew install wget` forcé |
| `poly install pip:requests` | `pip install requests` forcé |
| `poly install npm:lodash` | `npm install -g lodash` forcé |
| `poly install cargo:ripgrep` | `cargo install ripgrep` forcé |
| `poly install go:golang.org/x/tools/cmd/goimports` | `go install <module>@latest` forcé |

Ajoute `@version` pour figer une version : `poly install requests@2.31.0`.
pip, npm et cargo la transmettent directement ; l'adapter tap n'offre
que la version fixée dans sa formule et échoue si tu en demandes une
autre ; `go` la traite comme la version/tag du module.

## Commandes

| Commande | Fait |
|---|---|
| `poly install [[adapter:]name[@version] ...]` | installe des paquets, routage auto ou forcé ; sans argument, installe depuis `poly.json` |
| `poly remove name` | désinstalle, via l'adapter qui a fait l'installation |
| `poly list` | affiche tout ce que poly a installé, et par quel biais |
| `poly search [adapter:]name` | vérifie l'existence et la dernière version sur tous les adapters |
| `poly info [adapter:]name` | version, résumé, site web et statut d'installation, par adapter |
| `poly init` | écrit `poly.json` à partir de tes paquets actuellement installés |
| `poly upgrade` | met à jour chaque paquet installé vers la dernière version de son adapter |
| `poly self-update` | met à jour le binaire poly lui-même vers la dernière release GitHub |
| `poly account` | affiche ton e-mail/username/bio/formule connectés |
| `poly login` / `poly logout` | connexion/déconnexion à ton compte Poly (débloque Pro) |
| `poly version` | affiche la version buildée de poly et, si connecté, ton compte/ta formule |

L'état vit dans `~/.poly/manifest.json`. Les binaires tap atterrissent
dans `~/.poly/bin` (ajoute-le à ton `PATH`). Les identifiants de connexion
vivent dans `~/.poly/credentials.json` (mode 0600). `poly self-update` et,
pour les comptes Pro, `poly upgrade` tournent aussi automatiquement en
arrière-plan (throttlé à ~1×/jour, loggé dans `~/.poly/auto-update.log`).

## Installations reproductibles avec poly.json

`poly init` écrit un `poly.json` dans le dossier courant, listant chaque
paquet installé par poly, figé à sa version exacte :

```json
{
  "packages": [
    "tap:ripgrep@15.1.0",
    "npm:eslint@9.2.0",
    "pip:requests@2.31.0"
  ]
}
```

Commit-le, et n'importe qui (ou une machine CI) peut reproduire le même
environnement avec un simple `poly install` — sans argument.

## Poly Pro

Poly est et reste 100% gratuit et open source — tout le catalogue tap
intégré (`ripgrep`, `fd`, `jq`, ...) est gratuit aussi, volontairement :
Pro n'a pas vocation à retenir des outils que les gens attendent
gratuits. Le vrai avantage mesurable aujourd'hui :

- `poly install a b c` (plusieurs paquets en une commande) les installe
  **séquentiellement** en version gratuite ; connecté avec une formule
  Pro active (`poly login`), la même commande les installe **en
  parallèle**, et `poly upgrade` tourne automatiquement en arrière-plan
  au lieu de seulement à la demande.

Gère ton compte (inscription, formule) sur
[poly.candygate.eu/account.html](https://poly.candygate.eu/account.html),
propulsé par Supabase Auth — voir `internal/account/account.go` pour la
logique côté client et la table `public.profiles` pour où vit le statut
de formule.

## Adapters

- **pip** — appelle `pip3`/`pip` en sous-processus, en streamant sa
  sortie en direct pour que tu voies la progression de pip lui-même.
  La recherche interroge l'API JSON de PyPI
  (`pypi.org/pypi/<name>/json`), une recherche par nom exact — PyPI n'a
  plus d'API de recherche libre publique.
- **npm** — appelle `npm install -g`, même sortie streamée en direct.
  La recherche interroge `registry.npmjs.org/<name>/latest`.
- **brew** — appelle `brew install`/`brew uninstall` en local, même
  sortie streamée en direct. La recherche interroge l'API publique
  `formulae.brew.sh/api/formula/<name>.json`, donc ça marche même sans
  brew installé (seule l'installation en a besoin).
- **cargo** — appelle `cargo install --force`/`cargo uninstall`. La
  recherche interroge l'API publique `crates.io/api/v1/crates/<name>`.
- **go** — appelle `go install <module>@version`. Les paquets ici sont
  des chemins d'import de module complets, pas des noms courts (ex.
  `golang.org/x/tools/cmd/goimports`), donc c'est en pratique utilisé
  uniquement via un préfixe `go:` explicite. La recherche remonte le
  chemin jusqu'à trouver le module englobant via le proxy public des
  modules Go, la même résolution que fait `go install` en interne.
- **tap** — installe des binaires précompilés directement depuis une
  URL fixée, avec une barre de progression en direct pendant le
  téléchargement, vérifiés par un checksum SHA-256, puis extraits
  (`.tar.gz`/`.zip`) ou copiés dans `~/.poly/bin`. Pas besoin de
  runtime Python, Node, Rust, ou Homebrew.

## Ajouter une formule tap

Dépose un fichier YAML dans `~/.poly/taps/<name>.yaml` (ceci surcharge
toute formule intégrée du même nom — voir
`internal/registry/embedded/taps` pour celles fournies avec poly) :

```yaml
name: ripgrep
description: "Line-oriented search tool that recursively searches directories for a regex pattern"
homepage: "https://github.com/BurntSushi/ripgrep"
version: "15.1.0"
binary: rg
tier: free   # omets pour gratuit, ou "pro" pour la réserver à une formule Pro active
artifacts:
  darwin_arm64:
    url: "https://github.com/BurntSushi/ripgrep/releases/download/15.1.0/ripgrep-15.1.0-aarch64-apple-darwin.tar.gz"
    sha256: "378e973289176ca0c6054054ee7f631a065874a352bf43f0fa60ef079b6ba715"
  darwin_amd64: { url: "...", sha256: "..." }
  linux_amd64: { url: "...", sha256: "..." }
  linux_arm64: { url: "...", sha256: "..." }
  windows_amd64: { url: "...", sha256: "..." }
```

La clé d'artifact est `<GOOS>_<GOARCH>`. `poly install <name>` choisit
l'entrée qui correspond à la machine sur laquelle il tourne. Formules
intégrées aujourd'hui, toutes gratuites : `ripgrep`, `fd`, `jq` — voir
`internal/registry/embedded/taps`.

## Structure du projet

```
main.go                          point d'entrée
cmd/                              commandes cobra (install, upgrade, init, info, account, self-update, ...)
internal/manifest/                lecture/écriture de ~/.poly/manifest.json
internal/lockfile/                lecture/écriture de poly.json
internal/adapters/                interface Adapter + implémentations pip, npm, brew, cargo, go, tap
internal/registry/embedded/       formules tap intégrées au binaire
internal/account/                 client Supabase Auth (login/formule/profil)
internal/selfupdate/              télécharge + vérifie + remplace le binaire poly
installers/                       sources des installeurs .pkg macOS et .exe Windows
site/                             site marketing, doc, compte, communauté
scripts/build-all.sh              build cross-plateforme pour les releases
```

## État

Fonctionne : install/remove/list/search/info/upgrade/init sur pip, npm,
brew, cargo, go et tap ; version figée ; auto-update et (Pro) upgrade
auto des paquets ; installeurs macOS/Windows ; système de compte
Supabase avec annuaire communautaire. Pas encore construit : installeurs
signés/notariés, une UI de recherche couvrant le catalogue complet de
chaque adapter (vs. recherche par nom exact), et les gestionnaires de
paquets des distros Linux (apt/dnf/pacman).
