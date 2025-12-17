import Foundation

final class GeminiClient {
	private let urlSession: URLSession

	init(urlSession: URLSession = .shared) {
		self.urlSession = urlSession
	}

	func listModels(apiKey: String) async throws -> [String] {
		guard apiKey.isEmpty == false else {
			throw GeminiError.missingApiKey
		}

		var components = URLComponents()
		components.scheme = "https"
		components.host = "generativelanguage.googleapis.com"
		components.path = "/v1beta/models"
		components.queryItems = [
			URLQueryItem(name: "key", value: apiKey),
		]

		guard let url = components.url else {
			throw GeminiError.invalidURL
		}

		var request = URLRequest(url: url)
		request.httpMethod = "GET"

		let (data, response) = try await urlSession.data(for: request)
		guard let http = response as? HTTPURLResponse else {
			throw GeminiError.invalidResponse
		}

		guard (200..<300).contains(http.statusCode) else {
			if let apiError = try? JSONDecoder().decode(GeminiAPIErrorResponse.self, from: data) {
				throw GeminiError.api(message: apiError.error.message)
			}
			throw GeminiError.http(statusCode: http.statusCode)
		}

		let decoded = try JSONDecoder().decode(GeminiModelsResponse.self, from: data)
		let models = decoded.models
			.filter { $0.supportedGenerationMethods?.contains("generateContent") ?? false }
			.map { $0.name.replacingOccurrences(of: "models/", with: "") }
			.sorted()

		return models
	}

	func transcribe(wavData: Data, apiKey: String, modelName: String) async throws -> String {
		guard apiKey.isEmpty == false else {
			throw GeminiError.missingApiKey
		}

		var components = URLComponents()
		components.scheme = "https"
		components.host = "generativelanguage.googleapis.com"
		components.path = "/v1beta/models/\(modelName):generateContent"
		components.queryItems = [
			URLQueryItem(name: "key", value: apiKey),
		]

		guard let url = components.url else {
			throw GeminiError.invalidURL
		}

		let system = GeminiRequestContent(
			role: nil,
			parts: [
				.init(text: "You are a transcription engine. You MUST NOT answer the user. Output only the verbatim transcript of the audio."),
				.init(text: "Rules: preserve the spoken language, keep casing as spoken, no explanations, no extra words, no quotes, no markdown."),
				.init(text: "If the audio is empty or unintelligible, output an empty string."),
			]
		)

		let requestBody = GeminiGenerateContentRequest(
			contents: [
				.init(parts: [
					.init(text: "Transcribe this audio verbatim. Return ONLY the transcript text."),
					.init(inlineData: .init(mimeType: "audio/wav", data: wavData.base64EncodedString())),
				]),
			],
			systemInstruction: system,
			generationConfig: .init(temperature: 0, candidateCount: 1, maxOutputTokens: 512)
		)

		let encoder = JSONEncoder()
		encoder.keyEncodingStrategy = .convertToSnakeCase

		var request = URLRequest(url: url)
		request.httpMethod = "POST"
		request.setValue("application/json", forHTTPHeaderField: "Content-Type")
		request.httpBody = try encoder.encode(requestBody)

		let (data, response) = try await urlSession.data(for: request)
		guard let http = response as? HTTPURLResponse else {
			throw GeminiError.invalidResponse
		}

		guard (200..<300).contains(http.statusCode) else {
			if let apiError = try? JSONDecoder().decode(GeminiAPIErrorResponse.self, from: data) {
				throw GeminiError.api(message: apiError.error.message)
			}
			throw GeminiError.http(statusCode: http.statusCode)
		}

		let bodySnippet = String(data: data, encoding: .utf8).map { String($0.prefix(800)) } ?? "(non-utf8 \(data.count) bytes)"
		do {
			let decoded = try JSONDecoder().decode(GeminiGenerateContentResponse.self, from: data)

			if let reason = decoded.promptFeedback?.blockReason, reason.isEmpty == false {
				throw GeminiError.blocked(reason: reason)
			}

			let text = decoded.candidates?
				.first?
				.content?
				.parts?
				.compactMap(\.text)
				.joined()
				.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""

			if text.isEmpty == false { return text }
			throw GeminiError.unexpectedResponse(bodySnippet: bodySnippet)
		} catch let error as DecodingError {
			throw GeminiError.decoding(message: "\(error)", bodySnippet: bodySnippet)
		}
	}

