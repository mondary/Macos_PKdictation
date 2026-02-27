//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework ApplicationServices -framework AVFoundation -framework Speech -framework Carbon -framework QuartzCore

#import <Cocoa/Cocoa.h>
#import <Speech/Speech.h>
#import <AVFoundation/AVFoundation.h>
#import <ApplicationServices/ApplicationServices.h>
#import <Carbon/Carbon.h>
#import <QuartzCore/QuartzCore.h>
#import <math.h>

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
static CGFloat gMaxMenuTextWidth = 280.0;
static NSString *gLocaleIdentifier = @"";
static const int64_t gMinHoldToRecordMs = 250;
static uint64_t gPendingStartSeq = 0;

static NSStatusItem *gStatusItem = nil;
static CFMachPortRef gEventTap = NULL;
typedef NS_ENUM(NSInteger, PKTGlassTheme) {
	PKTGlassThemeLight = 0,
	PKTGlassThemeDark = 1,
};
typedef NS_ENUM(NSInteger, PKTStatusIconStyle) {
	PKTStatusIconStyleWave = 0,
	PKTStatusIconStyleMicro = 1,
};
typedef NS_ENUM(NSInteger, PKTSpinnerPattern) {
	PKTSpinnerPatternWave = 0,
	PKTSpinnerPatternSpinner = 1,
	PKTSpinnerPatternPulse = 2,
	PKTSpinnerPatternCross = 3,
	PKTSpinnerPatternBurst = 4,
	PKTSpinnerPatternArrowMove = 5,
	PKTSpinnerPatternSineWave = 6,
};
typedef NS_ENUM(NSInteger, PKTSpinnerColor) {
	PKTSpinnerColorMagenta = 0,
	PKTSpinnerColorRed = 1,
	PKTSpinnerColorCyan = 2,
	PKTSpinnerColorGreen = 3,
	PKTSpinnerColorOrange = 4,
	PKTSpinnerColorPurple = 5,
};
static NSInteger gGlassTheme = PKTGlassThemeDark;
static NSInteger gStatusIconStyle = PKTStatusIconStyleMicro;
static NSInteger gSpinnerPattern = PKTSpinnerPatternSpinner;
static NSInteger gSpinnerColor = PKTSpinnerColorMagenta;
static NSImage *gStatusBaseIcon = nil;
static NSPopover *gPopover = nil;
static NSVisualEffectView *gPopoverBackground = nil;
static NSTextField *gPopoverHotkeyLabel = nil;
static NSButton *gPopoverSettingsButton = nil;
static NSTextField *gPopoverHistoryHeader = nil;
static NSButton *gPopoverHistoryButtons[10] = { nil };
static NSButton *gPopoverQuitButton = nil;
static NSStackView *gPopoverStack = nil;
static id gMenuHandler = nil;
static id gFlagsChangedMonitor = nil;
static id gFlagsChangedLocalMonitor = nil;
static BOOL gModifierIsDown = NO;
static BOOL gDidShowAccessibilityAlert = NO;
static NSWindow *gSettingsWindow = nil;
static NSVisualEffectView *gSettingsBackground = nil;
static NSView *gSettingsContent = nil;
static NSButton *gSettingsAutoPasteCheckbox = nil;
static NSSlider *gSettingsMenuWidthSlider = nil;
static NSTextField *gSettingsMenuWidthValueLabel = nil;
static NSSegmentedControl *gSettingsThemeSegment = nil;
static NSSegmentedControl *gSettingsStatusIconSegment = nil;
static NSSegmentedControl *gSettingsPatternTopSegment = nil;
static NSSegmentedControl *gSettingsPatternBottomSegment = nil;
static NSButton *gSettingsColorButtons[6] = { nil };
static NSView *gSettingsPreviewBackground = nil;
static NSView *gSettingsPreviewSpinner = nil;
static NSView *gSettingsPreviewDots[9] = { nil };
static NSPanel *gNotchWindow = nil;
static NSView *gNotchBackground = nil;
static NSView *gNotchSpinner = nil;
static NSView *gNotchDots[9] = { nil };
static NSTextField *gNotchLabel = nil;
static BOOL gNotchShown = NO;
static NSTimer *gSpinnerTimer = nil;
static CFTimeInterval gSpinnerStartTime = 0;

static void startRecording(void);
static void stopRecording(void);
static void updateMenuState(void);
static bool copyToClipboard(NSString *text);
static void addTranscriptToHistory(NSString *text);
static void showSettingsWindow(void);
static bool isModifierHotKeyCode(CGKeyCode keycode);
static bool isHotKeyDownForFlags(NSEventModifierFlags flags);
static NSString *hotkeyTitle(void);
static void togglePopover(void);
static void closePopover(void);
static void ensurePopover(void);
static void applyGlassTheme(void);
static NSImage *makeStatusIcon(BOOL recording);
static void updateStatusItemIcon(void);
static void applySettingsTheme(void);
static void ensureNotchWindow(void);
static void showNotch(void);
static void hideNotch(void);
static void refreshSpinnerVisuals(void);
static void syncSpinnerSettingsUI(void);

@interface MenuHandler : NSObject
@end

@implementation MenuHandler
- (void)statusItemClicked:(id)sender {
	(void)sender;
	togglePopover();
}
- (void)popoverOpenSettings:(id)sender {
	(void)sender;
	closePopover();
	showSettingsWindow();
}
- (void)popoverCopyHistory:(id)sender {
	NSButton *b = (NSButton *)sender;
	if (![b isKindOfClass:[NSButton class]]) return;
	NSInteger idx = b.tag;
	if (!gTranscriptHistory) return;
	if (idx < 0 || idx >= (NSInteger)gTranscriptHistory.count) return;
	(void)copyToClipboard(gTranscriptHistory[(NSUInteger)idx]);
	closePopover();
}
- (void)popoverQuit:(id)sender {
	(void)sender;
	closePopover();
	[NSApp terminate:nil];
}
- (void)settingsToggleAutoPaste:(id)sender {
	NSButton *b = (NSButton *)sender;
	if (![b isKindOfClass:[NSButton class]]) return;
	gAutoPasteEnabled = (b.state == NSControlStateValueOn);
	[[NSUserDefaults standardUserDefaults] setBool:gAutoPasteEnabled forKey:@"autoPasteEnabled"];
	updateMenuState();
}
- (void)settingsMenuWidthChanged:(id)sender {
	NSSlider *s = (NSSlider *)sender;
	if (![s isKindOfClass:[NSSlider class]]) return;
	gMaxMenuTextWidth = round(s.doubleValue);
	[[NSUserDefaults standardUserDefaults] setDouble:gMaxMenuTextWidth forKey:@"maxMenuTextWidth"];
	if (gSettingsMenuWidthValueLabel) {
		gSettingsMenuWidthValueLabel.stringValue = [NSString stringWithFormat:@"%.0f px", gMaxMenuTextWidth];
	}
	updateMenuState();
}
- (void)settingsThemeChanged:(id)sender {
	NSSegmentedControl *seg = (NSSegmentedControl *)sender;
	if (![seg isKindOfClass:[NSSegmentedControl class]]) return;
	gGlassTheme = seg.selectedSegment;
	[[NSUserDefaults standardUserDefaults] setInteger:gGlassTheme forKey:@"glassTheme"];
	applyGlassTheme();
	updateMenuState();
}
- (void)settingsStatusIconChanged:(id)sender {
	NSSegmentedControl *seg = (NSSegmentedControl *)sender;
	if (![seg isKindOfClass:[NSSegmentedControl class]]) return;
	NSInteger v = seg.selectedSegment;
	if (v != PKTStatusIconStyleWave && v != PKTStatusIconStyleMicro) return;
	gStatusIconStyle = v;
	[[NSUserDefaults standardUserDefaults] setInteger:gStatusIconStyle forKey:@"statusIconStyle"];
	updateStatusItemIcon();
	updateMenuState();
}
- (void)settingsPatternTopChanged:(id)sender {
	NSSegmentedControl *seg = (NSSegmentedControl *)sender;
	if (![seg isKindOfClass:[NSSegmentedControl class]]) return;
	NSInteger s = seg.selectedSegment;
	if (s < 0 || s > 3) return;
	gSpinnerPattern = s;
	[[NSUserDefaults standardUserDefaults] setInteger:gSpinnerPattern forKey:@"spinnerPattern"];
	syncSpinnerSettingsUI();
	refreshSpinnerVisuals();
}
- (void)settingsPatternBottomChanged:(id)sender {
	NSSegmentedControl *seg = (NSSegmentedControl *)sender;
	if (![seg isKindOfClass:[NSSegmentedControl class]]) return;
	NSInteger s = seg.selectedSegment;
	if (s < 0 || s > 2) return;
	gSpinnerPattern = 4 + s;
	[[NSUserDefaults standardUserDefaults] setInteger:gSpinnerPattern forKey:@"spinnerPattern"];
	syncSpinnerSettingsUI();
	refreshSpinnerVisuals();
}
- (void)settingsColorClicked:(id)sender {
	NSButton *b = (NSButton *)sender;
	if (![b isKindOfClass:[NSButton class]]) return;
	NSInteger idx = b.tag;
	if (idx < PKTSpinnerColorMagenta || idx > PKTSpinnerColorPurple) return;
	gSpinnerColor = idx;
	[[NSUserDefaults standardUserDefaults] setInteger:gSpinnerColor forKey:@"spinnerColor"];
	syncSpinnerSettingsUI();
	refreshSpinnerVisuals();
}
@end

