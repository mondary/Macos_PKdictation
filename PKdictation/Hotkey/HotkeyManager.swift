import AppKit
import ApplicationServices
import Carbon.HIToolbox
import Foundation

final class HotkeyManager: ObservableObject {
	@Published private(set) var isRunning: Bool = false
	@Published private(set) var lastErrorMessage: String?

	var onPress: (() -> Void)?
	var onRelease: (() -> Void)?
	var onDoublePress: (() -> Void)?

	private let lock = NSLock()
	private var hotkey: Hotkey
	private var consumeEvents: Bool
	private var isPressed: Bool = false
	private var lastPressTime: TimeInterval = 0
	private var inputMonitoringTrustedCached: Bool = false
	private var lastInputMonitoringTrustedCheck: TimeInterval = 0
	private let inputMonitoringTrustedCheckInterval: TimeInterval = 2.0

	private var eventTap: CFMachPort?
	private var runLoopSource: CFRunLoopSource?

	init(hotkey: Hotkey, consumeEvents: Bool) {
		self.hotkey = hotkey
		self.consumeEvents = consumeEvents
	}

	func start() {
		lock.lock()
		defer { lock.unlock() }

		guard eventTap == nil else { return }

		lastErrorMessage = nil

		let mask = CGEventMask(
			(1 << CGEventType.keyDown.rawValue)
				| (1 << CGEventType.keyUp.rawValue)
				| (1 << CGEventType.flagsChanged.rawValue)
				| (1 << CGEventType.tapDisabledByTimeout.rawValue)
				| (1 << CGEventType.tapDisabledByUserInput.rawValue)
		)

		let callback: CGEventTapCallBack = { proxy, type, event, userInfo in
			guard let userInfo else { return Unmanaged.passRetained(event) }
			let manager = Unmanaged<HotkeyManager>.fromOpaque(userInfo).takeUnretainedValue()
			return manager.handle(proxy: proxy, type: type, event: event)
		}

		let userInfo = UnsafeMutableRawPointer(Unmanaged.passUnretained(self).toOpaque())
		guard let tap = CGEvent.tapCreate(
			tap: .cgSessionEventTap,
			place: .headInsertEventTap,
			options: .defaultTap,
			eventsOfInterest: mask,
			callback: callback,
			userInfo: userInfo
		) else {
			isRunning = false
			let axTrusted = AccessibilityPermission.isTrusted()
			let axStatus = axTrusted ? "granted" : "not granted"
			let imTrusted = InputMonitoringPermission.isTrusted()
			let imStatus = imTrusted ? "granted" : "not granted"
			lastErrorMessage =
				"Cannot start hotkey listener.\nAccessibility: \(axStatus). Input Monitoring: \(imStatus).\nTry: enable both, then click “Retry hotkey” (or relaunch)."
			Task { @MainActor in
				LogStore.shared.log("Hotkey listener failed to start (AX=\(axStatus), IM=\(imStatus)).")
			}
			Task { @MainActor in
				if axTrusted == false { AccessibilityPermission.requestPrompt() }
				if imTrusted == false { InputMonitoringPermission.requestPrompt() }
			}
			return
		}

		let source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0)
		CFRunLoopAddSource(CFRunLoopGetMain(), source, .commonModes)
		CGEvent.tapEnable(tap: tap, enable: true)

		inputMonitoringTrustedCached = InputMonitoringPermission.isTrusted()
		lastInputMonitoringTrustedCheck = CFAbsoluteTimeGetCurrent()

