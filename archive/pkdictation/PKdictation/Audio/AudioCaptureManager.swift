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
	nonisolated(unsafe) private let writeGroup = DispatchGroup()
	nonisolated(unsafe) private var lastLevelsUpdateNanos: UInt64 = 0
	private let audioProcessingQueue = DispatchQueue(label: "PKdictation.AudioCapture.processing", qos: .userInitiated)
	private var recordingStartedAt: Date?

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
		self.recordingStartedAt = Date()
		self.lastLevelsUpdateNanos = 0
		self.isRecording = true
	}

	/// Stops recording, optionally keeping a small "tail" to avoid truncating the last syllables.
	/// Also waits briefly to let the last tap callback finish writing before the file is consumed.
	func stop(tailPaddingMs: Int = 600, finalizeDelayMs: Int = 250) async throws -> URL {
		guard isRecording else {
			throw AudioCaptureError.notRecording
		}

		if tailPaddingMs > 0 {
			try? await Task.sleep(nanoseconds: UInt64(tailPaddingMs) * 1_000_000)
		}

		engine?.inputNode.removeTap(onBus: 0)

		if finalizeDelayMs > 0 {
			try? await Task.sleep(nanoseconds: UInt64(finalizeDelayMs) * 1_000_000)
		}

		let waitResult = writeGroup.wait(timeout: .now() + .seconds(2))
		if waitResult == .timedOut {
			LogStore.shared.log("Audio finalize: timed out waiting for pending writes (2s).")
		}

		engine?.stop()

		engine = nil
		converter = nil
		audioFile = nil
		isRecording = false

		guard let url = currentRecordingURL else {
			throw AudioCaptureError.missingRecordingURL
		}

		let elapsedMs: Int
		if let startedAt = recordingStartedAt {
			elapsedMs = Int((Date().timeIntervalSince(startedAt) * 1000).rounded())
		} else {
			elapsedMs = -1
		}
		recordingStartedAt = nil

		let sizeBytes: Int64
		if let attrs = try? FileManager.default.attributesOfItem(atPath: url.path),
		   let size = attrs[.size] as? NSNumber
		{
			sizeBytes = size.int64Value
		} else {
			sizeBytes = -1
		}
		LogStore.shared.log("Audio captured: \(elapsedMs)ms, wavSize=\(sizeBytes) bytes, file=\(url.lastPathComponent)")
		if sizeBytes >= 0 && sizeBytes < 2048 {
			LogStore.shared.log("Audio warning: WAV file is very small (\(sizeBytes) bytes). Mic permission or capture may have failed.")
		}

		return url
	}

	nonisolated private func process(buffer: AVAudioPCMBuffer, converter: AVAudioConverter, recordFormat: AVAudioFormat, file: AVAudioFile) {
		// The audio tap callback runs on a real‑time audio thread. Avoid heavy work (conversion + file I/O)
		// here; instead, copy the buffer and process it on a serial queue.
		guard let copied = copyBuffer(buffer) else { return }
		writeGroup.enter()
		audioProcessingQueue.async { [weak self] in
			defer { self?.writeGroup.leave() }
			self?.processCopiedBuffer(copied, converter: converter, recordFormat: recordFormat, file: file)
		}
	}

	nonisolated private func processCopiedBuffer(_ buffer: AVAudioPCMBuffer, converter: AVAudioConverter, recordFormat: AVAudioFormat, file: AVAudioFile) {
		// Avoid flooding the main actor with level updates (audio tap can fire ~40–100x/sec).
		let now = DispatchTime.now().uptimeNanoseconds
		let minInterval: UInt64 = 33_000_000 // ~30 fps
		if now &- lastLevelsUpdateNanos >= minInterval {
			lastLevelsUpdateNanos = now
			let newLevels = AudioLevelMeter.levels(from: buffer, barCount: 16, gain: 8)
			Task { @MainActor [weak self] in
				self?.levels = newLevels.map { Double($0) }
			}
		}

		let inputFrameCount = AVAudioFrameCount(buffer.frameLength)
		let sampleRateRatio = recordFormat.sampleRate / buffer.format.sampleRate
		let outputFrameCapacity = AVAudioFrameCount(Double(inputFrameCount) * sampleRateRatio) + 16

		guard let outputBuffer = AVAudioPCMBuffer(pcmFormat: recordFormat, frameCapacity: outputFrameCapacity) else { return }

		var conversionError: NSError?
		var didProvideInput = false
		let status = converter.convert(to: outputBuffer, error: &conversionError) { _, outStatus in
			if didProvideInput {
				outStatus.pointee = .endOfStream
				return nil
			}
			didProvideInput = true
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

	nonisolated private func copyBuffer(_ buffer: AVAudioPCMBuffer) -> AVAudioPCMBuffer? {
		let format = buffer.format
		let frames = buffer.frameLength
		guard let copy = AVAudioPCMBuffer(pcmFormat: format, frameCapacity: frames) else { return nil }
		copy.frameLength = frames

		let channels = Int(format.channelCount)

		if let src = buffer.floatChannelData, let dst = copy.floatChannelData {
			let bytesPerChannel = Int(frames) * MemoryLayout<Float>.size
			for ch in 0..<channels {
				memcpy(dst[ch], src[ch], bytesPerChannel)
			}
			return copy
		}

		if let src = buffer.int16ChannelData, let dst = copy.int16ChannelData {
			let bytesPerChannel = Int(frames) * MemoryLayout<Int16>.size
			for ch in 0..<channels {
				memcpy(dst[ch], src[ch], bytesPerChannel)
			}
			return copy
		}

		if let src = buffer.int32ChannelData, let dst = copy.int32ChannelData {
			let bytesPerChannel = Int(frames) * MemoryLayout<Int32>.size
			for ch in 0..<channels {
				memcpy(dst[ch], src[ch], bytesPerChannel)
			}
			return copy
		}

		return nil
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
