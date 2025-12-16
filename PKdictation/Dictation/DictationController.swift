import AppKit
import Carbon.HIToolbox
import Foundation

@MainActor
final class DictationController: ObservableObject {
	enum Phase: Equatable {
		case idle
		case listening
		case processing
		case done
		case error(message: String)
	}

	@Published private(set) var phase: Phase = .idle
	@Published private(set) var transcript: String = ""

	let audio = AudioCaptureManager()
	private let settings: SettingsStore
	private let gemini: GeminiClient
	private var lastFrontmostAppPID: pid_t?
	private var isLocked: Bool = false

	weak var overlay: (any OverlayPresenting)?

	init(settings: SettingsStore, gemini: GeminiClient = GeminiClient()) {
		self.settings = settings
		self.gemini = gemini
	}

	func hotkeyPress() {
		if isLocked {
			if phase == .listening {
				LogStore.shared.log("Hotkey press: stopping locked recording.")
				isLocked = false
				stop()
			} else {
				isLocked = false
			}
			return
		}
		start()
	}

	func hotkeyRelease() {
		if isLocked { return }
		stop()
	}

	func hotkeyDoublePress() {
		guard isLocked == false else { return }
		isLocked = true
		LogStore.shared.log("Locked dictation enabled (double-press). Press again to stop.")
		if phase != .listening {
			start()
		}
	}

	func start() {
		guard phase != .listening else { return }
		lastFrontmostAppPID = NSWorkspace.shared.frontmostApplication?.processIdentifier
		transcript = ""
		phase = .listening
		overlay?.show()
		LogStore.shared.log("Recording started.")

		Task {
			do {
				try await audio.start()
			} catch {
				LogStore.shared.log("Recording failed: \(error.localizedDescription)")
				phase = .error(message: error.localizedDescription)
			}
		}
	}

	func stop() {
		guard phase == .listening else { return }
		phase = .processing
		LogStore.shared.log("Recording stopped. Sending to Gemini…")

		let url: URL
		do {
			url = try audio.stop()
		} catch AudioCaptureError.notRecording {
			LogStore.shared.log("Stop ignored: not recording.")
			phase = .idle
			overlay?.hide()
			return
		} catch {
			LogStore.shared.log("Stop failed: \(error.localizedDescription)")
			phase = .error(message: error.localizedDescription)
			return
		}

		Task { [weak self] in
			guard let self else { return }

			do {
				guard let apiKey = self.settings.loadGeminiApiKey(), apiKey.isEmpty == false else {
					throw DictationError.missingGeminiApiKey
				}

				let audioData = try Data(contentsOf: url)
				let text = try await self.gemini.transcribe(
					wavData: audioData,
					apiKey: apiKey,
					modelName: self.settings.geminiModelName
				)

				let cleaned = text.trimmingCharacters(in: .whitespacesAndNewlines)
				self.transcript = cleaned
				self.copyToPasteboard(cleaned)
				if self.settings.autoPasteTranscript {
					self.pasteIntoActiveAppFromClipboard()
				}
				self.phase = .done
				LogStore.shared.log("Gemini transcription OK (\(cleaned.count) chars).")
			} catch {
				LogStore.shared.log("Gemini transcription failed: \(error.localizedDescription)")
				self.phase = .error(message: error.localizedDescription)
			}

			try? await Task.sleep(nanoseconds: 4_000_000_000)
			if self.phase == .done || (ifCaseError(self.phase) != nil) {
				self.phase = .idle
				self.overlay?.hide()
			}
		}
	}

	var statusText: String {
		switch phase {
		case .idle:
			return "Idle"
		case .listening:
			return "Listening…"
		case .processing:
			return "Processing…"
		case .done:
			return "Done"
		case .error(let message):
			return "Error: \(message)"
		}
	}

	private func copyToPasteboard(_ text: String) {
		guard text.isEmpty == false else { return }
		let pb = NSPasteboard.general
		pb.clearContents()
		pb.setString(text, forType: .string)
	}

	private func pasteIntoActiveAppFromClipboard() {
		guard transcript.isEmpty == false else { return }
		guard NSRunningApplication.current.isActive == false else {
			LogStore.shared.log("Auto-paste skipped (PKdictation is active).")
			return
		}

		if let pid = lastFrontmostAppPID {
			LogStore.shared.log("Auto-paste: sending Cmd+V to frontmost app pid=\(pid).")
		} else {
			LogStore.shared.log("Auto-paste: sending Cmd+V.")
		}

		guard let keyDown = CGEvent(keyboardEventSource: nil, virtualKey: CGKeyCode(kVK_ANSI_V), keyDown: true),
			  let keyUp = CGEvent(keyboardEventSource: nil, virtualKey: CGKeyCode(kVK_ANSI_V), keyDown: false)
		else {
			LogStore.shared.log("Auto-paste failed: cannot create CGEvent.")
			return
		}

		keyDown.flags = .maskCommand
		keyUp.flags = .maskCommand

		keyDown.post(tap: .cghidEventTap)
		keyUp.post(tap: .cghidEventTap)
	}
}

private enum DictationError: LocalizedError {
	case missingGeminiApiKey

	var errorDescription: String? {
		switch self {
		case .missingGeminiApiKey:
			return "Missing Gemini API key. Open Settings to add it."
		}
	}
}

private func ifCaseError(_ phase: DictationController.Phase) -> String? {
	if case .error(let message) = phase { return message }
	return nil
}