static void setHotKeyCode(uint16_t v) {
	gHotKeyCode = v;
}

static void updateStatusItemTitle(void) {
	updateStatusItemIcon();
}

static NSImage *makeStatusIcon(BOOL recording) {
	if (gStatusIconStyle == PKTStatusIconStyleWave) {
		if (@available(macOS 11.0, *)) {
			NSImage *wave = [NSImage imageWithSystemSymbolName:@"waveform" accessibilityDescription:@"PKvoice"];
			if (wave) {
				wave.template = YES;
				return wave;
			}
		}
		// Fallback to bundled icon if SF Symbols isn't available.
	}
	if (!gStatusBaseIcon) return nil;
	const CGFloat s = 18.0;

	NSImage *out = [[NSImage alloc] initWithSize:NSMakeSize(s, s)];
	[out lockFocus];

	[gStatusBaseIcon drawInRect:NSMakeRect(0, 0, s, s) fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1.0 respectFlipped:YES hints:nil];

	if (recording) {
		NSRect dotRect = NSMakeRect(s - 6.5, s - 6.5, 5.0, 5.0);
		[[NSColor systemRedColor] setFill];
		[[NSBezierPath bezierPathWithOvalInRect:dotRect] fill];
	}

	[out unlockFocus];
	return out;
}

static void updateStatusItemIcon(void) {
	if (!gStatusItem || !gStatusItem.button) return;
	if (gStatusIconStyle == PKTStatusIconStyleMicro && !gStatusBaseIcon) {
		NSBundle *bundle = [NSBundle mainBundle];
		NSString *iconPath = [bundle pathForResource:@"PKvoice" ofType:@"icns"];
		if (iconPath) {
			gStatusBaseIcon = [[NSImage alloc] initWithContentsOfFile:iconPath];
		}
	}

	NSImage *img = makeStatusIcon(gIsRecording);
	if (!img) return;

	gStatusItem.button.image = img;
	if (gStatusIconStyle == PKTStatusIconStyleWave) {
		gStatusItem.button.contentTintColor = gIsRecording ? [NSColor systemRedColor] : nil;
	} else {
		gStatusItem.button.contentTintColor = nil;
	}
	gStatusItem.button.title = @"";
	gStatusItem.button.attributedTitle = [[NSAttributedString alloc] initWithString:@""];
	gStatusItem.button.imagePosition = NSImageOnly;
}

