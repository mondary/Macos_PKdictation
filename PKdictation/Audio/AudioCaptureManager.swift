import AVFoundation
import Foundation

@MainActor
final class AudioCaptureManager: ObservableObject {
	@Published private(set) var isRecording = false
	@Published private(set) var levels: [Double] = Array(repeating: 0, count: 16)

	private var engine: AVAudioEngine?
	private var audioFile: AVAudioFile?
	private var converter: AVAudioConverter?
	private var currentRecordingURL: URL?

	func start() async throws {
		guard !isRecording else { return }

		let granted = await requestMicrophonePermission()
		guard granted else {
			throw AudioCaptureError.microphonePermissionDenied
		}

		let engine = AVAudioEngine()
		let inputNode = engine.inputNode
		let inputFormat = inputNode.outputFormat(forBus: 0)

		guard let recordFormat = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: 16_000, channels: 1, interleaved: true) else {
			throw AudioCaptureError.invalidAudioFormat
		}

		let converter = AVAudioConverter(from: inputFormat, to: recordFormat)
		guard let converter else {
			throw AudioCaptureError.cannotCreateConverter
		}

		let url = try makeNewRecordingURL()
		let file = try AVAudioFile(forWriting: url, settings: recordFormat.settings, commonFormat: recordFormat.commonFormat, interleaved: recordFormat.isInterleaved)

		inputNode.installTap(onBus: 0, bufferSize: 1024, format: inputFormat) { [weak self] buffer, _ in
			guard let self else { return }
			self.process(buffer: buffer, converter: converter, recordFormat: recordFormat, file: file)
		}

		engine.prepare()
		try engine.start()

		self.engine = engine
		self.audioFile = file
		self.converter = converter
		self.currentRecordingURL = url
		self.isRecording = true
	}

	func stop() throws -> URL {
		guard isRecording else {
			throw AudioCaptureError.notRecording
		}

		engine?.inputNode.removeTap(onBus: 0)
		engine?.stop()

		engine = nil
		converter = nil
		audioFile = nil
		isRecording = false

		guard let url = currentRecordingURL else {
			throw AudioCaptureError.missingRecordingURL
		}
		return url
	}

	private func process(buffer: AVAudioPCMBuffer, converter: AVAudioConverter, recordFormat: AVAudioFormat, file: AVAudioFile) {
		let newLevels = AudioLevelMeter.levels(from: buffer, barCount: 16, gain: 8)
		Task { @MainActor [weak self] in
			self?.levels = newLevels.map { Double($0) }
		}

		let inputFrameCount = AVAudioFrameCount(buffer.frameLength)
		let sampleRateRatio = recordFormat.sampleRate / buffer.format.sampleRate
		let outputFrameCapacity = AVAudioFrameCount(Double(inputFrameCount) * sampleRateRatio) + 16

		guard let outputBuffer = AVAudioPCMBuffer(pcmFormat: recordFormat, frameCapacity: outputFrameCapacity) else { return }

		var conversionError: NSError?
		let status = converter.convert(to: outputBuffer, error: &conversionError) { _, outStatus in
			outStatus.pointee = .haveData
			return buffer
		}

		if status == .error || conversionError != nil {
			return
		}

		do {
			try file.write(from: outputBuffer)
		} catch {
			return
		}
	}

	private func makeNewRecordingURL() throws -> URL {
		let dir = URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true)
			.appendingPathComponent("PKdictation", isDirectory: true)
		try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)

		let formatter = ISO8601DateFormatter()
		formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
		let fileName = "recording-\(formatter.string(from: Date())).wav"
		return dir.appendingPathComponent(fileName)
	}

	private func requestMicrophonePermission() async -> Bool {
		switch AVCaptureDevice.authorizationStatus(for: .audio) {
		case .authorized:
			return true
		case .denied, .restricted:
			return false
		case .notDetermined:
			return await withCheckedContinuation { continuation in
				AVCaptureDevice.requestAccess(for: .audio) { granted in
					continuation.resume(returning: granted)
				}
			}
		@unknown default:
			return false
		}
	}
}

enum AudioCaptureError: LocalizedError {
	case microphonePermissionDenied
	case invalidAudioFormat
	case cannotCreateConverter
	case notRecording
	case missingRecordingURL

	var errorDescription: String? {
		switch self {
		case .microphonePermissionDenied:
			return "Microphone permission denied."
		case .invalidAudioFormat:
			return "Invalid audio format."
		case .cannotCreateConverter:
			return "Cannot create audio converter."
		case .notRecording:
			return "Not recording."
		case .missingRecordingURL:
			return "Missing recording file."
		}
	}
}
