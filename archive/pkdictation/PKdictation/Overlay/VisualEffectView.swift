import AppKit

final class RoundedVisualEffectView: NSVisualEffectView {
	init(
		material: NSVisualEffectView.Material,
		blendingMode: NSVisualEffectView.BlendingMode,
		cornerRadius: CGFloat,
		borderColor: NSColor,
		borderWidth: CGFloat
	) {
		super.init(frame: .zero)

		self.material = material
		self.blendingMode = blendingMode
		state = .active

		wantsLayer = true
		layer?.cornerRadius = cornerRadius
		layer?.masksToBounds = true
		layer?.borderColor = borderColor.cgColor
		layer?.borderWidth = borderWidth
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}
}
