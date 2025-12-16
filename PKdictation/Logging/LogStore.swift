import Foundation

@MainActor
final class LogStore: ObservableObject {
	static let shared = LogStore()

	@Published private(set) var lines: [String] = []

	private let maxLines = 300

	func log(_ message: String) {
		let ts = Self.timestamp()
		let line = "[\(ts)] \(message)"
		lines.append(line)
		if lines.count > maxLines {
			lines.removeFirst(lines.count - maxLines)
		}
	}

	func dump() -> String {
		lines.joined(separator: "\n")
	}

	private static func timestamp() -> String {
		let f = DateFormatter()
		f.locale = Locale(identifier: "en_US_POSIX")
		f.dateFormat = "HH:mm:ss"
		return f.string(from: Date())
	}
}

