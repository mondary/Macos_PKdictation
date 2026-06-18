# Macos_PKdictation (monorepo)

Monorepo rassemblant mes apps macOS de transcription/dictée vocale **push-to-talk**.
Historiquement 3 dépôts séparés, fusionnés ici en gardant tout l'historique git.

> URL du dépôt : https://github.com/mondary/Macos_PKdictation
> (le nom `PKdictation` est conservé car c'est le dépôt le plus ancien ; il sert désormais de monorepo)

## Structure

```
.
├── apps/                       ← apps vivantes, maintenues
│   └── pkvoice/                ← app principale (Go + cgo)
└── archive/                    ← apps archivées, en lecture seule
    ├── pkdictation/            ← essai Swift/Xcode (abandonné)
    └── pktranscript/           ← MVP Go (ancêtre direct de pkvoice)
```

## Apps

### `apps/pkvoice` — app principale

App macOS en Go (bindings Objective-C via cgo) :

- Push-to-talk global (touche configurable, `Fn` par défaut)
- Transcription via le framework Apple **Speech** (offline)
- **Nettoyage IA** optionnel de la transcription (multi-provider : OpenAI, Claude, Gemini, OpenRouter, Z.AI)
- **Traduction** via raccourci `Fn+Ctrl` (langue cible configurable)
- Menu barre avec historique, settings, et toggle auto-paste
- **Notch animé** pendant l'enregistrement (patterns personnalisables)
- UI localisée (FR/ENG)

➡️ Voir [`apps/pkvoice/README.md`](apps/pkvoice/README.md) pour le build et l'utilisation.

### `archive/pkdictation` — voie Swift (abandonnée)

Essai d'app menu-bar en **Swift / Xcode** avec transcription via **Gemini API**.
2 commits seulement (`first commit`, `bof`), jamais terminée. Code modulaire propre
(modules `Audio/`, `Hotkey/`, `Gemini/`, `Keychain/`, `Overlay/`, `Logging/`).
Archivée pour référence.

### `archive/pktranscript` — MVP Go (absorbé dans pkvoice)

Premier MVP en Go de la chaîne. C'est l'ancêtre direct de `pkvoice` :
le tout premier commit de `pkvoice` (`Initial PKvoice scaffold from PKTranscript`)
copie intégralement son `run_darwin.go` (936 lignes), puis 33 commits plus tard
ce fichier est passé à 2 864 lignes avec le multi-provider IA, le notch, etc.

Archivé pour traçabilité historique — **tout son code est déjà dans `apps/pkvoice/`**.

## Pourquoi un monorepo ?

- Éviter la confusion entre 3 dépôts qui faisaient la même chose
- Centraliser l'historique (PKtranscript → PKvoice → ...)
- Garder un seul dépôt vivant (`apps/`) tout en préservant les anciens (`archive/`)

## Historique git

L'historique de chaque app est **entièrement préservé** dans ce monorepo
(via `git filter-repo --to-subdirectory-filter`) :

- `git log -- apps/pkvoice` → 34 commits de `mondary/Macos_PKvoice`
- `git log -- archive/pktranscript` → 17 commits de `mondary/Macos_PKtranscript`
- `git log -- archive/pkdictation` → commits historiques de ce dépôt

Les dépôts GitHub originaux `Macos_PKvoice` et `Macos_PKtranscript` sont archivés
(lecture seule) et pointent vers ce monorepo.
