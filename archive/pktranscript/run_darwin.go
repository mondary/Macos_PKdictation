//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework AVFoundation -framework Speech -framework Carbon

#import <Cocoa/Cocoa.h>
#import <Speech/Speech.h>
#import <AVFoundation/AVFoundation.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>

static uint16_t gHotKeyCode = 0x61; // F6 by default
static bool gIsRecording = false;
static bool gPasteWhenFinal = false;
static bool gAutoPasteEnabled = true;
static bool gCopyWhenFinal = false;

static SFSpeechRecognizer *gRecognizer = nil;
static SFSpeechAudioBufferRecognitionRequest *gRequest = nil;
static SFSpeechRecognitionTask *gTask = nil;
static AVAudioEngine *gEngine = nil;

static NSString *gLatestTranscript = @"";
static NSMutableArray<NSString *> *gTranscriptHistory = nil;
static bool gDidCommitTranscript = false;

static NSStatusItem *gStatusItem = nil;
static CFMachPortRef gEventTap = NULL;
static NSMenuItem *gTranscriptToggleItem = nil;
static NSMenuItem *gHotkeyItem = nil;
static NSMenuItem *gHistoryHeaderItem = nil;
static NSMenuItem *gHistoryItems[10] = { nil };
static id gMenuHandler = nil;
static id gFlagsChangedMonitor = nil;
static id gFlagsChangedLocalMonitor = nil;
static BOOL gFnIsDown = NO;
static BOOL gDidShowAccessibilityAlert = NO;

static void startRecording(void);
static void stopRecording(void);
static void updateMenuState(void);
static void copyToClipboard(NSString *text);
static void reopenMenuSoon(void);
static void addTranscriptToHistory(NSString *text);

@interface MenuHandler : NSObject
@end

@implementation MenuHandler
- (void)menuToggleTranscript:(id)sender {
	(void)sender;
	gAutoPasteEnabled = !gAutoPasteEnabled;
	updateMenuState();
}
- (void)menuCopyHistory:(id)sender {
	NSMenuItem *item = (NSMenuItem *)sender;
	if (![item isKindOfClass:[NSMenuItem class]]) return;
	id obj = item.representedObject;
	if (![obj isKindOfClass:[NSString class]]) return;
	copyToClipboard((NSString *)obj);
}
@end

static void setHotKeyCode(uint16_t v) {
	gHotKeyCode = v;
}

static void updateStatusItemTitle(void) {
	if (!gStatusItem) return;
	if (gIsRecording) {
		gStatusItem.button.title = @"● PKT";
	} else {
		gStatusItem.button.title = @"PKT";
	}
}

static void copyToClipboard(NSString *text) {
	if (!text) return;
	NSString *trim = [text stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
	if (trim.length == 0) return;

	NSPasteboard *pb = [NSPasteboard generalPasteboard];
	[pb clearContents];
	[pb setString:trim forType:NSPasteboardTypeString];
}

static void pasteClipboard(void) {
	// Simulate Cmd+V to paste at current cursor location.
	CGEventRef keyDown = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)kVK_ANSI_V, true);
	CGEventRef keyUp = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)kVK_ANSI_V, false);
	CGEventSetFlags(keyDown, kCGEventFlagMaskCommand);
	CGEventSetFlags(keyUp, kCGEventFlagMaskCommand);
	CGEventPost(kCGHIDEventTap, keyDown);
	CGEventPost(kCGHIDEventTap, keyUp);
	CFRelease(keyDown);
	CFRelease(keyUp);
}

static bool ensureAccessibilityTrusted(bool prompt) {
	if (!prompt) return AXIsProcessTrusted();
	const void *keys[] = { kAXTrustedCheckOptionPrompt };
	const void *vals[] = { kCFBooleanTrue };
	CFDictionaryRef opts = CFDictionaryCreate(kCFAllocatorDefault, keys, vals, 1, &kCFCopyStringDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	Boolean trusted = AXIsProcessTrustedWithOptions(opts);
	CFRelease(opts);
	return trusted;
}

static void showAccessibilityAlertOnce(void) {
	if (gDidShowAccessibilityAlert) return;
	gDidShowAccessibilityAlert = YES;
	NSAlert *a = [[NSAlert alloc] init];
	a.messageText = @"Autorisation requise : Accessibilité";
	a.informativeText = @"Pour coller automatiquement (Cmd+V), active PKTranscript dans Réglages Système → Confidentialité et sécurité → Accessibilité, puis relance l’app.";
	[a addButtonWithTitle:@"OK"];
	[a runModal];
}

static void reopenMenuSoon(void) {
	if (!gStatusItem || !gStatusItem.menu) return;
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(80 * NSEC_PER_MSEC)), dispatch_get_main_queue(), ^{
		// Re-open the status item menu right after an action (menus close automatically on click).
		if (gStatusItem.button) [gStatusItem.button performClick:nil];
	});
}

