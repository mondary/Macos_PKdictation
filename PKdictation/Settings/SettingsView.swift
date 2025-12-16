import AppKit
import Combine

@MainActor
final class SettingsWindowController: NSWindowController {
	private let viewController: SettingsViewController

	init(settings: SettingsStore, hotkeyManager: HotkeyManager) {
		self.viewController = SettingsViewController(settings: settings, hotkeyManager: hotkeyManager)

		let window = NSWindow(
			contentRect: NSRect(x: 0, y: 0, width: 520, height: 420),
			styleMask: [.titled, .closable, .miniaturizable],
			backing: .buffered,
			defer: false
		)
		window.title = "PKdictation Settings"
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
final class SettingsViewController: NSViewController, NSComboBoxDelegate {
	private let settings: SettingsStore
	private let hotkeyManager: HotkeyManager
	private var cancellables: Set<AnyCancellable> = []

	private let hotkeyLabel = NSTextField(labelWithString: "")
	private let recordHotkeyButton = NSButton(title: "Record…", target: nil, action: nil)
	private let useFnButton = NSButton(title: "Use Fn", target: nil, action: nil)
	private let useRightCommandButton = NSButton(title: "Use Right ⌘", target: nil, action: nil)
	private let consumeEventsCheckbox = NSButton(checkboxWithTitle: "Consume shortcut keystrokes", target: nil, action: nil)
	private let autoPasteCheckbox = NSButton(checkboxWithTitle: "Auto-paste transcript (Cmd+V)", target: nil, action: nil)
	private let permissionButtonsRow = NSStackView()
	private let enableAccessibilityButton = NSButton(title: "Enable Accessibility…", target: nil, action: nil)
	private let enableInputMonitoringButton = NSButton(title: "Enable Input Monitoring…", target: nil, action: nil)
	private let retryHotkeyButton = NSButton(title: "Retry hotkey", target: nil, action: nil)
	private let hotkeyErrorLabel = NSTextField(labelWithString: "")
	private let permissionsStatusLabel = NSTextField(labelWithString: "")
	private let microphoneStatusLabel = NSTextField(labelWithString: "")
	private let enableMicrophoneButton = NSButton(title: "Enable Microphone…", target: nil, action: nil)

	private let apiKeyField = NSSecureTextField()
	private let apiKeySaveButton = NSButton(title: "Save", target: nil, action: nil)
	private let apiKeyRemoveButton = NSButton(title: "Remove", target: nil, action: nil)
	private let apiKeyStatusLabel = NSTextField(labelWithString: "")
	private let apiKeyCheckButton = NSButton(title: "Check key", target: nil, action: nil)
	private let apiKeyValidityLabel = NSTextField(labelWithString: "")
	private let apiKeyLinkButton = NSButton(title: "Open AI Studio API keys…", target: nil, action: nil)
	private let apiKeyErrorLabel = NSTextField(labelWithString: "")

	private let modelComboBox = NSComboBox()
	private let refreshModelsButton = NSButton(title: "Refresh models", target: nil, action: nil)
	private let modelStatusLabel = NSTextField(labelWithString: "")
	private let modelErrorLabel = NSTextField(labelWithString: "")

	private let logsTextView = NSTextView()
	private let copyLogsButton = NSButton(title: "Copy logs", target: nil, action: nil)

	private var isRecordingHotkey = false
	private var hotkeyMonitor: Any?
	private var modelObserver: Any?
	private var apiKeyObserver: Any?
	private var logsCancellable: AnyCancellable?

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
		setUpUI()
		bind()
	}

	override func viewWillDisappear() {
		super.viewWillDisappear()
		stopHotkeyRecording()
		if let modelObserver {
			NotificationCenter.default.removeObserver(modelObserver)
			self.modelObserver = nil
		}
		if let apiKeyObserver {
			NotificationCenter.default.removeObserver(apiKeyObserver)
			self.apiKeyObserver = nil
		}
		logsCancellable?.cancel()
		logsCancellable = nil
	}

	private func setUpUI() {
		let root = NSStackView()
		root.orientation = .vertical
		root.alignment = .leading
		root.spacing = 18
		root.translatesAutoresizingMaskIntoConstraints = false
		view.addSubview(root)

		NSLayoutConstraint.activate([
			root.leadingAnchor.constraint(equalTo: view.leadingAnchor, constant: 16),
			root.trailingAnchor.constraint(equalTo: view.trailingAnchor, constant: -16),
			root.topAnchor.constraint(equalTo: view.topAnchor, constant: 16),
			root.bottomAnchor.constraint(lessThanOrEqualTo: view.bottomAnchor, constant: -16),
		])

		root.addArrangedSubview(sectionTitle("Push-to-talk"))
		root.addArrangedSubview(pushToTalkSection())

		root.addArrangedSubview(sectionTitle("Gemini API key"))
		root.addArrangedSubview(apiKeySection())

		root.addArrangedSubview(sectionTitle("Gemini"))
		root.addArrangedSubview(modelSection())

		root.addArrangedSubview(sectionTitle("Logs"))
		root.addArrangedSubview(logsSection())
	}

	private func pushToTalkSection() -> NSView {
		let container = NSStackView()
		container.orientation = .vertical
		container.alignment = .leading
		container.spacing = 10

		let hotkeyRow = HotkeyRecorderRow(
			hotkeyLabel: hotkeyLabel,
			recordButton: recordHotkeyButton,
			useFnButton: useFnButton,
			useRightCommandButton: useRightCommandButton
		)
		recordHotkeyButton.target = self
		recordHotkeyButton.action = #selector(toggleHotkeyRecording)
		useFnButton.target = self
		useFnButton.action = #selector(useFn)
		useRightCommandButton.target = self
		useRightCommandButton.action = #selector(useRightCommand)

		consumeEventsCheckbox.target = self
		consumeEventsCheckbox.action = #selector(toggleConsumeEvents)

		autoPasteCheckbox.target = self
		autoPasteCheckbox.action = #selector(toggleAutoPaste)

		enableAccessibilityButton.target = self
		enableAccessibilityButton.action = #selector(enableAccessibility)
		enableInputMonitoringButton.target = self
		enableInputMonitoringButton.action = #selector(enableInputMonitoring)
		retryHotkeyButton.target = self
		retryHotkeyButton.action = #selector(retryHotkey)

		enableMicrophoneButton.target = self
		enableMicrophoneButton.action = #selector(enableMicrophone)

		permissionButtonsRow.orientation = .horizontal
		permissionButtonsRow.alignment = .centerY
		permissionButtonsRow.spacing = 10
		permissionButtonsRow.addArrangedSubview(enableAccessibilityButton)
		permissionButtonsRow.addArrangedSubview(enableInputMonitoringButton)
		permissionButtonsRow.addArrangedSubview(retryHotkeyButton)

		hotkeyErrorLabel.textColor = .systemRed
		hotkeyErrorLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		hotkeyErrorLabel.lineBreakMode = .byWordWrapping
		hotkeyErrorLabel.maximumNumberOfLines = 3
		hotkeyErrorLabel.isHidden = true

		permissionsStatusLabel.textColor = .secondaryLabelColor
		permissionsStatusLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)

		microphoneStatusLabel.textColor = .secondaryLabelColor
		microphoneStatusLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)

