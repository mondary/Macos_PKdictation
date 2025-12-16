import Foundation
import Security

enum Keychain {
	static func saveString(_ value: String, service: String, account: String) throws {
		let data = Data(value.utf8)
		let query: [CFString: Any] = [
			kSecClass: kSecClassGenericPassword,
			kSecAttrService: service,
			kSecAttrAccount: account,
		]

		let attributes: [CFString: Any] = [
			kSecValueData: data,
		]

		let status = SecItemCopyMatching(query as CFDictionary, nil)
		if status == errSecSuccess {
			let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
			guard updateStatus == errSecSuccess else { throw KeychainError(status: updateStatus) }
			return
		}

		if status == errSecItemNotFound {
			var addQuery = query
			addQuery[kSecValueData] = data
			let addStatus = SecItemAdd(addQuery as CFDictionary, nil)
			guard addStatus == errSecSuccess else { throw KeychainError(status: addStatus) }
			return
		}

		throw KeychainError(status: status)
	}

	static func readString(service: String, account: String) throws -> String? {
		let query: [CFString: Any] = [
			kSecClass: kSecClassGenericPassword,
			kSecAttrService: service,
			kSecAttrAccount: account,
			kSecReturnData: true,
			kSecMatchLimit: kSecMatchLimitOne,
		]

		var result: CFTypeRef?
		let status = SecItemCopyMatching(query as CFDictionary, &result)

		if status == errSecItemNotFound { return nil }
		guard status == errSecSuccess else { throw KeychainError(status: status) }

		guard let data = result as? Data else { return nil }
		return String(data: data, encoding: .utf8)
	}

	static func deleteItem(service: String, account: String) throws {
		let query: [CFString: Any] = [
			kSecClass: kSecClassGenericPassword,
			kSecAttrService: service,
			kSecAttrAccount: account,
		]

		let status = SecItemDelete(query as CFDictionary)
		if status == errSecItemNotFound { return }
		guard status == errSecSuccess else { throw KeychainError(status: status) }
	}
}

struct KeychainError: LocalizedError {
	let status: OSStatus

	var errorDescription: String? {
		let message = SecCopyErrorMessageString(status, nil) as String?
		return message ?? "Keychain error (\(status))."
	}
}

