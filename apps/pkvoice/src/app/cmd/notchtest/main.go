//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

typedef NS_ENUM(NSInteger, PKNIconStyle) {
	PKNIconStyleWave = 0,
	PKNIconStyleMicro = 1,
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
static NSInteger gIconStyle = PKNIconStyleWave;
static NSImage *gMicroIcon = nil;
static id gAppDelegate = nil;

static void ensureNotchWindow(void);
static void positionNotchWindow(void);
static void updateNotchAppearance(void);
static void updateSpinnerFrame(void);
static void startSpinner(void);
static void stopSpinner(void);
static void showNotch(void);
static void hideNotch(void);
static void toggleNotch(void);
static void createControlWindow(id target);

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

static NSColor *spinnerAccentColor(void) {
	if (gIconStyle == PKNIconStyleMicro) {
		return [NSColor colorWithCalibratedRed:1.0 green:74.0/255.0 blue:74.0/255.0 alpha:1.0];
	}
	// Inspired by user's CSS sample (#FF14CC)
	return [NSColor colorWithCalibratedRed:1.0 green:20.0/255.0 blue:204.0/255.0 alpha:1.0];
}

static NSColor *spinnerGlowColor(void) {
	if (gIconStyle == PKNIconStyleMicro) {
		return [NSColor colorWithCalibratedRed:1.0 green:170.0/255.0 blue:170.0/255.0 alpha:1.0];
	}
	return [NSColor colorWithCalibratedRed:1.0 green:163.0/255.0 blue:235.0/255.0 alpha:1.0];
}

static void applyDotStyle(NSView *dot, BOOL active) {
	if (!dot || !dot.layer) return;
	dot.layer.backgroundColor = (active ? spinnerAccentColor() : spinnerBaseColor()).CGColor;
	dot.layer.shadowColor = spinnerGlowColor().CGColor;
	dot.layer.shadowOpacity = active ? 0.95 : 0.0;
	dot.layer.shadowRadius = active ? 6.0 : 0.0;
	dot.layer.shadowOffset = CGSizeZero;
	dot.layer.transform = active ? CATransform3DMakeScale(1.12, 1.12, 1.0) : CATransform3DIdentity;
}

static void updateSpinnerFrame(void) {
	if (!gSpinnerContainer) return;
	static const NSInteger order[8] = {0, 1, 2, 5, 8, 7, 6, 3}; // clockwise perimeter
	NSInteger activeIdx = order[(NSUInteger)(gSpinnerStep % 8)];

	for (NSInteger i = 0; i < 9; i++) {
		BOOL active = (i == activeIdx);
		if (i == 4) active = NO; // center stays muted like a matrix hub
		applyDotStyle(gSpinnerDots[i], active);
	}

	gSpinnerStep = (gSpinnerStep + 1) % 8;
}

static void startSpinner(void) {
	if (gSpinnerTimer) return;
	updateSpinnerFrame();
	gSpinnerTimer = [NSTimer scheduledTimerWithTimeInterval:0.25 repeats:YES block:^(__unused NSTimer *timer) {
		updateSpinnerFrame();
	}];
	[[NSRunLoop mainRunLoop] addTimer:gSpinnerTimer forMode:NSRunLoopCommonModes];
}

static void stopSpinner(void) {
	if (gSpinnerTimer) {
		[gSpinnerTimer invalidate];
		gSpinnerTimer = nil;
	}
	for (NSInteger i = 0; i < 9; i++) {
		applyDotStyle(gSpinnerDots[i], NO);
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
		gNotchLabel.stringValue = @"Listening…";
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

	const CGFloat dotSize = 6.0;
	const CGFloat gap = 2.0;
	for (NSInteger i = 0; i < 9; i++) {
		NSInteger row = i / 3;
		NSInteger col = i % 3;
		NSView *d = [[NSView alloc] initWithFrame:NSMakeRect(col * (dotSize + gap), (2 - row) * (dotSize + gap), dotSize, dotSize)];
		d.wantsLayer = YES;
		d.layer.cornerRadius = 1.5;
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
		[gSpinnerContainer.widthAnchor constraintEqualToConstant:22],
		[gSpinnerContainer.heightAnchor constraintEqualToConstant:22],

		[gNotchLabel.leadingAnchor constraintEqualToAnchor:gSpinnerContainer.trailingAnchor constant:10],
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
	updateNotchAppearance();
	// Reuse the selector to preview two spinner accents.
	if (gIconStyle == PKNIconStyleMicro) {
		for (NSInteger i = 0; i < 9; i++) {
			if (gSpinnerDots[i] && gSpinnerDots[i].layer) {
				gSpinnerDots[i].layer.borderWidth = 0.0;
			}
		}
	}
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

	NSRect frame = NSMakeRect(0, 0, 420, 190);
	gControlWindow = [[NSWindow alloc] initWithContentRect:frame
		styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable
		backing:NSBackingStoreBuffered
		defer:NO];
	gControlWindow.title = @"PKvoice Notch Test";
	gControlWindow.releasedWhenClosed = NO;

	NSView *content = gControlWindow.contentView;

	NSTextField *title = [NSTextField labelWithString:@"Tester le notch indépendamment de PKvoice"];
	title.font = [NSFont boldSystemFontOfSize:14];
	title.frame = NSMakeRect(20, 146, 380, 22);
	[content addSubview:title];

	NSTextField *hint = [NSTextField labelWithString:@"Le notch s'affiche en haut-centre (abaissé pour le test). Utilise les boutons ci-dessous pour le montrer / cacher."];
	hint.font = [NSFont systemFontOfSize:12];
	hint.textColor = [NSColor secondaryLabelColor];
	hint.frame = NSMakeRect(20, 124, 380, 18);
	[content addSubview:hint];

	NSTextField *iconLabel = [NSTextField labelWithString:@"Style spinner"];
	iconLabel.frame = NSMakeRect(20, 92, 80, 20);
	[content addSubview:iconLabel];

	gIconSegment = [NSSegmentedControl segmentedControlWithLabels:@[ @"Wave", @"Micro" ]
		trackingMode:NSSegmentSwitchTrackingSelectOne
		target:target
		action:@selector(iconStyleChanged:)];
	gIconSegment.frame = NSMakeRect(100, 88, 150, 28);
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
