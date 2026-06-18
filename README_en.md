<div align="center">

![Monorepo icon](icon.png)

# Macos_PKdictation

**Monorepo of macOS push-to-talk apps (voice transcription / dictation)**

</div>

[🇬🇧 EN](README_en.md) · [🇫🇷 FR](README.md)

✨ Three push-to-talk voice transcription apps — historically separate — merged into a single repo while keeping the full git history of each. One living app (`apps/pkvoice`), two archived for traceability.

---

## ✅ Features

| App | Status | Language | Transcription | Highlights |
|---|---|---|---|---|
| **PKvoice** | 🟢 active | Go + cgo | Apple Speech (offline) | multi-AI, translation, animated notch, FR/ENG UI |
| [**PKdictation**](https://github.com/mondary/Macos_PKdictation) → [archive](https://github.com/mondary/Macos_PKdictation/tree/main/archive/pkdictation) | 📦 archive | Swift / Xcode | Gemini API (cloud) | Swift attempt, abandoned |
| [**PKtranscript**](https://github.com/mondary/Macos_PKtranscript) → [archive](https://github.com/mondary/Macos_PKdictation/tree/main/archive/pktranscript) | 📦 archive | Go + cgo | Apple Speech (offline) | MVP, absorbed into PKvoice |

> 🔗 **Archived GitHub repositories** (read-only):
> - [`mondary/Macos_PKvoice`](https://github.com/mondary/Macos_PKvoice) — former PKvoice repo, now merged here into [`apps/pkvoice/`](apps/pkvoice/)
> - [`mondary/Macos_PKtranscript`](https://github.com/mondary/Macos_PKtranscript) — former PKtranscript repo, now merged here into [`archive/pktranscript/`](archive/pktranscript/)
> - [`mondary/Macos_PKdictation`](https://github.com/mondary/Macos_PKdictation) — this repo (now serving as the monorepo)

## 🧠 Usage

### Active app — PKvoice

1. Download `PKvoice.app` from the [latest release](https://github.com/mondary/Macos_PKdictation/releases/latest).
2. Drag it to `/Applications`.
3. Launch it → `waveform` icon shows up in the menu bar.
4. Grant permissions: **Microphone**, **Speech Recognition**, **Accessibility**, **Input Monitoring**.
5. **Hold `Fn`** to talk → release to paste the transcript at the cursor.
6. **`Fn` + `Ctrl`**: cleaned transcript + translation in the configured language.

➡️ Full details in [`apps/pkvoice/README.md`](apps/pkvoice/README.md).

### Archived apps — PKdictation / PKtranscript

Available in releases as `*-deprecated.app` for archival purposes. **Unmaintained**, kept for historical traceability. All their functional code is already in PKvoice.

## ⚙️ Settings

From the menu bar icon → **Settings…**:
- AI provider (`OpenAI`, `Claude`, `Gemini`, `OpenRouter`, `Z.AI`) + API keys
- AI model + cleanup toggle (removes `uh`, repetitions, false starts)
- Translation language (`EN/FR/ES/DE/IT`)
- Push-to-talk key (interactive capture, `Fn` by default)
- Speech recognition locale (FR-FR by default)
- Menu bar icon (`Wave` or `Micro`)
- Notch animation (patterns: `Wave`, `Spinner`, `Pulse`, `Cross`, `Burst`, `ArrowMove`, `Sine Wave`)
- UI language (`FR/ENG`)

## 📦 Build & Package

### PKvoice (active app)

```sh
cd apps/pkvoice
./src/build-app.sh                    # builds release/PKvoice.app
APP_VERSION=2.3 ./src/build-app.sh    # explicit version
open release/PKvoice.app
```

### PKtranscript (archive)

```sh
cd archive/pktranscript
./build-app.sh                        # builds dist/PKTranscript.app
```

### PKdictation (archive, requires Xcode)

```sh
cd archive/pkdictation
make export                           # builds dist/PKdictation.app
open dist/PKdictation.app
```

## 🚀 Install (from release)

1. Go to [Releases](https://github.com/mondary/Macos_PKdictation/releases/latest).
2. Download `PKvoice.app.zip` (or `PKdictation-deprecated.app.zip` / `PKtranscript-deprecated.app.zip` for archives).
3. Unzip and drag the `.app` to `/Applications`.
4. First launch: **right-click → Open** (unsigned app on macOS).
5. Grant permissions in **System Settings → Privacy & Security**:
   - Microphone
   - Speech Recognition
   - Accessibility (for global `Cmd+V`)
   - Input Monitoring (for the global hotkey)

## 🗂️ Repo structure

```
.
├── README.md                  ← French version
├── README_en.md               ← this file (EN)
├── icon.png                   ← monorepo icon
├── apps/
│   └── pkvoice/               ← 🟢 living app (34 commits of history)
└── archive/
    ├── pkdictation/           ← 📦 Swift archive (abandoned)
    └── pktranscript/          ← 📦 Go MVP archive (absorbed into pkvoice)
```

The full history of each app is preserved:
```sh
git log -- apps/pkvoice           # → 34 commits (ex mondary/Macos_PKvoice)
git log -- archive/pktranscript   # → 17 commits (ex mondary/Macos_PKtranscript)
git log -- archive/pkdictation    # → original commits of this repo
```

## 🧾 Changelog

- `2026-06-18`: **Monorepo migration** — merged `Macos_PKvoice` (archived) and `Macos_PKtranscript` (archived) into `Macos_PKdictation`. Full history preserved (34 + 17 commits) via `git filter-repo`. New bilingual README + GitHub release of the 3 apps.
- `2026-02-27`: PKvoice v2.3 — `Fn+Ctrl` translation shortcut, AI connection test, full multi-provider.
- `2026-02-26`: PKvoice v2.0 — rename `PKTranscript → PKvoice`, project restructure.
- `2025-12-17`: PKtranscript MVP 🟢.
- `2025-12-16`: PKdictation `first commit`.

## 🔗 Links

- French version: [`README.md`](README.md)
- Living app: [`apps/pkvoice/`](apps/pkvoice/)
- Releases: https://github.com/mondary/Macos_PKdictation/releases
- Archived repos: [Macos_PKvoice](https://github.com/mondary/Macos_PKvoice) · [Macos_PKtranscript](https://github.com/mondary/Macos_PKtranscript)
