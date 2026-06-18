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
	private let accessibilityStatusLabel = NSTextField(labelWithString: "")
	private let inputMonitoringStatusLabel = NSTextField(labelWithString: "")
	private let hotkeyListenerStatusLabel = NSTextField(labelWithString: "")
	private let hotkeyErrorLabel = NSTextField(labelWithString: "")
	private let installWarningLabel = NSTextField(labelWithString: "")
	private let revealAppButton = NSButton(title: "Show app in Finder", target: nil, action: nil)
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

	private let chatField = NSTextField()
	private let chatSendButton = NSButton(title: "Send", target: nil, action: nil)
	private let chatClearButton = NSButton(title: "Clear", target: nil, action: nil)
	private let chatStatusLabel = NSTextField(labelWithString: "")
	private let chatTextView = NSTextView()

	private let logsTextView = NSTextView()
	private let copyLogsButton = NSButton(title: "Copy logs", target: nil, action: nil)
	private let openLogFileButton = NSButton(title: "Open log file", target: nil, action: nil)
	private let revealLogFileButton = NSButton(title: "Reveal in Finder", target: nil, action: nil)
	private let logFilePathLabel = NSTextField(labelWithString: "")

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
		view = NSView(frame: NSRect(x: 0, y: 0, width: 520, height: 420))
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
		let scroll = NSScrollView()
		scroll.hasVerticalScroller = true
		scroll.drawsBackground = false
		scroll.borderType = .noBorder
		scroll.translatesAutoresizingMaskIntoConstraints = false
		view.addSubview(scroll)

		NSLayoutConstraint.activate([
			scroll.leadingAnchor.constraint(equalTo: view.leadingAnchor),
			scroll.trailingAnchor.constraint(equalTo: view.trailingAnchor),
			scroll.topAnchor.constraint(equalTo: view.topAnchor),
			scroll.bottomAnchor.constraint(equalTo: view.bottomAnchor),
		])

		let document = NSView()
		document.translatesAutoresizingMaskIntoConstraints = false
		scroll.documentView = document

		let root = NSStackView()
		root.orientation = .vertical
		root.alignment = .leading
		root.spacing = 18
		root.translatesAutoresizingMaskIntoConstraints = false
		document.addSubview(root)

		NSLayoutConstraint.activate([
			document.leadingAnchor.constraint(equalTo: scroll.contentView.leadingAnchor),
			document.trailingAnchor.constraint(equalTo: scroll.contentView.trailingAnchor),
			document.topAnchor.constraint(equalTo: scroll.contentView.topAnchor),
			document.bottomAnchor.constraint(equalTo: scroll.contentView.bottomAnchor),
			document.widthAnchor.constraint(equalTo: scroll.contentView.widthAnchor),

			root.leadingAnchor.constraint(equalTo: document.leadingAnchor, constant: 16),
			root.trailingAnchor.constraint(equalTo: document.trailingAnchor, constant: -16),
			root.topAnchor.constraint(equalTo: document.topAnchor, constant: 16),
			root.bottomAnchor.constraint(equalTo: document.bottomAnchor, constant: -16),
		])

		root.addArrangedSubview(sectionTitle("Push-to-talk"))
		root.addArrangedSubview(pushToTalkSection())

		root.addArrangedSubview(sectionTitle("Gemini API key"))
		root.addArrangedSubview(apiKeySection())

		root.addArrangedSubview(sectionTitle("Gemini"))
		root.addArrangedSubview(modelSection())

		root.addArrangedSubview(sectionTitle("Ask AI"))
		root.addArrangedSubview(chatSection())

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

		revealAppButton.target = self
		revealAppButton.action = #selector(revealAppInFinder)

		[accessibilityStatusLabel, inputMonitoringStatusLabel, hotkeyListenerStatusLabel].forEach { label in
			label.textColor = .secondaryLabelColor
			label.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
			label.lineBreakMode = .byTruncatingTail
		}

		permissionButtonsRow.orientation = .vertical
		permissionButtonsRow.alignment = .leading
		permissionButtonsRow.spacing = 8

		permissionButtonsRow.addArrangedSubview(permissionRow(button: enableAccessibilityButton, statusLabel: accessibilityStatusLabel))
		permissionButtonsRow.addArrangedSubview(permissionRow(button: enableInputMonitoringButton, statusLabel: inputMonitoringStatusLabel))
		permissionButtonsRow.addArrangedSubview(permissionRow(button: retryHotkeyButton, statusLabel: hotkeyListenerStatusLabel))

		hotkeyErrorLabel.textColor = .systemRed
		hotkeyErrorLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		hotkeyErrorLabel.lineBreakMode = .byWordWrapping
		hotkeyErrorLabel.maximumNumberOfLines = 3
		hotkeyErrorLabel.isHidden = true

		permissionsStatusLabel.isHidden = true

		microphoneStatusLabel.textColor = .secondaryLabelColor
		microphoneStatusLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)

		installWarningLabel.textColor = .systemOrange
		installWarningLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		installWarningLabel.lineBreakMode = .byWordWrapping
		installWarningLabel.maximumNumberOfLines = 2
		installWarningLabel.isHidden = true

		revealAppButton.isHidden = true

		container.addArrangedSubview(hotkeyRow)
		container.addArrangedSubview(consumeEventsCheckbox)
		container.addArrangedSubview(autoPasteCheckbox)
		container.addArrangedSubview(microphoneStatusLabel)
		container.addArrangedSubview(enableMicrophoneButton)
		container.addArrangedSubview(installWarningLabel)
		container.addArrangedSubview(revealAppButton)
		container.addArrangedSubview(permissionButtonsRow)
		container.addArrangedSubview(hotkeyErrorLabel)

		return container
	}

	private func permissionRow(button: NSButton, statusLabel: NSTextField) -> NSView {
		let row = NSStackView()
		row.orientation = .horizontal
		row.alignment = .centerY
		row.spacing = 10

		button.setContentHuggingPriority(.required, for: .horizontal)
		statusLabel.setContentHuggingPriority(.defaultLow, for: .horizontal)

		row.addArrangedSubview(button)
		row.addArrangedSubview(statusLabel)

		let spacer = NSView()
		spacer.translatesAutoresizingMaskIntoConstraints = false
		spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
		spacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
		row.addArrangedSubview(spacer)
		return row
	}

	private func apiKeySection() -> NSView {
		let container = NSStackView()
		container.orientation = .vertical
		container.alignment = .leading
		container.spacing = 10

		apiKeyField.placeholderString = "Paste your Gemini API key"
		apiKeyField.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			apiKeyField.widthAnchor.constraint(equalToConstant: 360),
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

		let keyRow = NSStackView()
		keyRow.orientation = .horizontal
		keyRow.alignment = .centerY
		keyRow.spacing = 10
		keyRow.addArrangedSubview(apiKeyField)
		keyRow.addArrangedSubview(apiKeyCheckButton)

		container.addArrangedSubview(apiKeyLinkButton)
		container.addArrangedSubview(keyRow)
		container.addArrangedSubview(buttonsRow)
		container.addArrangedSubview(apiKeyValidityLabel)
		container.addArrangedSubview(apiKeyErrorLabel)

		return container
	}

	private func chatSection() -> NSView {
		let container = NSStackView()
		container.orientation = .vertical
		container.alignment = .leading
		container.spacing = 10

		chatField.placeholderString = "Ask something…"
		chatField.target = self
		chatField.action = #selector(sendChat)
		chatField.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			chatField.widthAnchor.constraint(equalToConstant: 360),
		])

		chatSendButton.target = self
		chatSendButton.action = #selector(sendChat)
		chatClearButton.target = self
		chatClearButton.action = #selector(clearChat)

		let inputRow = NSStackView()
		inputRow.orientation = .horizontal
		inputRow.alignment = .centerY
		inputRow.spacing = 10
		inputRow.addArrangedSubview(chatField)
		inputRow.addArrangedSubview(chatSendButton)
		inputRow.addArrangedSubview(chatClearButton)

		let scroll = NSScrollView()
		scroll.hasVerticalScroller = true
		scroll.borderType = .bezelBorder
		scroll.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			scroll.widthAnchor.constraint(equalToConstant: 460),
			scroll.heightAnchor.constraint(equalToConstant: 140),
		])

		chatTextView.isEditable = false
		chatTextView.isSelectable = true
		chatTextView.font = NSFont.systemFont(ofSize: 12)
		chatTextView.textContainerInset = NSSize(width: 10, height: 10)
		chatTextView.isVerticallyResizable = true
		chatTextView.isHorizontallyResizable = false
		chatTextView.autoresizingMask = [.width]
		chatTextView.textContainer?.widthTracksTextView = true
		chatTextView.textContainer?.containerSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
		chatTextView.frame = NSRect(x: 0, y: 0, width: 460, height: 140)
		scroll.documentView = chatTextView

		chatStatusLabel.textColor = .secondaryLabelColor
		chatStatusLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		chatStatusLabel.stringValue = "Uses your Gemini API key and selected model."

		container.addArrangedSubview(inputRow)
		container.addArrangedSubview(scroll)
		container.addArrangedSubview(chatStatusLabel)
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
		logsTextView.isVerticallyResizable = true
		logsTextView.isHorizontallyResizable = false
		logsTextView.autoresizingMask = [.width]
		logsTextView.textContainerInset = NSSize(width: 10, height: 10)
		logsTextView.textContainer?.widthTracksTextView = true
		logsTextView.textContainer?.containerSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
		logsTextView.frame = NSRect(x: 0, y: 0, width: 480, height: 140)
		scroll.documentView = logsTextView

		NSLayoutConstraint.activate([
			scroll.widthAnchor.constraint(equalToConstant: 480),
			scroll.heightAnchor.constraint(equalToConstant: 140),
		])

		copyLogsButton.target = self
		copyLogsButton.action = #selector(copyLogs)

		openLogFileButton.target = self
		openLogFileButton.action = #selector(openLogFile)

		revealLogFileButton.target = self
		revealLogFileButton.action = #selector(revealLogFile)

		logFilePathLabel.textColor = .secondaryLabelColor
		logFilePathLabel.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
		logFilePathLabel.lineBreakMode = .byTruncatingMiddle
		logFilePathLabel.maximumNumberOfLines = 1

		let buttonsRow = NSStackView()
		buttonsRow.orientation = .horizontal
		buttonsRow.alignment = .centerY
		buttonsRow.spacing = 10
		buttonsRow.addArrangedSubview(copyLogsButton)
		buttonsRow.addArrangedSubview(openLogFileButton)
		buttonsRow.addArrangedSubview(revealLogFileButton)

		container.addArrangedSubview(scroll)
		container.addArrangedSubview(buttonsRow)
		container.addArrangedSubview(logFilePathLabel)
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
			updateInstallWarningUI()

		chatSendButton.keyEquivalent = "\r"
		chatTextView.string = ""

		logsCancellable = LogStore.shared.$lines
			.receive(on: RunLoop.main)
			.sink { [weak self] lines in
				guard let self else { return }
				self.logsTextView.string = lines.joined(separator: "\n")
				self.logsTextView.scrollToEndOfDocument(nil)
			}

		if let url = LogStore.shared.logFileURL() {
			logFilePathLabel.stringValue = "Log file: \(url.path)"
		} else {
			logFilePathLabel.stringValue = "Log file: (unavailable)"
		}
	}

	private func updatePermissionUI() {
		let ax = AccessibilityPermission.isTrusted()
		let im = InputMonitoringPermission.isTrusted()
		let hotkey = hotkeyManager.isRunning

		updatePermissionStatusLabel()
	}

	private func updateInstallWarningUI() {
		let bundlePath = Bundle.main.bundleURL.path
		let looksLikeDerivedData = bundlePath.contains("/DerivedData/") || bundlePath.contains("/.derivedData/")
		let isInApplications = bundlePath.hasPrefix("/Applications/")

		if looksLikeDerivedData || !isInApplications {
			installWarningLabel.isHidden = false
			installWarningLabel.stringValue =
				"Tip: for macOS 15 permissions, run PKdictation from /Applications (not from DerivedData)."
			revealAppButton.isHidden = false
		} else {
			installWarningLabel.isHidden = true
			revealAppButton.isHidden = true
		}
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

		accessibilityStatusLabel.stringValue = ax ? "✅ Granted" : "❌ Not granted"

		if ax == false {
			inputMonitoringStatusLabel.stringValue = "ℹ️ Grant Accessibility first"
		} else {
			inputMonitoringStatusLabel.stringValue = im ? "✅ Granted" : "❌ Not granted"
		}

		hotkeyListenerStatusLabel.stringValue = hotkey ? "✅ Running" : "❌ Not running"
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

	@objc private func revealAppInFinder() {
		let url = Bundle.main.bundleURL
		NSWorkspace.shared.activateFileViewerSelecting([url])
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

	@objc private func clearChat() {
		chatTextView.string = ""
		chatStatusLabel.stringValue = "Cleared."
	}

	@objc private func sendChat() {
		let prompt = chatField.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
		guard prompt.isEmpty == false else { return }

		guard let apiKey = settings.loadGeminiApiKey(), apiKey.isEmpty == false else {
			chatStatusLabel.stringValue = "Missing Gemini API key. Add it above first."
			return
		}

		let modelName = settings.geminiModelName
		chatSendButton.isEnabled = false
		chatStatusLabel.stringValue = "Thinking…"

		appendChat(role: "You", text: prompt)
		chatField.stringValue = ""

		Task { @MainActor in
			defer { self.chatSendButton.isEnabled = true }
			do {
				let answer = try await GeminiClient().chat(prompt: prompt, apiKey: apiKey, modelName: modelName)
				self.appendChat(role: "AI", text: answer)
				self.chatStatusLabel.stringValue = "OK"
				LogStore.shared.log("Chat OK (\(answer.count) chars).")
			} catch {
				self.chatStatusLabel.stringValue = "Error: \(error.localizedDescription)"
				self.appendChat(role: "Error", text: error.localizedDescription)
				if let detailed = (error as? LocalizedError)?.failureReason, detailed.isEmpty == false {
					self.appendChat(role: "Details", text: detailed)
				}
				LogStore.shared.log("Chat failed: \(error.localizedDescription)")
			}
		}
	}

	private func appendChat(role: String, text: String) {
		let cleaned = text.trimmingCharacters(in: .whitespacesAndNewlines)
		guard cleaned.isEmpty == false else { return }
		let existing = chatTextView.string
		let line = "\(role): \(cleaned)"
		chatTextView.string = existing.isEmpty ? line : (existing + "\n\n" + line)
		chatTextView.scrollToEndOfDocument(nil)
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

	@objc private func openLogFile() {
		guard let url = LogStore.shared.logFileURL() else { return }
		NSWorkspace.shared.open(url)
	}

	@objc private func revealLogFile() {
		guard let url = LogStore.shared.logFileURL() else { return }
		NSWorkspace.shared.activateFileViewerSelecting([url])
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
