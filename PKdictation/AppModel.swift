import Combine
import Foundation

@MainActor
final class AppModel: ObservableObject {
	static let shared = AppModel()

	let settings: SettingsStore
	let dictation: DictationController
	let hotkeyManager: HotkeyManager

	private var cancellables: Set<AnyCancellable> = []

	private init() {
		let settings = SettingsStore()
		self.settings = settings
		self.dictation = DictationController(settings: settings)

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