		eventTap = tap
		runLoopSource = source
		isRunning = true
		Task { @MainActor in
			LogStore.shared.log("Hotkey listener started.")
		}
	}

	func stop() {
		lock.lock()
		defer { lock.unlock() }

		guard let tap = eventTap, let source = runLoopSource else { return }
		CGEvent.tapEnable(tap: tap, enable: false)
		CFRunLoopRemoveSource(CFRunLoopGetMain(), source, .commonModes)
		eventTap = nil
		runLoopSource = nil
		isRunning = false
		Task { @MainActor in
			LogStore.shared.log("Hotkey listener stopped.")
		}
	}

	func update(hotkey: Hotkey) {
		lock.lock()
		defer { lock.unlock() }
		self.hotkey = hotkey
		self.isPressed = false
	}

	func update(consumeEvents: Bool) {
		lock.lock()
		defer { lock.unlock() }
		self.consumeEvents = consumeEvents
	}

	private func handle(proxy: CGEventTapProxy, type: CGEventType, event: CGEvent) -> Unmanaged<CGEvent>? {
		if type == .tapDisabledByTimeout || type == .tapDisabledByUserInput {
			lock.lock()
			let tap = eventTap
			lock.unlock()
			if let tap {
				CGEvent.tapEnable(tap: tap, enable: true)
			}
			return Unmanaged.passRetained(event)
		}

		let keyCode = UInt16(event.getIntegerValueField(.keyboardEventKeycode))
		let currentMods = Self.modifierFlags(from: event.flags)
		let now = CFAbsoluteTimeGetCurrent()

		if type == .flagsChanged {
			lock.lock()
			let hk = hotkey
			lock.unlock()

			let isFn = (hk.keyCode == nil && hk.modifiers.intersection(Hotkey.relevantModifiers) == [.function])
			let isRightCmd = (hk.keyCode == UInt16(kVK_RightCommand))
			if (isFn && keyCode == UInt16(kVK_Function)) || (isRightCmd && keyCode == UInt16(kVK_RightCommand)) {
				let flagsRaw = event.flags.rawValue
				Task { @MainActor in
					LogStore.shared.log("flagsChanged keyCode=\(keyCode) flags=0x\(String(flagsRaw, radix: 16)) mods=\(currentMods)")
				}
			}
		}

		let action: Action
		let shouldConsume: Bool
		let userIsTypingInApp = Self.userIsTypingInApp()
		let inputMonitoringTrusted = inputMonitoringTrusted(now: now)
		lock.lock()
		action = Self.actionForEvent(
			type: type,
			keyCode: keyCode,
			currentModifiers: currentMods,
			hotkey: hotkey,
			inputMonitoringTrusted: inputMonitoringTrusted,
			isPressed: &isPressed,
			event: event
		)
		shouldConsume = consumeEvents && action.shouldConsume && !userIsTypingInApp
		lock.unlock()

		switch action {
		case .none:
			break
		case .press:
			let isFnDoublePress = Self.isFnDoublePress(hotkey: hotkey, now: CFAbsoluteTimeGetCurrent(), lastPressTime: &lastPressTime)
			Task { @MainActor in
				LogStore.shared.log("Hotkey press (\(hotkey.displayString))\(isFnDoublePress ? " [double]" : "")")
			}
			DispatchQueue.main.async { [weak self] in self?.onPress?() }
			if isFnDoublePress {
				DispatchQueue.main.async { [weak self] in self?.onDoublePress?() }
			}
		case .release:
			Task { @MainActor in
				LogStore.shared.log("Hotkey release (\(hotkey.displayString))")
			}
			DispatchQueue.main.async { [weak self] in self?.onRelease?() }
		}

		if shouldConsume {
			return nil
		}

		return Unmanaged.passRetained(event)
	}

	private func inputMonitoringTrusted(now: TimeInterval) -> Bool {
		lock.lock()
		let needsRefresh = (now - lastInputMonitoringTrustedCheck) >= inputMonitoringTrustedCheckInterval
		lock.unlock()

		if needsRefresh {
			let current = InputMonitoringPermission.isTrusted()
			lock.lock()
			inputMonitoringTrustedCached = current
			lastInputMonitoringTrustedCheck = now
			lock.unlock()
			return current
		}

		lock.lock()
		let cached = inputMonitoringTrustedCached
		lock.unlock()
		return cached
	}

	private static func userIsTypingInApp() -> Bool {
		guard Thread.isMainThread else { return false }
		guard NSRunningApplication.current.isActive else { return false }
		guard let keyWindow = NSApp.keyWindow else { return false }
		return keyWindow.firstResponder is NSTextView
	}

	private static func isFnDoublePress(hotkey: Hotkey, now: TimeInterval, lastPressTime: inout TimeInterval) -> Bool {
		guard hotkey.keyCode == nil else {
			lastPressTime = now
			return false
		}
		let required = hotkey.modifiers.intersection(Hotkey.relevantModifiers)
		guard required == [.function] else {
			lastPressTime = now
			return false
		}
		let delta = now - lastPressTime
		lastPressTime = now
		return delta > 0 && delta < 0.35
	}

	private static func modifierFlags(from flags: CGEventFlags) -> NSEvent.ModifierFlags {
		var result: NSEvent.ModifierFlags = []
		if flags.contains(.maskControl) { result.insert(.control) }
		if flags.contains(.maskAlternate) { result.insert(.option) }
		if flags.contains(.maskShift) { result.insert(.shift) }
		if flags.contains(.maskCommand) { result.insert(.command) }
		if flags.contains(.maskSecondaryFn) { result.insert(.function) }
		return result
	}

	private enum Action: Equatable {
		case none
		case press
		case release

		var shouldConsume: Bool {
			switch self {
			case .none:
				return false
			case .press, .release:
				return true
			}
		}
	}

	private static func actionForEvent(
		type: CGEventType,
		keyCode: UInt16,
		currentModifiers: NSEvent.ModifierFlags,
		hotkey: Hotkey,
		inputMonitoringTrusted: Bool,
		isPressed: inout Bool,
		event: CGEvent
	) -> Action {
		var relevantCurrent = currentModifiers.intersection(Hotkey.relevantModifiers)

		// Fn is often reported inconsistently via flags; treat it specially.
		let fnDown = fnIsDownNow(event: event, inputMonitoringTrusted: inputMonitoringTrusted)
		if fnDown {
			relevantCurrent.insert(.function)
		} else {
			relevantCurrent.remove(.function)
		}

		let required = hotkey.modifiers.intersection(Hotkey.relevantModifiers)
		let modifiersMatch = relevantCurrent.isSuperset(of: required)

		if let requiredKeyCode = hotkey.keyCode {
			if Hotkey.modifierFlag(forKeyCode: requiredKeyCode) != nil {
				guard type == .flagsChanged else { return .none }

				let activeNow = modifierKeyIsDownNow(
					requiredKeyCode: requiredKeyCode,
					event: event,
					currentModifiers: currentModifiers,
					modifiersMatch: modifiersMatch,
					inputMonitoringTrusted: inputMonitoringTrusted
				)

				if activeNow && !isPressed {
					isPressed = true
					return .press
				}
				if !activeNow && isPressed {
					isPressed = false
					return .release
				}

				return .none
			}

			guard keyCode == requiredKeyCode, modifiersMatch else { return .none }

			if type == .keyDown {
				let isAutoRepeat = event.getIntegerValueField(.keyboardEventAutorepeat) != 0
				if isAutoRepeat || isPressed { return .none }
				isPressed = true
				return .press
			}

			if type == .keyUp {
				if isPressed == false { return .none }
				isPressed = false
				return .release
			}

			return .none
		}

		// Modifier-only hotkey (e.g. Fn)
		guard required.isEmpty == false, type == .flagsChanged else { return .none }

		// Special-case Fn-only: some systems deliver flagsChanged for Fn but without maskSecondaryFn,
		// and keyState may be unavailable without Input Monitoring. In that case, we fall back to toggling
		// on each flagsChanged event for keyCode==kVK_Function.
		if required == [.function], keyCode == UInt16(kVK_Function) {
			let downNowKnown = fnDown
			if downNowKnown {
				let downNow = true
				if downNow && !isPressed {
					isPressed = true
					return .press
				}
			} else {
				if isPressed {
					isPressed = false
					return .release
				} else {
					isPressed = true
					return .press
				}
			}
			return .none
		}

		let downNow: Bool = modifiersMatch
		if downNow && !isPressed {
			isPressed = true
			return .press
		}
		if !downNow && isPressed {
			isPressed = false
			return .release
		}
		return .none
	}

	private static func modifierKeyIsDownNow(
		requiredKeyCode: UInt16,
		event: CGEvent,
		currentModifiers: NSEvent.ModifierFlags,
		modifiersMatch: Bool,
		inputMonitoringTrusted: Bool
	) -> Bool {
		guard let mod = Hotkey.modifierFlag(forKeyCode: requiredKeyCode) else { return false }

		if requiredKeyCode == UInt16(kVK_Function) {
			return fnIsDownNow(event: event, inputMonitoringTrusted: inputMonitoringTrusted) && modifiersMatch
		}

		let mask: CGEventFlags
		switch mod {
		case .control:
			mask = .maskControl
		case .option:
			mask = .maskAlternate
		case .shift:
			mask = .maskShift
		case .command:
			mask = .maskCommand
		case .function:
			mask = .maskSecondaryFn
		default:
			return false
		}

		guard event.flags.contains(mask) else { return false }

		// For left/right modifier keys, only treat the modifier as down when the event is about that key.
		// This prevents "Right⌘" from triggering when only Left⌘ changes.
		if event.getIntegerValueField(.keyboardEventKeycode) != Int64(requiredKeyCode) {
			return false
		}

		return currentModifiers.contains(mod) && modifiersMatch
	}

	private static func fnIsDownNow(event: CGEvent, inputMonitoringTrusted: Bool) -> Bool {
		if event.flags.contains(.maskSecondaryFn) { return true }
		if inputMonitoringTrusted {
			return CGEventSource.keyState(.combinedSessionState, key: CGKeyCode(kVK_Function))
		}
		return false
	}
}
