# PKvoice (macOS / Go)

Une petite app macOS en Go (avec bindings Objective‑C via cgo) qui :

- écoute une touche globale en mode *push‑to‑talk*
- enregistre votre voix pendant que vous maintenez la touche
- transcrit via le framework Apple **Speech**
- copie le texte dans le presse‑papiers et le colle à l’endroit du curseur (Cmd+V)

## Structure du dossier

- `README.md` : documentation projet
- `src/` : script de build + dossiers du projet
- `src/app/` : code Go (module + sources)
- `src/assets/` : icônes et assets
- `release/` : sorties de build (`.app`)

## Prérequis

- macOS 12+ recommandé
- Go 1.22+
- Autorisations macOS à accorder à l’app :
  - **Microphone**
  - **Reconnaissance vocale**
  - **Surveillance de saisie** (*Input Monitoring*) pour capter la touche globale
  - **Accessibilité** pour envoyer Cmd+V

## Build (en .app)

```bash
./src/build-app.sh
open release/PKvoice.app
```

Build avec version explicite :

```bash
APP_VERSION=1.10 APP_BUILD=1 ./src/build-app.sh
```

## Test du notch (sandbox séparé)

App de test dédiée pour travailler le notch sans casser `PKvoice` :

```bash
./src/build-notch-test.sh
open release/PKvoiceNotchTest.app
```

Cette app ouvre une petite fenêtre de contrôle avec :
- `Afficher` / `Masquer` / `Toggle`
- choix d'icône `Wave` / `Micro`
- `Recentrer` (si tu changes d'écran / espace)
- le notch de test est volontairement **abaissé** sous la barre de menu pour être plus visible

Note:
- lance de préférence le sandbox via `open release/PKvoiceNotchTest.app` (mode GUI normal)

À la première exécution, macOS va demander les autorisations. Si ça ne colle pas, vérifiez :

- Réglages Système → Confidentialité et sécurité → **Accessibilité**
- Réglages Système → Confidentialité et sécurité → **Surveillance de saisie**

## Utilisation

- Par défaut : maintenir **Fn** pour parler, relâcher pour coller la transcription.
- Menu barre “PKT” :
  - **Transcript (auto-paste)** : toggle (si OFF, ça copie seulement dans le clipboard, sans coller)
  - **Settings…** : ouvre la fenêtre de réglages
  - **Historique (10)** : affiche les 10 dernières transcriptions (clic pour copier)
  - Quitter : *Quitter PKvoice*

### Choisir une touche / locale

L’app accepte des flags si vous lancez le binaire directement (dans l’app bundle : `Contents/MacOS/pkvoice`) :

```bash
release/PKvoice.app/Contents/MacOS/pkvoice --hotkey f7 --locale fr-FR
```

`--hotkey` accepte aussi un keycode macOS (ex: `0x61` pour F6).

Hotkeys utiles (push-to-talk) :
- `fn` (maintenir)
- `rcmd` / `cmd` (maintenir)
- `ropt` / `lopt` (maintenir)

## Changelog

- `2026-02-26` : Correctif `PKvoiceNotchTest` : suppression d'une combinaison invalide de `collectionBehavior` (`canJoinAllSpaces` + `moveToActiveSpace`) qui provoquait un crash AppKit ; alignement partiel sur le pattern `NSPanel` de `notchprompt`.
- `2026-02-26` : `PKvoiceNotchTest` rendu plus visible (affichage différé, niveau de fenêtre plus haut, contour clair, position abaissée sous la barre de menu).
- `2026-02-26` : Ajout de `PKvoiceNotchTest.app` (sandbox séparé) pour tester le notch indépendamment de `PKvoice`.
- `2026-02-26` : Correction de la version par défaut du build à `1.10` (affichée dans l'app et injectée dans `Info.plist`).
- `2026-02-26` : La fenêtre Settings affiche maintenant la version de l'app (`CFBundleShortVersionString` + build).
- `2026-02-26` : Le build accepte `APP_VERSION` et `APP_BUILD` et les injecte dans `Info.plist`.
- `2026-02-26` : Le choix d'icône menubar affiche directement les icônes (wave + picto micro) au lieu de labels texte.
- `2026-02-26` : Ajout d'une option de réglages pour choisir l'icône menubar (`Wave` ou `Micro`) avec persistance (`NSUserDefaults`).
- `2026-02-26` : Renommage du projet `PKTranscript` -> `PKvoice` (bundle, binaire, assets, README).
- `2026-02-26` : Restructuration du dossier (`README.md` + `src/` + `release/`) et adaptation du script de build.
- `2026-02-26` : Initialisation du scaffold `PKvoice` à partir de `Macos_PKtranscript`.
