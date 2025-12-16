import AppKit

@MainActor
final class OverlayContentView: NSView {
	private let background: RoundedVisualEffectView
	private let iconView: NSImageView
	private let statusLabel: NSTextField
	private let transcriptLabel: NSTextField
	private let spectrumView: SpectrumBarsView

	override init(frame frameRect: NSRect) {
		self.background = RoundedVisualEffectView(
			material: .hudWindow,
			blendingMode: .withinWindow,
			cornerRadius: 16,
			borderColor: NSColor.white.withAlphaComponent(0.12),
			borderWidth: 1
		)

		self.iconView = NSImageView()
		self.statusLabel = NSTextField(labelWithString: "")
		self.transcriptLabel = NSTextField(labelWithString: "")
		self.spectrumView = SpectrumBarsView()

		super.init(frame: frameRect)

		wantsLayer = true
		setUpView()
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}

	func update(phase: DictationController.Phase, statusText: String, transcript: String) {
		let isListening = (phase == .listening)
		iconView.image = NSImage(systemSymbolName: isListening ? "mic.fill" : "waveform", accessibilityDescription: "PKdictation")
		iconView.contentTintColor = isListening ? .systemRed : .secondaryLabelColor

		statusLabel.stringValue = statusText

		let trimmed = transcript.trimmingCharacters(in: .whitespacesAndNewlines)
		transcriptLabel.stringValue = trimmed
		transcriptLabel.isHidden = trimmed.isEmpty
	}

	func update(levels: [Double]) {
		spectrumView.levels = levels
	}

	private func setUpView() {
		background.translatesAutoresizingMaskIntoConstraints = false
		addSubview(background)

		NSLayoutConstraint.activate([
			background.leadingAnchor.constraint(equalTo: leadingAnchor),
			background.trailingAnchor.constraint(equalTo: trailingAnchor),
			background.topAnchor.constraint(equalTo: topAnchor),
			background.bottomAnchor.constraint(equalTo: bottomAnchor),
		])

		let hStack = NSStackView()
		hStack.orientation = .horizontal
		hStack.alignment = .centerY
		hStack.spacing = 12
		hStack.translatesAutoresizingMaskIntoConstraints = false
		background.addSubview(hStack)

		NSLayoutConstraint.activate([
			hStack.leadingAnchor.constraint(equalTo: background.leadingAnchor, constant: 14),
			hStack.trailingAnchor.constraint(equalTo: background.trailingAnchor, constant: -14),
			hStack.topAnchor.constraint(equalTo: background.topAnchor, constant: 14),
			hStack.bottomAnchor.constraint(equalTo: background.bottomAnchor, constant: -14),
		])

		iconView.translatesAutoresizingMaskIntoConstraints = false
		iconView.symbolConfiguration = NSImage.SymbolConfiguration(pointSize: 20, weight: .semibold)
		iconView.contentTintColor = .secondaryLabelColor
		iconView.image = NSImage(systemSymbolName: "waveform", accessibilityDescription: "PKdictation")

		NSLayoutConstraint.activate([
			iconView.widthAnchor.constraint(equalToConstant: 22),
			iconView.heightAnchor.constraint(equalToConstant: 22),
		])

		hStack.addArrangedSubview(iconView)

		let vStack = NSStackView()
		vStack.orientation = .vertical
		vStack.alignment = .leading
		vStack.spacing = 8
		vStack.translatesAutoresizingMaskIntoConstraints = false
		hStack.addArrangedSubview(vStack)

		statusLabel.font = NSFont.systemFont(ofSize: 15, weight: .semibold)
		statusLabel.textColor = .labelColor
		vStack.addArrangedSubview(statusLabel)

		transcriptLabel.font = NSFont.systemFont(ofSize: 13, weight: .regular)
		transcriptLabel.textColor = .labelColor
		transcriptLabel.lineBreakMode = .byTruncatingTail
		transcriptLabel.maximumNumberOfLines = 3
		transcriptLabel.isHidden = true
		vStack.addArrangedSubview(transcriptLabel)

		spectrumView.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			spectrumView.heightAnchor.constraint(equalToConstant: 38),
			spectrumView.widthAnchor.constraint(greaterThanOrEqualToConstant: 140),
		])
		vStack.addArrangedSubview(spectrumView)
	}
}

final class SpectrumBarsView: NSView {
	var levels: [Double] = Array(repeating: 0, count: 16) {
		didSet { needsDisplay = true }
	}

	override var isOpaque: Bool { false }

	override func draw(_ dirtyRect: NSRect) {
		super.draw(dirtyRect)

		guard levels.isEmpty == false else { return }

		let barCount = levels.count
		let barWidth: CGFloat = 5
		let spacing: CGFloat = 3
		let minHeight: CGFloat = 4
		let maxExtra: CGFloat = 34

		let totalWidth = CGFloat(barCount) * barWidth + CGFloat(max(0, barCount - 1)) * spacing
		let startX = max(0, (bounds.width - totalWidth) / 2)

		let color = NSColor.controlAccentColor.withAlphaComponent(0.85)
		color.setFill()

		for (index, level) in levels.enumerated() {
			let value = max(0, min(1, level))
			let height = minHeight + value * maxExtra
			let x = startX + CGFloat(index) * (barWidth + spacing)
			let rect = NSRect(x: x, y: 0, width: barWidth, height: height)
			let path = NSBezierPath(roundedRect: rect, xRadius: barWidth / 2, yRadius: barWidth / 2)
			path.fill()
		}
	}
}
