# PKdictation (macOS)

Menu-bar dictation app (push-to-talk) that records your voice, sends it to the Gemini API for transcription, and shows the result in a small overlay bubble (with an audio “spectrum”).

## Features

- Menu bar app (no Dock icon)
- Push-to-talk global hotkey (default: `Fn`, configurable)
- Overlay bubble with live audio meter + status + transcription
- Copies the final transcript to the clipboard automatically
- Gemini API key stored in macOS Keychain

## Tester l’app (pas besoin d’Xcode UI)

Depuis la racine du repo:

```sh
make export
open dist/PKdictation.app
```

Au lancement, l’app **n’a pas d’icône dans le Dock** : cherchez l’icône `waveform` dans la barre de menu.

Ensuite, dans le menu de l’icône → **Settings…** :

1. Collez votre **Gemini API key** (stockée dans le Keychain).
2. Vérifiez le **model** (par défaut `gemini-2.0-flash`).
3. Configurez le **push-to-talk** (par défaut `Fn`, ou `Right ⌘`).

Permissions nécessaires :

- **Microphone** (demandé au 1er enregistrement)
- **Privacy & Security → Accessibility** (et parfois **Input Monitoring**) pour le hotkey global

### macOS 15 (conseil important)

Pour que macOS te propose plus facilement **PKdictation** dans les listes **Accessibilité / Input Monitoring**, lance l’app depuis un emplacement “stable” :

1. `make export`
2. Copie `dist/PKdictation.app` dans `/Applications`
3. Lance-la depuis `/Applications`, puis redemande les permissions (Settings → Push-to-talk)

## Usage

- Hold your push-to-talk shortcut (default `Fn`) to record.
- Release to send the audio to Gemini and get the transcript.
- The transcript is displayed in the overlay and copied to the clipboard.

## Dépannage (phrase tronquée à la fin)

Si tu constates que la fin de tes phrases est parfois coupée, l’app ajoute maintenant un petit “tail padding” (≈250ms) avant de fermer le fichier audio, puis attend brièvement (≈120ms) avant de lire/transcrire. Tu peux vérifier dans **Settings → Logs** que la ligne `Stop: keeping 250ms tail...` apparaît.

## Build (CLI)

```sh
xcodebuild -project PKdictation.xcodeproj -scheme PKdictation -configuration Debug -derivedDataPath ./.derivedData build
```

## Build/Run (Make)

```sh
make build
make run
make relaunch
make status
make quit
make export
make crash
make test
make clean
```

Note: dans certains environnements sandboxés, `open` peut être bloqué. Dans ce cas, lancez `dist/PKdictation.app` via Finder (double-clic) ou depuis Terminal.app.
