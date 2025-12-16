import AppKit

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
	private var statusItem: NSStatusItem?
	private var overlay: OverlayWindowController?
	private var settingsWindowController: SettingsWindowController?

	func applicationDidFinishLaunching(_ notification: Notification) {
		NSApp.setActivationPolicy(.accessory)
		setUpMainMenu()
		setUpOverlay()
		setUpStatusItem()
		AppModel.shared.hotkeyManager.start()
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
		menu.addItem(NSMenuItem(title: "Start recording", action: #selector(startRecording), keyEquivalent: ""))
		menu.addItem(NSMenuItem(title: "Stop recording", action: #selector(stopRecording), keyEquivalent: ""))
		menu.addItem(.separator())
		menu.addItem(NSMenuItem(title: "Settings…", action: #selector(openSettings), keyEquivalent: ","))
		menu.addItem(.separator())
		menu.addItem(NSMenuItem(title: "Quit", action: #selector(quit), keyEquivalent: "q"))
		item.menu = menu

		statusItem = item
	}

	@objc private func startRecording() {
		AppModel.shared.dictation.start()
	}

	@objc private func stopRecording() {
		AppModel.shared.dictation.stop()
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
}
