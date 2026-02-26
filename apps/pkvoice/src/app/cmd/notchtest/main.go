//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

typedef NS_ENUM(NSInteger, PKNIconStyle) {
	PKNIconStyleWave = 0,
	PKNIconStyleMicro = 1,
};

static NSPanel *gNotchWindow = nil;
static NSView *gNotchBackground = nil;
static NSImageView *gNotchIconView = nil;
static NSTextField *gNotchLabel = nil;
static NSWindow *gControlWindow = nil;
static NSSegmentedControl *gIconSegment = nil;
static NSInteger gIconStyle = PKNIconStyleWave;
static NSImage *gMicroIcon = nil;

static void ensureNotchWindow(void);
static void positionNotchWindow(void);
static void updateNotchAppearance(void);
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
	if (gNotchIconView) {
		gNotchIconView.image = currentIcon();
		gNotchIconView.contentTintColor = (gIconStyle == PKNIconStyleWave) ? [NSColor whiteColor] : nil;
	}
	if (gNotchLabel) {
		gNotchLabel.stringValue = (gIconStyle == PKNIconStyleWave) ? @"Notch test · Wave" : @"Notch test · Micro";
	}
}

static void ensureNotchWindow(void) {
	if (gNotchWindow) return;

	NSRect frame = NSMakeRect(0, 0, 300, 48);
	gNotchWindow = [[NSPanel alloc] initWithContentRect:frame
		styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
		backing:NSBackingStoreBuffered
		defer:NO];
	gNotchWindow.releasedWhenClosed = NO;
	gNotchWindow.opaque = NO;
	gNotchWindow.backgroundColor = [NSColor clearColor];
	gNotchWindow.hasShadow = YES;
	gNotchWindow.level = NSStatusWindowLevel;
	gNotchWindow.hidesOnDeactivate = NO;
	gNotchWindow.ignoresMouseEvents = YES;
	gNotchWindow.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces | NSWindowCollectionBehaviorMoveToActiveSpace | NSWindowCollectionBehaviorTransient | NSWindowCollectionBehaviorFullScreenAuxiliary;

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

	NSView *dot = [NSView new];
	dot.translatesAutoresizingMaskIntoConstraints = NO;
	dot.wantsLayer = YES;
	dot.layer.cornerRadius = 4.5;
	dot.layer.masksToBounds = YES;
	dot.layer.backgroundColor = [NSColor systemRedColor].CGColor;
	[content addSubview:dot];

	gNotchIconView = [NSImageView new];
	gNotchIconView.translatesAutoresizingMaskIntoConstraints = NO;
	gNotchIconView.imageScaling = NSImageScaleProportionallyDown;
	[content addSubview:gNotchIconView];

	gNotchLabel = [NSTextField labelWithString:@"Notch test"];
	gNotchLabel.translatesAutoresizingMaskIntoConstraints = NO;
	gNotchLabel.font = [NSFont systemFontOfSize:12 weight:NSFontWeightSemibold];
	gNotchLabel.textColor = [NSColor whiteColor];
	gNotchLabel.lineBreakMode = NSLineBreakByTruncatingTail;
	[content addSubview:gNotchLabel];

	[NSLayoutConstraint activateConstraints:@[
		[dot.leadingAnchor constraintEqualToAnchor:content.leadingAnchor constant:14],
		[dot.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
		[dot.widthAnchor constraintEqualToConstant:9],
		[dot.heightAnchor constraintEqualToConstant:9],

		[gNotchIconView.leadingAnchor constraintEqualToAnchor:dot.trailingAnchor constant:9],
		[gNotchIconView.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
		[gNotchIconView.widthAnchor constraintEqualToConstant:18],
		[gNotchIconView.heightAnchor constraintEqualToConstant:18],

		[gNotchLabel.leadingAnchor constraintEqualToAnchor:gNotchIconView.trailingAnchor constant:8],
		[gNotchLabel.trailingAnchor constraintEqualToAnchor:content.trailingAnchor constant:-14],
		[gNotchLabel.centerYAnchor constraintEqualToAnchor:content.centerYAnchor],
	]];

	updateNotchAppearance();
	positionNotchWindow();
}

static void showNotch(void) {
	ensureNotchWindow();
	updateNotchAppearance();
	positionNotchWindow();
	[gNotchWindow orderFrontRegardless];
}

static void hideNotch(void) {
	if (!gNotchWindow) return;
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
	createControlWindow(self);
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(250 * NSEC_PER_MSEC)), dispatch_get_main_queue(), ^{
		showNotch();
	});
	[NSApp activateIgnoringOtherApps:YES];
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

	NSTextField *iconLabel = [NSTextField labelWithString:@"Icône"];
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
		PKNHandler *handler = [PKNHandler new];
		[NSApp setDelegate:handler];
		[NSApp run];
	}
}
*/
import "C"

func main() {
	C.runApp()
}
