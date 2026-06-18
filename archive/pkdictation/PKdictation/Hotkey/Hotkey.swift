import AppKit
import Carbon.HIToolbox

struct Hotkey: Equatable {
	var keyCode: UInt16?
	var modifiers: NSEvent.ModifierFlags

	static let relevantModifiers: NSEvent.ModifierFlags = [.control, .option, .shift, .command, .function]
	static let defaultPushToTalk = Hotkey(keyCode: nil, modifiers: [.function])
	static let rightCommandPushToTalk = Hotkey(keyCode: UInt16(kVK_RightCommand), modifiers: [.command])

	static func modifierFlag(forKeyCode keyCode: UInt16) -> NSEvent.ModifierFlags? {
		switch keyCode {
		case UInt16(kVK_Control), UInt16(kVK_RightControl):
			return .control
		case UInt16(kVK_Option), UInt16(kVK_RightOption):
			return .option
		case UInt16(kVK_Shift), UInt16(kVK_RightShift):
			return .shift
		case UInt16(kVK_Command), UInt16(kVK_RightCommand):
			return .command
		case UInt16(kVK_Function):
			return .function
		default:
			return nil
		}
	}

	var displayString: String {
		var mods = modifiers.intersection(Self.relevantModifiers)
		var parts: [String] = []

		if let keyCode, let modForKeyCode = Self.modifierFlag(forKeyCode: keyCode) {
			mods.remove(modForKeyCode)
		}

		if mods.contains(.control) { parts.append("⌃") }
		if mods.contains(.option) { parts.append("⌥") }
		if mods.contains(.shift) { parts.append("⇧") }
		if mods.contains(.command) { parts.append("⌘") }
		if mods.contains(.function) { parts.append("Fn") }

		if let keyCode {
			parts.append(KeyCodeTranslator.displayString(for: keyCode))
		}

		return parts.isEmpty ? "None" : parts.joined()
	}
}

private enum KeyCodeTranslator {
	static func displayString(for keyCode: UInt16) -> String {
		if let special = specialKeyStrings[keyCode] {
			return special
		}

		guard
			let inputSource = TISCopyCurrentKeyboardLayoutInputSource()?.takeRetainedValue(),
			let rawLayoutData = TISGetInputSourceProperty(inputSource, kTISPropertyUnicodeKeyLayoutData)
		else {
			return "KeyCode \(keyCode)"
		}

		let layoutData = unsafeBitCast(rawLayoutData, to: CFData.self)
		guard let ptr = CFDataGetBytePtr(layoutData) else {
			return "KeyCode \(keyCode)"
		}

		let keyboardLayout = UnsafeRawPointer(ptr).assumingMemoryBound(to: UCKeyboardLayout.self)
		var deadKeyState: UInt32 = 0
		var length: Int = 0
		var chars = [UniChar](repeating: 0, count: 8)

		let result = UCKeyTranslate(
			keyboardLayout,
			keyCode,
			UInt16(kUCKeyActionDisplay),
			0,
			UInt32(LMGetKbdType()),
			0,
			&deadKeyState,
			chars.count,
			&length,
			&chars
		)

		guard result == noErr, length > 0 else {
			return "KeyCode \(keyCode)"
		}

		return String(utf16CodeUnits: chars, count: length).uppercased()
	}

	private static let specialKeyStrings: [UInt16: String] = [
		UInt16(kVK_Command): "Left⌘",
		UInt16(kVK_RightCommand): "Right⌘",
		UInt16(kVK_Shift): "Left⇧",
		UInt16(kVK_RightShift): "Right⇧",
		UInt16(kVK_Option): "Left⌥",
		UInt16(kVK_RightOption): "Right⌥",
		UInt16(kVK_Control): "Left⌃",
		UInt16(kVK_RightControl): "Right⌃",
		UInt16(kVK_Function): "Fn",
		36: "↩︎", // Return
		48: "⇥", // Tab
		49: "Space",
		51: "⌫", // Delete
		53: "⎋", // Escape
		117: "⌦", // Forward Delete
		123: "←",
		124: "→",
		125: "↓",
		126: "↑",
	]
}
