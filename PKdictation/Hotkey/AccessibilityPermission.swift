import AppKit
import ApplicationServices

enum AccessibilityPermission {
	static func isTrusted() -> Bool {
		AXIsProcessTrusted()
	}

	static func requestPrompt() {
		let options = [kAXTrustedCheckOptionPrompt.takeRetainedValue() as String: true] as CFDictionary
		AXIsProcessTrustedWithOptions(options)
	}

	static func openSettings() {
		guard let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility") else { return }
		NSWorkspace.shared.open(url)
	}
}

