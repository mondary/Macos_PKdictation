# PKdictation — Walkthrough / Changelog

Ce document résume l’app, comment l’utiliser, et les changements effectués récemment dans ce repo.

## Fonctionnalités principales

- App macOS **menu‑bar** (pas d’icône Dock).
- Dictée **push‑to‑talk** via hotkey global (Fn par défaut, configurable).
- **Overlay** type “notch” avec spectre audio pendant l’écoute.
- Envoi audio WAV à **Gemini** pour transcription.
- Copie automatique du transcript au **presse‑papier**, avec option **auto‑paste** (Cmd+V) vers l’app active.
- **Historique** des transcriptions (menu + fenêtre “Show all history…”).
- Gestion des permissions:
  - Microphone
  - Accessibility / Input Monitoring (pour hotkey global)
- Logs:
  - logs en mémoire (affichés dans Settings)
  - logs fichier (Application Support)

## Usage (rapide)

1. Ouvre l’app (menu bar).
2. Settings → ajoute la **Gemini API key** (Keychain).
3. Choisis le **model**.
4. Configure le **push‑to‑talk** + permissions.
5. Maintiens le hotkey → enregistre, relâche → transcription + copie au presse‑papier.

## Modifications / améliorations effectuées

### Stabilité / anti‑freeze

- `PKdictation/Hotkey/HotkeyManager.swift`
  - Évite d’appeler `CGPreflightListenEventAccess()` à haute fréquence via `InputMonitoringPermission.isTrusted()` dans le callback de l’event tap.
  - Ajout d’un **cache** “Input Monitoring trusted” (rafraîchi toutes les ~2s) utilisé par la logique Fn.
- `PKdictation/Audio/AudioCaptureManager.swift`
  - Throttle des updates `levels` envoyés au main actor (≈30 fps) pour éviter de saturer l’UI.

### Overlay “notch” (look & placement)

- `PKdictation/Overlay/OverlayWindowController.swift`
  - Ajout d’un **overhang** (haut partiellement hors‑écran) pour un rendu “notch qui descend depuis l’extérieur”.
- `PKdictation/Overlay/OverlayView.swift`
  - Ajustement layout/padding pour que le contenu soit sous l’overhang.
  - Fond avec **corner radius** plus large + **dégradé** subtil.

### Historique: clic = copie + collage

- `PKdictation/AppDelegate.swift`
  - Clic sur un item d’historique:
    - copie dans le presse‑papier
    - envoie `Cmd+V` à la dernière app externe au premier plan (pour coller dans le champ sélectionné).
  - Tracking du dernier `pid` externe via `NSWorkspace.didActivateApplicationNotification`.

### Fenêtre “Show all history…”

- `PKdictation/AppDelegate.swift`
  - Correction du sizing/layout (évite fenêtre minuscule/vide).
  - Amélioration du `NSTextView` (monospace, padding, scroll correct).
  - Ajout bouton **Clear all** (avec confirmation).
- `PKdictation/AppModel.swift`
  - Ajout `HistoryStore.clearAll()` (efface + persiste).

### Menu “Start recording” en toggle

- `PKdictation/AppDelegate.swift`
  - Le titre du menu devient **Start recording / Stop recording** selon `DictationController.phase`.
  - Ajout d’un `toolTip` pour rappeler le hotkey “hold/release”.
  - `receive(on: RunLoop.main)` sur les subscriptions + refresh immédiat après clic.

### Settings: chat / requêtes IA + scroll

- `PKdictation/Gemini/GeminiClient.swift`
  - Ajout `chat(prompt:apiKey:modelName:)` (texte → texte) via `generateContent`.
  - Récupération du texte en joignant toutes les parts (robuste si plusieurs parts).
- `PKdictation/Settings/SettingsView.swift`
  - Ajout section **Ask AI**:
    - champ de saisie + boutons Send/Clear
    - zone de conversation scrollable
  - Les Settings sont maintenant dans un **NSScrollView** (sections accessibles même si la fenêtre est petite).

### Settings: statuts permissions alignés aux boutons

- `PKdictation/Settings/SettingsView.swift`
  - Affichage des statuts **en face** de:
    - Enable Accessibility…
    - Enable Input Monitoring…
    - Retry hotkey
  - L’ancien label “tout‑en‑un” est masqué.

### Settings: logs visibles

- `PKdictation/Settings/SettingsView.swift`
  - Fix du `NSTextView` logs (frame/resize/textContainer) pour éviter l’affichage vide même si les lignes existent.

## Notes build

- `make build` peut échouer si deux builds utilisent le même `-derivedDataPath` (DB lock). Utiliser un path unique si besoin.

