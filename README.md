<div align="center">

![Monorepo icon](icon.png)

# Macos_PKdictation

**Monorepo d'apps macOS push-to-talk (transcription / dictée vocale)**

</div>

[🇫🇷 FR](README.md) · [🇬🇧 EN](README_en.md)

✨ Trois apps de transcription vocale push-to-talk, historiquement séparées, fusionnées dans un seul dépôt en gardant tout l'historique git. Une seule app vivante (`apps/pkvoice`), deux archives pour trace historique.

---

## ✅ Fonctionnalités

| App | Statut | Langage | Transcription | Particularités |
|---|---|---|---|---|
| **PKvoice** | 🟢 active | Go + cgo | Apple Speech (offline) | multi-IA, traduction, notch animé, UI FR/ENG |
| [**PKdictation**](https://github.com/mondary/Macos_PKdictation) → [archive](https://github.com/mondary/Macos_PKdictation/tree/main/archive/pkdictation) | 📦 archive | Swift / Xcode | Gemini API (cloud) | essai Swift, abandonné |
| [**PKtranscript**](https://github.com/mondary/Macos_PKtranscript) → [archive](https://github.com/mondary/Macos_PKdictation/tree/main/archive/pktranscript) | 📦 archive | Go + cgo | Apple Speech (offline) | MVP, absorbé dans PKvoice |

> 🔗 **Dépôts GitHub archivés** (lecture seule) :
> - [`mondary/Macos_PKvoice`](https://github.com/mondary/Macos_PKvoice) — ancien dépôt de PKvoice, désormais fusionné ici dans [`apps/pkvoice/`](apps/pkvoice/)
> - [`mondary/Macos_PKtranscript`](https://github.com/mondary/Macos_PKtranscript) — ancien dépôt de PKtranscript, désormais fusionné ici dans [`archive/pktranscript/`](archive/pktranscript/)
> - [`mondary/Macos_PKdictation`](https://github.com/mondary/Macos_PKdictation) — ce dépôt (qui sert désormais de monorepo)

## 🧠 Utilisation

### App active — PKvoice

1. Téléchargez `PKvoice.app` depuis la [dernière release](https://github.com/mondary/Macos_PKdictation/releases/latest).
2. Glissez-la dans `/Applications`.
3. Lancez-la → icône `waveform` apparaît dans la barre de menu.
4. Autorisez : **Microphone**, **Reconnaissance vocale**, **Accessibilité**, **Surveillance de saisie**.
5. **Maintenez `Fn`** pour parler → relâchez pour coller la transcription au curseur.
6. **`Fn` + `Ctrl`** : transcription nettoyée + traduction dans la langue configurée.

➡️ Détails complets dans [`apps/pkvoice/README.md`](apps/pkvoice/README.md).

### Apps archivées — PKdictation / PKtranscript

Disponibles dans les releases en `*-deprecated.app` pour archival. **Non maintenues**, conservées pour trace historique. Tout leur code fonctionnel est déjà dans PKvoice.

## ⚙️ Réglages

Dans le menu barre (icône waveform) → **Settings…** :
- Provider IA (`OpenAI`, `Claude`, `Gemini`, `OpenRouter`, `Z.AI`) + clés API
- Modèle IA + toggle nettoyage (supprime `euh`, répétitions, faux départs)
- Langue de traduction (`EN/FR/ES/DE/IT`)
- Touche push-to-talk (capture interactive, `Fn` par défaut)
- Locale de reconnaissance vocale (FR-FR par défaut)
- Icône barre de menu (`Wave` ou `Micro`)
- Animation du notch (patterns : `Wave`, `Spinner`, `Pulse`, `Cross`, `Burst`, `ArrowMove`, `Sine Wave`)
- Langue de l'UI (`FR/ENG`)

## 📦 Build & Package

### PKvoice (app active)

```sh
cd apps/pkvoice
./src/build-app.sh                    # génère release/PKvoice.app
APP_VERSION=2.3 ./src/build-app.sh    # version explicite
open release/PKvoice.app
```

### PKtranscript (archive)

```sh
cd archive/pktranscript
./build-app.sh                        # génère dist/PKTranscript.app
```

### PKdictation (archive, nécessite Xcode)

```sh
cd archive/pkdictation
make export                           # génère dist/PKdictation.app
open dist/PKdictation.app
```

## 🚀 Installation (depuis la release)

1. Allez sur [Releases](https://github.com/mondary/Macos_PKdictation/releases/latest).
2. Téléchargez `PKvoice.app.zip` (ou `PKdictation-deprecated.app.zip` / `PKtranscript-deprecated.app.zip` pour les archives).
3. Décompressez et glissez le `.app` dans `/Applications`.
4. Premier lancement : **clic droit → Ouvrir** ( macOS : app non signée).
5. Accordez les permissions dans **Réglages Système → Confidentialité et sécurité** :
   - Microphone
   - Reconnaissance vocale (Speech)
   - Accessibilité (pour `Cmd+V` global)
   - Surveillance de saisie (pour la touche globale)

## 🗂️ Structure du dépôt

```
.
├── README.md                  ← ce fichier (FR)
├── README_en.md               ← version anglaise
├── icon.png                   ← icône du monorepo
├── apps/
│   └── pkvoice/               ← 🟢 app vivante (34 commits d'historique)
└── archive/
    ├── pkdictation/           ← 📦 archive Swift (abandonnée)
    └── pktranscript/          ← 📦 archive MVP Go (absorbé dans pkvoice)
```

L'historique complet de chaque app est préservé :
```sh
git log -- apps/pkvoice           # → 34 commits (ex mondary/Macos_PKvoice)
git log -- archive/pktranscript   # → 17 commits (ex mondary/Macos_PKtranscript)
git log -- archive/pkdictation    # → commits originaux de ce dépôt
```

## 🧾 Changelog

- `2026-06-18` : **Migration monorepo** — fusion de `Macos_PKvoice` (archive) et `Macos_PKtranscript` (archive) dans `Macos_PKdictation`. Historique complet préservé (34 + 17 commits) via `git filter-repo`. Nouveau README bilingue + release GitHub des 3 apps.
- `2026-02-27` : PKvoice v2.3 — raccourci traduction `Fn+Ctrl`, test de connexion IA, multi-provider complet.
- `2026-02-26` : PKvoice v2.0 — renommage `PKTranscript → PKvoice`, restructuration du projet.
- `2025-12-17` : PKtranscript MVP 🟢.
- `2025-12-16` : PKdictation `first commit`.

## 🔗 Liens

- Version anglaise : [`README_en.md`](README_en.md)
- App vivante : [`apps/pkvoice/`](apps/pkvoice/)
- Releases : https://github.com/mondary/Macos_PKdictation/releases
- Dépôts archivés : [Macos_PKvoice](https://github.com/mondary/Macos_PKvoice) · [Macos_PKtranscript](https://github.com/mondary/Macos_PKtranscript)
