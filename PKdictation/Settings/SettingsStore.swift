import AppKit
import Foundation

@MainActor
final class SettingsStore: ObservableObject {
	@Published private(set) var hasGeminiApiKey: Bool = false

	@Published var pushToTalkHotkey: Hotkey {
		didSet { saveHotkey(pushToTalkHotkey) }
	}

	@Published var consumeHotkeyEvents: Bool {
		didSet { UserDefaults.standard.set(consumeHotkeyEvents, forKey: Keys.consumeHotkeyEvents) }
	}

	@Published var autoPasteTranscript: Bool {
		didSet { UserDefaults.standard.set(autoPasteTranscript, forKey: Keys.autoPasteTranscript) }
	}

	@Published var geminiModelName: String {
		didSet { UserDefaults.standard.set(geminiModelName, forKey: Keys.geminiModelName) }
	}

	init() {
		self.geminiModelName = UserDefaults.standard.string(forKey: Keys.geminiModelName) ?? "gemini-2.5-flash-lite"
		self.consumeHotkeyEvents = UserDefaults.standard.object(forKey: Keys.consumeHotkeyEvents) as? Bool ?? true
		self.autoPasteTranscript = UserDefaults.standard.object(forKey: Keys.autoPasteTranscript) as? Bool ?? true
		self.pushToTalkHotkey = SettingsStore.loadHotkey()

		let currentKey = (try? Keychain.readString(service: Keys.keychainService, account: Keys.geminiApiKeyAccount)) ?? nil
		if let currentKey, currentKey.isEmpty == false {
			self.hasGeminiApiKey = true
			return
		}

		// Migration: PKDication → PKdictation (service name changed).
		let legacyKey = (try? Keychain.readString(service: Keys.legacyKeychainService, account: Keys.geminiApiKeyAccount)) ?? nil
		if let legacyKey, legacyKey.isEmpty == false {
			try? Keychain.saveString(legacyKey, service: Keys.keychainService, account: Keys.geminiApiKeyAccount)
			try? Keychain.deleteItem(service: Keys.legacyKeychainService, account: Keys.geminiApiKeyAccount)
			self.hasGeminiApiKey = true
		} else {
			self.hasGeminiApiKey = false
		}
	}

	func loadGeminiApiKey() -> String? {
		let current = (try? Keychain.readString(service: Keys.keychainService, account: Keys.geminiApiKeyAccount)) ?? nil
		if let current { return current }
		return (try? Keychain.readString(service: Keys.legacyKeychainService, account: Keys.geminiApiKeyAccount)) ?? nil
	}

	func saveGeminiApiKey(_ apiKey: String) throws {
		let trimmed = apiKey.trimmingCharacters(in: .whitespacesAndNewlines)
		guard trimmed.isEmpty == false else {
			try deleteGeminiApiKey()
			return
		}
		try Keychain.saveString(trimmed, service: Keys.keychainService, account: Keys.geminiApiKeyAccount)
		hasGeminiApiKey = true
	}

	func deleteGeminiApiKey() throws {
		try Keychain.deleteItem(service: Keys.keychainService, account: Keys.geminiApiKeyAccount)
		try? Keychain.deleteItem(service: Keys.legacyKeychainService, account: Keys.geminiApiKeyAccount)
		hasGeminiApiKey = false
	}

	private enum Keys {
		static let geminiModelName = "geminiModelName"
		static let keychainService = "PKdictation"
		static let legacyKeychainService = "PKDication"
		static let geminiApiKeyAccount = "gemini_api_key"
		static let hotkeyKeyCode = "pushToTalkHotkeyKeyCode"
		static let hotkeyModifiers = "pushToTalkHotkeyModifiers"
		static let consumeHotkeyEvents = "consumeHotkeyEvents"
		static let autoPasteTranscript = "autoPasteTranscript"
	}

	private static func loadHotkey() -> Hotkey {
		let defaults = UserDefaults.standard
		guard
			let keyCodeValue = defaults.object(forKey: Keys.hotkeyKeyCode) as? Int,
			let modifiersValue = defaults.object(forKey: Keys.hotkeyModifiers) as? UInt64
		else {
			return .defaultPushToTalk
		}

		let keyCode: UInt16? = keyCodeValue < 0 ? nil : UInt16(clamping: keyCodeValue)
		let modifiers = NSEvent.ModifierFlags(rawValue: UInt(modifiersValue))
		return Hotkey(keyCode: keyCode, modifiers: modifiers)
	}

	private func saveHotkey(_ hotkey: Hotkey) {
		UserDefaults.standard.set(hotkey.keyCode.map(Int.init) ?? -1, forKey: Keys.hotkeyKeyCode)
		UserDefaults.standard.set(UInt64(hotkey.modifiers.rawValue), forKey: Keys.hotkeyModifiers)
	}
}
