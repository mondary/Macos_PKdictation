import AppKit
import Combine
import Carbon.HIToolbox

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
	private var statusItem: NSStatusItem?
	private var overlay: OverlayWindowController?
	private var settingsWindowController: SettingsWindowController?
	private var recordingMenuItem: NSMenuItem?
	private var historyPlaceholderItem: NSMenuItem?
	private var historyMenuItems: [NSMenuItem] = []
	private var showAllHistoryMenuItem: NSMenuItem?
	private var historyWindowController: HistoryWindowController?
	private var onboardingWindowController: OnboardingWindowController?
	private var cancellables: Set<AnyCancellable> = []
	private var appActivationObserver: Any?
	private var lastExternalFrontmostAppPID: pid_t?

	func applicationDidFinishLaunching(_ notification: Notification) {
		NSApp.setActivationPolicy(.accessory)
		startTrackingFrontmostApp()
		setUpMainMenu()
		setUpOverlay()
		setUpStatusItem()
		AppModel.shared.hotkeyManager.start()
		showOnboardingIfNeeded()
	}

	private func startTrackingFrontmostApp() {
		let selfPID = ProcessInfo.processInfo.processIdentifier

		if let current = NSWorkspace.shared.frontmostApplication,
		   current.processIdentifier != selfPID
		{
			lastExternalFrontmostAppPID = current.processIdentifier
		}

		let nc = NSWorkspace.shared.notificationCenter
		appActivationObserver = nc.addObserver(forName: NSWorkspace.didActivateApplicationNotification, object: nil, queue: .main) { [weak self] note in
			guard let self else { return }
			guard let app = note.userInfo?[NSWorkspace.applicationUserInfoKey] as? NSRunningApplication else { return }
			guard app.processIdentifier != selfPID else { return }
			self.lastExternalFrontmostAppPID = app.processIdentifier
		}
	}

	private func setUpMainMenu() {
		let mainMenu = NSMenu()

		let appMenuItem = NSMenuItem()
		mainMenu.addItem(appMenuItem)

		let appMenu = NSMenu()
		appMenuItem.submenu = appMenu
		appMenu.addItem(NSMenuItem(title: "Settings…", action: #selector(openSettings), keyEquivalent: ","))
		appMenu.addItem(.separator())
		appMenu.addItem(NSMenuItem(title: "Quit PKdictation", action: #selector(quit), keyEquivalent: "q"))

		let editMenuItem = NSMenuItem()
		mainMenu.addItem(editMenuItem)

		let editMenu = NSMenu(title: "Edit")
		editMenuItem.submenu = editMenu
		editMenu.addItem(NSMenuItem(title: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x"))
		editMenu.addItem(NSMenuItem(title: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c"))
		editMenu.addItem(NSMenuItem(title: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v"))
		editMenu.addItem(NSMenuItem(title: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a"))

		NSApp.mainMenu = mainMenu
	}

	private func setUpOverlay() {
		let overlay = OverlayWindowController(controller: AppModel.shared.dictation)
		AppModel.shared.dictation.overlay = overlay
		self.overlay = overlay
	}

	private func setUpStatusItem() {
		let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
		if let button = item.button {
			button.image = NSImage(systemSymbolName: "waveform", accessibilityDescription: "PKdictation")
			button.imagePosition = .imageOnly
		}

		let menu = NSMenu()
		let recordItem = NSMenuItem(title: "Start recording", action: #selector(toggleRecording), keyEquivalent: "")
		menu.addItem(recordItem)
		menu.addItem(.separator())
		let historyTitle = NSMenuItem(title: "History", action: nil, keyEquivalent: "")
		historyTitle.isEnabled = false
		menu.addItem(historyTitle)
		let historyPlaceholder = NSMenuItem(title: "No history yet", action: nil, keyEquivalent: "")
		historyPlaceholder.isEnabled = false
		menu.addItem(historyPlaceholder)
		let showAllHistory = NSMenuItem(title: "Show all history…", action: #selector(showAllHistory), keyEquivalent: "")
		menu.addItem(showAllHistory)
		menu.addItem(.separator())
		menu.addItem(NSMenuItem(title: "Settings…", action: #selector(openSettings), keyEquivalent: ","))
		menu.addItem(.separator())
		menu.addItem(NSMenuItem(title: "Quit", action: #selector(quit), keyEquivalent: "q"))
		item.menu = menu

		statusItem = item
		recordingMenuItem = recordItem
		historyPlaceholderItem = historyPlaceholder
		showAllHistoryMenuItem = showAllHistory
		updateRecordingMenuItem()
		rebuildHistoryMenu()

		AppModel.shared.settings.$pushToTalkHotkey
			.receive(on: RunLoop.main)
			.sink { [weak self] _ in
				self?.updateRecordingMenuItem()
			}
			.store(in: &cancellables)

		AppModel.shared.dictation.$phase
			.receive(on: RunLoop.main)
			.sink { [weak self] _ in
				self?.updateRecordingMenuItem()
			}
			.store(in: &cancellables)

		AppModel.shared.history.$entries
			.receive(on: RunLoop.main)
			.sink { [weak self] _ in
				self?.rebuildHistoryMenu()
			}
			.store(in: &cancellables)
	}

	private func updateRecordingMenuItem() {
		let hotkey = AppModel.shared.settings.pushToTalkHotkey.displayString
		let phase = AppModel.shared.dictation.phase

		switch phase {
		case .idle, .done, .error:
			recordingMenuItem?.title = "Start recording"
			recordingMenuItem?.toolTip = "Push-to-talk: hold \(hotkey)"
			recordingMenuItem?.isEnabled = true
		case .listening:
			recordingMenuItem?.title = "Stop recording"
			recordingMenuItem?.toolTip = "Push-to-talk: release \(hotkey)"
			recordingMenuItem?.isEnabled = true
		case .processing:
			recordingMenuItem?.title = "Processing…"
			recordingMenuItem?.toolTip = nil
			recordingMenuItem?.isEnabled = false
		}
	}

	private func rebuildHistoryMenu() {
		guard let menu = statusItem?.menu else { return }
		guard let placeholder = historyPlaceholderItem else { return }

		for item in historyMenuItems {
			menu.removeItem(item)
		}
		historyMenuItems.removeAll()

		let entries = AppModel.shared.history.recent(limit: 15)
		placeholder.isHidden = !entries.isEmpty

		guard entries.isEmpty == false else { return }

		let insertIndex = menu.index(of: placeholder)
		for (offset, entry) in entries.enumerated() {
			let title = Self.menuTitle(for: entry.text, max: 72)
			let item = NSMenuItem(title: title, action: #selector(selectHistoryItem(_:)), keyEquivalent: "")
			item.toolTip = entry.text
			item.representedObject = entry.text
			menu.insertItem(item, at: insertIndex + offset)
			historyMenuItems.append(item)
		}
	}

	private static func menuTitle(for text: String, max: Int) -> String {
		let cleaned = text.trimmingCharacters(in: .whitespacesAndNewlines)
		guard cleaned.count > max else { return cleaned }
		let idx = cleaned.index(cleaned.startIndex, offsetBy: max - 1)
		return String(cleaned[..<idx]) + "…"
	}

	@objc private func toggleRecording() {
		switch AppModel.shared.dictation.phase {
		case .listening:
			AppModel.shared.dictation.stop()
		case .processing:
			break
		case .idle, .done, .error:
			AppModel.shared.dictation.start()
		}
		updateRecordingMenuItem()
	}

	@objc private func selectHistoryItem(_ sender: NSMenuItem) {
		guard let text = sender.representedObject as? String, text.isEmpty == false else { return }
		let pb = NSPasteboard.general
		pb.clearContents()
		pb.setString(text, forType: .string)
		LogStore.shared.log("History item copied to clipboard (\(text.count) chars).")
		pasteIntoLastExternalAppFromClipboard()
	}

	private func pasteIntoLastExternalAppFromClipboard() {
		let pid = lastExternalFrontmostAppPID
		if let pid {
			LogStore.shared.log("History paste: sending Cmd+V to pid=\(pid).")
		} else {
			LogStore.shared.log("History paste: sending Cmd+V (no pid).")
		}

		guard let keyDown = CGEvent(keyboardEventSource: nil, virtualKey: CGKeyCode(kVK_ANSI_V), keyDown: true),
			  let keyUp = CGEvent(keyboardEventSource: nil, virtualKey: CGKeyCode(kVK_ANSI_V), keyDown: false)
		else {
			LogStore.shared.log("History paste failed: cannot create CGEvent.")
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

	@objc private func showAllHistory() {
		if historyWindowController == nil {
			historyWindowController = HistoryWindowController(history: AppModel.shared.history)
		}
		NSApp.activate(ignoringOtherApps: true)
		historyWindowController?.showWindow(nil)
		historyWindowController?.window?.makeKeyAndOrderFront(nil)
	}

	@objc private func openSettings() {
		if settingsWindowController == nil {
			settingsWindowController = SettingsWindowController(
				settings: AppModel.shared.settings,
				hotkeyManager: AppModel.shared.hotkeyManager
			)
		}

		NSApp.activate(ignoringOtherApps: true)
		settingsWindowController?.showWindow(nil)
		settingsWindowController?.window?.makeKeyAndOrderFront(nil)
	}

	@objc private func quit() {
		NSApp.terminate(nil)
	}

	private func showOnboardingIfNeeded() {
		let defaults = UserDefaults.standard
		let didShow = defaults.bool(forKey: "PKdictation.didShowOnboarding")
		guard didShow == false else { return }
		defaults.set(true, forKey: "PKdictation.didShowOnboarding")

		DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) { [weak self] in
			guard let self else { return }
			if self.onboardingWindowController == nil {
				self.onboardingWindowController = OnboardingWindowController(
					settings: AppModel.shared.settings,
					hotkeyManager: AppModel.shared.hotkeyManager
				)
			}
			NSApp.activate(ignoringOtherApps: true)
			self.onboardingWindowController?.showWindow(nil)
			self.onboardingWindowController?.window?.makeKeyAndOrderFront(nil)
		}
	}
}

@MainActor
final class HistoryWindowController: NSWindowController {
	private let viewController: HistoryViewController

	init(history: HistoryStore) {
		self.viewController = HistoryViewController(history: history)

		let window = NSWindow(
			contentRect: NSRect(x: 0, y: 0, width: 680, height: 520),
			styleMask: [.titled, .closable, .miniaturizable, .resizable],
			backing: .buffered,
			defer: false
		)
		window.title = "PKdictation History"
		window.center()
		window.contentViewController = viewController
		window.isReleasedWhenClosed = false

		super.init(window: window)
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}
}

@MainActor
final class HistoryViewController: NSViewController {
	private let history: HistoryStore
	private var cancellable: AnyCancellable?

	private let textView = NSTextView()
	private let copyAllButton = NSButton(title: "Copy all", target: nil, action: nil)
	private let clearAllButton = NSButton(title: "Clear all", target: nil, action: nil)

	init(history: HistoryStore) {
		self.history = history
		super.init(nibName: nil, bundle: nil)
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}

	override func loadView() {
		// Don't disable autoresizing translation on the root view; otherwise AppKit may not size it
		// correctly inside the window, which can lead to a tiny/empty window.
		view = NSView(frame: NSRect(x: 0, y: 0, width: 680, height: 520))

		let scroll = NSScrollView()
		scroll.hasVerticalScroller = true
		scroll.borderType = .bezelBorder
		scroll.translatesAutoresizingMaskIntoConstraints = false
		view.addSubview(scroll)

		textView.isEditable = false
		textView.isSelectable = true
		textView.font = NSFont.monospacedSystemFont(ofSize: 12, weight: .regular)
		textView.textContainerInset = NSSize(width: 10, height: 10)
		textView.isVerticallyResizable = true
		textView.isHorizontallyResizable = false
		textView.autoresizingMask = [.width]
		textView.textContainer?.widthTracksTextView = true
		textView.textContainer?.containerSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
		scroll.documentView = textView

		copyAllButton.target = self
		copyAllButton.action = #selector(copyAll)
		copyAllButton.translatesAutoresizingMaskIntoConstraints = false
		clearAllButton.target = self
		clearAllButton.action = #selector(clearAll)
		clearAllButton.translatesAutoresizingMaskIntoConstraints = false

		let buttonsRow = NSStackView(views: [clearAllButton, spacerView(), copyAllButton])
		buttonsRow.orientation = .horizontal
		buttonsRow.alignment = .centerY
		buttonsRow.spacing = 10
		buttonsRow.translatesAutoresizingMaskIntoConstraints = false
		view.addSubview(buttonsRow)

		NSLayoutConstraint.activate([
			scroll.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 14),
			scroll.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -14),
			scroll.topAnchor.constraint(equalTo: view.topAnchor, constant: 14),
			scroll.bottomAnchor.constraint(equalTo: buttonsRow.topAnchor, constant: -12),

			buttonsRow.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 14),
			buttonsRow.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -14),
			buttonsRow.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -14),
		])

		refresh()
		cancellable = history.$entries.sink { [weak self] _ in self?.refresh() }
	}

	private func spacerView() -> NSView {
		let spacer = NSView()
		spacer.translatesAutoresizingMaskIntoConstraints = false
		spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
		spacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
		return spacer
	}

	private func refresh() {
		textView.string = history.allText()
	}

	@objc private func copyAll() {
		let text = history.allText()
		let pb = NSPasteboard.general
		pb.clearContents()
		pb.setString(text, forType: .string)
	}

	@objc private func clearAll() {
		let alert = NSAlert()
		alert.messageText = "Clear all history?"
		alert.informativeText = "This will remove all saved transcripts from PKdictation."
		alert.addButton(withTitle: "Clear")
		alert.addButton(withTitle: "Cancel")
		alert.alertStyle = .warning

		let response = alert.runModal()
		guard response == .alertFirstButtonReturn else { return }
		history.clearAll()
	}
}

@MainActor
final class OnboardingWindowController: NSWindowController {
	private let viewController: OnboardingViewController

	init(settings: SettingsStore, hotkeyManager: HotkeyManager) {
		self.viewController = OnboardingViewController(settings: settings, hotkeyManager: hotkeyManager)

		let window = NSWindow(
			contentRect: NSRect(x: 0, y: 0, width: 560, height: 640),
			styleMask: [.titled, .closable, .miniaturizable],
			backing: .buffered,
			defer: false
		)
		window.title = "Welcome to PKdictation"
		window.center()
		window.contentViewController = viewController
		window.isReleasedWhenClosed = false

		super.init(window: window)
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}
}

@MainActor
final class OnboardingViewController: NSViewController {
	private let settings: SettingsStore
	private let hotkeyManager: HotkeyManager

	private var embedded: SettingsViewController?
	private let finishButton = NSButton(title: "Finish setup", target: nil, action: nil)
	private let laterButton = NSButton(title: "Not now", target: nil, action: nil)

	init(settings: SettingsStore, hotkeyManager: HotkeyManager) {
		self.settings = settings
		self.hotkeyManager = hotkeyManager
		super.init(nibName: nil, bundle: nil)
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}

	override func loadView() {
		view = NSView()
		view.translatesAutoresizingMaskIntoConstraints = false

		let title = NSTextField(labelWithString: "Set up PKdictation")
		title.font = .systemFont(ofSize: 18, weight: .semibold)

		let subtitle = NSTextField(
			labelWithString: "Add your Gemini API key, choose a model, set your push‑to‑talk shortcut, then grant the required permissions."
		)
		subtitle.textColor = .secondaryLabelColor
		subtitle.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		subtitle.maximumNumberOfLines = 3
		subtitle.lineBreakMode = .byWordWrapping

		let header = NSStackView(views: [title, subtitle])
		header.orientation = .vertical
		header.alignment = .leading
		header.spacing = 6

		let settingsVC = SettingsViewController(settings: settings, hotkeyManager: hotkeyManager)
		embedded = settingsVC
		addChild(settingsVC)

		let settingsContainer = NSView()
		settingsContainer.translatesAutoresizingMaskIntoConstraints = false
		settingsContainer.addSubview(settingsVC.view)
		settingsVC.view.translatesAutoresizingMaskIntoConstraints = false

		NSLayoutConstraint.activate([
			settingsVC.view.leadingAnchor.constraint(equalTo: settingsContainer.leadingAnchor),
			settingsVC.view.trailingAnchor.constraint(equalTo: settingsContainer.trailingAnchor),
			settingsVC.view.topAnchor.constraint(equalTo: settingsContainer.topAnchor),
			settingsVC.view.bottomAnchor.constraint(equalTo: settingsContainer.bottomAnchor),
		])

		finishButton.target = self
		finishButton.action = #selector(finishSetup)
		finishButton.keyEquivalent = "\r"

		laterButton.target = self
		laterButton.action = #selector(dismissOnboarding)

		let buttons = NSStackView(views: [laterButton, finishButton])
		buttons.orientation = .horizontal
		buttons.alignment = .centerY
		buttons.spacing = 10

		let spacer = NSView()
		spacer.translatesAutoresizingMaskIntoConstraints = false
		spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
		spacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

		let footer = NSStackView(views: [spacer, buttons])
		footer.orientation = .horizontal
		footer.alignment = .centerY

		let root = NSStackView(views: [header, settingsContainer, footer])
		root.orientation = .vertical
		root.alignment = .leading
		root.spacing = 14
		root.translatesAutoresizingMaskIntoConstraints = false
		view.addSubview(root)

		NSLayoutConstraint.activate([
			root.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
			root.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
			root.topAnchor.constraint(equalTo: view.topAnchor, constant: 16),
			root.bottomAnchor.constraint(equalTo: view.bottomAnchor, constant: -16),
		])
	}

	@objc private func finishSetup() {
		UserDefaults.standard.set(true, forKey: "PKdictation.didCompleteOnboarding")
		view.window?.close()
	}

	@objc private func dismissOnboarding() {
		view.window?.close()
	}
}
