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

À la première exécution, macOS va demander les autorisations. Si ça ne colle pas, vérifiez :

- Réglages Système → Confidentialité et sécurité → **Accessibilité**
- Réglages Système → Confidentialité et sécurité → **Surveillance de saisie**

## Utilisation

- Par défaut : maintenir **Fn** pour parler, relâcher pour coller la transcription.
- Un indicateur visuel type **notch** apparaît en haut de l'écran pendant l'enregistrement.
- Note : avec les touches modificateur (ex. `Fn`), il faut maintenir ~`250 ms` avant le démarrage (anti faux déclenchements), donc le notch n'apparaît qu'après ce délai.
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

- `2026-02-26` : Correctif stabilité : notch simplifié (fenêtre borderless classique) pour éviter un crash au déclenchement de l'enregistrement (`Fn`).
- `2026-02-26` : Notch d'enregistrement renforcé (plus visible, animation d'apparition/disparition, meilleure visibilité en espaces plein écran).
- `2026-02-26` : Correction de la version par défaut du build à `1.10` (affichée dans l'app et injectée dans `Info.plist`).
- `2026-02-26` : Ajout d'un overlay type **notch** (haut-centre) visible pendant l'enregistrement pour indiquer clairement l'écoute en cours.
- `2026-02-26` : La fenêtre Settings affiche maintenant la version de l'app (`CFBundleShortVersionString` + build).
- `2026-02-26` : Le build accepte `APP_VERSION` et `APP_BUILD` et les injecte dans `Info.plist`.
- `2026-02-26` : Le choix d'icône menubar affiche directement les icônes (wave + picto micro) au lieu de labels texte.
- `2026-02-26` : Ajout d'une option de réglages pour choisir l'icône menubar (`Wave` ou `Micro`) avec persistance (`NSUserDefaults`).
- `2026-02-26` : Renommage du projet `PKTranscript` -> `PKvoice` (bundle, binaire, assets, README).
- `2026-02-26` : Restructuration du dossier (`README.md` + `src/` + `release/`) et adaptation du script de build.
- `2026-02-26` : Initialisation du scaffold `PKvoice` à partir de `Macos_PKtranscript`.
