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
static NSInteger gGlassTheme = PKTGlassThemeDark;
static NSInteger gStatusIconStyle = PKTStatusIconStyleMicro;
static NSImage *gStatusBaseIcon = nil;
static NSPopover *gPopover = nil;
static NSVisualEffectView *gPopoverBackground = nil;
static NSTextField *gPopoverHotkeyLabel = nil;
static NSButton *gPopoverAutoPasteCheckbox = nil;
static NSButton *gPopoverSettingsButton = nil;
static NSTextField *gPopoverHistoryHeader = nil;
static NSButton *gPopoverHistoryButtons[10] = { nil };
static NSButton *gPopoverQuitButton = nil;
static NSStackView *gPopoverStack = nil;
static NSPanel *gRecordingNotchWindow = nil;
static NSVisualEffectView *gRecordingNotchBackground = nil;
static NSImageView *gRecordingNotchIconView = nil;
static NSTextField *gRecordingNotchLabel = nil;
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
static void ensureRecordingNotch(void);
static void showRecordingNotch(void);
static void hideRecordingNotch(void);
static void updateRecordingNotchState(void);

@interface MenuHandler : NSObject
@end

@implementation MenuHandler
- (void)statusItemClicked:(id)sender {
	(void)sender;
	togglePopover();
}
- (void)popoverToggleAutoPaste:(id)sender {
	NSButton *b = (NSButton *)sender;
	if (![b isKindOfClass:[NSButton class]]) return;
	gAutoPasteEnabled = (b.state == NSControlStateValueOn);
	[[NSUserDefaults standardUserDefaults] setBool:gAutoPasteEnabled forKey:@"autoPasteEnabled"];
	updateMenuState();
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
	updateRecordingNotchState();
	updateMenuState();
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

static void updateMenuState(void) {
	if (gSettingsAutoPasteCheckbox) gSettingsAutoPasteCheckbox.state = gAutoPasteEnabled ? NSControlStateValueOn : NSControlStateValueOff;
	if (gPopoverAutoPasteCheckbox) gPopoverAutoPasteCheckbox.state = gAutoPasteEnabled ? NSControlStateValueOn : NSControlStateValueOff;
	if (gSettingsMenuWidthSlider) gSettingsMenuWidthSlider.doubleValue = gMaxMenuTextWidth;
	if (gSettingsMenuWidthValueLabel) gSettingsMenuWidthValueLabel.stringValue = [NSString stringWithFormat:@"%.0f px", gMaxMenuTextWidth];
	if (gSettingsThemeSegment) gSettingsThemeSegment.selectedSegment = gGlassTheme;
	if (gSettingsStatusIconSegment) gSettingsStatusIconSegment.selectedSegment = gStatusIconStyle;
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

static NSImage *makeRecordingNotchIcon(void) {
	if (gStatusIconStyle == PKTStatusIconStyleWave) {
		if (@available(macOS 11.0, *)) {
			NSImage *wave = [NSImage imageWithSystemSymbolName:@"waveform" accessibilityDescription:@"Recording"];
			if (wave) {
				[wave setSize:NSMakeSize(15, 15)];
				wave.template = YES;
				return wave;
			}
		}
	}

	if (!gStatusBaseIcon) {
		NSBundle *bundle = [NSBundle mainBundle];
		NSString *iconPath = [bundle pathForResource:@"PKvoice" ofType:@"icns"];
		if (iconPath) {
			gStatusBaseIcon = [[NSImage alloc] initWithContentsOfFile:iconPath];
		}
	}
	if (!gStatusBaseIcon) return nil;

	NSImage *img = [gStatusBaseIcon copy];
	[img setSize:NSMakeSize(15, 15)];
	return img;
}

static void positionRecordingNotchWindow(void) {
	if (!gRecordingNotchWindow) return;
	NSScreen *screen = [NSScreen mainScreen];
	if (!screen) {
		NSArray<NSScreen *> *screens = [NSScreen screens];
		if (screens.count > 0) screen = screens[0];
	}
	if (!screen) return;

	NSRect visible = screen.visibleFrame;
	NSRect current = gRecordingNotchWindow.frame;
	CGFloat x = round(NSMidX(visible) - current.size.width / 2.0);
	CGFloat y = round(NSMaxY(visible) - current.size.height - 6.0);
	[gRecordingNotchWindow setFrame:NSMakeRect(x, y, current.size.width, current.size.height) display:NO];
}

static void updateRecordingNotchState(void) {
	if (!gRecordingNotchWindow) return;
	if (gRecordingNotchLabel) {
		gRecordingNotchLabel.stringValue = gIsRecording ? @"Enregistrement en cours" : @"Enregistrement";
	}
	if (gRecordingNotchIconView) {
		gRecordingNotchIconView.image = makeRecordingNotchIcon();
		gRecordingNotchIconView.contentTintColor = [NSColor whiteColor];
	}
	if (gRecordingNotchBackground) {
		gRecordingNotchBackground.material = NSVisualEffectMaterialHUDWindow;
		gRecordingNotchBackground.blendingMode = NSVisualEffectBlendingModeWithinWindow;
		gRecordingNotchBackground.state = NSVisualEffectStateActive;
	}
}

static void ensureRecordingNotch(void) {
	if (gRecordingNotchWindow) return;

	NSRect frame = NSMakeRect(0, 0, 260, 42);
	gRecordingNotchWindow = [[NSPanel alloc] initWithContentRect:frame
		styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
		backing:NSBackingStoreBuffered
		defer:NO];
	gRecordingNotchWindow.releasedWhenClosed = NO;
	gRecordingNotchWindow.opaque = NO;
	gRecordingNotchWindow.backgroundColor = [NSColor clearColor];
	gRecordingNotchWindow.hasShadow = YES;
	gRecordingNotchWindow.level = NSStatusWindowLevel + 1;
	gRecordingNotchWindow.hidesOnDeactivate = NO;
	gRecordingNotchWindow.movable = NO;
	gRecordingNotchWindow.ignoresMouseEvents = YES;
	gRecordingNotchWindow.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorTransient;

	gRecordingNotchBackground = [NSVisualEffectView new];
	gRecordingNotchBackground.frame = ((NSView *)gRecordingNotchWindow.contentView).bounds;
	gRecordingNotchBackground.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
	gRecordingNotchBackground.wantsLayer = YES;
	gRecordingNotchBackground.layer.cornerRadius = 14.0;
	gRecordingNotchBackground.layer.masksToBounds = YES;
	[gRecordingNotchWindow setContentView:gRecordingNotchBackground];

	NSView *content = gRecordingNotchBackground;

	NSView *dot = [NSView new];
	dot.translatesAutoresizingMaskIntoConstraints = NO;
	dot.wantsLayer = YES;
	dot.layer.cornerRadius = 4.0;
	dot.layer.masksToBounds = YES;
	dot.layer.backgroundColor = [NSColor systemRedColor].CGColor;
	[content addSubview:dot];

	gRecordingNotchIconView = [NSImageView new];
	gRecordingNotchIconView.translatesAutoresizingMaskIntoConstraints = NO;
	gRecordingNotchIconView.imageScaling = NSImageScaleProportionallyDown;
	[content addSubview:gRecordingNotchIconView];

	gRecordingNotchLabel = [NSTextField labelWithString:@"Enregistrement en cours"];
	gRecordingNotchLabel.translatesAutoresizingMaskIntoConstraints = NO;
	gRecordingNotchLabel.font = [NSFont systemFontOfSize:12 weight:NSFontWeightSemibold];
	gRecordingNotchLabel.textColor = [NSColor whiteColor];
	gRecordingNotchLabel.lineBreakMode = NSLineBreakByTruncatingTail;
	[content addSubview:gRecordingNotchLabel];

	[NSLayoutConstraint activateConstraints:@[
		[dot.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:14],
		[dot.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
		[dot.widthAnchor constraintEqualToConstant:8],
		[dot.heightAnchor constraintEqualToConstant:8],

		[gRecordingNotchIconView.leadingAnchor constraintEqualToAnchor:dot.trailingAnchor constant:8],
		[gRecordingNotchIconView.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
		[gRecordingNotchIconView.widthAnchor constraintEqualToConstant:15],
		[gRecordingNotchIconView.heightAnchor constraintEqualToConstant:15],

		[gRecordingNotchLabel.leadingAnchor constraintEqualToAnchor:gRecordingNotchIconView.trailingAnchor constant:8],
		[gRecordingNotchLabel.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-14],
		[gRecordingNotchLabel.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
	]];

	updateRecordingNotchState();
	positionRecordingNotchWindow();
}

static void showRecordingNotch(void) {
	ensureRecordingNotch();
	updateRecordingNotchState();
	positionRecordingNotchWindow();
	[gRecordingNotchWindow orderFrontRegardless];
}

static void hideRecordingNotch(void) {
	if (!gRecordingNotchWindow) return;
	[gRecordingNotchWindow orderOut:nil];
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

	gPopoverAutoPasteCheckbox = [NSButton checkboxWithTitle:@"Auto-paste (Cmd+V)" target:gMenuHandler action:@selector(popoverToggleAutoPaste:)];
	gPopoverAutoPasteCheckbox.translatesAutoresizingMaskIntoConstraints = NO;

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
	[gPopoverStack addArrangedSubview:gPopoverAutoPasteCheckbox];
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
		return;
	}

	NSRect frame = NSMakeRect(0, 0, 460, 350);
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
	NSString *appBuild = [[[NSBundle mainBundle] objectForInfoDictionaryKey:@"CFBundleVersion"] description] ?: @"?";
	NSTextField *subtitle = [NSTextField labelWithString:[NSString stringWithFormat:@"%@\nLocale : %@\nVersion : %@ (build %@)", hotkey, locale, appVersion, appBuild]];
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

	NSTextField *placeholder = [NSTextField labelWithString:@"Autres réglages (raccourci, locale, etc.) : à venir"];
	placeholder.textColor = [NSColor tertiaryLabelColor];
	placeholder.font = [NSFont systemFontOfSize:12];
	placeholder.translatesAutoresizingMaskIntoConstraints = NO;
	[content addSubview:placeholder];

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

		[placeholder.topAnchor constraintEqualToAnchor:iconLabel.bottomAnchor constant:18],
		[placeholder.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:18],
		[placeholder.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-18],
	]];

	applySettingsTheme();
	[gSettingsWindow center];
	[NSApp activateIgnoringOtherApps:YES];
	[gSettingsWindow makeKeyAndOrderFront:gSettingsWindow];
}

static void stopRecording(void) {
	if (!gIsRecording) return;
	gIsRecording = false;
	updateStatusItemTitle();
	updateMenuState();
	hideRecordingNotch();

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
	showRecordingNotch();

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
		updateMenuState();
		hideRecordingNotch();
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
