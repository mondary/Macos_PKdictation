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

		let tailPaddingMs = 600
		let finalizeDelayMs = 250

		Task.detached(priority: .userInitiated) { [weak self] in
			guard let self else { return }

			do {
				await MainActor.run {
					LogStore.shared.log("Stop: keeping \(tailPaddingMs)ms tail + \(finalizeDelayMs)ms finalize delay to avoid truncation.")
				}

				let url: URL
				do {
					url = try await Task { @MainActor in
						try await self.audio.stop(tailPaddingMs: tailPaddingMs, finalizeDelayMs: finalizeDelayMs)
					}.value
				} catch AudioCaptureError.notRecording {
					await MainActor.run {
						LogStore.shared.log("Stop ignored: not recording.")
						self.phase = .idle
						self.overlay?.hide()
					}
					return
				}

				let (apiKey, modelName, autoPaste) = await MainActor.run {
					(self.settings.loadGeminiApiKey(), self.settings.geminiModelName, self.settings.autoPasteTranscript)
				}
				guard let apiKey, apiKey.isEmpty == false else { throw DictationError.missingGeminiApiKey }

				let audioData = try Data(contentsOf: url)
				if audioData.count < 1024 {
					await MainActor.run {
						LogStore.shared.log("Audio read looks too small (\(audioData.count) bytes). File: \(url.lastPathComponent)")
					}
					throw DictationError.emptyOrInvalidAudio(sizeBytes: audioData.count)
				}

				if isValidWavData(audioData) == false {
					await MainActor.run {
						LogStore.shared.log("Audio read is not a valid WAV (missing RIFF/WAVE). size=\(audioData.count) file=\(url.lastPathComponent)")
					}
					throw DictationError.emptyOrInvalidAudio(sizeBytes: audioData.count)
				}

				await MainActor.run {
					LogStore.shared.log("Sending audio to Gemini (wavBytes=\(audioData.count), model=\(modelName)).")
				}
				let text = try await self.gemini.transcribe(
					wavData: audioData,
					apiKey: apiKey,
					modelName: modelName
				)

				let cleaned = text.trimmingCharacters(in: .whitespacesAndNewlines)
				await MainActor.run {
					self.transcript = cleaned
					AppModel.shared.history.add(cleaned)
					self.copyToPasteboard(cleaned)
					if autoPaste {
						Task { @MainActor in
							try? await Task.sleep(nanoseconds: 75_000_000)
							self.pasteIntoActiveAppFromClipboard()
						}
					}
					self.phase = .done
					LogStore.shared.log("Gemini transcription OK (\(cleaned.count) chars).")
				}
			} catch {
				await MainActor.run {
					LogStore.shared.log("Gemini transcription failed: \(error.localizedDescription)")
					if let detailed = (error as? LocalizedError)?.failureReason, detailed.isEmpty == false {
						LogStore.shared.log("Gemini transcription failure details: \(detailed)")
					}
					self.phase = .error(message: error.localizedDescription)
				}
			}

			try? await Task.sleep(nanoseconds: 4_000_000_000)
			await MainActor.run {
				if self.phase == .done || (ifCaseError(self.phase) != nil) {
					self.phase = .idle
					self.overlay?.hide()
				}
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
		if let roundTrip = pb.string(forType: .string), roundTrip.count != text.count {
			LogStore.shared.log("Pasteboard warning: wrote \(text.count) chars, read back \(roundTrip.count).")
		}
	}

	private func pasteIntoActiveAppFromClipboard() {
		guard transcript.isEmpty == false else { return }
		guard NSRunningApplication.current.isActive == false else {
			LogStore.shared.log("Auto-paste skipped (PKdictation is active).")
			return
		}

		let pid = lastFrontmostAppPID
		if let pid {
			LogStore.shared.log("Auto-paste: sending Cmd+V to frontmost app pid=\(pid).")
		} else {
			LogStore.shared.log("Auto-paste: sending Cmd+V (no pid).")
		}

		guard let keyDown = CGEvent(keyboardEventSource: nil, virtualKey: CGKeyCode(kVK_ANSI_V), keyDown: true),
			  let keyUp = CGEvent(keyboardEventSource: nil, virtualKey: CGKeyCode(kVK_ANSI_V), keyDown: false)
		else {
			LogStore.shared.log("Auto-paste failed: cannot create CGEvent.")
			return
		}

		keyDown.flags = .maskCommand
		keyUp.flags = .maskCommand

		if let pid {
			keyDown.postToPid(pid)
			keyUp.postToPid(pid)
		} else {
			keyDown.post(tap: .cghidEventTap)
			keyUp.post(tap: .cghidEventTap)
		}
	}
}

private enum DictationError: LocalizedError {
	case missingGeminiApiKey
	case emptyOrInvalidAudio(sizeBytes: Int)

	var errorDescription: String? {
		switch self {
		case .missingGeminiApiKey:
			return "Missing Gemini API key. Open Settings to add it."
		case .emptyOrInvalidAudio(let sizeBytes):
			return "Recorded audio looks empty/invalid (\(sizeBytes) bytes). Try again (and ensure Mic permission is granted)."
		}
	}
}

private func ifCaseError(_ phase: DictationController.Phase) -> String? {
	if case .error(let message) = phase { return message }
	return nil
}

private func isValidWavData(_ data: Data) -> Bool {
	guard data.count >= 12 else { return false }
	let riff = data.prefix(4)
	let wave = data.subdata(in: 8..<12)
	return riff == Data([0x52, 0x49, 0x46, 0x46]) && wave == Data([0x57, 0x41, 0x56, 0x45]) // "RIFF" + "WAVE"
}