		container.addArrangedSubview(hotkeyRow)
		container.addArrangedSubview(consumeEventsCheckbox)
		container.addArrangedSubview(autoPasteCheckbox)
		container.addArrangedSubview(microphoneStatusLabel)
		container.addArrangedSubview(enableMicrophoneButton)
		container.addArrangedSubview(permissionButtonsRow)
		container.addArrangedSubview(permissionsStatusLabel)
		container.addArrangedSubview(hotkeyErrorLabel)

		return container
	}

	private func apiKeySection() -> NSView {
		let container = NSStackView()
		container.orientation = .vertical
		container.alignment = .leading
		container.spacing = 10

		apiKeyField.placeholderString = "Paste your Gemini API key"
		apiKeyField.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			apiKeyField.widthAnchor.constraint(equalToConstant: 420),
		])

		apiKeySaveButton.target = self
		apiKeySaveButton.action = #selector(saveApiKey)
		apiKeyRemoveButton.target = self
		apiKeyRemoveButton.action = #selector(removeApiKey)
		apiKeyCheckButton.target = self
		apiKeyCheckButton.action = #selector(checkApiKey)

		apiKeyLinkButton.target = self
		apiKeyLinkButton.action = #selector(openApiKeysPage)
		apiKeyLinkButton.toolTip = "https://aistudio.google.com/api-keys"
		apiKeyLinkButton.isBordered = false
		apiKeyLinkButton.bezelStyle = .regularSquare
		apiKeyLinkButton.alignment = .left
		apiKeyLinkButton.setContentHuggingPriority(.required, for: .horizontal)
		let linkTitle = NSMutableAttributedString(string: apiKeyLinkButton.title)
		linkTitle.addAttributes(
			[
				.foregroundColor: NSColor.linkColor,
				.underlineStyle: NSUnderlineStyle.single.rawValue,
			],
			range: NSRange(location: 0, length: linkTitle.length)
		)
		apiKeyLinkButton.attributedTitle = linkTitle

		let buttonsRow = NSStackView()
		buttonsRow.orientation = .horizontal
		buttonsRow.alignment = .centerY
		buttonsRow.spacing = 10
		buttonsRow.addArrangedSubview(apiKeySaveButton)
		buttonsRow.addArrangedSubview(apiKeyRemoveButton)
		buttonsRow.addArrangedSubview(apiKeyCheckButton)

		let spacer = NSView()
		spacer.translatesAutoresizingMaskIntoConstraints = false
		spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
		spacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
		buttonsRow.addArrangedSubview(spacer)

		apiKeyStatusLabel.textColor = .secondaryLabelColor
		buttonsRow.addArrangedSubview(apiKeyStatusLabel)

		apiKeyValidityLabel.textColor = .secondaryLabelColor
		apiKeyValidityLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)

		apiKeyErrorLabel.textColor = .systemRed
		apiKeyErrorLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		apiKeyErrorLabel.lineBreakMode = .byWordWrapping
		apiKeyErrorLabel.maximumNumberOfLines = 3
		apiKeyErrorLabel.isHidden = true

		container.addArrangedSubview(apiKeyField)
		container.addArrangedSubview(buttonsRow)
		container.addArrangedSubview(apiKeyValidityLabel)
		container.addArrangedSubview(apiKeyLinkButton)
		container.addArrangedSubview(apiKeyErrorLabel)

		return container
	}

	private func modelSection() -> NSView {
		let container = NSStackView()
		container.orientation = .vertical
		container.alignment = .leading
		container.spacing = 10

		modelComboBox.usesDataSource = false
		modelComboBox.isEditable = true
		modelComboBox.completes = true
		modelComboBox.delegate = self
		modelComboBox.placeholderString = "Model (e.g. gemini-2.0-flash)"
		modelComboBox.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			modelComboBox.widthAnchor.constraint(equalToConstant: 420),
		])

		refreshModelsButton.target = self
		refreshModelsButton.action = #selector(refreshModels)

		modelStatusLabel.textColor = .secondaryLabelColor
		modelStatusLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)

		modelErrorLabel.textColor = .systemRed
		modelErrorLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		modelErrorLabel.lineBreakMode = .byWordWrapping
		modelErrorLabel.maximumNumberOfLines = 3
		modelErrorLabel.isHidden = true

		let row = NSStackView()
		row.orientation = .horizontal
		row.alignment = .centerY
		row.spacing = 10
		row.addArrangedSubview(modelComboBox)
		row.addArrangedSubview(refreshModelsButton)

		container.addArrangedSubview(row)
		container.addArrangedSubview(modelStatusLabel)
		container.addArrangedSubview(modelErrorLabel)
		return container
	}

	private func logsSection() -> NSView {
		let container = NSStackView()
		container.orientation = .vertical
		container.alignment = .leading
		container.spacing = 10

		let scroll = NSScrollView()
		scroll.hasVerticalScroller = true
		scroll.borderType = .bezelBorder
		scroll.translatesAutoresizingMaskIntoConstraints = false
		scroll.drawsBackground = true

		logsTextView.isEditable = false
		logsTextView.isSelectable = true
		logsTextView.drawsBackground = false
		logsTextView.font = NSFont.monospacedSystemFont(ofSize: 11, weight: .regular)
		scroll.documentView = logsTextView

		NSLayoutConstraint.activate([
			scroll.widthAnchor.constraint(equalToConstant: 480),
			scroll.heightAnchor.constraint(equalToConstant: 140),
		])

		copyLogsButton.target = self
		copyLogsButton.action = #selector(copyLogs)

		container.addArrangedSubview(scroll)
		container.addArrangedSubview(copyLogsButton)
		return container
	}

	private func bind() {
		settings.$pushToTalkHotkey
			.receive(on: RunLoop.main)
			.sink { [weak self] hotkey in
				self?.hotkeyLabel.stringValue = hotkey.displayString
			}
			.store(in: &cancellables)

		settings.$consumeHotkeyEvents
			.receive(on: RunLoop.main)
			.sink { [weak self] consume in
				self?.consumeEventsCheckbox.state = consume ? .on : .off
			}
			.store(in: &cancellables)

		settings.$autoPasteTranscript
			.receive(on: RunLoop.main)
			.sink { [weak self] enabled in
				self?.autoPasteCheckbox.state = enabled ? .on : .off
			}
			.store(in: &cancellables)

		settings.$hasGeminiApiKey
			.receive(on: RunLoop.main)
			.sink { [weak self] hasKey in
				let key = self?.settings.loadGeminiApiKey() ?? ""
				let suffix = hasKey && key.isEmpty == false ? " (\(Self.redactedApiKey(key)))" : ""
				self?.apiKeyStatusLabel.stringValue = hasKey ? "Saved in Keychain\(suffix)" : "Not set"
				self?.apiKeyRemoveButton.isHidden = !hasKey
			}
			.store(in: &cancellables)

		settings.$geminiModelName
			.receive(on: RunLoop.main)
			.sink { [weak self] model in
				guard let self else { return }
				if self.modelComboBox.stringValue != model {
					self.modelComboBox.stringValue = model
				}
			}
			.store(in: &cancellables)

		hotkeyManager.$isRunning
			.receive(on: RunLoop.main)
			.sink { [weak self] _ in
				self?.updatePermissionUI()
			}
			.store(in: &cancellables)

		hotkeyManager.$lastErrorMessage
			.receive(on: RunLoop.main)
			.sink { [weak self] message in
				self?.hotkeyErrorLabel.stringValue = message ?? ""
				self?.hotkeyErrorLabel.isHidden = (message?.isEmpty ?? true)
				self?.updatePermissionUI()
			}
			.store(in: &cancellables)

		modelObserver = NotificationCenter.default.addObserver(
			forName: NSControl.textDidChangeNotification,
			object: modelComboBox,
			queue: .main
		) { [weak self] _ in
			guard let self else { return }
			self.settings.geminiModelName = self.modelComboBox.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
		}

		apiKeyObserver = NotificationCenter.default.addObserver(
			forName: NSControl.textDidChangeNotification,
			object: apiKeyField,
			queue: .main
		) { [weak self] _ in
			guard let self else { return }
			self.scheduleAutoSaveApiKey()
		}

		hotkeyLabel.stringValue = settings.pushToTalkHotkey.displayString
		consumeEventsCheckbox.state = settings.consumeHotkeyEvents ? .on : .off
		autoPasteCheckbox.state = settings.autoPasteTranscript ? .on : .off
		if let existingKey = settings.loadGeminiApiKey(), existingKey.isEmpty == false {
			apiKeyField.stringValue = existingKey
		}
		let suffix = settings.hasGeminiApiKey ? " (\(Self.redactedApiKey(settings.loadGeminiApiKey() ?? "")))" : ""
		apiKeyStatusLabel.stringValue = settings.hasGeminiApiKey ? "Saved in Keychain\(suffix)" : "Not set"
		apiKeyValidityLabel.stringValue = "Key: not checked yet"
		apiKeyRemoveButton.isHidden = !settings.hasGeminiApiKey
		seedModelList()
		modelComboBox.stringValue = settings.geminiModelName
		modelStatusLabel.stringValue = "Tip: click “Refresh models” to load from your API key."
		updatePermissionUI()
		updatePermissionStatusLabel()
		updateMicrophoneUI()

		logsCancellable = LogStore.shared.$lines
			.receive(on: RunLoop.main)
			.sink { [weak self] lines in
				guard let self else { return }
				self.logsTextView.string = lines.joined(separator: "\n")
				self.logsTextView.scrollToEndOfDocument(nil)
			}
	}

	private func updatePermissionUI() {
		let ax = AccessibilityPermission.isTrusted()
		let im = InputMonitoringPermission.isTrusted()
		let hotkey = hotkeyManager.isRunning

		enableAccessibilityButton.isHidden = ax
		// Only suggest IM once AX is granted (otherwise it's just noise).
		enableInputMonitoringButton.isHidden = !ax || im
		retryHotkeyButton.isHidden = hotkey && ax

		let showRow = (!ax) || (!im && ax) || (!hotkey)
		permissionButtonsRow.isHidden = !showRow
		updatePermissionStatusLabel()
	}

	private func updateMicrophoneUI() {
		let status = MicrophonePermission.status()
		switch status {
		case .authorized:
			microphoneStatusLabel.stringValue = "✅ Microphone: granted"
			enableMicrophoneButton.isHidden = true
		case .notDetermined:
			microphoneStatusLabel.stringValue = "ℹ️ Microphone: not requested yet"
			enableMicrophoneButton.isHidden = false
		case .denied, .restricted:
			microphoneStatusLabel.stringValue = "❌ Microphone: denied (open Settings to enable)"
			enableMicrophoneButton.isHidden = false
		@unknown default:
			microphoneStatusLabel.stringValue = "ℹ️ Microphone: unknown"
			enableMicrophoneButton.isHidden = false
		}
	}

	private func updatePermissionStatusLabel() {
		let ax = AccessibilityPermission.isTrusted()
		let im = InputMonitoringPermission.isTrusted()
		let hotkey = hotkeyManager.isRunning
		let axText = ax ? "✅ Accessibility: granted" : "❌ Accessibility: not granted"
		let hotkeyText = hotkey ? "✅ Hotkey listener: running" : "❌ Hotkey listener: not running"
		let imText: String
		if im {
			imText = "✅ Input Monitoring: granted"
		} else if hotkey {
			imText = "ℹ️ Input Monitoring: not granted (may be OK)"
		} else {
			imText = "❌ Input Monitoring: not granted (may be required)"
		}

		permissionsStatusLabel.stringValue = "\(axText)   \(imText)   \(hotkeyText)"
	}

	@objc private func toggleHotkeyRecording() {
		isRecordingHotkey ? stopHotkeyRecording() : startHotkeyRecording()
	}

	private func startHotkeyRecording() {
		stopHotkeyRecording()
		isRecordingHotkey = true
		recordHotkeyButton.title = "Press keys…"

		var pendingModifier: (keyCode: UInt16, modifiers: NSEvent.ModifierFlags)?

		hotkeyMonitor = NSEvent.addLocalMonitorForEvents(matching: [.keyDown, .flagsChanged]) { [weak self] event in
			guard let self else { return event }
			let mods = event.modifierFlags.intersection(Hotkey.relevantModifiers)

			if event.type == .keyDown {
				self.settings.pushToTalkHotkey = Hotkey(keyCode: event.keyCode, modifiers: mods)
				self.stopHotkeyRecording()
				return nil
			}

			guard event.type == .flagsChanged else { return event }
			guard let flag = Hotkey.modifierFlag(forKeyCode: event.keyCode) else { return nil }

			let isDownNow = mods.contains(flag)
			if isDownNow {
				pendingModifier = (event.keyCode, mods)
				return nil
			}

			guard let pending = pendingModifier, pending.keyCode == event.keyCode else { return nil }

			let recorded: Hotkey
			if flag == .function && pending.modifiers == [.function] {
				recorded = .defaultPushToTalk
			} else {
				recorded = Hotkey(keyCode: pending.keyCode, modifiers: pending.modifiers)
			}

			self.settings.pushToTalkHotkey = recorded
			self.stopHotkeyRecording()
			return nil
		}
	}

	private func stopHotkeyRecording() {
		isRecordingHotkey = false
		recordHotkeyButton.title = "Record…"
		if let hotkeyMonitor {
			NSEvent.removeMonitor(hotkeyMonitor)
			self.hotkeyMonitor = nil
		}
	}

	@objc private func useFn() {
		settings.pushToTalkHotkey = .defaultPushToTalk
	}

	@objc private func useRightCommand() {
		settings.pushToTalkHotkey = .rightCommandPushToTalk
	}

	@objc private func toggleConsumeEvents(_ sender: NSButton) {
		settings.consumeHotkeyEvents = (sender.state == .on)
	}

	@objc private func toggleAutoPaste(_ sender: NSButton) {
		settings.autoPasteTranscript = (sender.state == .on)
	}

	@objc private func enableAccessibility() {
		AccessibilityPermission.openSettings()
		AccessibilityPermission.requestPrompt()
		updatePermissionUI()
		DispatchQueue.main.asyncAfter(deadline: .now() + 0.4) { [weak self] in
			self?.retryHotkey()
		}
	}

	@objc private func enableInputMonitoring() {
		InputMonitoringPermission.openSettings()
		InputMonitoringPermission.requestPrompt()
		updatePermissionUI()
		DispatchQueue.main.asyncAfter(deadline: .now() + 0.4) { [weak self] in
			self?.retryHotkey()
		}
	}

	@objc private func retryHotkey() {
		LogStore.shared.log("Retrying hotkey listener…")
		hotkeyManager.stop()
		hotkeyManager.start()
		updatePermissionUI()
	}

	@objc private func enableMicrophone() {
		switch MicrophonePermission.status() {
		case .authorized:
			updateMicrophoneUI()
		case .notDetermined:
			MicrophonePermission.request { _ in
				DispatchQueue.main.async { [weak self] in
					LogStore.shared.log("Microphone permission requested.")
					self?.updateMicrophoneUI()
				}
			}
		case .denied, .restricted:
			MicrophonePermission.openSettings()
		@unknown default:
			MicrophonePermission.openSettings()
		}
	}

	@objc private func openApiKeysPage() {
		guard let url = URL(string: "https://aistudio.google.com/api-keys") else { return }
		NSWorkspace.shared.open(url)
	}

	@objc private func saveApiKey() {
		apiKeyErrorLabel.isHidden = true
		do {
			try settings.saveGeminiApiKey(apiKeyField.stringValue)
		} catch {
			apiKeyErrorLabel.stringValue = error.localizedDescription
			apiKeyErrorLabel.isHidden = false
		}
	}

	@objc private func removeApiKey() {
		apiKeyErrorLabel.isHidden = true
		do {
			try settings.deleteGeminiApiKey()
			apiKeyField.stringValue = ""
		} catch {
			apiKeyErrorLabel.stringValue = error.localizedDescription
			apiKeyErrorLabel.isHidden = false
		}
	}

	@objc private func refreshModels() {
		modelErrorLabel.isHidden = true
		modelStatusLabel.stringValue = "Loading models…"
		refreshModelsButton.isEnabled = false

		let apiKey = settings.loadGeminiApiKey() ?? ""
		Task { @MainActor in
			defer { self.refreshModelsButton.isEnabled = true }
			do {
				let models = try await GeminiClient().listModels(apiKey: apiKey)
				self.modelComboBox.removeAllItems()
				self.modelComboBox.addItems(withObjectValues: models)
				self.modelStatusLabel.stringValue = models.isEmpty ? "No models returned." : "Loaded \(models.count) models."
				if self.modelComboBox.stringValue.isEmpty == false {
					self.modelComboBox.selectItem(withObjectValue: self.modelComboBox.stringValue)
				}
			} catch {
				self.modelErrorLabel.stringValue = error.localizedDescription
				self.modelErrorLabel.isHidden = false
				self.modelStatusLabel.stringValue = "Failed to load models."
			}
		}
	}

	@objc private func checkApiKey() {
		apiKeyErrorLabel.isHidden = true
		apiKeyValidityLabel.stringValue = "Checking key…"
		apiKeyCheckButton.isEnabled = false

		let apiKey = settings.loadGeminiApiKey() ?? ""
		Task { @MainActor in
			defer { self.apiKeyCheckButton.isEnabled = true }
			do {
				let models = try await GeminiClient().listModels(apiKey: apiKey)
				let countText = models.isEmpty ? "0 models" : "\(models.count) models"
				self.apiKeyValidityLabel.stringValue = "✅ Key OK (\(countText))"
				LogStore.shared.log("API key check OK (\(countText)).")
			} catch {
				self.apiKeyValidityLabel.stringValue = "❌ Key check failed"
				self.apiKeyErrorLabel.stringValue = error.localizedDescription
				self.apiKeyErrorLabel.isHidden = false
				LogStore.shared.log("API key check failed: \(error.localizedDescription)")
			}
		}
	}

	func comboBoxSelectionDidChange(_ notification: Notification) {
		guard let combo = notification.object as? NSComboBox, combo === modelComboBox else { return }
		let idx = modelComboBox.indexOfSelectedItem
		if idx >= 0, let selected = modelComboBox.itemObjectValue(at: idx) as? String {
			settings.geminiModelName = selected.trimmingCharacters(in: .whitespacesAndNewlines)
		} else {
			settings.geminiModelName = modelComboBox.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
		}
	}

	@objc private func copyLogs() {
		let pb = NSPasteboard.general
		pb.clearContents()
		pb.setString(LogStore.shared.dump(), forType: .string)
	}

	private var pendingApiKeySaveWorkItem: DispatchWorkItem?

	private func scheduleAutoSaveApiKey(immediate: Bool = false) {
		pendingApiKeySaveWorkItem?.cancel()

		let work = DispatchWorkItem { [weak self] in
			guard let self else { return }
			let value = self.apiKeyField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
			guard value.count >= 20 || value.isEmpty else { return }
			self.apiKeyErrorLabel.isHidden = true
			do {
				try self.settings.saveGeminiApiKey(value)
			} catch {
				self.apiKeyErrorLabel.stringValue = error.localizedDescription
				self.apiKeyErrorLabel.isHidden = false
			}
		}

		pendingApiKeySaveWorkItem = work
		DispatchQueue.main.asyncAfter(deadline: .now() + (immediate ? 0 : 0.4), execute: work)
	}

	private func seedModelList() {
		let defaults = [
			"gemini-2.5-flash-lite",
			"gemini-2.5-flash",
			"gemini-2.0-flash-lite",
			"gemini-2.0-flash",
			"gemini-2.0-pro",
			"gemini-1.5-flash",
			"gemini-1.5-pro",
		]
		modelComboBox.removeAllItems()
		modelComboBox.addItems(withObjectValues: defaults)
	}

	private static func redactedApiKey(_ key: String) -> String {
		let trimmed = key.trimmingCharacters(in: .whitespacesAndNewlines)
		guard trimmed.count >= 12 else { return "•••" }
		let start = trimmed.prefix(6)
		let end = trimmed.suffix(4)
		return "\(start)…\(end)"
	}
}

private func sectionTitle(_ title: String) -> NSTextField {
	let label = NSTextField(labelWithString: title)
	label.font = .systemFont(ofSize: 13, weight: .semibold)
	label.textColor = .secondaryLabelColor
	return label
}
