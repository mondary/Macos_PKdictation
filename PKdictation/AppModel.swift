import Combine
import Foundation

@MainActor
final class AppModel: ObservableObject {
	static let shared = AppModel()

	let settings: SettingsStore
	let dictation: DictationController
	let hotkeyManager: HotkeyManager
	let history: HistoryStore

	private var cancellables: Set<AnyCancellable> = []

	private init() {
		let settings = SettingsStore()
		self.settings = settings
		self.dictation = DictationController(settings: settings)
		self.history = HistoryStore()

		let hotkeyManager = HotkeyManager(hotkey: settings.pushToTalkHotkey, consumeEvents: settings.consumeHotkeyEvents)
		self.hotkeyManager = hotkeyManager

		hotkeyManager.onPress = { [weak dictation = self.dictation] in
			Task { @MainActor in
				dictation?.hotkeyPress()
			}
		}
		hotkeyManager.onRelease = { [weak dictation = self.dictation] in
			Task { @MainActor in
				dictation?.hotkeyRelease()
			}
		}
		hotkeyManager.onDoublePress = { [weak dictation = self.dictation] in
			Task { @MainActor in
				dictation?.hotkeyDoublePress()
			}
		}

		settings.$pushToTalkHotkey
			.sink { [weak hotkeyManager] hotkey in
				hotkeyManager?.update(hotkey: hotkey)
			}
			.store(in: &cancellables)

		settings.$consumeHotkeyEvents
			.sink { [weak hotkeyManager] consume in
				hotkeyManager?.update(consumeEvents: consume)
			}
			.store(in: &cancellables)
	}
}

@MainActor
final class HistoryStore: ObservableObject {
	static let maxItems = 200

	@Published private(set) var entries: [Entry]

	struct Entry: Codable, Equatable, Identifiable {
		let id: UUID
		let createdAt: Date
		let text: String
	}

	private let storageKey = "PKdictation.history.v1"

	init() {
		self.entries = Self.load(from: storageKey)
	}

	func add(_ text: String) {
		let cleaned = text.trimmingCharacters(in: .whitespacesAndNewlines)
		guard cleaned.isEmpty == false else { return }

		if let last = entries.first, last.text == cleaned {
			return
		}

		entries.insert(.init(id: UUID(), createdAt: Date(), text: cleaned), at: 0)
		if entries.count > Self.maxItems {
			entries.removeLast(entries.count - Self.maxItems)
		}
		save()
	}

	func recent(limit: Int) -> [Entry] {
		Array(entries.prefix(max(0, limit)))
	}

	func allText() -> String {
		let formatter = ISO8601DateFormatter()
		formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
		return entries
			.map { "[\(formatter.string(from: $0.createdAt))] \($0.text)" }
			.joined(separator: "\n\n")
	}

	func clearAll() {
		entries.removeAll()
		save()
	}

	private func save() {
		guard let data = try? JSONEncoder().encode(entries) else { return }
		UserDefaults.standard.set(data, forKey: storageKey)
	}

	private static func load(from key: String) -> [Entry] {
		guard let data = UserDefaults.standard.data(forKey: key) else { return [] }
		return (try? JSONDecoder().decode([Entry].self, from: data)) ?? []
	}
}
