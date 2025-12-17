# PKTranscript (macOS / Go)

Une petite app macOS en Go (avec bindings Objective‑C via cgo) qui :

- écoute une touche globale en mode *push‑to‑talk*
- enregistre votre voix pendant que vous maintenez la touche
- transcrit via le framework Apple **Speech**
- copie le texte dans le presse‑papiers et le colle à l’endroit du curseur (Cmd+V)

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
./build-app.sh
open dist/PKTranscript.app
```

À la première exécution, macOS va demander les autorisations. Si ça ne colle pas, vérifiez :

- Réglages Système → Confidentialité et sécurité → **Accessibilité**
- Réglages Système → Confidentialité et sécurité → **Surveillance de saisie**

## Utilisation

- Par défaut : maintenir **Fn** pour parler, relâcher pour coller la transcription.
- Menu barre “PKT” :
  - **Start/Stop** : toggle pour lancer/arrêter l’enregistrement sans hotkey
  - **Transcript (auto-paste)** : toggle (si OFF, ça copie seulement dans le clipboard, sans coller)
  - **Copier transcript** : copie la dernière transcription
  - Quitter : *Quitter PKTranscript*

### Choisir une touche / locale

L’app accepte des flags si vous lancez le binaire directement (dans l’app bundle : `Contents/MacOS/pktranscript`) :

```bash
dist/PKTranscript.app/Contents/MacOS/pktranscript --hotkey f7 --locale fr-FR
```

`--hotkey` accepte aussi un keycode macOS (ex: `0x61` pour F6).
