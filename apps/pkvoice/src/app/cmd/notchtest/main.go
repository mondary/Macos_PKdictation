//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <math.h>

typedef NS_ENUM(NSInteger, PKNIconStyle) {
	PKNIconStyleWave = 0,
	PKNIconStyleMicro = 1,
};

typedef NS_ENUM(NSInteger, PKNSpinnerPattern) {
	PKNSpinnerPatternWave = 0,
	PKNSpinnerPatternSpinner = 1,
	PKNSpinnerPatternPulse = 2,
	PKNSpinnerPatternCross = 3,
	PKNSpinnerPatternBurst = 4,
	PKNSpinnerPatternArrowMove = 5,
	PKNSpinnerPatternSineWave = 6,
};

@interface PKNOverlayPanel : NSPanel
@end

@implementation PKNOverlayPanel
- (BOOL)canBecomeKeyWindow { return YES; }
- (BOOL)canBecomeMainWindow { return NO; }
@end

static NSPanel *gNotchWindow = nil;
static NSView *gNotchBackground = nil;
static NSView *gSpinnerContainer = nil;
static NSView *gSpinnerDots[9] = { nil };
static NSTimer *gSpinnerTimer = nil;
static NSInteger gSpinnerStep = 0;
static NSTextField *gNotchLabel = nil;
static NSWindow *gControlWindow = nil;
static NSSegmentedControl *gIconSegment = nil;
static NSPopUpButton *gPatternPopup = nil;
static NSColorWell *gAccentColorWell = nil;
static NSInteger gIconStyle = PKNIconStyleWave;
static NSInteger gSpinnerPattern = PKNSpinnerPatternSpinner;
static NSImage *gMicroIcon = nil;
static id gAppDelegate = nil;
static CFTimeInterval gSpinnerStartTime = 0;

static void ensureNotchWindow(void);
static void positionNotchWindow(void);
static void updateNotchAppearance(void);
static void updateSpinnerFrame(void);
static CGFloat spinnerIntensityForDot(NSInteger i, double tNorm);
static CFTimeInterval spinnerCycleDuration(void);
static void startSpinner(void);
static void stopSpinner(void);
static void showNotch(void);
static void hideNotch(void);
static void toggleNotch(void);
static void createControlWindow(id target);
static NSString *spinnerPatternTitle(void);
static NSColor *currentAccentColor(void);

static NSImage *loadMicroIcon(void) {
	if (gMicroIcon) return gMicroIcon;
	NSString *iconPath = [[NSBundle mainBundle] pathForResource:@"PKvoice" ofType:@"icns"];
	if (!iconPath) return nil;
	gMicroIcon = [[NSImage alloc] initWithContentsOfFile:iconPath];
	return gMicroIcon;
}

static NSImage *makeWaveIcon(void) {
	if (@available(macOS 11.0, *)) {
		NSImage *img = [NSImage imageWithSystemSymbolName:@"waveform" accessibilityDescription:@"Wave"];
		if (img) {
			[img setSize:NSMakeSize(18, 18)];
			img.template = YES;
			return img;
		}
	}
	return nil;
}

static NSImage *currentIcon(void) {
	if (gIconStyle == PKNIconStyleMicro) {
		NSImage *img = [loadMicroIcon() copy];
		if (img) {
			[img setSize:NSMakeSize(18, 18)];
			return img;
		}
	}
	NSImage *wave = makeWaveIcon();
	if (wave) return wave;
	NSImage *img = [loadMicroIcon() copy];
	if (img) [img setSize:NSMakeSize(18, 18)];
	return img;
}

static NSColor *spinnerBaseColor(void) {
	return [NSColor colorWithCalibratedWhite:0.20 alpha:1.0]; // #333333
}

static NSColor *presetAccentColor(void) {
	if (gIconStyle == PKNIconStyleMicro) {
		return [NSColor colorWithCalibratedRed:1.0 green:74.0/255.0 blue:74.0/255.0 alpha:1.0];
	}
	// Inspired by user's CSS sample (#FF14CC)
	return [NSColor colorWithCalibratedRed:1.0 green:20.0/255.0 blue:204.0/255.0 alpha:1.0];
}

static NSColor *currentAccentColor(void) {
	if (gAccentColorWell && gAccentColorWell.color) return gAccentColorWell.color;
	return presetAccentColor();
}

static NSColor *spinnerAccentColor(void) {
	return currentAccentColor();
}

