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

		let requestBody = GeminiGenerateContentRequest(
			contents: [
				.init(parts: [
					.init(text: "Transcribe this audio. Return only the transcript, without quotes."),
					.init(inlineData: .init(mimeType: "audio/wav", data: wavData.base64EncodedString())),
				]),
			],
			generationConfig: .init(temperature: 0)
		)

		var request = URLRequest(url: url)
		request.httpMethod = "POST"
		request.setValue("application/json", forHTTPHeaderField: "Content-Type")
		request.httpBody = try JSONEncoder().encode(requestBody)

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

		let decoded = try JSONDecoder().decode(GeminiGenerateContentResponse.self, from: data)
		guard let text = decoded.candidates.first?.content.parts.compactMap(\.text).first else {
			throw GeminiError.missingText
		}
		return text
	}
}

private struct GeminiGenerateContentRequest: Encodable {
	let contents: [GeminiRequestContent]
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
}

private struct GeminiGenerateContentResponse: Decodable {
	let candidates: [Candidate]

	struct Candidate: Decodable {
		let content: GeminiResponseContent
	}
}

private struct GeminiResponseContent: Decodable {
	let parts: [GeminiResponsePart]
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
		}
	}
}
