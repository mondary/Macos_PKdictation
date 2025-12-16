import AppKit

final class HotkeyRecorderRow: NSStackView {
	init(hotkeyLabel: NSTextField, recordButton: NSButton, useFnButton: NSButton, useRightCommandButton: NSButton) {
		super.init(frame: .zero)

		orientation = .horizontal
		alignment = .centerY
		spacing = 12

		hotkeyLabel.font = NSFont.monospacedSystemFont(ofSize: 13, weight: .regular)
		hotkeyLabel.textColor = .labelColor

		hotkeyLabel.translatesAutoresizingMaskIntoConstraints = false
		NSLayoutConstraint.activate([
			hotkeyLabel.widthAnchor.constraint(greaterThanOrEqualToConstant: 140),
		])

		addArrangedSubview(hotkeyLabel)
		addArrangedSubview(recordButton)
		addArrangedSubview(useFnButton)
		addArrangedSubview(useRightCommandButton)

		let spacer = NSView()
		spacer.translatesAutoresizingMaskIntoConstraints = false
		spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
		spacer.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
		addArrangedSubview(spacer)

		toolTip = "Configure the push-to-talk shortcut. Default is Fn."
	}

	@available(*, unavailable)
	required init(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}
}
