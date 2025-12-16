import AppKit

@main
enum PKdictationMain {
	@MainActor
	static func main() {
		autoreleasepool {
			let app = NSApplication.shared
			let delegate = AppDelegate()
			app.delegate = delegate
			app.run()
		}
	}
}
