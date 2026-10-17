// Deprecated WebView compatibility renderer. This is a separate App Sandbox
// executable, never the application's WKWebView and never an in-process fallback.
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <Security/Security.h>
#import <unistd.h>
#import <signal.h>
#import <sys/resource.h>
#import <math.h>
#import <CoreGraphics/CoreGraphics.h>

static NSString *nonce;
static void fail(NSString *why) { fprintf(stderr, "macOS preview: %s\n", why.UTF8String); exit(1); }
static NSData *readExact(NSUInteger length) {
    NSMutableData *data = [NSMutableData dataWithLength:length]; NSUInteger offset = 0;
    while (offset < length) { ssize_t n = read(STDIN_FILENO, (char *)data.mutableBytes + offset, length-offset); if (n <= 0) fail(@"parent pipe closed"); offset += n; }
    return data;
}
static void writeData(NSData *data) {
    NSUInteger offset=0; while(offset<data.length) { ssize_t n=write(STDOUT_FILENO,(const char *)data.bytes+offset,data.length-offset); if(n<=0) exit(1);offset+=n; }
}
static NSDictionary *readPacket(void) {
    NSData *prefix=readExact(4); uint32_t n; memcpy(&n,prefix.bytes,4); n=CFSwapInt32LittleToHost(n);
    if(n<1 || n>4096) fail(@"invalid IPC header length");
    id value=[NSJSONSerialization JSONObjectWithData:readExact(n) options:0 error:nil];
    if(![value isKindOfClass:NSDictionary.class] || ![value[@"version"] isEqual:@1] || ![value[@"nonce"] isEqual:nonce]) fail(@"invalid IPC header");
    return value;
}
static void packet(NSString *kind, NSDictionary *fields) {
    NSMutableDictionary *p=[@{@"version":@1,@"kind":kind,@"nonce":nonce} mutableCopy]; [p addEntriesFromDictionary:fields];
    NSData *raw=[NSJSONSerialization dataWithJSONObject:p options:0 error:nil]; if(!raw || raw.length>4096) fail(@"oversize IPC header");
    uint32_t n=CFSwapInt32HostToLittle((uint32_t)raw.length); writeData([NSData dataWithBytes:&n length:4]);writeData(raw);
}
static NSDictionary *signingInfo(SecCodeRef code) {
    CFDictionaryRef info=NULL;
    if(SecCodeCheckValidity(code,kSecCSStrictValidate,NULL)!=errSecSuccess || SecCodeCopySigningInformation(code,kSecCSSigningInformation,&info)!=errSecSuccess) fail(@"invalid live code signature");
    return CFBridgingRelease(info);
}
static void requireCode(SecCodeRef code, NSString *identifier, NSString *team) {
    NSString *text=[NSString stringWithFormat:@"anchor apple generic and identifier \"%@\" and certificate leaf[subject.OU] = \"%@\"",identifier,team];
    SecRequirementRef requirement=NULL;
    if(SecRequirementCreateWithString((__bridge CFStringRef)text,kSecCSDefaultFlags,&requirement)!=errSecSuccess) fail(@"invalid code requirement");
    OSStatus status=SecCodeCheckValidity(code,kSecCSStrictValidate,requirement);CFRelease(requirement);
    if(status!=errSecSuccess) fail(@"untrusted process identity");
}
static void verifyIdentity(void) {
    SecCodeRef self=NULL; if(SecCodeCopySelf(kSecCSDefaultFlags,&self)!=errSecSuccess) fail(@"cannot identify helper");
    NSDictionary *info=signingInfo(self);
    NSString *team=info[(__bridge NSString *)kSecCodeInfoTeamIdentifier];
    NSRegularExpression *pattern=[NSRegularExpression regularExpressionWithPattern:@"^[A-Z0-9]{10}$" options:0 error:nil];
    if(![team isKindOfClass:NSString.class] || [pattern numberOfMatchesInString:team options:0 range:NSMakeRange(0,team.length)]!=1) fail(@"missing signing team");
    requireCode(self,@"org.whaleisle.preview",team);
    NSDictionary *ent=info[(__bridge NSString *)kSecCodeInfoEntitlementsDict];
    id sandbox=ent[@"com.apple.security.app-sandbox"];
    if(![ent isKindOfClass:NSDictionary.class] || ent.count!=1 || sandbox!=(__bridge id)kCFBooleanTrue) fail(@"helper requires exactly App Sandbox, without network or file grants");
    CFRelease(self);
    pid_t parent=getppid();if(parent<=1) fail(@"missing parent");
    NSDictionary *attrs=@{(__bridge NSString *)kSecGuestAttributePid:@(parent)};SecCodeRef host=NULL;
    if(SecCodeCopyGuestWithAttributes(NULL,(__bridge CFDictionaryRef)attrs,kSecCSDefaultFlags,&host)!=errSecSuccess) fail(@"cannot identify parent");
    requireCode(host,@"ai.deepseek.harness.gui",team);CFRelease(host);
}