static NSColor *spinnerGlowColor(void) {
	NSColor *accent = [currentAccentColor() colorUsingColorSpace:[NSColorSpace genericRGBColorSpace]];
	if (!accent) {
		if (gIconStyle == PKNIconStyleMicro) {
			return [NSColor colorWithCalibratedRed:1.0 green:170.0/255.0 blue:170.0/255.0 alpha:1.0];
		}
		return [NSColor colorWithCalibratedRed:1.0 green:163.0/255.0 blue:235.0/255.0 alpha:1.0];
	}
	CGFloat r = 1, g = 1, b = 1, a = 1;
	[accent getRed:&r green:&g blue:&b alpha:&a];
	CGFloat mix = 0.55; // blend toward white for glow
	return [NSColor colorWithCalibratedRed:(r * (1.0 - mix) + 1.0 * mix)
	                                 green:(g * (1.0 - mix) + 1.0 * mix)
	                                  blue:(b * (1.0 - mix) + 1.0 * mix)
	                                 alpha:1.0];
}

static NSString *spinnerPatternTitle(void) {
	switch (gSpinnerPattern) {
	case PKNSpinnerPatternWave: return @"Wave";
	case PKNSpinnerPatternSpinner: return @"Spinner";
	case PKNSpinnerPatternPulse: return @"Pulse";
	case PKNSpinnerPatternCross: return @"Cross";
	case PKNSpinnerPatternBurst: return @"Burst";
	case PKNSpinnerPatternArrowMove: return @"ArrowMove";
	case PKNSpinnerPatternSineWave: return @"Sine Wave";
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

static CFTimeInterval spinnerCycleDuration(void) {
	switch (gSpinnerPattern) {
	case PKNSpinnerPatternWave:
		return 1.50;
	case PKNSpinnerPatternSpinner:
		return 1.00;
	case PKNSpinnerPatternPulse:
		return 1.75;
	case PKNSpinnerPatternCross:
		return 1.30;
	case PKNSpinnerPatternBurst:
		return 1.25;
	case PKNSpinnerPatternArrowMove:
		return 1.75;
	case PKNSpinnerPatternSineWave:
		return 1.25;
	default:
		return 1.20;
	}
}

static CGFloat spinnerIntensityForDot(NSInteger i, double tNorm) {
	CGFloat t = (CGFloat)tNorm;
	NSInteger row = i / 3;
	NSInteger col = i % 3;
	BOOL isCenter = (i == 4);

	switch (gSpinnerPattern) {
	case PKNSpinnerPatternSpinner: {
		// CSS-inspired spinner: center always on, orthogonals rotate.
		if (isCenter) return 1.0;
		static const NSInteger ringDots[4] = {1, 5, 7, 3};
		static const CGFloat phases[4] = {0.00, 0.25, 0.50, 0.75};
		for (NSInteger k = 0; k < 4; k++) {
			if (i == ringDots[k]) return phasePulse(t, phases[k], 0.18);
		}
		return 0.0;
	}
	case PKNSpinnerPatternWave: {
		// Diagonal traveling wave across the 3x3 matrix.
		CGFloat phase = (CGFloat)(row + col) / 4.0;
		CGFloat a = phasePulse(t, phase, 0.16);
		CGFloat b = 0.45 * phasePulse(t, fmod(phase + 0.50, 1.0), 0.16);
		return clamp01(fmax(a, b));
	}
	case PKNSpinnerPatternPulse: {
		// Center pulse with delayed rings (orthogonals then corners).
		CGFloat delay = 0.0;
		if (!isCenter) {
			NSInteger manhattan = labs(row - 1) + labs(col - 1);
			delay = (manhattan == 1) ? 0.09 : 0.17;
		}
		return phasePulse(t, delay, isCenter ? 0.26 : 0.20);
	}
	case PKNSpinnerPatternCross: {
		// Alternate X and + groups with center bridging.
		static const NSInteger xDots[] = {0, 2, 6, 8};
		static const NSInteger plusDots[] = {1, 3, 5, 7};
		CGFloat xI = fmax(phasePulse(t, 0.00, 0.22), phasePulse(t, 0.50, 0.22));
		CGFloat plusI = fmax(phasePulse(t, 0.25, 0.20), phasePulse(t, 0.75, 0.20));
		if (isCenter) return clamp01(fmax(xI, plusI));
		if (dotInList(i, xDots, 4)) return xI;
		if (dotInList(i, plusDots, 4)) return plusI;
		return 0.0;
	}
	case PKNSpinnerPatternBurst: {
		// Center -> orthogonals -> corners burst.
		CGFloat phase = 0.0;
		if (isCenter) phase = 0.00;
		else {
			NSInteger manhattan = labs(row - 1) + labs(col - 1);
			phase = (manhattan == 1) ? 0.12 : 0.22;
		}
		return phasePulse(t, phase, isCenter ? 0.20 : 0.18);
	}
	case PKNSpinnerPatternArrowMove: {
		// Three right-pointing arrow positions moving left -> center -> right.
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
	case PKNSpinnerPatternSineWave: {
		// Column-based phase shift to mimic a sine wave motion across matrix.
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

static void updateSpinnerFrame(void) {
	if (!gSpinnerContainer) return;
	if (gSpinnerStartTime <= 0) gSpinnerStartTime = CACurrentMediaTime();
	CFTimeInterval now = CACurrentMediaTime();
	CFTimeInterval dur = spinnerCycleDuration();
	double tNorm = (dur > 0) ? fmod((now - gSpinnerStartTime) / dur, 1.0) : 0.0;
	for (NSInteger i = 0; i < 9; i++) {
		applyDotStyle(gSpinnerDots[i], spinnerIntensityForDot(i, tNorm));
	}
}

static void startSpinner(void) {
	if (gSpinnerTimer) return;
	gSpinnerStartTime = CACurrentMediaTime();
	updateSpinnerFrame();
	gSpinnerTimer = [NSTimer scheduledTimerWithTimeInterval:(1.0 / 30.0) repeats:YES block:^(__unused NSTimer *timer) {
		updateSpinnerFrame();
	}];
	[[NSRunLoop mainRunLoop] addTimer:gSpinnerTimer forMode:NSRunLoopCommonModes];
}

static void stopSpinner(void) {
	if (gSpinnerTimer) {
		[gSpinnerTimer invalidate];
		gSpinnerTimer = nil;
	}
	gSpinnerStartTime = 0;
	for (NSInteger i = 0; i < 9; i++) {
		applyDotStyle(gSpinnerDots[i], 0.0);
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
	// Keep it clearly below the menu bar / hardware notch for testing visibility.
	CGFloat y = round(NSMaxY(visible) - frame.size.height - 36.0);
	[gNotchWindow setFrame:NSMakeRect(x, y, frame.size.width, frame.size.height) display:NO];
}

static void updateNotchAppearance(void) {
	if (!gNotchWindow) return;
	if (gNotchLabel) {
		gNotchLabel.stringValue = [NSString stringWithFormat:@"Listening… · %@", spinnerPatternTitle()];
	}
}

static void ensureNotchWindow(void) {
	if (gNotchWindow) return;

	NSRect frame = NSMakeRect(0, 0, 316, 52);
	gNotchWindow = [[PKNOverlayPanel alloc] initWithContentRect:frame
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
	gNotchWindow.becomesKeyOnlyIfNeeded = YES;

	gNotchBackground = [NSView new];
	gNotchBackground.frame = ((NSView *)gNotchWindow.contentView).bounds;
	gNotchBackground.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
	gNotchBackground.wantsLayer = YES;
	gNotchBackground.layer.cornerRadius = 16.0;
	gNotchBackground.layer.masksToBounds = YES;
	gNotchBackground.layer.backgroundColor = [NSColor colorWithCalibratedWhite:0.08 alpha:0.95].CGColor;
	gNotchBackground.layer.borderWidth = 1.0;
	gNotchBackground.layer.borderColor = [NSColor colorWithCalibratedWhite:1.0 alpha:0.22].CGColor;
	[gNotchWindow setContentView:gNotchBackground];

	NSView *content = gNotchBackground;

	gSpinnerContainer = [NSView new];
	gSpinnerContainer.translatesAutoresizingMaskIntoConstraints = NO;
	gSpinnerContainer.wantsLayer = YES;
	[content addSubview:gSpinnerContainer];

	const CGFloat dotSize = 5.0;
	const CGFloat gap = 1.0;
	for (NSInteger i = 0; i < 9; i++) {
		NSInteger row = i / 3;
		NSInteger col = i % 3;
		NSView *d = [[NSView alloc] initWithFrame:NSMakeRect(col * (dotSize + gap), (2 - row) * (dotSize + gap), dotSize, dotSize)];
		d.wantsLayer = YES;
		d.layer.cornerRadius = 1.0;
		d.layer.masksToBounds = NO;
		d.layer.backgroundColor = spinnerBaseColor().CGColor;
		gSpinnerDots[i] = d;
		[gSpinnerContainer addSubview:d];
	}

	gNotchLabel = [NSTextField labelWithString:@"Notch test"];
	gNotchLabel.translatesAutoresizingMaskIntoConstraints = NO;
	gNotchLabel.font = [NSFont systemFontOfSize:12 weight:NSFontWeightSemibold];
	gNotchLabel.textColor = [NSColor whiteColor];
	gNotchLabel.lineBreakMode = NSLineBreakByTruncatingTail;
	[content addSubview:gNotchLabel];

	[NSLayoutConstraint activateConstraints:@[
		[gSpinnerContainer.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:14],
		[gSpinnerContainer.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
		[gSpinnerContainer.widthAnchor constraintEqualToConstant:17],
		[gSpinnerContainer.heightAnchor constraintEqualToConstant:17],

		[gNotchLabel.leadingAnchor constraintEqualToAnchor:gSpinnerContainer.trailingAnchor constant:9],
		[gNotchLabel.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-14],
		[gNotchLabel.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
	]];

	updateNotchAppearance();
	stopSpinner();
	positionNotchWindow();
}

static void showNotch(void) {
	ensureNotchWindow();
	updateNotchAppearance();
	positionNotchWindow();
	startSpinner();
	[gNotchWindow orderFrontRegardless];
	[gNotchWindow makeKeyAndOrderFront:nil];
}

static void hideNotch(void) {
	if (!gNotchWindow) return;
	stopSpinner();
	[gNotchWindow orderOut:nil];
}

static void toggleNotch(void) {
	ensureNotchWindow();
	if (gNotchWindow.visible) {
		hideNotch();
	} else {
		showNotch();
	}
}

@interface PKNHandler : NSObject <NSApplicationDelegate>
@end

@implementation PKNHandler
- (void)applicationDidFinishLaunching:(NSNotification *)notification {
	(void)notification;
	@try {
		NSLog(@"[PKvoiceNotchTest] didFinishLaunching");
		createControlWindow(self);
		dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(250 * NSEC_PER_MSEC)), dispatch_get_main_queue(), ^{
			@try {
				NSLog(@"[PKvoiceNotchTest] showing notch");
				showNotch();
			} @catch (NSException *e) {
				NSLog(@"[PKvoiceNotchTest] showNotch exception: %@ %@", e.name, e.reason);
			}
		});
		[NSApp activateIgnoringOtherApps:YES];
	} @catch (NSException *e) {
		NSLog(@"[PKvoiceNotchTest] launch exception: %@ %@", e.name, e.reason);
	}
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)sender {
	(void)sender;
	return YES;
}

- (void)showNotchAction:(id)sender {
	(void)sender;
	showNotch();
}

- (void)hideNotchAction:(id)sender {
	(void)sender;
	hideNotch();
}

- (void)toggleNotchAction:(id)sender {
	(void)sender;
	toggleNotch();
}

- (void)repositionAction:(id)sender {
	(void)sender;
	positionNotchWindow();
	if (gNotchWindow && gNotchWindow.visible) [gNotchWindow orderFrontRegardless];
}

- (void)iconStyleChanged:(id)sender {
	NSSegmentedControl *seg = (NSSegmentedControl *)sender;
	if (![seg isKindOfClass:[NSSegmentedControl class]]) return;
	gIconStyle = seg.selectedSegment;
	if (gAccentColorWell) {
		gAccentColorWell.color = presetAccentColor();
	}
	updateNotchAppearance();
	updateSpinnerFrame();
	if (gNotchWindow && gNotchWindow.visible) [gNotchWindow orderFrontRegardless];
}

- (void)patternChanged:(id)sender {
	NSPopUpButton *popup = (NSPopUpButton *)sender;
	if (![popup isKindOfClass:[NSPopUpButton class]]) return;
	NSInteger idx = popup.indexOfSelectedItem;
	if (idx < 0) return;
	gSpinnerPattern = idx;
	gSpinnerStartTime = CACurrentMediaTime();
	updateNotchAppearance();
	updateSpinnerFrame();
	if (gNotchWindow && gNotchWindow.visible) [gNotchWindow orderFrontRegardless];
}

- (void)accentColorChanged:(id)sender {
	NSColorWell *well = (NSColorWell *)sender;
	if (![well isKindOfClass:[NSColorWell class]]) return;
	if (well.color) {
		gAccentColorWell.color = well.color;
	}
	updateSpinnerFrame();
	if (gNotchWindow && gNotchWindow.visible) [gNotchWindow orderFrontRegardless];
}

- (void)quitAction:(id)sender {
	(void)sender;
	[NSApp terminate:nil];
}
@end

static NSButton *makeBtn(NSString *title, id target, SEL action, NSRect frame) {
	NSButton *b = [NSButton buttonWithTitle:title target:target action:action];
	b.frame = frame;
	b.bezelStyle = NSBezelStyleRounded;
	return b;
}

static void createControlWindow(id target) {
	if (gControlWindow) return;

	NSRect frame = NSMakeRect(0, 0, 420, 275);
	gControlWindow = [[NSWindow alloc] initWithContentRect:frame
		styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable
		backing:NSBackingStoreBuffered
		defer:NO];
	gControlWindow.title = @"PKvoice Notch Test";
	gControlWindow.releasedWhenClosed = NO;

	NSView *content = gControlWindow.contentView;

	NSTextField *title = [NSTextField labelWithString:@"Tester le notch indépendamment de PKvoice"];
	title.font = [NSFont boldSystemFontOfSize:14];
	title.frame = NSMakeRect(20, 231, 380, 22);
	[content addSubview:title];

	NSTextField *hint = [NSTextField labelWithString:@"Le notch s'affiche en haut-centre (abaissé pour le test). Utilise les boutons ci-dessous pour le montrer / cacher."];
	hint.font = [NSFont systemFontOfSize:12];
	hint.textColor = [NSColor secondaryLabelColor];
	hint.frame = NSMakeRect(20, 209, 380, 18);
	[content addSubview:hint];

	NSTextField *iconLabel = [NSTextField labelWithString:@"Couleur"];
	iconLabel.frame = NSMakeRect(20, 172, 80, 20);
	[content addSubview:iconLabel];

	gIconSegment = [NSSegmentedControl segmentedControlWithLabels:@[ @"Wave", @"Micro" ]
		trackingMode:NSSegmentSwitchTrackingSelectOne
		target:target
		action:@selector(iconStyleChanged:)];
	gIconSegment.frame = NSMakeRect(100, 168, 150, 28);
	gIconSegment.selectedSegment = gIconStyle;

	if (@available(macOS 11.0, *)) {
		NSImage *wave = [NSImage imageWithSystemSymbolName:@"waveform" accessibilityDescription:@"Wave"];
		if (wave) {
			[wave setSize:NSMakeSize(16, 16)];
			[gIconSegment setLabel:@"" forSegment:0];
			[gIconSegment setImage:wave forSegment:0];
		}
	}
	NSImage *micro = [loadMicroIcon() copy];
	if (micro) {
		[micro setSize:NSMakeSize(16, 16)];
		[gIconSegment setLabel:@"" forSegment:1];
		[gIconSegment setImage:micro forSegment:1];
	}
	[content addSubview:gIconSegment];

	NSTextField *accentLabel = [NSTextField labelWithString:@"Couleur"];
	accentLabel.frame = NSMakeRect(20, 136, 80, 20);
	[content addSubview:accentLabel];

	gAccentColorWell = [[NSColorWell alloc] initWithFrame:NSMakeRect(100, 132, 60, 28)];
	gAccentColorWell.color = presetAccentColor();
	gAccentColorWell.target = target;
	gAccentColorWell.action = @selector(accentColorChanged:);
	[content addSubview:gAccentColorWell];

	NSTextField *patternLabel = [NSTextField labelWithString:@"Pattern"];
	patternLabel.frame = NSMakeRect(20, 94, 80, 20);
	[content addSubview:patternLabel];

	gPatternPopup = [[NSPopUpButton alloc] initWithFrame:NSMakeRect(100, 90, 200, 28) pullsDown:NO];
	[gPatternPopup addItemsWithTitles:@[
		@"Wave",
		@"Spinner",
		@"Pulse",
		@"Cross",
		@"Burst",
		@"ArrowMove",
		@"Sine Wave"
	]];
	[gPatternPopup selectItemAtIndex:gSpinnerPattern];
	gPatternPopup.target = target;
	gPatternPopup.action = @selector(patternChanged:);
	[content addSubview:gPatternPopup];

	[content addSubview:makeBtn(@"Afficher", target, @selector(showNotchAction:), NSMakeRect(20, 34, 90, 30))];
	[content addSubview:makeBtn(@"Masquer", target, @selector(hideNotchAction:), NSMakeRect(120, 34, 90, 30))];
	[content addSubview:makeBtn(@"Toggle", target, @selector(toggleNotchAction:), NSMakeRect(220, 34, 90, 30))];
	[content addSubview:makeBtn(@"Recentrer", target, @selector(repositionAction:), NSMakeRect(320, 34, 80, 30))];

	[gControlWindow center];
	[gControlWindow makeKeyAndOrderFront:nil];
}

static void runApp(void) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
		gAppDelegate = [PKNHandler new];
		[NSApp setDelegate:gAppDelegate];
		[NSApp run];
	}
}
*/
import "C"

func main() {
	C.runApp()
}
