import Foundation

@MainActor
protocol OverlayPresenting: AnyObject {
	func show()
	func hide()
}