@interface Preview : NSObject <NSWindowDelegate>
@property(strong) NSWindow *window;
@property(strong) WebView *web;
@property BOOL initialNavigation;
@property BOOL capture;
@property BOOL inlineSeat;
@property BOOL openUI;
@property NSUInteger pressedButtons;
@property(strong) NSMutableDictionary<NSNumber *,NSView *> *mouseTargets;
@property NSPoint pointer;
@property BOOL loaded;
@property NSInteger width;
@property NSTimeInterval deadline;
@end
@implementation Preview
- (NSData *)framePNG {
    NSBitmapImageRep *bitmap=[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:self.width pixelsHigh:700 bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
    [self.web displayIfNeeded];[self.web cacheDisplayInRect:NSMakeRect(0,0,self.width,700) toBitmapImageRep:bitmap];
    NSData *png=[bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    if(!png || png.length>8*1024*1024) fail(@"invalid inline preview frame");
    return png;
}
// Only the fixed trusted parent-shell queue is inspected. The opaque viewer
// has no native binding, filesystem/clipboard/URL API, or direct IPC access.
- (void)emitOpenUIEvents {
    if(!self.openUI || !self.loaded) return;
    NSString *json=[self.web stringByEvaluatingJavaScriptFromString:@"(()=>{if(window.__previewState&&window.__previewState.error)throw new Error('OpenUI shell failed');return JSON.stringify(window.__whaleOpenUIDrain());})()"];
    if(!json || [json lengthOfBytesUsingEncoding:NSUTF8StringEncoding]>16*(65536+1)+2) fail(@"invalid OpenUI event batch");
    id batch=[NSJSONSerialization JSONObjectWithData:[json dataUsingEncoding:NSUTF8StringEncoding] options:0 error:nil];
    if(![batch isKindOfClass:NSArray.class] || [batch count]>16) fail(@"invalid OpenUI event count");
    for(id event in batch) {
        if(![event isKindOfClass:NSDictionary.class]) fail(@"invalid OpenUI event");
        NSData *raw=[NSJSONSerialization dataWithJSONObject:event options:0 error:nil];
        if(!raw || raw.length<1 || raw.length>65536) fail(@"invalid OpenUI event size");
        packet(@"openui-event",@{@"bytes":@(raw.length)});writeData(raw);
    }
}
- (void)pollOpenUI:(NSTimer *)timer { [self emitOpenUIEvents]; }
- (void)emitFrame {
    NSData *png=[self framePNG];
    packet(@"frame",@{@"bytes":@(png.length),@"width":@(self.width),@"height":@700});writeData(png);
}
- (void)mouseButton:(NSInteger)button down:(BOOL)down {
    NSEventType type;
    if(button==1) type=down?NSEventTypeLeftMouseDown:NSEventTypeLeftMouseUp;
    else if(button==3) type=down?NSEventTypeRightMouseDown:NSEventTypeRightMouseUp;
    else type=down?NSEventTypeOtherMouseDown:NSEventTypeOtherMouseUp;
    NSUInteger bit=1UL<<button;if(down) self.pressedButtons|=bit;else self.pressedButtons&=~bit;
    NSEvent *event=[NSEvent mouseEventWithType:type location:self.pointer modifierFlags:0 timestamp:NSProcessInfo.processInfo.systemUptime windowNumber:self.window.windowNumber context:nil eventNumber:0 clickCount:1 pressure:down?1:0];
    // Dispatch only to this helper's view. A never-key hidden window must not
    // consume the first click for activation; drag/up retain the down target.
    if(!self.mouseTargets) self.mouseTargets=[NSMutableDictionary dictionary];
    NSView *target=self.mouseTargets[@(button)];
    if(down) {
        target=[self.web hitTest:[self.web.superview convertPoint:self.pointer fromView:nil]];
        if(target) {self.mouseTargets[@(button)]=target;[self.window makeFirstResponder:target];}
    }
    if(button==1) {if(down) [target mouseDown:event];else [target mouseUp:event];}
    else if(button==3) {if(down) [target rightMouseDown:event];else [target rightMouseUp:event];}
    else {if(down) [target otherMouseDown:event];else [target otherMouseUp:event];}
    if(!down) [self.mouseTargets removeObjectForKey:@(button)];
}
- (void)releaseInput {
    for(NSInteger button=1;button<=3;button++) if(self.pressedButtons&(1UL<<button)) [self mouseButton:button down:NO];
}
- (void)input:(NSDictionary *)v {
    if(![v isKindOfClass:NSDictionary.class]) fail(@"invalid inline input");
    NSString *kind=v[@"kind"];
    if([kind isEqual:@"pointer"]) {
        NSInteger x=[v[@"x"] integerValue],y=[v[@"y"] integerValue],button=[v[@"button"] integerValue];
        if(x<0 || x>=self.width || y<0 || y>=700 || button<0 || button>5) fail(@"invalid pointer input");
        self.pointer=NSMakePoint(x+0.5,699.5-y);
        if(button==4 || button==5) {
            CGEventRef cg=CGEventCreateScrollWheelEvent(NULL,kCGScrollEventUnitPixel,1,button==4?64:-64);
            if(!cg) fail(@"cannot construct scroll input");
            NSPoint screen=[self.window convertPointToScreen:self.pointer];
            NSScreen *primary=NSScreen.screens.firstObject;if(!primary) fail(@"no screen for local scroll coordinates");
            CGEventSetLocation(cg,CGPointMake(screen.x,NSMaxY(primary.frame)-screen.y));
            CGEventSetIntegerValueField(cg,kCGMouseEventWindowUnderMousePointer,self.window.windowNumber);
            CGEventSetIntegerValueField(cg,kCGMouseEventWindowUnderMousePointerThatCanHandleThisEvent,self.window.windowNumber);
            NSEvent *event=[NSEvent eventWithCGEvent:cg];CFRelease(cg);
            NSPoint location=event.locationInWindow;
            if(event.windowNumber!=self.window.windowNumber || fabs(location.x-self.pointer.x)>1 || fabs(location.y-self.pointer.y)>1) fail(@"native scroll event could not bind to helper window");
            NSView *target=[self.web hitTest:[self.web.superview convertPoint:self.pointer fromView:nil]];
            [target scrollWheel:event];return;
        }
        if(button) {[self mouseButton:button down:[v[@"down"] boolValue]];return;}
        NSEventType type=NSEventTypeMouseMoved;
        if(self.pressedButtons&(1UL<<1)) type=NSEventTypeLeftMouseDragged;
        else if(self.pressedButtons&(1UL<<3)) type=NSEventTypeRightMouseDragged;
        else if(self.pressedButtons&(1UL<<2)) type=NSEventTypeOtherMouseDragged;
        NSEvent *event=[NSEvent mouseEventWithType:type location:self.pointer modifierFlags:0 timestamp:NSProcessInfo.processInfo.systemUptime windowNumber:self.window.windowNumber context:nil eventNumber:0 clickCount:0 pressure:0];
        NSView *target;
        if(type==NSEventTypeLeftMouseDragged) target=self.mouseTargets[@1];
        else if(type==NSEventTypeRightMouseDragged) target=self.mouseTargets[@3];
        else if(type==NSEventTypeOtherMouseDragged) target=self.mouseTargets[@2];
        else target=[self.web hitTest:[self.web.superview convertPoint:self.pointer fromView:nil]];
        if(type==NSEventTypeLeftMouseDragged) [target mouseDragged:event];
        else if(type==NSEventTypeRightMouseDragged) [target rightMouseDragged:event];
        else if(type==NSEventTypeOtherMouseDragged) [target otherMouseDragged:event];
        else [target mouseMoved:event];
        return;
    }
    if([kind isEqual:@"text"]) {
        NSString *text=v[@"text"];
        if(![text isKindOfClass:NSString.class] || text.length==0 || [text lengthOfBytesUsingEncoding:NSUTF8StringEncoding]>1024 || [text rangeOfString:[NSString stringWithFormat:@"%C",(unichar)0]].location!=NSNotFound) fail(@"invalid text input");
        id target=self.window.firstResponder;
        if([target respondsToSelector:@selector(insertText:replacementRange:)]) [(id<NSTextInputClient>)target insertText:text replacementRange:NSMakeRange(NSNotFound,0)];
        else if([target respondsToSelector:@selector(insertText:)]) [target insertText:text];
        return;
    }
    if([kind isEqual:@"key"]) {
        NSDictionary *keys=@{@"Return":@[@13,@36],@"Escape":@[@27,@53],@"BackSpace":@[@127,@51],@"Tab":@[@9,@48],@"Delete":@[@(NSDeleteFunctionKey),@117],@"Home":@[@(NSHomeFunctionKey),@115],@"End":@[@(NSEndFunctionKey),@119],@"Prior":@[@(NSPageUpFunctionKey),@116],@"Next":@[@(NSPageDownFunctionKey),@121],@"Left":@[@(NSLeftArrowFunctionKey),@123],@"Right":@[@(NSRightArrowFunctionKey),@124],@"Up":@[@(NSUpArrowFunctionKey),@126],@"Down":@[@(NSDownArrowFunctionKey),@125]};
        NSArray *entry=keys[v[@"key"]];if(!entry) fail(@"invalid key input");
        unichar character=[entry[0] unsignedShortValue];NSString *text=[NSString stringWithCharacters:&character length:1];
        for(NSNumber *down in @[@YES,@NO]) {
            NSEvent *event=[NSEvent keyEventWithType:down.boolValue?NSEventTypeKeyDown:NSEventTypeKeyUp location:NSZeroPoint modifierFlags:0 timestamp:NSProcessInfo.processInfo.systemUptime windowNumber:self.window.windowNumber context:nil characters:text charactersIgnoringModifiers:text isARepeat:NO keyCode:[entry[1] unsignedShortValue]];
            if(down.boolValue) [self.window.firstResponder keyDown:event];else [self.window.firstResponder keyUp:event];
        }
        return;
    }
    fail(@"unsupported inline input kind");
}
// Program updates are typed data for one fixed shell function. They cannot
// change the iframe, theme/state, native operation, or any external resource.
- (void)renderOpenUI:(NSDictionary *)next {
    if(!self.openUI || self.capture || ![next isKindOfClass:NSDictionary.class] || next.count!=3) fail(@"invalid OpenUI render mode");
    NSString *instance=next[@"instanceId"],*program=next[@"program"];id streaming=next[@"streaming"];
    NSCharacterSet *nonhex=[[NSCharacterSet characterSetWithCharactersInString:@"0123456789abcdefABCDEF"] invertedSet];
    if(![instance isKindOfClass:NSString.class] || instance.length!=64 || [instance rangeOfCharacterFromSet:nonhex].location!=NSNotFound || ![program isKindOfClass:NSString.class] || program.length<1 || [program lengthOfBytesUsingEncoding:NSUTF8StringEncoding]>262144 || (streaming!=(__bridge id)kCFBooleanTrue && streaming!=(__bridge id)kCFBooleanFalse)) fail(@"invalid OpenUI render payload");
    NSData *raw=[NSJSONSerialization dataWithJSONObject:next options:0 error:nil];
    if(!raw || raw.length>2*1024*1024) fail(@"OpenUI render exceeds limit");
    NSString *script=[NSString stringWithFormat:@"window.__whaleOpenUIRender(%@);true",[[NSString alloc] initWithData:raw encoding:NSUTF8StringEncoding]];
    NSString *result=[self.web stringByEvaluatingJavaScriptFromString:script];
    if(![result isEqual:@"true"]) fail(@"OpenUI render rejected by shell");
}
- (void)command:(NSDictionary *)p {
    NSString *kind=p[@"kind"];
    if([kind isEqual:@"render"]) [self renderOpenUI:p[@"renderBody"]];
    else if([kind isEqual:@"theme"]) [self theme:p];
    else if(self.inlineSeat && [kind isEqual:@"release"]) [self releaseInput];
    else if(self.inlineSeat && [kind isEqual:@"poll"]) {if(p[@"input"]) [self input:p[@"input"]];}
    else fail(@"unexpected IPC command");
    if(self.inlineSeat) {
        // Deliver input first, then capture on a later main-loop turn. The parent
        // permits only one outstanding request and validates every PNG frame.
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW,33*NSEC_PER_MSEC),dispatch_get_main_queue(),^{[self emitOpenUIEvents];[self emitFrame];});
    }
}
- (void)windowWillClose:(NSNotification *)note { exit(0); }
- (void)webView:(WebView *)sender decidePolicyForNavigationAction:(NSDictionary *)action request:(NSURLRequest *)request frame:(WebFrame *)frame decisionListener:(id<WebPolicyDecisionListener>)listener {
    NSString *url=request.URL.absoluteString ?: @"";
    if(self.initialNavigation && frame==self.web.mainFrame && ([url isEqual:@"about:blank"] || url.length==0)) { self.initialNavigation=NO;[listener use];return; }
    // Only the trusted shell's opaque srcdoc child may be initialized. No page
    // navigation, redirects, external protocols or arbitrary nested frames.
    if(frame!=self.web.mainFrame && frame.parentFrame==self.web.mainFrame && [url isEqual:@"about:srcdoc"]) { [listener use];return; }
    [listener ignore];
}
- (void)webView:(WebView *)sender decidePolicyForNewWindowAction:(NSDictionary *)action request:(NSURLRequest *)request newFrameName:(NSString *)name decisionListener:(id<WebPolicyDecisionListener>)listener { [listener ignore]; }
- (void)webView:(WebView *)sender decidePolicyForMIMEType:(NSString *)type request:(NSURLRequest *)request frame:(WebFrame *)frame decisionListener:(id<WebPolicyDecisionListener>)listener { if([type isEqual:@"text/html"] && [request.URL.scheme isEqual:@"about"]) [listener use];else [listener ignore]; }
- (NSURLRequest *)webView:(WebView *)sender resource:(id)identifier willSendRequest:(NSURLRequest *)request redirectResponse:(NSURLResponse *)response fromDataSource:(WebDataSource *)source {
    if(response) return nil;
    NSString *url=request.URL.absoluteString ?: @"";
    return ([request.URL.scheme isEqual:@"data"] || [url isEqual:@"about:blank"] || [url isEqual:@"about:srcdoc"]) ? request : nil;
}
- (WebView *)webView:(WebView *)sender createWebViewWithRequest:(NSURLRequest *)request { return nil; }
- (void)webView:(WebView *)sender runJavaScriptAlertPanelWithMessage:(NSString *)message initiatedByFrame:(WebFrame *)frame {}
- (BOOL)webView:(WebView *)sender runJavaScriptConfirmPanelWithMessage:(NSString *)message initiatedByFrame:(WebFrame *)frame { return NO; }
- (NSString *)webView:(WebView *)sender runJavaScriptTextInputPanelWithPrompt:(NSString *)prompt defaultText:(NSString *)text initiatedByFrame:(WebFrame *)frame { return nil; }
- (void)webView:(WebView *)sender runOpenPanelForFileButtonWithResultListener:(id<WebOpenPanelResultListener>)listener { [listener cancel]; }
- (void)webView:(WebView *)sender runOpenPanelForFileButtonWithResultListener:(id<WebOpenPanelResultListener>)listener allowMultipleFiles:(BOOL)multiple { [listener cancel]; }
- (NSArray *)webView:(WebView *)sender contextMenuItemsForElement:(NSDictionary *)element defaultMenuItems:(NSArray *)items { return @[]; }
- (BOOL)webView:(WebView *)sender shouldPerformAction:(SEL)action fromSender:(id)source { return NO; }
- (void)downloadDidBegin:(WebDownload *)download { [download cancel]; }
- (void)webView:(WebView *)sender didFailLoadWithError:(NSError *)error forFrame:(WebFrame *)frame { fail(@"legacy WebView load failed"); }
- (void)webView:(WebView *)sender didFailProvisionalLoadWithError:(NSError *)error forFrame:(WebFrame *)frame { fail(@"legacy WebView provisional load failed"); }
- (void)poll:(NSTimer *)timer {
    if(self.loaded) return;
    if(NSDate.timeIntervalSinceReferenceDate>self.deadline) fail(@"legacy WebView initialization timed out");
    NSString *json=[self.web stringByEvaluatingJavaScriptFromString:@"JSON.stringify(window.__previewState||null)"];
    NSDictionary *state=[NSJSONSerialization JSONObjectWithData:[json dataUsingEncoding:NSUTF8StringEncoding] options:0 error:nil];
    if(![state isKindOfClass:NSDictionary.class]) return;
    if([state[@"error"] isKindOfClass:NSString.class] && [state[@"error"] length]>0) fail(@"preview shell failed to initialize");
    if(![state[@"ready"] isEqual:@YES]) return;
    double measured=[state[@"height"] doubleValue];if(!isfinite(measured) || measured<1 || measured>9007199254740991.0) fail(@"invalid content height");
    self.loaded=YES;[timer invalidate];
    packet(@"loaded",@{@"browserPID":@(getpid())});
    if(!self.capture) {
        if(self.inlineSeat) { [self.window makeFirstResponder:self.web];[self emitFrame]; }
        else { [self.window makeKeyAndOrderFront:nil]; [NSApp activateIgnoringOtherApps:YES];
            if(self.openUI) [NSTimer scheduledTimerWithTimeInterval:0.15 target:self selector:@selector(pollOpenUI:) userInfo:nil repeats:YES];
        }
        return;
    }
    NSInteger height=MIN(4000,(NSInteger)ceil(measured));
    [self.window setContentSize:NSMakeSize(self.width,height)];[self.web setFrame:NSMakeRect(0,0,self.width,height)];
    // Wait for the resized frame to be painted before reading the view's pixels.
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW,150*NSEC_PER_MSEC),dispatch_get_main_queue(),^{
        NSBitmapImageRep *bitmap=[[NSBitmapImageRep alloc] initWithBitmapDataPlanes:NULL pixelsWide:self.width pixelsHigh:height bitsPerSample:8 samplesPerPixel:4 hasAlpha:YES isPlanar:NO colorSpaceName:NSDeviceRGBColorSpace bytesPerRow:0 bitsPerPixel:0];
        [self.web displayIfNeeded];[self.web cacheDisplayInRect:self.web.bounds toBitmapImageRep:bitmap];
        NSData *png=[bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];if(!png) fail(@"capture unavailable");
        NSArray *messages=[state[@"consoleMessages"] isKindOfClass:NSArray.class]?state[@"consoleMessages"]:@[];
        NSDictionary *result=@{@"png":[png base64EncodedStringWithOptions:0],@"width":@(self.width),@"contentHeight":@((NSInteger)ceil(measured)),@"capturedHeight":@(height),@"consoleMessages":messages};
        NSData *body=[NSJSONSerialization dataWithJSONObject:result options:0 error:nil];if(!body || body.length>90*1024*1024) fail(@"capture exceeds limit");
        packet(@"capture",@{@"bytes":@(body.length)});writeData(body);
    });
}
- (void)theme:(NSDictionary *)p {
    NSString *appearance=p[@"appearance"];if(![appearance isEqual:@"dark"] && ![appearance isEqual:@"light"]) fail(@"invalid theme");
    BOOL dark=[appearance isEqual:@"dark"];
    NSMutableDictionary *vars=[@{@"--bg":dark?@"#171717":@"#ffffff",@"--background":dark?@"#171717":@"#ffffff",@"--surface":dark?@"#242424":@"#f5f5f5",@"--text":dark?@"#eeeeee":@"#171717",@"--text-secondary":dark?@"#aaaaaa":@"#666666",@"--border":dark?@"#444444":@"#dddddd",@"--accent":dark?@"#60a5fa":@"#2563eb"} mutableCopy];
    NSDictionary *overrides=p[@"variables"] ?: @{};if(![overrides isKindOfClass:NSDictionary.class]) fail(@"invalid theme tokens");
    for(NSString *key in overrides) { id value=overrides[key];if(!vars[key] || ![value isKindOfClass:NSString.class] || [value length]<1 || [value length]>128) fail(@"invalid theme token");vars[key]=value; }
    NSData *raw=[NSJSONSerialization dataWithJSONObject:@{@"type":@"whale-visual-reply-theme",@"appearance":appearance,@"variables":vars} options:0 error:nil];
    NSString *script=[NSString stringWithFormat:@"window.__previewTheme(%@);true",[[NSString alloc] initWithData:raw encoding:NSUTF8StringEncoding]];
    [self.web stringByEvaluatingJavaScriptFromString:script];
}
@end

