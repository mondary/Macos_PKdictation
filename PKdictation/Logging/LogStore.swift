import Foundation
import os

@MainActor
final class LogStore: ObservableObject {
	static let shared = LogStore()

	@Published private(set) var lines: [String] = []

	private let maxLines = 300
	private let logger = Logger(subsystem: Bundle.main.bundleIdentifier ?? "PKdictation", category: "app")
	private let fileQueue = DispatchQueue(label: "PKdictation.LogStore.file")

	func log(_ message: String) {
		let ts = Self.timestamp()
		let line = "[\(ts)] \(message)"
		lines.append(line)
		if lines.count > maxLines {
			lines.removeFirst(lines.count - maxLines)
		}

		logger.info("\(line, privacy: .public)")
		appendToFile(line)
	}

	func dump() -> String {
		lines.joined(separator: "\n")
	}

	func logFileURL() -> URL? {
		Self.makeLogFileURL()
	}

	private func appendToFile(_ line: String) {
		fileQueue.async {
			guard let url = Self.makeLogFileURL() else { return }
			let payload = (line + "\n").data(using: .utf8) ?? Data()
			do {
				let dir = url.deletingLastPathComponent()
				try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)

				if FileManager.default.fileExists(atPath: url.path) == false {
					try payload.write(to: url, options: [.atomic])
					return
				}

				let handle = try FileHandle(forWritingTo: url)
				try handle.seekToEnd()
				try handle.write(contentsOf: payload)
				try handle.close()
			} catch {
				// Best-effort: keep in-memory logs even if file writes fail.
			}
		}
	}

	private static func makeLogFileURL() -> URL? {
		let fm = FileManager.default
		guard let appSupport = fm.urls(for: .applicationSupportDirectory, in: .userDomainMask).first else { return nil }
		let dir = appSupport.appendingPathComponent("PKdictation/Logs", isDirectory: true)
		return dir.appendingPathComponent("PKdictation.log", isDirectory: false)
	}

	private static func timestamp() -> String {
		timestampFormatter.string(from: Date())
	}

	private static let timestampFormatter: DateFormatter = {
		let f = DateFormatter()
		f.locale = Locale(identifier: "en_US_POSIX")
		f.dateFormat = "HH:mm:ss"
		return f
	}()
}
