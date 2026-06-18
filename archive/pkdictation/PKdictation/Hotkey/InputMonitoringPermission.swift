import AppKit
import ApplicationServices

enum InputMonitoringPermission {
	static func isTrusted() -> Bool {
		CGPreflightListenEventAccess()
	}

	static func requestPrompt() {
		_ = CGRequestListenEventAccess()
	}

	static func openSettings() {
		guard let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_ListenEvent") else { return }
		NSWorkspace.shared.open(url)
	}
}