int main(int argc,const char *argv[]) { @autoreleasepool {
    if(argc!=3 || strcmp(argv[1],"--legacy-sandbox-helper-v1")) fail(@"parent handshake required");
    nonce=[NSString stringWithUTF8String:argv[2]];
    NSRegularExpression *pattern=[NSRegularExpression regularExpressionWithPattern:@"^[a-f0-9]{64}$" options:0 error:nil];
    if([pattern numberOfMatchesInString:nonce options:0 range:NSMakeRange(0,nonce.length)]!=1) fail(@"invalid nonce");
    signal(SIGPIPE,SIG_DFL);
    // No inherited auxiliary files, sockets or application descriptors.
    struct rlimit limits;if(getrlimit(RLIMIT_NOFILE,&limits)==0) for(int fd=3;fd<(int)MIN(limits.rlim_cur,65536);fd++) close(fd);
    verifyIdentity();
    Class legacy=NSClassFromString(@"WebView");if(!legacy || ![legacy instancesRespondToSelector:@selector(initWithFrame:frameName:groupName:)]) fail(@"public legacy WebView unavailable on this macOS version");
    [NSApplication sharedApplication];[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    Preview *preview=[Preview new];
    // Ready means the independently signed single-process renderer is identified;
    // no untrusted document has been read or loaded at this point.
    packet(@"ready",@{@"helperPID":@(getpid()),@"browserPID":@(getpid()),@"openUI":@YES,@"openUIUpdate":@YES});
    NSDictionary *doc=readPacket();if(![doc[@"kind"] isEqual:@"document"]) fail(@"unsupported document request");
    NSInteger n=[doc[@"bytes"] integerValue],width=[doc[@"width"] integerValue];
    if(n<1 || n>25*1024*1024 || width<240 || width>1600) fail(@"invalid document size");
    NSString *html=[[NSString alloc] initWithData:readExact(n) encoding:NSUTF8StringEncoding];if(!html) fail(@"invalid document UTF-8");
    preview.width=width;preview.capture=[doc[@"capture"] boolValue];preview.inlineSeat=[doc[@"inline"] boolValue];preview.initialNavigation=YES;preview.openUI=[doc[@"openUI"] isEqual:@YES];
    if(preview.openUI && preview.capture) fail(@"OpenUI requires interactive mode");
    if(preview.inlineSeat && preview.capture) fail(@"capture requires a dedicated session");
    if(preview.inlineSeat) [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    preview.window=[[NSWindow alloc] initWithContentRect:NSMakeRect(0,0,width,700) styleMask:NSWindowStyleMaskTitled|NSWindowStyleMaskClosable|NSWindowStyleMaskResizable backing:NSBackingStoreBuffered defer:NO];
    preview.window.title=@"Isolated HTML preview (legacy compatibility)";preview.window.delegate=preview;
    preview.web=[[legacy alloc] initWithFrame:NSMakeRect(0,0,width,700) frameName:nil groupName:NSUUID.UUID.UUIDString];
    if(!preview.web) fail(@"legacy renderer creation failed");
    WebPreferences *prefs=[[WebPreferences alloc] initWithIdentifier:NSUUID.UUID.UUIDString];prefs.autosaves=NO;prefs.javaScriptEnabled=YES;prefs.javaEnabled=NO;prefs.plugInsEnabled=NO;prefs.javaScriptCanOpenWindowsAutomatically=NO;
    preview.web.shouldUpdateWhileOffscreen=YES;
    preview.web.preferences=prefs;preview.web.frameLoadDelegate=preview;preview.web.policyDelegate=preview;preview.web.resourceLoadDelegate=preview;preview.web.UIDelegate=preview;preview.web.downloadDelegate=preview;
    preview.web.autoresizingMask=NSViewWidthSizable|NSViewHeightSizable;preview.window.contentView=preview.web;
    preview.deadline=NSDate.timeIntervalSinceReferenceDate+15;
    [preview.web.mainFrame loadHTMLString:html baseURL:[NSURL URLWithString:@"about:blank"]];
    [NSTimer scheduledTimerWithTimeInterval:0.05 target:preview selector:@selector(poll:) userInfo:nil repeats:YES];
    dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED,0),^{ while(YES) { @autoreleasepool {
        NSDictionary *p=readPacket();
        if([p[@"kind"] isEqual:@"render"]) {
            NSInteger size=[p[@"bytes"] integerValue];
            if(!preview.openUI || preview.capture || size<1 || size>2*1024*1024) fail(@"invalid OpenUI render body length");
            id body=[NSJSONSerialization JSONObjectWithData:readExact(size) options:0 error:nil];
            if(![body isKindOfClass:NSDictionary.class]) fail(@"invalid OpenUI render JSON");
            NSMutableDictionary *withBody=[p mutableCopy];withBody[@"renderBody"]=body;p=withBody;
        }
        dispatch_sync(dispatch_get_main_queue(),^{ [preview command:p]; });
    } } });
    [NSApp run];
} return 0; }
