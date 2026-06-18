# Macos_PKdictation

Monorepo de mes apps macOS **push-to-talk** de transcription/dictée vocale.
Historiquement 3 dépôts séparés, fusionnés ici en gardant tout l'historique git.

🔗 https://github.com/mondary/Macos_PKdictation

---

## Les 3 apps

| App | Statut | Langage | Description | README |
|---|---|---|---|---|
| **PKvoice** | 🟢 active | Go + cgo | App principale : Speech + multi-IA + traduction + notch animé | [`apps/pkvoice/README.md`](apps/pkvoice/README.md) |
| **PKdictation** | 📦 archive | Swift / Xcode | Essai Swift via Gemini, abandonné | [`archive/pkdictation/README.md`](archive/pkdictation/README.md) |
| **PKtranscript** | 📦 archive | Go + cgo | MVP Go, ancêtre direct de PKvoice (code déjà absorbé) | [`archive/pktranscript/README.md`](archive/pktranscript/README.md) |

## Structure

```
.
├── apps/
│   └── pkvoice/            ← 🟢 app vivante (34 commits d'historique)
└── archive/
    ├── pkdictation/        ← 📦 archive (voie Swift abandonnée)
    └── pktranscript/       ← 📦 archive (MVP Go absorbé dans pkvoice)
```

## Détails

### `apps/pkvoice` — l'app à utiliser

Menu-bar push-to-talk (Go + cgo). Transcription **offline** via Apple Speech,
nettoyage IA optionnel (OpenAI / Claude / Gemini / OpenRouter / Z.AI),
traduction via `Fn+Ctrl`, notch animé, UI FR/ENG.

➡️ **Tout est dans [`apps/pkvoice/README.md`](apps/pkvoice/README.md)** : build, usage, settings.

### `archive/pkdictation`

Essai Swift/Xcode avec transcription **Gemini cloud**. Code modulaire propre
(`Audio/`, `Hotkey/`, `Gemini/`, `Keychain/`, `Overlay/`, `Logging/`)
mais jamais terminé (2 commits : `first commit`, `bof`). Gardé pour référence.

➡️ Voir [`archive/pkdictation/README.md`](archive/pkdictation/README.md)

### `archive/pktranscript`

Premier MVP Go de la chaîne. **Ancêtre direct de `pkvoice`** : le premier commit
de `pkvoice` (`Initial PKvoice scaffold from PKTranscript`) copie intégralement son
`run_darwin.go` (936 lignes), passé depuis à 2 864 lignes.

➡️ **Tout son code est déjà dans `apps/pkvoice/`** — l'archive est purement historique.
Voir [`archive/pktranscript/README.md`](archive/pktranscript/README.md)

---

## Historique git préservé

L'historique complet de chaque app est dans ce monorepo
(import via `git filter-repo --to-subdirectory-filter`) :

```sh
git log -- apps/pkvoice           # → 34 commits (mondary/Macos_PKvoice)
git log -- archive/pktranscript   # → 17 commits (mondary/Macos_PKtranscript)
git log -- archive/pkdictation    # → commits originaux de ce dépôt
```

Les dépôts GitHub `Macos_PKvoice` et `Macos_PKtranscript` sont archivés
en lecture seule et pointent vers ce monorepo.
