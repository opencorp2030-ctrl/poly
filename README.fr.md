# poly

[English](README.md) · **[Français](README.fr.md)**

Une commande, tous les gestionnaires de paquets. `poly` installe depuis
pip, npm, Homebrew, et des binaires vérifiés par checksum, derrière une
seule commande unifiée, sur macOS, Linux et Windows.

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

Nécessite Go 1.21+. Il n'y a pas encore d'installeur hébergé, donc il
faut compiler depuis les sources et mettre le binaire quelque part sur
ton `PATH` :

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

Cross-compile pour un autre OS/architecture avec `GOOS`/`GOARCH`, ou
lance `scripts/build-all.sh` pour builder les cinq cibles
(darwin/linux/windows × amd64/arm64) dans `dist/`.

## Comment la résolution de paquet fonctionne

`poly install <name>` essaie la recherche (`Search`) de chaque adapter
dans l'ordre — **tap → brew → pip → npm** — et installe via le premier
qui signale que le paquet existe. Force un backend précis avec un préfixe :

| Commande | Résout vers |
|---|---|
| `poly install ripgrep` | première correspondance : tap → brew → pip → npm |
| `poly install tap:ripgrep` | téléchargement binaire forcé depuis la formule tap |
| `poly install brew:wget` | `brew install wget` forcé |
| `poly install pip:requests` | `pip install requests` forcé |
| `poly install npm:lodash` | `npm install -g lodash` forcé |

Ajoute `@version` pour figer une version : `poly install requests@2.31.0`.
Les adapters pip et npm la transmettent directement ; l'adapter tap
n'offre que la version fixée dans sa formule et échoue si tu en demandes
une autre.

## Commandes

| Commande | Fait |
|---|---|
| `poly install [adapter:]name[@version] ...` | installe un ou plusieurs paquets, routage auto ou forcé |
| `poly remove name` | désinstalle, via l'adapter qui a fait l'installation |
| `poly list` | affiche tout ce que poly a installé, et par quel biais |
| `poly search [adapter:]name` | vérifie l'existence et la dernière version sur tous les adapters |
| `poly login` / `poly logout` | connexion/déconnexion à ton compte Poly (débloque Pro) |
| `poly version` | affiche la version buildée de poly et, si connecté, ton compte/ta formule |

L'état vit dans `~/.poly/manifest.json`. Les binaires tap atterrissent
dans `~/.poly/bin` (ajoute-le à ton `PATH`). Les identifiants de connexion
vivent dans `~/.poly/credentials.json` (mode 0600).

## Poly Pro

Poly est et reste 100% gratuit et open source. Deux vrais avantages
Pro :

- `poly install a b c` (plusieurs paquets en une commande) les installe
  **séquentiellement** en version gratuite ; connecté avec une formule Pro
  active (`poly login`), la même commande les installe **en parallèle** —
  un vrai gain mesurable, pas un ralentissement artificiel de la version
  gratuite.
- Le catalogue tap a des formules réservées à Pro (actuellement `fd`,
  `jq`) en plus des gratuites (actuellement `ripgrep`) — `poly search`
  les montre dans les deux cas, marquées `[pro]`, mais `poly install`
  bloque avec un message clair si tu n'es pas connecté en Pro.

Gère ton compte (inscription, formule) sur `site/account.html`, propulsé
par Supabase Auth — voir `internal/account/account.go` pour la logique
côté client et la table `public.profiles` pour où vit le statut de
formule.

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
- **tap** — installe des binaires précompilés directement depuis une
  URL fixée, avec une barre de progression en direct pendant le
  téléchargement, vérifiés par un checksum SHA-256, puis extraits
  (`.tar.gz`/`.zip`) ou copiés dans `~/.poly/bin`. Pas besoin de
  runtime Python, Node, ou Homebrew. Les formules peuvent être marquées
  `tier: pro` — voir plus bas.

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
intégrées aujourd'hui : `ripgrep` (gratuite), `fd` et `jq` (pro — voir
`internal/registry/embedded/taps`).

## Structure du projet

```
main.go                          point d'entrée
cmd/                              commandes cobra (install, remove, list, search, version)
internal/manifest/                lecture/écriture de ~/.poly/manifest.json
internal/adapters/                interface Adapter + implémentations pip, npm, tap
internal/registry/embedded/       formules tap intégrées au binaire
site/                             site marketing/doc (site/index.html)
scripts/build-all.sh              build cross-plateforme pour les releases
```

## État

Early-stage. Fonctionne : install/remove/list/search sur pip, npm et
tap ; version figée ; builds cross-plateforme. Pas encore construit :
un installeur/registre hébergé, plus de formules tap intégrées, et une
UI de recherche de paquets sur le site.