static bool copyToClipboard(NSString *text) {
	if (!text) return false;
	NSString *trim = [text stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
	if (trim.length == 0) return false;

	NSPasteboard *pb = [NSPasteboard generalPasteboard];
	[pb clearContents];
	[pb setString:trim forType:NSPasteboardTypeString];
	return true;
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
	a.informativeText = @"Pour coller automatiquement (Cmd+V), active PKvoice dans Réglages Système → Confidentialité et sécurité → Accessibilité, puis relance l’app.";
	[a addButtonWithTitle:@"OK"];
	[a runModal];
}

static void copyAndMaybePasteText(NSString *text, bool shouldPaste) {
	bool didCopy = copyToClipboard(text);
	if (!didCopy) return;
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

static bool isModifierHotKeyCode(CGKeyCode keycode) {
	switch (keycode) {
	case (CGKeyCode)kVK_Function:
	case (CGKeyCode)kVK_Shift:
	case (CGKeyCode)kVK_RightShift:
	case (CGKeyCode)kVK_Control:
	case (CGKeyCode)kVK_RightControl:
	case (CGKeyCode)kVK_Option:
	case (CGKeyCode)kVK_RightOption:
	case (CGKeyCode)kVK_Command:
	case (CGKeyCode)kVK_RightCommand:
		return true;
	default:
		return false;
	}
}

static bool isHotKeyDownForFlags(NSEventModifierFlags flags) {
	switch ((CGKeyCode)gHotKeyCode) {
	case (CGKeyCode)kVK_Function:
		return (flags & NSEventModifierFlagFunction) != 0;
	case (CGKeyCode)kVK_Shift:
	case (CGKeyCode)kVK_RightShift:
		return (flags & NSEventModifierFlagShift) != 0;
	case (CGKeyCode)kVK_Control:
	case (CGKeyCode)kVK_RightControl:
		return (flags & NSEventModifierFlagControl) != 0;
	case (CGKeyCode)kVK_Option:
	case (CGKeyCode)kVK_RightOption:
		return (flags & NSEventModifierFlagOption) != 0;
	case (CGKeyCode)kVK_Command:
	case (CGKeyCode)kVK_RightCommand:
		return (flags & NSEventModifierFlagCommand) != 0;
	default:
		return false;
	}
}

static NSString *spinnerPatternTitle(void) {
	switch (gSpinnerPattern) {
	case PKTSpinnerPatternWave: return @"Wave";
	case PKTSpinnerPatternSpinner: return @"Spinner";
	case PKTSpinnerPatternPulse: return @"Pulse";
	case PKTSpinnerPatternCross: return @"Cross";
	case PKTSpinnerPatternBurst: return @"Burst";
	case PKTSpinnerPatternArrowMove: return @"ArrowMove";
	case PKTSpinnerPatternSineWave: return @"Sine Wave";
	default: return @"Spinner";
	}
}

static CGFloat clamp01(CGFloat v) {
	if (v < 0.0) return 0.0;
	if (v > 1.0) return 1.0;
	return v;
}

static CGFloat smoothstep01(CGFloat t) {
	t = clamp01(t);
	return t * t * (3.0 - 2.0 * t);
}

static CGFloat wrapDist01(CGFloat a, CGFloat b) {
	CGFloat d = fabs(a - b);
	if (d > 1.0) d = fmod(d, 1.0);
	return fmin(d, 1.0 - d);
}

static CGFloat phasePulse(CGFloat t, CGFloat phase, CGFloat width) {
	CGFloat d = wrapDist01(t, phase);
	if (d >= width) return 0.0;
	return smoothstep01(1.0 - (d / width));
}

static BOOL dotInList(NSInteger idx, const NSInteger *list, NSInteger count) {
	for (NSInteger i = 0; i < count; i++) {
		if (list[i] == idx) return YES;
	}
	return NO;
}

static NSColor *spinnerBaseColor(void) {
	return [NSColor colorWithCalibratedWhite:0.20 alpha:1.0]; // #333333
}

static NSColor *spinnerPresetColor(NSInteger idx) {
	switch (idx) {
	case PKTSpinnerColorRed:
		return [NSColor colorWithCalibratedRed:1.0 green:74.0/255.0 blue:74.0/255.0 alpha:1.0];
	case PKTSpinnerColorCyan:
		return [NSColor colorWithCalibratedRed:60.0/255.0 green:220.0/255.0 blue:1.0 alpha:1.0];
	case PKTSpinnerColorGreen:
		return [NSColor colorWithCalibratedRed:80.0/255.0 green:240.0/255.0 blue:130.0/255.0 alpha:1.0];
	case PKTSpinnerColorOrange:
		return [NSColor colorWithCalibratedRed:1.0 green:154.0/255.0 blue:64.0/255.0 alpha:1.0];
	case PKTSpinnerColorPurple:
		return [NSColor colorWithCalibratedRed:168.0/255.0 green:120.0/255.0 blue:1.0 alpha:1.0];
	case PKTSpinnerColorMagenta:
	default:
		return [NSColor colorWithCalibratedRed:1.0 green:20.0/255.0 blue:204.0/255.0 alpha:1.0];
	}
}

static NSColor *spinnerAccentColor(void) {
	return spinnerPresetColor(gSpinnerColor);
}

static NSColor *spinnerGlowColor(void) {
	NSColor *accent = [spinnerAccentColor() colorUsingColorSpace:[NSColorSpace genericRGBColorSpace]];
	if (!accent) return [NSColor colorWithCalibratedWhite:1.0 alpha:1.0];
	CGFloat r = 1, g = 1, b = 1, a = 1;
	[accent getRed:&r green:&g blue:&b alpha:&a];
	CGFloat mix = 0.55;
	return [NSColor colorWithCalibratedRed:(r * (1.0 - mix) + 1.0 * mix)
	                                 green:(g * (1.0 - mix) + 1.0 * mix)
	                                  blue:(b * (1.0 - mix) + 1.0 * mix)
	                                 alpha:1.0];
}

static CFTimeInterval spinnerCycleDuration(void) {
	switch (gSpinnerPattern) {
	case PKTSpinnerPatternWave: return 1.50;
	case PKTSpinnerPatternSpinner: return 1.00;
	case PKTSpinnerPatternPulse: return 1.75;
	case PKTSpinnerPatternCross: return 1.30;
	case PKTSpinnerPatternBurst: return 1.25;
	case PKTSpinnerPatternArrowMove: return 1.75;
	case PKTSpinnerPatternSineWave: return 1.25;
	default: return 1.20;
	}
}

static CGFloat spinnerIntensityForDot(NSInteger i, double tNorm) {
	CGFloat t = (CGFloat)tNorm;
	NSInteger row = i / 3;
	NSInteger col = i % 3;
	BOOL isCenter = (i == 4);

	switch (gSpinnerPattern) {
	case PKTSpinnerPatternSpinner: {
		if (isCenter) return 1.0;
		static const NSInteger ringDots[4] = {1, 5, 7, 3};
		static const CGFloat phases[4] = {0.00, 0.25, 0.50, 0.75};
		for (NSInteger k = 0; k < 4; k++) {
			if (i == ringDots[k]) return phasePulse(t, phases[k], 0.18);
		}
		return 0.0;
	}
	case PKTSpinnerPatternWave: {
		CGFloat phase = (CGFloat)(row + col) / 4.0;
		CGFloat a = phasePulse(t, phase, 0.16);
		CGFloat b = 0.45 * phasePulse(t, fmod(phase + 0.50, 1.0), 0.16);
		return clamp01(fmax(a, b));
	}
	case PKTSpinnerPatternPulse: {
		CGFloat delay = 0.0;
		if (!isCenter) {
			NSInteger manhattan = labs(row - 1) + labs(col - 1);
			delay = (manhattan == 1) ? 0.09 : 0.17;
		}
		return phasePulse(t, delay, isCenter ? 0.26 : 0.20);
	}
	case PKTSpinnerPatternCross: {
		static const NSInteger xDots[] = {0, 2, 6, 8};
		static const NSInteger plusDots[] = {1, 3, 5, 7};
		CGFloat xI = fmax(phasePulse(t, 0.00, 0.22), phasePulse(t, 0.50, 0.22));
		CGFloat plusI = fmax(phasePulse(t, 0.25, 0.20), phasePulse(t, 0.75, 0.20));
		if (isCenter) return clamp01(fmax(xI, plusI));
		if (dotInList(i, xDots, 4)) return xI;
		if (dotInList(i, plusDots, 4)) return plusI;
		return 0.0;
	}
	case PKTSpinnerPatternBurst: {
		CGFloat phase = 0.0;
		if (!isCenter) {
			NSInteger manhattan = labs(row - 1) + labs(col - 1);
			phase = (manhattan == 1) ? 0.12 : 0.22;
		}
		return phasePulse(t, phase, isCenter ? 0.20 : 0.18);
	}
	case PKTSpinnerPatternArrowMove: {
		static const NSInteger frame0[] = {0, 3, 4, 6, 1};
		static const NSInteger frame1[] = {1, 4, 7, 2, 5};
		static const NSInteger frame2[] = {2, 5, 8, 4, 7};
		CGFloat f0 = phasePulse(t, 0.00, 0.16);
		CGFloat f1 = phasePulse(t, 0.33, 0.16);
		CGFloat f2 = phasePulse(t, 0.66, 0.16);
		CGFloat v = 0.0;
		if (dotInList(i, frame0, 5)) v = fmax(v, f0);
		if (dotInList(i, frame1, 5)) v = fmax(v, f1);
		if (dotInList(i, frame2, 5)) v = fmax(v, f2);
		return clamp01(v);
	}
	case PKTSpinnerPatternSineWave: {
		CGFloat colPhase = (CGFloat)col * 0.18;
		CGFloat rowOffset = (row == 1) ? 0.00 : 0.10;
		CGFloat a = phasePulse(t, fmod(colPhase + rowOffset, 1.0), 0.18);
		CGFloat b = 0.65 * phasePulse(t, fmod(colPhase + rowOffset + 0.50, 1.0), 0.18);
		return clamp01(fmax(a, b));
	}
	default:
		return 0.0;
	}
}

static void applyDotStyle(NSView *dot, CGFloat intensity) {
	if (!dot || !dot.layer) return;
	intensity = clamp01(intensity);
	BOOL active = intensity > 0.01;
	dot.layer.backgroundColor = (active ? spinnerAccentColor() : spinnerBaseColor()).CGColor;
	dot.layer.shadowColor = spinnerGlowColor().CGColor;
	dot.layer.shadowOpacity = active ? (0.25 + 0.70 * intensity) : 0.0;
	dot.layer.shadowRadius = active ? (1.0 + 3.0 * intensity) : 0.0;
	dot.layer.shadowOffset = CGSizeZero;
	CGFloat s = 1.0 + 0.08 * intensity;
	dot.layer.transform = active ? CATransform3DMakeScale(s, s, 1.0) : CATransform3DIdentity;
}

static void updateDots(NSView * __strong dots[9], double tNorm) {
	for (NSInteger i = 0; i < 9; i++) {
		applyDotStyle(dots[i], spinnerIntensityForDot(i, tNorm));
	}
}

static BOOL shouldAnimateSpinner(void) {
	BOOL settingsVisible = (gSettingsWindow && gSettingsWindow.visible && gSettingsPreviewSpinner);
	return gNotchShown || settingsVisible;
}

static void updateSpinnerFrame(void) {
	if (gSpinnerStartTime <= 0) gSpinnerStartTime = CACurrentMediaTime();
	CFTimeInterval now = CACurrentMediaTime();
	CFTimeInterval dur = spinnerCycleDuration();
	double tNorm = (dur > 0) ? fmod((now - gSpinnerStartTime) / dur, 1.0) : 0.0;
	updateDots(gNotchDots, tNorm);
	updateDots(gSettingsPreviewDots, tNorm);
}

static void syncSpinnerSettingsUI(void) {
	if (gSettingsPatternTopSegment && gSettingsPatternBottomSegment) {
		[gSettingsPatternTopSegment setSelectedSegment:-1];
		[gSettingsPatternBottomSegment setSelectedSegment:-1];
		if (gSpinnerPattern >= PKTSpinnerPatternWave && gSpinnerPattern <= PKTSpinnerPatternCross) {
			gSettingsPatternTopSegment.selectedSegment = gSpinnerPattern;
		} else {
			gSettingsPatternBottomSegment.selectedSegment = gSpinnerPattern - 4;
		}
	}
	for (NSInteger i = 0; i < 6; i++) {
		NSButton *b = gSettingsColorButtons[i];
		if (!b || !b.layer) continue;
		BOOL selected = (i == gSpinnerColor);
		b.layer.borderColor = selected ? [NSColor whiteColor].CGColor : [NSColor colorWithCalibratedWhite:1.0 alpha:0.25].CGColor;
		b.layer.borderWidth = selected ? 2.0 : 1.0;
	}
}

static void ensureSpinnerTimer(void) {
	if (gSpinnerTimer) return;
	gSpinnerStartTime = CACurrentMediaTime();
	updateSpinnerFrame();
	gSpinnerTimer = [NSTimer scheduledTimerWithTimeInterval:(1.0 / 30.0) repeats:YES block:^(__unused NSTimer *timer) {
		if (!shouldAnimateSpinner()) {
			[gSpinnerTimer invalidate];
			gSpinnerTimer = nil;
			return;
		}
		updateSpinnerFrame();
	}];
	[[NSRunLoop mainRunLoop] addTimer:gSpinnerTimer forMode:NSRunLoopCommonModes];
}

static void refreshSpinnerVisuals(void) {
	if (gNotchLabel) {
		gNotchLabel.stringValue = [NSString stringWithFormat:@"Recording · %@", spinnerPatternTitle()];
	}
	syncSpinnerSettingsUI();
	gSpinnerStartTime = CACurrentMediaTime();
	updateSpinnerFrame();
	if (shouldAnimateSpinner()) {
		ensureSpinnerTimer();
	} else if (gSpinnerTimer) {
		[gSpinnerTimer invalidate];
		gSpinnerTimer = nil;
	}
}

static void positionNotchWindow(void) {
	if (!gNotchWindow) return;
	NSScreen *screen = [NSScreen mainScreen];
	if (!screen) {
		NSArray<NSScreen *> *screens = [NSScreen screens];
		if (screens.count > 0) screen = screens[0];
	}
	if (!screen) return;
	NSRect visible = screen.visibleFrame;
	NSRect frame = gNotchWindow.frame;
	CGFloat x = round(NSMidX(visible) - frame.size.width / 2.0);
	CGFloat y = round(NSMaxY(visible) - frame.size.height - 8.0);
	[gNotchWindow setFrame:NSMakeRect(x, y, frame.size.width, frame.size.height) display:NO];
}

static void ensureNotchWindow(void) {
	if (gNotchWindow) return;

	NSRect frame = NSMakeRect(0, 0, 220, 38);
	gNotchWindow = [[NSPanel alloc] initWithContentRect:frame
		styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
		backing:NSBackingStoreBuffered
		defer:NO];
	gNotchWindow.releasedWhenClosed = NO;
	gNotchWindow.opaque = NO;
	gNotchWindow.backgroundColor = [NSColor clearColor];
	gNotchWindow.hasShadow = NO;
	gNotchWindow.level = NSScreenSaverWindowLevel;
	gNotchWindow.hidesOnDeactivate = NO;
	gNotchWindow.ignoresMouseEvents = YES;
	gNotchWindow.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorTransient | NSWindowCollectionBehaviorFullScreenAuxiliary | NSWindowCollectionBehaviorStationary | NSWindowCollectionBehaviorIgnoresCycle;

	gNotchBackground = [NSView new];
	gNotchBackground.frame = ((NSView *)gNotchWindow.contentView).bounds;
	gNotchBackground.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
	gNotchBackground.wantsLayer = YES;
	gNotchBackground.layer.cornerRadius = 12.0;
	gNotchBackground.layer.masksToBounds = YES;
	gNotchBackground.layer.backgroundColor = [NSColor colorWithCalibratedWhite:0.08 alpha:0.95].CGColor;
	gNotchBackground.layer.borderWidth = 1.0;
	gNotchBackground.layer.borderColor = [NSColor colorWithCalibratedWhite:1.0 alpha:0.22].CGColor;
	[gNotchWindow setContentView:gNotchBackground];

	NSView *content = gNotchBackground;
	gNotchSpinner = [NSView new];
	gNotchSpinner.translatesAutoresizingMaskIntoConstraints = NO;
	gNotchSpinner.wantsLayer = YES;
	[content addSubview:gNotchSpinner];

	const CGFloat dotSize = 4.0;
	const CGFloat gap = 1.0;
	for (NSInteger i = 0; i < 9; i++) {
		NSInteger row = i / 3;
		NSInteger col = i % 3;
		NSView *d = [[NSView alloc] initWithFrame:NSMakeRect(col * (dotSize + gap), (2 - row) * (dotSize + gap), dotSize, dotSize)];
		d.wantsLayer = YES;
		d.layer.cornerRadius = 1.0;
		d.layer.masksToBounds = NO;
		d.layer.backgroundColor = spinnerBaseColor().CGColor;
		gNotchDots[i] = d;
		[gNotchSpinner addSubview:d];
	}

	gNotchLabel = [NSTextField labelWithString:@"Recording"];
	gNotchLabel.translatesAutoresizingMaskIntoConstraints = NO;
	gNotchLabel.font = [NSFont systemFontOfSize:11 weight:NSFontWeightSemibold];
	gNotchLabel.textColor = [NSColor whiteColor];
	gNotchLabel.lineBreakMode = NSLineBreakByTruncatingTail;
	[content addSubview:gNotchLabel];

	[NSLayoutConstraint activateConstraints:@[
		[gNotchSpinner.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:12],
		[gNotchSpinner.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
		[gNotchSpinner.widthAnchor constraintEqualToConstant:14],
		[gNotchSpinner.heightAnchor constraintEqualToConstant:14],
		[gNotchLabel.leadingAnchor constraintEqualToAnchor:gNotchSpinner.trailingAnchor constant:8],
		[gNotchLabel.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-12],
		[gNotchLabel.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
	]];

	positionNotchWindow();
	refreshSpinnerVisuals();
}

static void showNotch(void) {
	ensureNotchWindow();
	gNotchShown = YES;
	positionNotchWindow();
	refreshSpinnerVisuals();
	[gNotchWindow orderFrontRegardless];
}

static void hideNotch(void) {
	if (!gNotchWindow) return;
	gNotchShown = NO;
	[gNotchWindow orderOut:nil];
	refreshSpinnerVisuals();
}

static void updateMenuState(void) {
	if (gSettingsAutoPasteCheckbox) gSettingsAutoPasteCheckbox.state = gAutoPasteEnabled ? NSControlStateValueOn : NSControlStateValueOff;
	if (gSettingsMenuWidthSlider) gSettingsMenuWidthSlider.doubleValue = gMaxMenuTextWidth;
	if (gSettingsMenuWidthValueLabel) gSettingsMenuWidthValueLabel.stringValue = [NSString stringWithFormat:@"%.0f px", gMaxMenuTextWidth];
	if (gSettingsThemeSegment) gSettingsThemeSegment.selectedSegment = gGlassTheme;
	if (gSettingsStatusIconSegment) gSettingsStatusIconSegment.selectedSegment = gStatusIconStyle;
	syncSpinnerSettingsUI();
	if (gPopoverHotkeyLabel) gPopoverHotkeyLabel.stringValue = hotkeyTitle() ?: @"";

	for (int i = 0; i < 10; i++) {
		NSButton *it = gPopoverHistoryButtons[i];
		if (!it) continue;
		if (!gTranscriptHistory || i >= (int)gTranscriptHistory.count) {
			it.hidden = YES;
			continue;
		}
		NSString *entry = gTranscriptHistory[i];
		NSString *full = [NSString stringWithFormat:@"%d. %@", i + 1, entry];
		it.title = truncateStringToMenuWidth(full, gMaxMenuTextWidth);
		it.hidden = NO;
	}

	BOOL hasHistory = (gTranscriptHistory && gTranscriptHistory.count > 0);
	if (gPopoverHistoryHeader) gPopoverHistoryHeader.hidden = !hasHistory;

	if (gPopover) {
		NSInteger count = gTranscriptHistory ? (NSInteger)gTranscriptHistory.count : 0;
		if (count > 10) count = 10;
		CGFloat base = 210.0;
		CGFloat row = 22.0;
		CGFloat h = base + row * (CGFloat)count;
		if (h < 210.0) h = 210.0;
		gPopover.contentSize = NSMakeSize(360.0, h);
	}
}

static NSString *hotkeyTitle(void) {
	switch ((CGKeyCode)gHotKeyCode) {
	case (CGKeyCode)kVK_Function:
		return @"Raccourci : Fn (maintenir)";
	case (CGKeyCode)kVK_RightCommand:
	case (CGKeyCode)kVK_Command:
		return @"Raccourci : Cmd (maintenir)";
	case (CGKeyCode)kVK_RightOption:
	case (CGKeyCode)kVK_Option:
		return @"Raccourci : Option (maintenir)";
	case (CGKeyCode)kVK_RightShift:
	case (CGKeyCode)kVK_Shift:
		return @"Raccourci : Shift (maintenir)";
	case (CGKeyCode)kVK_RightControl:
	case (CGKeyCode)kVK_Control:
		return @"Raccourci : Ctrl (maintenir)";
	default:
		break;
	}
	return [NSString stringWithFormat:@"Raccourci : keycode 0x%X", (unsigned)gHotKeyCode];
}

static void applyGlassTheme(void) {
	if (!gPopoverBackground) return;
	NSAppearanceName name = (gGlassTheme == PKTGlassThemeDark) ? NSAppearanceNameVibrantDark : NSAppearanceNameVibrantLight;
	NSAppearance *appearance = [NSAppearance appearanceNamed:name];
	if (gPopover) gPopover.appearance = appearance;
	gPopoverBackground.appearance = appearance;
	gPopoverBackground.material = (gGlassTheme == PKTGlassThemeDark) ? NSVisualEffectMaterialHUDWindow : NSVisualEffectMaterialPopover;
	gPopoverBackground.blendingMode = NSVisualEffectBlendingModeWithinWindow;
	gPopoverBackground.state = NSVisualEffectStateActive;

	applySettingsTheme();
}

static void applySettingsTheme(void) {
	if (!gSettingsWindow) return;
	NSAppearanceName name = (gGlassTheme == PKTGlassThemeDark) ? NSAppearanceNameVibrantDark : NSAppearanceNameVibrantLight;
	NSAppearance *appearance = [NSAppearance appearanceNamed:name];
	gSettingsWindow.appearance = appearance;
	if (gSettingsBackground) {
		gSettingsBackground.appearance = appearance;
		gSettingsBackground.material = (gGlassTheme == PKTGlassThemeDark) ? NSVisualEffectMaterialHUDWindow : NSVisualEffectMaterialPopover;
		gSettingsBackground.blendingMode = NSVisualEffectBlendingModeWithinWindow;
		gSettingsBackground.state = NSVisualEffectStateActive;
	}
}

static void ensurePopover(void) {
	if (gPopover) return;

	NSViewController *vc = [NSViewController new];
	gPopoverBackground = [NSVisualEffectView new];
	gPopoverBackground.translatesAutoresizingMaskIntoConstraints = NO;
	vc.view = gPopoverBackground;

	gPopover = [NSPopover new];
	gPopover.behavior = NSPopoverBehaviorTransient;
	gPopover.animates = YES;
	gPopover.contentViewController = vc;
	gPopover.contentSize = NSMakeSize(360.0, 260.0);

	NSView *content = gPopoverBackground;

	NSTextField *title = [NSTextField labelWithString:@"PKvoice"];
	title.font = [NSFont boldSystemFontOfSize:13];
	title.alignment = NSTextAlignmentLeft;
	title.translatesAutoresizingMaskIntoConstraints = NO;

	gPopoverHotkeyLabel = [NSTextField labelWithString:@""];
	gPopoverHotkeyLabel.font = [NSFont systemFontOfSize:12];
	gPopoverHotkeyLabel.textColor = [NSColor secondaryLabelColor];
	gPopoverHotkeyLabel.alignment = NSTextAlignmentLeft;
	gPopoverHotkeyLabel.translatesAutoresizingMaskIntoConstraints = NO;

	gPopoverSettingsButton = [NSButton buttonWithTitle:@"Settings…" target:gMenuHandler action:@selector(popoverOpenSettings:)];
	gPopoverSettingsButton.bordered = YES;
	gPopoverSettingsButton.bezelStyle = NSBezelStyleTexturedRounded;
	gPopoverSettingsButton.alignment = NSTextAlignmentLeft;
	gPopoverSettingsButton.controlSize = NSControlSizeSmall;
	gPopoverSettingsButton.translatesAutoresizingMaskIntoConstraints = NO;

	NSBox *sep1 = [NSBox new];
	sep1.boxType = NSBoxSeparator;
	sep1.translatesAutoresizingMaskIntoConstraints = NO;

	gPopoverHistoryHeader = [NSTextField labelWithString:@"Historique"];
	gPopoverHistoryHeader.font = [NSFont boldSystemFontOfSize:12];
	gPopoverHistoryHeader.alignment = NSTextAlignmentLeft;
	gPopoverHistoryHeader.translatesAutoresizingMaskIntoConstraints = NO;

	NSStackView *historyStack = [NSStackView new];
	historyStack.orientation = NSUserInterfaceLayoutOrientationVertical;
	historyStack.spacing = 4;
	historyStack.alignment = NSLayoutAttributeLeading;
	historyStack.translatesAutoresizingMaskIntoConstraints = NO;
	for (int i = 0; i < 10; i++) {
		NSButton *b = [NSButton buttonWithTitle:@"" target:gMenuHandler action:@selector(popoverCopyHistory:)];
		b.tag = i;
		b.bordered = NO;
		b.bezelStyle = NSBezelStyleRegularSquare;
		b.alignment = NSTextAlignmentLeft;
		b.font = [NSFont systemFontOfSize:12];
		b.hidden = YES;
		[b setButtonType:NSButtonTypeMomentaryPushIn];
		gPopoverHistoryButtons[i] = b;
		[historyStack addArrangedSubview:b];
	}

	NSBox *sep2 = [NSBox new];
	sep2.boxType = NSBoxSeparator;
	sep2.translatesAutoresizingMaskIntoConstraints = NO;

	gPopoverQuitButton = [NSButton buttonWithTitle:@"Quitter" target:gMenuHandler action:@selector(popoverQuit:)];
	gPopoverQuitButton.bordered = YES;
	gPopoverQuitButton.bezelStyle = NSBezelStyleTexturedRounded;
	gPopoverQuitButton.alignment = NSTextAlignmentLeft;
	gPopoverQuitButton.controlSize = NSControlSizeSmall;
	gPopoverQuitButton.translatesAutoresizingMaskIntoConstraints = NO;

	NSView *bottomSpacer = [NSView new];
	bottomSpacer.translatesAutoresizingMaskIntoConstraints = NO;
	[bottomSpacer setContentHuggingPriority:NSLayoutPriorityDefaultLow forOrientation:NSLayoutConstraintOrientationHorizontal];
	[gPopoverSettingsButton setContentHuggingPriority:NSLayoutPriorityRequired forOrientation:NSLayoutConstraintOrientationHorizontal];
	[gPopoverQuitButton setContentHuggingPriority:NSLayoutPriorityRequired forOrientation:NSLayoutConstraintOrientationHorizontal];

	NSStackView *bottomRow = [NSStackView new];
	bottomRow.orientation = NSUserInterfaceLayoutOrientationHorizontal;
	bottomRow.spacing = 8;
	bottomRow.distribution = NSStackViewDistributionFill;
	bottomRow.alignment = NSLayoutAttributeCenterY;
	bottomRow.translatesAutoresizingMaskIntoConstraints = NO;
	[bottomRow addArrangedSubview:gPopoverSettingsButton];
	[bottomRow addArrangedSubview:bottomSpacer];
	[bottomRow addArrangedSubview:gPopoverQuitButton];

	gPopoverStack = [NSStackView new];
	gPopoverStack.orientation = NSUserInterfaceLayoutOrientationVertical;
	gPopoverStack.spacing = 10;
	gPopoverStack.alignment = NSLayoutAttributeLeading;
	gPopoverStack.translatesAutoresizingMaskIntoConstraints = NO;

	[gPopoverStack addArrangedSubview:title];
	[gPopoverStack addArrangedSubview:gPopoverHotkeyLabel];
	[gPopoverStack addArrangedSubview:sep1];
	[gPopoverStack addArrangedSubview:gPopoverHistoryHeader];
	[gPopoverStack addArrangedSubview:historyStack];
	[gPopoverStack addArrangedSubview:sep2];
	[gPopoverStack addArrangedSubview:bottomRow];

	[content addSubview:gPopoverStack];

	[NSLayoutConstraint activateConstraints:@[
		[gPopoverStack.topAnchor constraintEqualToAnchor:content.topAnchor constant:14],
		[gPopoverStack.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:14],
		[gPopoverStack.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-14],
		[gPopoverStack.bottomAnchor constraintLessThanOrEqualToAnchor:content.bottomAnchor constant:-14],

		[bottomRow.leadingAnchor constraintEqualToAnchor:gPopoverStack.leadingAnchor],
		[bottomRow.trailingAnchor constraintEqualToAnchor:gPopoverStack.trailingAnchor],

		[sep1.heightAnchor constraintEqualToConstant:1],
		[sep2.heightAnchor constraintEqualToConstant:1],
	]];

	applyGlassTheme();
}

static void closePopover(void) {
	if (!gPopover) return;
	if (gPopover.isShown) [gPopover performClose:nil];
}

static void togglePopover(void) {
	if (!gStatusItem || !gStatusItem.button) return;
	ensurePopover();
	updateMenuState();
	applyGlassTheme();

	if (gPopover.isShown) {
		[gPopover performClose:nil];
		return;
	}
	[gPopover showRelativeToRect:gStatusItem.button.bounds ofView:gStatusItem.button preferredEdge:NSRectEdgeMinY];
}

static void showSettingsWindow(void) {
	if (gSettingsWindow) {
		updateMenuState();
		if (gSettingsMenuWidthSlider) gSettingsMenuWidthSlider.doubleValue = gMaxMenuTextWidth;
		if (gSettingsMenuWidthValueLabel) gSettingsMenuWidthValueLabel.stringValue = [NSString stringWithFormat:@"%.0f px", gMaxMenuTextWidth];
		applySettingsTheme();
		[NSApp activateIgnoringOtherApps:YES];
		[gSettingsWindow makeKeyAndOrderFront:gSettingsWindow];
		refreshSpinnerVisuals();
		return;
	}

	NSRect frame = NSMakeRect(0, 0, 520, 560);
	NSWindowStyleMask style = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable;
	gSettingsWindow = [[NSWindow alloc] initWithContentRect:frame styleMask:style backing:NSBackingStoreBuffered defer:NO];
	gSettingsWindow.title = @"PKvoice — Settings";
	gSettingsWindow.releasedWhenClosed = NO;

	gSettingsBackground = [NSVisualEffectView new];
	gSettingsBackground.frame = gSettingsWindow.contentView.bounds;
	gSettingsBackground.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
	[gSettingsWindow setContentView:gSettingsBackground];

	gSettingsContent = [NSView new];
	gSettingsContent.translatesAutoresizingMaskIntoConstraints = NO;
	[gSettingsBackground addSubview:gSettingsContent];
	[NSLayoutConstraint activateConstraints:@[
		[gSettingsContent.topAnchor constraintEqualToAnchor:gSettingsBackground.topAnchor],
		[gSettingsContent.leadingAnchor constraintEqualToAnchor:gSettingsBackground.leadingAnchor],
		[gSettingsContent.trailingAnchor constraintEqualToAnchor:gSettingsBackground.trailingAnchor],
		[gSettingsContent.bottomAnchor constraintEqualToAnchor:gSettingsBackground.bottomAnchor],
	]];

	NSView *content = gSettingsContent;

	NSTextField *title = [NSTextField labelWithString:@"Paramètres"];
	title.font = [NSFont boldSystemFontOfSize:18];
	title.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:title];

	NSString *hotkey = hotkeyTitle() ?: @"";
	NSString *locale = (gLocaleIdentifier && gLocaleIdentifier.length > 0) ? gLocaleIdentifier : @"system";
	NSString *appVersion = [[[NSBundle mainBundle] objectForInfoDictionaryKey:@"CFBundleShortVersionString"] description] ?: @"?";
	NSTextField *subtitle = [NSTextField labelWithString:[NSString stringWithFormat:@"%@\nLocale : %@\nVersion : %@", hotkey, locale, appVersion]];
	subtitle.font = [NSFont systemFontOfSize:12];
	subtitle.textColor = [NSColor secondaryLabelColor];
	subtitle.lineBreakMode = NSLineBreakByWordWrapping;
	subtitle.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:subtitle];

	gSettingsAutoPasteCheckbox = [NSButton checkboxWithTitle:@"Auto-paste (Cmd+V) après relâchement" target:gMenuHandler action:@selector(settingsToggleAutoPaste:)];
	gSettingsAutoPasteCheckbox.state = gAutoPasteEnabled ? NSControlStateValueOn : NSControlStateValueOff;
	gSettingsAutoPasteCheckbox.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsAutoPasteCheckbox];

	NSTextField *widthLabel = [NSTextField labelWithString:@"Largeur max (menu historique)"];
	widthLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:widthLabel];

	gSettingsMenuWidthValueLabel = [NSTextField labelWithString:[NSString stringWithFormat:@"%.0f px", gMaxMenuTextWidth]];
	gSettingsMenuWidthValueLabel.textColor = [NSColor secondaryLabelColor];
	gSettingsMenuWidthValueLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsMenuWidthValueLabel];

	gSettingsMenuWidthSlider = [NSSlider sliderWithValue:gMaxMenuTextWidth minValue:180 maxValue:420 target:gMenuHandler action:@selector(settingsMenuWidthChanged:)];
	gSettingsMenuWidthSlider.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsMenuWidthSlider];

	NSTextField *themeLabel = [NSTextField labelWithString:@"Style du menu"];
	themeLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:themeLabel];

	gSettingsThemeSegment = [NSSegmentedControl segmentedControlWithLabels:@[ @"Light", @"Dark" ] trackingMode:NSSegmentSwitchTrackingSelectOne target:gMenuHandler action:@selector(settingsThemeChanged:)];
	gSettingsThemeSegment.selectedSegment = gGlassTheme;
	if (@available(macOS 11.0, *)) {
		NSImage *sun = [NSImage imageWithSystemSymbolName:@"sun.max" accessibilityDescription:@"Light"];
		NSImage *moon = [NSImage imageWithSystemSymbolName:@"moon" accessibilityDescription:@"Dark"];
		if (sun && moon) {
			[gSettingsThemeSegment setLabel:@"" forSegment:0];
			[gSettingsThemeSegment setLabel:@"" forSegment:1];
			[gSettingsThemeSegment setImage:sun forSegment:0];
			[gSettingsThemeSegment setImage:moon forSegment:1];
		}
	}
	gSettingsThemeSegment.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsThemeSegment];

	NSTextField *iconLabel = [NSTextField labelWithString:@"Icône menubar"];
	iconLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:iconLabel];

	gSettingsStatusIconSegment = [NSSegmentedControl segmentedControlWithLabels:@[ @"Wave", @"Micro" ] trackingMode:NSSegmentSwitchTrackingSelectOne target:gMenuHandler action:@selector(settingsStatusIconChanged:)];
	gSettingsStatusIconSegment.selectedSegment = gStatusIconStyle;
	if (@available(macOS 11.0, *)) {
		NSImage *waveIcon = [NSImage imageWithSystemSymbolName:@"waveform" accessibilityDescription:@"Wave"];
		if (waveIcon) {
			[waveIcon setSize:NSMakeSize(16, 16)];
			[gSettingsStatusIconSegment setLabel:@"" forSegment:0];
			[gSettingsStatusIconSegment setImage:waveIcon forSegment:0];
		}
	}
	NSBundle *bundle = [NSBundle mainBundle];
	NSString *microIconPath = [bundle pathForResource:@"PKvoice" ofType:@"icns"];
	if (microIconPath) {
		NSImage *microIcon = [[NSImage alloc] initWithContentsOfFile:microIconPath];
		if (microIcon) {
			[microIcon setSize:NSMakeSize(16, 16)];
			[gSettingsStatusIconSegment setLabel:@"" forSegment:1];
			[gSettingsStatusIconSegment setImage:microIcon forSegment:1];
		}
	}
	gSettingsStatusIconSegment.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsStatusIconSegment];

	NSTextField *notchLabel = [NSTextField labelWithString:@"Notch FN (animation)"];
	notchLabel.font = [NSFont boldSystemFontOfSize:13];
	notchLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:notchLabel];

	NSTextField *patternLabel = [NSTextField labelWithString:@"Animation"];
	patternLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:patternLabel];

	gSettingsPatternTopSegment = [NSSegmentedControl segmentedControlWithLabels:@[ @"Wave", @"Spinner", @"Pulse", @"Cross" ]
		trackingMode:NSSegmentSwitchTrackingSelectOne
		target:gMenuHandler
		action:@selector(settingsPatternTopChanged:)];
	gSettingsPatternTopSegment.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsPatternTopSegment];

	gSettingsPatternBottomSegment = [NSSegmentedControl segmentedControlWithLabels:@[ @"Burst", @"Arrow", @"Sine" ]
		trackingMode:NSSegmentSwitchTrackingSelectOne
		target:gMenuHandler
		action:@selector(settingsPatternBottomChanged:)];
	gSettingsPatternBottomSegment.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:gSettingsPatternBottomSegment];

	NSTextField *colorLabel = [NSTextField labelWithString:@"Couleur"];
	colorLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:colorLabel];

	NSView *swatchRow = [NSView new];
	swatchRow.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:swatchRow];

	const CGFloat swatchSize = 20.0;
	const CGFloat swatchGap = 8.0;
	for (NSInteger i = 0; i < 6; i++) {
		NSButton *b = [NSButton buttonWithTitle:@"" target:gMenuHandler action:@selector(settingsColorClicked:)];
		b.tag = i;
		b.bordered = NO;
		b.frame = NSMakeRect(i * (swatchSize + swatchGap), 0, swatchSize, swatchSize);
		b.wantsLayer = YES;
		b.layer.cornerRadius = swatchSize / 2.0;
		b.layer.masksToBounds = YES;
		b.layer.backgroundColor = spinnerPresetColor(i).CGColor;
		b.layer.borderColor = [NSColor colorWithCalibratedWhite:1.0 alpha:0.25].CGColor;
		b.layer.borderWidth = 1.0;
		gSettingsColorButtons[i] = b;
		[swatchRow addSubview:b];
	}

	NSTextField *previewLabel = [NSTextField labelWithString:@"Prévisualisation"];
	previewLabel.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:previewLabel];

	gSettingsPreviewBackground = [NSView new];
	gSettingsPreviewBackground.translatesAutoresizingMaskIntoConstraints = NO;
	gSettingsPreviewBackground.wantsLayer = YES;
	gSettingsPreviewBackground.layer.cornerRadius = 12.0;
	gSettingsPreviewBackground.layer.masksToBounds = YES;
	gSettingsPreviewBackground.layer.backgroundColor = [NSColor colorWithCalibratedWhite:0.08 alpha:0.95].CGColor;
	gSettingsPreviewBackground.layer.borderColor = [NSColor colorWithCalibratedWhite:1.0 alpha:0.18].CGColor;
	gSettingsPreviewBackground.layer.borderWidth = 1.0;
	[content addSubview:gSettingsPreviewBackground];

	NSTextField *previewHint = [NSTextField labelWithString:@"FN: le notch animé apparaît pendant l'appui."];
	previewHint.translatesAutoresizingMaskIntoConstraints = NO;
	previewHint.font = [NSFont systemFontOfSize:11];
	previewHint.textColor = [NSColor colorWithCalibratedWhite:1.0 alpha:0.9];
	[gSettingsPreviewBackground addSubview:previewHint];

	gSettingsPreviewSpinner = [NSView new];
	gSettingsPreviewSpinner.translatesAutoresizingMaskIntoConstraints = NO;
	gSettingsPreviewSpinner.wantsLayer = YES;
	[gSettingsPreviewBackground addSubview:gSettingsPreviewSpinner];

	const CGFloat previewDotSize = 4.0;
	const CGFloat previewGap = 1.0;
	for (NSInteger i = 0; i < 9; i++) {
		NSInteger row = i / 3;
		NSInteger col = i % 3;
		NSView *d = [[NSView alloc] initWithFrame:NSMakeRect(col * (previewDotSize + previewGap), (2 - row) * (previewDotSize + previewGap), previewDotSize, previewDotSize)];
		d.wantsLayer = YES;
		d.layer.cornerRadius = 1.0;
		d.layer.masksToBounds = NO;
		d.layer.backgroundColor = spinnerBaseColor().CGColor;
		gSettingsPreviewDots[i] = d;
		[gSettingsPreviewSpinner addSubview:d];
	}

	[NSLayoutConstraint activateConstraints:@[
		[title.topAnchor constraintEqualToAnchor:content.topAnchor constant:18],
		[title.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],
		[title.trailingAnchor constraintLessThanOrEqualToAnchor:content.trailingAnchor constant:-18],

		[subtitle.topAnchor constraintEqualToAnchor:title.bottomAnchor constant:6],
		[subtitle.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],
		[subtitle.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],

		[gSettingsAutoPasteCheckbox.topAnchor constraintEqualToAnchor:subtitle.bottomAnchor constant:16],
		[gSettingsAutoPasteCheckbox.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],
		[gSettingsAutoPasteCheckbox.trailingAnchor constraintLessThanOrEqualToAnchor:content.trailingAnchor constant:-18],

		[widthLabel.topAnchor constraintEqualToAnchor:gSettingsAutoPasteCheckbox.bottomAnchor constant:18],
		[widthLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[gSettingsMenuWidthValueLabel.centerYAnchor constraintEqualToAnchor:widthLabel.centerYAnchor],
		[gSettingsMenuWidthValueLabel.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],

		[gSettingsMenuWidthSlider.topAnchor constraintEqualToAnchor:widthLabel.bottomAnchor constant:8],
		[gSettingsMenuWidthSlider.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],
		[gSettingsMenuWidthSlider.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],

		[themeLabel.topAnchor constraintEqualToAnchor:gSettingsMenuWidthSlider.bottomAnchor constant:16],
		[themeLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[gSettingsThemeSegment.centerYAnchor constraintEqualToAnchor:themeLabel.centerYAnchor],
		[gSettingsThemeSegment.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],

		[iconLabel.topAnchor constraintEqualToAnchor:themeLabel.bottomAnchor constant:16],
		[iconLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[gSettingsStatusIconSegment.centerYAnchor constraintEqualToAnchor:iconLabel.centerYAnchor],
		[gSettingsStatusIconSegment.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],

		[notchLabel.topAnchor constraintEqualToAnchor:iconLabel.bottomAnchor constant:20],
		[notchLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[patternLabel.topAnchor constraintEqualToAnchor:notchLabel.bottomAnchor constant:12],
		[patternLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[gSettingsPatternTopSegment.centerYAnchor constraintEqualToAnchor:patternLabel.centerYAnchor],
		[gSettingsPatternTopSegment.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],
		[gSettingsPatternTopSegment.widthAnchor constraintEqualToConstant:320],

		[gSettingsPatternBottomSegment.topAnchor constraintEqualToAnchor:gSettingsPatternTopSegment.bottomAnchor constant:8],
		[gSettingsPatternBottomSegment.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],
		[gSettingsPatternBottomSegment.widthAnchor constraintEqualToConstant:240],

		[colorLabel.topAnchor constraintEqualToAnchor:gSettingsPatternBottomSegment.bottomAnchor constant:14],
		[colorLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[swatchRow.centerYAnchor constraintEqualToAnchor:colorLabel.centerYAnchor],
		[swatchRow.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],
		[swatchRow.widthAnchor constraintEqualToConstant:(swatchSize * 6.0 + swatchGap * 5.0)],
		[swatchRow.heightAnchor constraintEqualToConstant:swatchSize],

		[previewLabel.topAnchor constraintEqualToAnchor:colorLabel.bottomAnchor constant:14],
		[previewLabel.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],

		[gSettingsPreviewBackground.topAnchor constraintEqualToAnchor:previewLabel.bottomAnchor constant:8],
		[gSettingsPreviewBackground.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],
		[gSettingsPreviewBackground.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],
		[gSettingsPreviewBackground.heightAnchor constraintEqualToConstant:74],
		[gSettingsPreviewBackground.bottomAnchor constraintLessThanOrEqualToAnchor:content.bottomAnchor constant:-18],

		[gSettingsPreviewSpinner.leadingAnchor constraintEqualToAnchor:gSettingsPreviewBackground.leadingAnchor constant:12],
		[gSettingsPreviewSpinner.centerYAnchor constraintEqualToAnchor:gSettingsPreviewBackground.centerYAnchor],
		[gSettingsPreviewSpinner.widthAnchor constraintEqualToConstant:14],
		[gSettingsPreviewSpinner.heightAnchor constraintEqualToConstant:14],

		[previewHint.leadingAnchor constraintEqualToAnchor:gSettingsPreviewSpinner.trailingAnchor constant:8],
		[previewHint.trailingAnchor constraintEqualToAnchor:gSettingsPreviewBackground.trailingAnchor constant:-12],
		[previewHint.centerYAnchor constraintEqualToAnchor:gSettingsPreviewBackground.centerYAnchor],
	]];

	syncSpinnerSettingsUI();
	applySettingsTheme();
	[gSettingsWindow center];
	[NSApp activateIgnoringOtherApps:YES];
	[gSettingsWindow makeKeyAndOrderFront:gSettingsWindow];
	refreshSpinnerVisuals();
}

static void stopRecording(void) {
	if (!gIsRecording) return;
	gIsRecording = false;
	hideNotch();
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
		hideNotch();
		return;
	}

	gIsRecording = true;
	showNotch();
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
		hideNotch();
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

	if (isModifierHotKeyCode(keycode)) {
		// Modifier hotkeys are handled via NSEvent flagsChanged monitors (more reliable).
		return event;
	}

	if (type == kCGEventKeyDown) {
		dispatch_async(dispatch_get_main_queue(), ^{
			showNotch();
			startRecording();
		});
	} else if (type == kCGEventKeyUp) {
		dispatch_async(dispatch_get_main_queue(), ^{
			stopRecording();
			hideNotch();
		});
	}
	return event;
}

static void setupStatusBar(void) {
	gStatusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
	gStatusItem.button.title = @"";
	// Handler implemented above.
	gMenuHandler = [MenuHandler new];
	if (!gTranscriptHistory) gTranscriptHistory = [NSMutableArray arrayWithCapacity:10];

	gStatusItem.menu = nil;
	gStatusItem.button.target = gMenuHandler;
	gStatusItem.button.action = @selector(statusItemClicked:);
	updateStatusItemIcon();

	ensurePopover();
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

		NSUserDefaults *defaults = [NSUserDefaults standardUserDefaults];
		if ([defaults objectForKey:@"glassTheme"] != nil) {
			NSInteger t = [defaults integerForKey:@"glassTheme"];
			if (t == PKTGlassThemeLight || t == PKTGlassThemeDark) gGlassTheme = t;
		}
		if ([defaults objectForKey:@"maxMenuTextWidth"] != nil) {
			double w = [defaults doubleForKey:@"maxMenuTextWidth"];
			if (w >= 180.0 && w <= 420.0) gMaxMenuTextWidth = (CGFloat)w;
		}
		if ([defaults objectForKey:@"autoPasteEnabled"] != nil) {
			gAutoPasteEnabled = [defaults boolForKey:@"autoPasteEnabled"];
		}
		if ([defaults objectForKey:@"statusIconStyle"] != nil) {
			NSInteger s = [defaults integerForKey:@"statusIconStyle"];
			if (s == PKTStatusIconStyleWave || s == PKTStatusIconStyleMicro) gStatusIconStyle = s;
		}
		if ([defaults objectForKey:@"spinnerPattern"] != nil) {
			NSInteger p = [defaults integerForKey:@"spinnerPattern"];
			if (p >= PKTSpinnerPatternWave && p <= PKTSpinnerPatternSineWave) gSpinnerPattern = p;
		}
		if ([defaults objectForKey:@"spinnerColor"] != nil) {
			NSInteger c = [defaults integerForKey:@"spinnerColor"];
			if (c >= PKTSpinnerColorMagenta && c <= PKTSpinnerColorPurple) gSpinnerColor = c;
		}

		requestPermissions();
		// Prompt early for Accessibility so Cmd+V paste can work.
		(void)ensureAccessibilityTrusted(true);

		NSString *locale = nil;
		if (localeCString && strlen(localeCString) > 0) {
			locale = [NSString stringWithUTF8String:localeCString];
		}
		gLocaleIdentifier = locale ?: @"";
		if (locale) {
			gRecognizer = [[SFSpeechRecognizer alloc] initWithLocale:[NSLocale localeWithLocaleIdentifier:locale]];
		} else {
			gRecognizer = [[SFSpeechRecognizer alloc] init];
		}

		setupStatusBar();
		updateStatusItemTitle();

		// Modifier keys (Fn/Cmd/Option/Shift/Ctrl) may not produce reliable keyDown/up events; listen to modifier flag changes.
		void (^modifierFlagsHandler)(NSEvent *) = ^(NSEvent *e) {
			if (!isModifierHotKeyCode((CGKeyCode)gHotKeyCode)) return;
			BOOL down = isHotKeyDownForFlags(e.modifierFlags);
			if (down == gModifierIsDown) return;
			gModifierIsDown = down;
			if (down) {
				showNotch();
				// Delay start so a quick tap (e.g. Fn for emojis) doesn't trigger paste/clipboard side effects.
				uint64_t seq = ++gPendingStartSeq;
				dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(gMinHoldToRecordMs * NSEC_PER_MSEC)), dispatch_get_main_queue(), ^{
					if (!gModifierIsDown) return;
					if (gIsRecording) return;
					if (seq != gPendingStartSeq) return;
					startRecording();
				});
				return;
			}

			// Key released: cancel any pending start; stop only if we actually started.
			++gPendingStartSeq;
			hideNotch();
			dispatch_async(dispatch_get_main_queue(), ^{
				if (gIsRecording) stopRecording();
			});
		};
		gFlagsChangedMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskFlagsChanged handler:modifierFlagsHandler];
		gFlagsChangedLocalMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskFlagsChanged handler:^NSEvent * _Nullable(NSEvent * _Nonnull e) {
			modifierFlagsHandler(e);
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
