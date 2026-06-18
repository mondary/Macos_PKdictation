import AppKit
import AVFoundation

enum MicrophonePermission {
	static func status() -> AVAuthorizationStatus {
		AVCaptureDevice.authorizationStatus(for: .audio)
	}

	static func isAuthorized() -> Bool {
		status() == .authorized
	}

	static func request(_ completion: @escaping (Bool) -> Void) {
		AVCaptureDevice.requestAccess(for: .audio, completionHandler: completion)
	}

	static func openSettings() {
		guard let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Microphone") else { return }
		NSWorkspace.shared.open(url)
	}
}

