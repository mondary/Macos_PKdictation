import Accelerate
import AVFoundation

enum AudioLevelMeter {
	static func levels(from buffer: AVAudioPCMBuffer, barCount: Int, gain: Float) -> [Float] {
		guard barCount > 0,
			  let channelData = buffer.floatChannelData?[0]
		else {
			return Array(repeating: 0, count: max(0, barCount))
		}

		let frameCount = Int(buffer.frameLength)
		guard frameCount > 0 else {
			return Array(repeating: 0, count: barCount)
		}

		let samplesPerBar = max(1, frameCount / barCount)
		var result = [Float](repeating: 0, count: barCount)

		for index in 0..<barCount {
			let start = index * samplesPerBar
			let end = min(frameCount, start + samplesPerBar)
			if start >= end { break }

			var meanMagnitude: Float = 0
			vDSP_meamgv(channelData.advanced(by: start), 1, &meanMagnitude, vDSP_Length(end - start))

			let scaled = min(1, meanMagnitude * gain)
			result[index] = scaled
		}

		return result
	}
}

