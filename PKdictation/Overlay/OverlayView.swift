import AppKit

@MainActor
final class OverlayContentView: NSView {
	private let topOverhang: CGFloat = 14
	private let background: NotchBackgroundView
	private let iconView: NSImageView
	private let waveformView: WaveformLineView
	private let micView: NSImageView

	override init(frame frameRect: NSRect) {
		self.background = NotchBackgroundView(cornerRadius: 20)

		self.iconView = NSImageView()
		self.waveformView = WaveformLineView()
		self.micView = NSImageView()

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
		iconView.image = NSImage(systemSymbolName: "waveform", accessibilityDescription: "PKdictation")
		iconView.contentTintColor = isListening ? NSColor.white.withAlphaComponent(0.95) : NSColor.white.withAlphaComponent(0.7)

		micView.image = NSImage(systemSymbolName: "mic.fill", accessibilityDescription: "Microphone")
		micView.contentTintColor = isListening ? NSColor.white.withAlphaComponent(0.95) : NSColor.white.withAlphaComponent(0.7)

		waveformView.isActive = isListening
	}

	func update(levels: [Double]) {
		waveformView.levels = levels
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
		hStack.spacing = 10
		hStack.translatesAutoresizingMaskIntoConstraints = false
		background.addSubview(hStack)

		NSLayoutConstraint.activate([
			hStack.leadingAnchor.constraint(equalTo: background.leadingAnchor, constant: 12),
			hStack.trailingAnchor.constraint(equalTo: background.trailingAnchor, constant: -12),
			// Keep content below the offscreen overhang.
			hStack.topAnchor.constraint(equalTo: background.topAnchor, constant: topOverhang + 8),
			hStack.bottomAnchor.constraint(equalTo: background.bottomAnchor, constant: -10),
		])

		iconView.translatesAutoresizingMaskIntoConstraints = false
		iconView.symbolConfiguration = NSImage.SymbolConfiguration(pointSize: 14, weight: .semibold)
		iconView.contentTintColor = NSColor.white.withAlphaComponent(0.85)
		iconView.image = NSImage(systemSymbolName: "waveform", accessibilityDescription: "PKdictation")

		NSLayoutConstraint.activate([
			iconView.widthAnchor.constraint(equalToConstant: 18),
			iconView.heightAnchor.constraint(equalToConstant: 18),
		])

		hStack.addArrangedSubview(iconView)

		waveformView.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			waveformView.heightAnchor.constraint(equalToConstant: 18),
			waveformView.widthAnchor.constraint(greaterThanOrEqualToConstant: 200),
		])
		hStack.addArrangedSubview(waveformView)

		let spacer = NSView()
		spacer.translatesAutoresizingMaskIntoConstraints = false
		spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
		spacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
		hStack.addArrangedSubview(spacer)

		micView.translatesAutoresizingMaskIntoConstraints = false
		micView.symbolConfiguration = NSImage.SymbolConfiguration(pointSize: 14, weight: .semibold)
		micView.contentTintColor = NSColor.white.withAlphaComponent(0.9)
		micView.image = NSImage(systemSymbolName: "mic.fill", accessibilityDescription: "Microphone")
		NSLayoutConstraint.activate([
			micView.widthAnchor.constraint(equalToConstant: 18),
			micView.heightAnchor.constraint(equalToConstant: 18),
		])
		hStack.addArrangedSubview(micView)
	}
}

final class NotchBackgroundView: NSView {
	private let cornerRadius: CGFloat
	private let gradient = CAGradientLayer()

	init(cornerRadius: CGFloat) {
		self.cornerRadius = cornerRadius
		super.init(frame: .zero)
		wantsLayer = true
		guard let layer else { return }
		layer.cornerRadius = cornerRadius
		layer.masksToBounds = true

		gradient.startPoint = CGPoint(x: 0.5, y: 1.0)
		gradient.endPoint = CGPoint(x: 0.5, y: 0.0)
		gradient.colors = [
			NSColor.black.withAlphaComponent(0.90).cgColor,
			NSColor.black.withAlphaComponent(0.82).cgColor,
		]
		layer.addSublayer(gradient)

		layer.borderColor = NSColor.white.withAlphaComponent(0.10).cgColor
		layer.borderWidth = 1
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}

	override func layout() {
		super.layout()
		gradient.frame = bounds
		gradient.cornerRadius = cornerRadius
	}
}

final class WaveformLineView: NSView {
	var levels: [Double] = Array(repeating: 0, count: 16) {
		didSet { needsDisplay = true }
	}

	var isActive: Bool = false {
		didSet { needsDisplay = true }
	}

	override var isOpaque: Bool { false }

	override func draw(_ dirtyRect: NSRect) {
		super.draw(dirtyRect)

		let inset: CGFloat = 2
		let rect = bounds.insetBy(dx: inset, dy: inset)
		guard rect.width > 6, rect.height > 6 else { return }

		let midY = rect.midY
		// Stronger visual amplitude (still clamped to view bounds).
		let amplitudeMax = rect.height * 0.78

		let raw = levels
		let normalized: [CGFloat]
		if raw.isEmpty {
			normalized = Array(repeating: 0, count: 16)
		} else {
			normalized = raw.map { CGFloat(max(0, min(1, $0))) }
		}

		let energy = normalized.reduce(CGFloat(0)) { $0 + $1 } / CGFloat(max(1, normalized.count))
		let isSilent = energy < 0.02 || isActive == false

		let pointsCount = max(16, normalized.count)
		let stepX = rect.width / CGFloat(pointsCount - 1)

		let path = NSBezierPath()
		path.lineJoinStyle = .round
		path.lineCapStyle = .round
		path.lineWidth = 2

		for idx in 0..<pointsCount {
			let t = CGFloat(idx) / CGFloat(pointsCount - 1)
			let baseRaw = isSilent ? CGFloat(0) : sample(normalized, t: t)
			// Boost small values more than large ones for a visible wave when speaking.
			let base = min(1, pow(baseRaw, 0.55) * 1.35)

			// Turn "energy" into a clean-ish sine that still follows the mic level.
			let wave = sin(t * .pi * 6) * base
			let y = midY + wave * amplitudeMax
			let x = rect.minX + CGFloat(idx) * stepX
			let p = NSPoint(x: x, y: y)
			if idx == 0 { path.move(to: p) } else { path.line(to: p) }
		}

		let stroke = (isActive ? NSColor.white : NSColor.white.withAlphaComponent(0.6)).withAlphaComponent(isSilent ? 0.55 : 0.9)
		stroke.setStroke()
		path.stroke()

		// Subtle center baseline for readability when silent.
		if isSilent {
			let baseline = NSBezierPath()
			baseline.lineWidth = 1
			baseline.move(to: NSPoint(x: rect.minX, y: midY))
			baseline.line(to: NSPoint(x: rect.maxX, y: midY))
			NSColor.white.withAlphaComponent(0.18).setStroke()
			baseline.stroke()
		}
	}

	private func sample(_ values: [CGFloat], t: CGFloat) -> CGFloat {
		guard values.isEmpty == false else { return 0 }
		if values.count == 1 { return values[0] }

		let scaled = t * CGFloat(values.count - 1)
		let i0 = max(0, min(values.count - 1, Int(floor(scaled))))
		let i1 = min(values.count - 1, i0 + 1)
		let frac = scaled - CGFloat(i0)
		return values[i0] * (1 - frac) + values[i1] * frac
	}
}