static void copyAndMaybePasteText(NSString *text, bool shouldPaste) {
	copyToClipboard(text);
	if (!shouldPaste) return;

	if (!ensureAccessibilityTrusted(true)) {
		dispatch_async(dispatch_get_main_queue(), ^{
			showAccessibilityAlertOnce();
		});
		return;
	}
	pasteClipboard();
}

static NSString *truncateStringToMenuWidth(NSString *s, CGFloat maxWidth) {
	if (!s) return @"";
	NSString *trim = [s stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
	if (trim.length == 0) return @"";

	NSFont *font = [NSFont menuFontOfSize:0];
	NSDictionary *attrs = @{ NSFontAttributeName : font };
	if ([trim sizeWithAttributes:attrs].width <= maxWidth) return trim;

	NSString *ellipsis = @"…";
	CGFloat ellW = [ellipsis sizeWithAttributes:attrs].width;
	if (ellW >= maxWidth) return ellipsis;

	NSUInteger lo = 0;
	NSUInteger hi = trim.length;
	NSUInteger best = 0;
	while (lo <= hi) {
		NSUInteger mid = (lo + hi) / 2;
		NSRange r = [trim rangeOfComposedCharacterSequencesForRange:NSMakeRange(0, mid)];
		NSString *candidate = [[trim substringWithRange:r] stringByAppendingString:ellipsis];
		CGFloat w = [candidate sizeWithAttributes:attrs].width;
		if (w <= maxWidth) {
			best = r.length;
			lo = mid + 1;
		} else {
			if (mid == 0) break;
			hi = mid - 1;
		}
	}
	NSRange r = [trim rangeOfComposedCharacterSequencesForRange:NSMakeRange(0, best)];
	return [[trim substringWithRange:r] stringByAppendingString:ellipsis];
}

static void addTranscriptToHistory(NSString *text) {
	if (!text) return;
	NSString *trim = [text stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
	if (trim.length == 0) return;

	if (!gTranscriptHistory) {
		gTranscriptHistory = [NSMutableArray arrayWithCapacity:10];
	}
	if (gTranscriptHistory.count > 0) {
		NSString *top = gTranscriptHistory[0];
		if ([top isEqualToString:trim]) return;
	}
	[gTranscriptHistory insertObject:trim atIndex:0];
	while (gTranscriptHistory.count > 10) {
		[gTranscriptHistory removeLastObject];
	}
}

static void updateMenuState(void) {
	const CGFloat kMaxMenuTextWidth = 280.0;
	if (gTranscriptToggleItem) gTranscriptToggleItem.state = gAutoPasteEnabled ? NSControlStateValueOn : NSControlStateValueOff;

	BOOL hasHistory = (gTranscriptHistory && gTranscriptHistory.count > 0);
	if (gHistoryHeaderItem) {
		gHistoryHeaderItem.hidden = !hasHistory;
	}
	for (int i = 0; i < 10; i++) {
		NSMenuItem *it = gHistoryItems[i];
		if (!it) continue;
		if (!hasHistory || i >= (int)gTranscriptHistory.count) {
			it.hidden = YES;
			it.representedObject = nil;
			it.title = @"";
			continue;
		}
		NSString *entry = gTranscriptHistory[i];
		NSString *full = [NSString stringWithFormat:@"%d. %@", i + 1, entry];
		it.title = truncateStringToMenuWidth(full, kMaxMenuTextWidth);
		it.representedObject = entry;
		it.enabled = YES;
		it.hidden = NO;
	}
}

static NSString *hotkeyTitle(void) {
	if (gHotKeyCode == (uint16_t)kVK_Function) return @"Raccourci : Fn (maintenir)";
	return [NSString stringWithFormat:@"Raccourci : keycode 0x%X", (unsigned)gHotKeyCode];
}

static void stopRecording(void) {
	if (!gIsRecording) return;
	gIsRecording = false;
	updateStatusItemTitle();
	updateMenuState();

	if (gEngine && gEngine.isRunning) {
		[gEngine stop];
		AVAudioInputNode *input = [gEngine inputNode];
		[input removeTapOnBus:0];
	}
	if (gRequest) {
		[gRequest endAudio];
	}
	gPasteWhenFinal = gAutoPasteEnabled;
	gCopyWhenFinal = !gAutoPasteEnabled;
}

static void startRecording(void) {
	if (gIsRecording) return;
	if (!gRecognizer) return;
	if (!gRecognizer.isAvailable) {
		NSLog(@"Speech recognizer not available");
		return;
	}

	gIsRecording = true;
	gDidCommitTranscript = false;
	gPasteWhenFinal = false;
	gCopyWhenFinal = false;
	gLatestTranscript = @"";
	updateStatusItemTitle();
	updateMenuState();

	if (gTask) {
		[gTask cancel];
		gTask = nil;
	}
	gRequest = [[SFSpeechAudioBufferRecognitionRequest alloc] init];
	gRequest.shouldReportPartialResults = YES;

	gEngine = [[AVAudioEngine alloc] init];
	AVAudioInputNode *input = [gEngine inputNode];
	AVAudioFormat *format = [input outputFormatForBus:0];

	[input installTapOnBus:0 bufferSize:2048 format:format block:^(AVAudioPCMBuffer *buffer, AVAudioTime *when) {
		if (gRequest) {
			[gRequest appendAudioPCMBuffer:buffer];
		}
	}];

	NSError *err = nil;
	[gEngine prepare];
	if (![gEngine startAndReturnError:&err]) {
		NSLog(@"Audio engine start error: %@", err);
		gIsRecording = false;
		updateStatusItemTitle();
		return;
	}

	gTask = [gRecognizer recognitionTaskWithRequest:gRequest resultHandler:^(SFSpeechRecognitionResult *result, NSError *error) {
		if (result) {
			gLatestTranscript = result.bestTranscription.formattedString ?: @"";
			dispatch_async(dispatch_get_main_queue(), ^{
				updateMenuState();
			});
		}
		if (error) {
			NSLog(@"Recognition error: %@", error);
		}

		BOOL isFinal = result ? result.isFinal : NO;
		if ((isFinal || error) && !gDidCommitTranscript) {
			gDidCommitTranscript = true;
			NSString *toCommit = gLatestTranscript;
			dispatch_async(dispatch_get_main_queue(), ^{
				addTranscriptToHistory(toCommit);
				updateMenuState();
			});
		}
		if ((isFinal || error) && gPasteWhenFinal) {
			NSString *toPaste = gLatestTranscript;
			gPasteWhenFinal = false;
			gCopyWhenFinal = false;
			dispatch_async(dispatch_get_main_queue(), ^{
				copyAndMaybePasteText(toPaste, true);
			});
		} else if ((isFinal || error) && gCopyWhenFinal) {
			NSString *toCopy = gLatestTranscript;
			gPasteWhenFinal = false;
			gCopyWhenFinal = false;
			dispatch_async(dispatch_get_main_queue(), ^{
				copyAndMaybePasteText(toCopy, false);
				updateMenuState();
			});
		}
	}];
}

static CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon) {
	if (type == kCGEventTapDisabledByTimeout) {
		if (gEventTap) CGEventTapEnable(gEventTap, true);
		return event;
	}
	if (type != kCGEventKeyDown && type != kCGEventKeyUp && type != kCGEventFlagsChanged) return event;

	CGKeyCode keycode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
	if (keycode != gHotKeyCode) return event;

	if (type == kCGEventFlagsChanged && keycode == (CGKeyCode)kVK_Function) {
		// Fn is handled via NSEvent monitors (more reliable). Avoid double-trigger here.
		return event;
	}

	if (type == kCGEventKeyDown) {
		dispatch_async(dispatch_get_main_queue(), ^{
			startRecording();
		});
	} else if (type == kCGEventKeyUp) {
		dispatch_async(dispatch_get_main_queue(), ^{
			stopRecording();
		});
	}
	return event;
}

static void setupStatusBar(void) {
	gStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
	gStatusItem.button.title = @"PKT";

	NSMenu *menu = [[NSMenu alloc] init];
	menu.autoenablesItems = NO;
	// Menu handler implemented below.
	gMenuHandler = [MenuHandler new];
	if (!gTranscriptHistory) gTranscriptHistory = [NSMutableArray arrayWithCapacity:10];

	gHotkeyItem = [[NSMenuItem alloc] initWithTitle:hotkeyTitle() action:nil keyEquivalent:@""];
	gHotkeyItem.enabled = NO;
	[menu addItem:gHotkeyItem];

	[menu addItem:[NSMenuItem separatorItem]];

	gTranscriptToggleItem = [[NSMenuItem alloc] initWithTitle:@"Transcript (auto-paste)" action:@selector(menuToggleTranscript:) keyEquivalent:@""];
	gTranscriptToggleItem.target = gMenuHandler;
	[menu addItem:gTranscriptToggleItem];

	[menu addItem:[NSMenuItem separatorItem]];

	gHistoryHeaderItem = [[NSMenuItem alloc] initWithTitle:@"Historique (10)" action:nil keyEquivalent:@""];
	gHistoryHeaderItem.enabled = NO;
	[menu addItem:gHistoryHeaderItem];

	for (int i = 0; i < 10; i++) {
		gHistoryItems[i] = [[NSMenuItem alloc] initWithTitle:@"" action:@selector(menuCopyHistory:) keyEquivalent:@""];
		gHistoryItems[i].target = gMenuHandler;
		gHistoryItems[i].indentationLevel = 1;
		gHistoryItems[i].hidden = YES;
		[menu addItem:gHistoryItems[i]];
	}

	[menu addItem:[NSMenuItem separatorItem]];
	NSMenuItem *quit = [[NSMenuItem alloc] initWithTitle:@"Quitter PKTranscript" action:@selector(terminate:) keyEquivalent:@"q"];
	quit.enabled = YES;
	[menu addItem:quit];
	gStatusItem.menu = menu;

	updateMenuState();
}

static void requestPermissions(void) {
	[SFSpeechRecognizer requestAuthorization:^(SFSpeechRecognizerAuthorizationStatus status) {
		NSLog(@"Speech auth status: %ld", (long)status);
	}];

	[AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {
		NSLog(@"Microphone access: %@", granted ? @"granted" : @"denied");
	}];
}

static void runApp(const char *localeCString) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

		requestPermissions();
		// Prompt early for Accessibility so Cmd+V paste can work.
		(void)ensureAccessibilityTrusted(true);

		NSString *locale = nil;
		if (localeCString && strlen(localeCString) > 0) {
			locale = [NSString stringWithUTF8String:localeCString];
		}
		if (locale) {
			gRecognizer = [[SFSpeechRecognizer alloc] initWithLocale:[NSLocale localeWithLocaleIdentifier:locale]];
		} else {
			gRecognizer = [[SFSpeechRecognizer alloc] init];
		}

		setupStatusBar();
		updateStatusItemTitle();

		// Fn is a modifier key and may not produce keyDown/up events; listen to modifier flag changes.
		void (^fnFlagsHandler)(NSEvent *) = ^(NSEvent *e) {
			if (gHotKeyCode != (uint16_t)kVK_Function) return;
			BOOL down = (e.modifierFlags & NSEventModifierFlagFunction) != 0;
			if (down == gFnIsDown) return;
			gFnIsDown = down;
			dispatch_async(dispatch_get_main_queue(), ^{
				if (down) startRecording(); else stopRecording();
			});
		};
		gFlagsChangedMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskFlagsChanged handler:fnFlagsHandler];
		gFlagsChangedLocalMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskFlagsChanged handler:^NSEvent * _Nullable(NSEvent * _Nonnull e) {
			fnFlagsHandler(e);
			return e;
		}];

		CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp) | CGEventMaskBit(kCGEventFlagsChanged);
		gEventTap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, 0, mask, eventTapCallback, NULL);
		if (!gEventTap) {
			NSLog(@"Failed to create event tap. Check Input Monitoring permission.");
		} else {
			CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, gEventTap, 0);
			CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
			CGEventTapEnable(gEventTap, true);
			CFRelease(source);
		}

		[NSApp run];
	}
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func Run(hotkeyKeycode uint16, locale string) error {
	if hotkeyKeycode == 0 {
		return errors.New("hotkey keycode invalide")
	}
	C.setHotKeyCode(C.uint16_t(hotkeyKeycode))
	cLocale := C.CString(locale)
	defer C.free(unsafe.Pointer(cLocale))
	C.runApp(cLocale)
	return nil
}