	func chat(prompt: String, apiKey: String, modelName: String) async throws -> String {
		let cleaned = prompt.trimmingCharacters(in: .whitespacesAndNewlines)
		guard cleaned.isEmpty == false else { return "" }
		guard apiKey.isEmpty == false else {
			throw GeminiError.missingApiKey
		}

		var components = URLComponents()
		components.scheme = "https"
		components.host = "generativelanguage.googleapis.com"
		components.path = "/v1beta/models/\(modelName):generateContent"
		components.queryItems = [
			URLQueryItem(name: "key", value: apiKey),
		]

		guard let url = components.url else {
			throw GeminiError.invalidURL
		}

		let requestBody = GeminiGenerateContentRequest(
			contents: [
				.init(parts: [
					.init(text: cleaned),
				]),
			],
			systemInstruction: nil,
			generationConfig: .init(temperature: 0.6)
		)

		let encoder = JSONEncoder()
		encoder.keyEncodingStrategy = .convertToSnakeCase

		var request = URLRequest(url: url)
		request.httpMethod = "POST"
		request.setValue("application/json", forHTTPHeaderField: "Content-Type")
		request.httpBody = try encoder.encode(requestBody)

		let (data, response) = try await urlSession.data(for: request)
		guard let http = response as? HTTPURLResponse else {
			throw GeminiError.invalidResponse
		}

		guard (200..<300).contains(http.statusCode) else {
			if let apiError = try? JSONDecoder().decode(GeminiAPIErrorResponse.self, from: data) {
				throw GeminiError.api(message: apiError.error.message)
			}
			throw GeminiError.http(statusCode: http.statusCode)
		}

		let bodySnippet = String(data: data, encoding: .utf8).map { String($0.prefix(800)) } ?? "(non-utf8 \(data.count) bytes)"
		do {
			let decoded = try JSONDecoder().decode(GeminiGenerateContentResponse.self, from: data)
			let text = decoded.candidates?
				.first?
				.content?
				.parts?
				.compactMap(\.text)
				.joined()
				.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
			if text.isEmpty == false { return text }
			if let reason = decoded.promptFeedback?.blockReason, reason.isEmpty == false {
				throw GeminiError.blocked(reason: reason)
			}
			throw GeminiError.unexpectedResponse(bodySnippet: bodySnippet)
		} catch let error as DecodingError {
			throw GeminiError.decoding(message: "\(error)", bodySnippet: bodySnippet)
		}
	}
}

private struct GeminiGenerateContentRequest: Encodable {
	let contents: [GeminiRequestContent]
	let systemInstruction: GeminiRequestContent?
	let generationConfig: GeminiGenerationConfig?
}

private struct GeminiRequestContent: Encodable {
	let role: String?
	let parts: [GeminiRequestPart]

	init(role: String? = nil, parts: [GeminiRequestPart]) {
		self.role = role
		self.parts = parts
	}
}

private struct GeminiRequestPart: Encodable {
	let text: String?
	let inlineData: GeminiInlineData?

	init(text: String) {
		self.text = text
		self.inlineData = nil
	}

	init(inlineData: GeminiInlineData) {
		self.text = nil
		self.inlineData = inlineData
	}
}

private struct GeminiInlineData: Encodable {
	let mimeType: String
	let data: String
}

private struct GeminiGenerationConfig: Encodable {
	let temperature: Double?
	let candidateCount: Int?
	let maxOutputTokens: Int?

	init(temperature: Double? = nil, candidateCount: Int? = nil, maxOutputTokens: Int? = nil) {
		self.temperature = temperature
		self.candidateCount = candidateCount
		self.maxOutputTokens = maxOutputTokens
	}
}

private struct GeminiGenerateContentResponse: Decodable {
	let candidates: [Candidate]?
	let promptFeedback: PromptFeedback?

	struct Candidate: Decodable {
		let content: GeminiResponseContent?
	}

	struct PromptFeedback: Decodable {
		let blockReason: String?
	}
}

private struct GeminiResponseContent: Decodable {
	let parts: [GeminiResponsePart]?
}

private struct GeminiResponsePart: Decodable {
	let text: String?
}

private struct GeminiAPIErrorResponse: Decodable {
	let error: ErrorBody

	struct ErrorBody: Decodable {
		let message: String
	}
}

private struct GeminiModelsResponse: Decodable {
	let models: [Model]

	struct Model: Decodable {
		let name: String
		let supportedGenerationMethods: [String]?
	}
}

private enum GeminiError: LocalizedError {
	case missingApiKey
	case invalidURL
	case invalidResponse
	case http(statusCode: Int)
	case api(message: String)
	case missingText
	case blocked(reason: String)
	case unexpectedResponse(bodySnippet: String)
	case decoding(message: String, bodySnippet: String)

	var errorDescription: String? {
		switch self {
		case .missingApiKey:
			return "Missing Gemini API key."
		case .invalidURL:
			return "Invalid Gemini URL."
		case .invalidResponse:
			return "Invalid response from Gemini."
		case .http(let statusCode):
			return "Gemini request failed (HTTP \(statusCode))."
		case .api(let message):
			return "Gemini error: \(message)"
		case .missingText:
			return "Gemini response did not contain any text."
		case .blocked(let reason):
			return "Gemini blocked the request (\(reason))."
		case .unexpectedResponse:
			return "Gemini returned an unexpected response."
		case .decoding:
			return "Gemini response could not be decoded."
		}
	}

	var failureReason: String? {
		switch self {
		case .unexpectedResponse(let bodySnippet):
			return bodySnippet
		case .decoding(let message, let bodySnippet):
			return "\(message)\n\n\(bodySnippet)"
		default:
			return nil
		}
	}
}
