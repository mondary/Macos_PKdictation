import AppKit
import Combine

@MainActor
final class OverlayWindowController: NSWindowController, OverlayPresenting {
	private let panel: NSPanel
	private let content: OverlayContentView
	private var cancellables: Set<AnyCancellable> = []
	private let topOverhang: CGFloat = 14

	init(controller: DictationController) {
		let contentView = OverlayContentView()

		let panel = NSPanel(
			contentRect: NSRect(x: 0, y: 0, width: 360, height: 44 + topOverhang),
			styleMask: [.borderless, .nonactivatingPanel],
			backing: .buffered,
			defer: false
		)
		panel.isFloatingPanel = true
		panel.level = .statusBar
		panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
		panel.isOpaque = false
		panel.backgroundColor = .clear
		panel.hasShadow = true
		panel.ignoresMouseEvents = true
		panel.contentView = contentView

		self.panel = panel
		self.content = contentView
		super.init(window: panel)

		bind(controller: controller)
		content.update(phase: controller.phase, statusText: controller.statusText, transcript: controller.transcript)
		content.update(levels: controller.audio.levels)
	}

	@available(*, unavailable)
	required init?(coder: NSCoder) {
		fatalError("init(coder:) has not been implemented")
	}

	func show() {
		guard panel.isVisible == false else { return }
		reposition()
		panel.orderFrontRegardless()
	}

	func hide() {
		panel.orderOut(nil)
	}

	private func reposition() {
		guard let screen = NSScreen.main else { return }
		let frame = screen.frame
		let size = panel.frame.size
		let x = frame.midX - size.width / 2
		// Push the top portion slightly offscreen so the shape feels like it "comes out"
		// from outside the screen into the content area (notch effect).
		let y = frame.maxY - size.height + topOverhang
		panel.setFrameOrigin(NSPoint(x: x, y: y))
	}

	private func bind(controller: DictationController) {
		controller.$phase
			.receive(on: RunLoop.main)
			.sink { [weak self, weak controller] phase in
				guard let self, let controller else { return }
				self.content.update(phase: phase, statusText: controller.statusText, transcript: controller.transcript)
			}
			.store(in: &cancellables)

		controller.$transcript
			.receive(on: RunLoop.main)
			.sink { [weak self, weak controller] transcript in
				guard let self, let controller else { return }
				self.content.update(phase: controller.phase, statusText: controller.statusText, transcript: transcript)
			}
			.store(in: &cancellables)

		controller.audio.$levels
			.receive(on: RunLoop.main)
			.sink { [weak self] levels in
				self?.content.update(levels: levels)
			}
			.store(in: &cancellables)
	}
}
