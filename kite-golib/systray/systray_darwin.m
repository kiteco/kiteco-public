#import <Cocoa/Cocoa.h>
#include "systray.h"

@interface MenuItem : NSObject
{
@public
  NSNumber *menuId;
  NSNumber *submenuId;
  NSString *title;
  NSString *tooltip;
  short disabled;
  short checked;
  short separator;
  short submenu;

}
- (id)initWithId:(int)theMenuId
       withTitle:(const char *)theTitle
     withTooltip:(const char *)theTooltip
    withDisabled:(short)theDisabled
     withChecked:(short)theChecked;

- (id)initSeparatorWithId:(int)theMenuId;

- (id)initSubmenuWithId:(int)theMenuId
              withTitle:(const char *)theTitle
            withTooltip:(const char *)theTooltip;
@end
@implementation MenuItem

- (id)initWithId:(int)theMenuId
       withTitle:(const char *)theTitle
     withTooltip:(const char *)theTooltip
    withDisabled:(short)theDisabled
     withChecked:(short)theChecked
{
  menuId = [NSNumber numberWithInt:theMenuId];
  submenuId = nil;
  title = [[NSString alloc] initWithCString:theTitle
                                   encoding:NSUTF8StringEncoding];
  tooltip = [[NSString alloc] initWithCString:theTooltip
                                     encoding:NSUTF8StringEncoding];
  disabled = theDisabled;
  checked = theChecked;
  separator = 0;
  submenu = 0;
  return self;
}

- (id)initSeparatorWithId:(int)theMenuId {
  menuId = [NSNumber numberWithInt:theMenuId];
  submenuId = nil;
  title = nil;
  tooltip = nil;
  disabled = 0;
  checked = 0;
  separator = 1;
  submenu = 0;
  return self;
}

- (id)initSubmenuWithId:(int)theMenuId
              withTitle:(const char *)theTitle
            withTooltip:(const char *)theTooltip
{
  menuId = [NSNumber numberWithInt:theMenuId];
  submenuId = nil;
  title = [[NSString alloc] initWithCString:theTitle
                                   encoding:NSUTF8StringEncoding];
  tooltip = [[NSString alloc] initWithCString:theTooltip
                                     encoding:NSUTF8StringEncoding];
  disabled = 0;
  checked = 0;
  separator = 0;
  submenu = 1;
  return self;
}

@end

@interface MenuItemRegistry : NSObject <NSMenuDelegate>
@property (strong) NSMenu *menu;
@property (strong) NSMutableDictionary *submenus;
@property (strong, nonatomic) NSStatusItem *statusItem;
- (void)addOrUpdateMenuItem:(MenuItem *)item;
- (void)addSeparator:(MenuItem *)item;
- (void)addOrUpdateSubmenu:(MenuItem *)item;
- (void)addOrUpdateSubmenuItem:(MenuItem *)item;
- (IBAction)menuHandler:(id)sender;
- (void)hang;
- (void)show:(NSString*)title tooltip:(NSString*)tooltip icon:(NSImage*)icon;
- (void)hide;
+ (MenuItemRegistry *)sharedRegistry;
@end

@implementation MenuItemRegistry

- (id)init {
  self = [super init];
  self.menu = [[NSMenu alloc] init];
  [self.menu setAutoenablesItems:FALSE];
  [self.menu setDelegate:self];
  self.submenus = [[NSMutableDictionary alloc] init];
  cgoSystrayReady();
  return self;
}

- (void)addOrUpdateMenuItem:(MenuItem*)item {
  NSMenuItem* menuItem;
  int existedMenuIndex = [self.menu indexOfItemWithRepresentedObject:item->menuId];
  if (existedMenuIndex == -1) {
    menuItem = [self.menu addItemWithTitle:item->title action:@selector(menuHandler:) keyEquivalent:@""];
    [menuItem setTarget:self];
    [menuItem setRepresentedObject:item->menuId];
  } else {
    menuItem = [self.menu itemAtIndex:existedMenuIndex];
    [menuItem setTitle:item->title];
  }

  [menuItem setToolTip:item->tooltip];
  if (item->disabled == 1) {
    [menuItem setEnabled:FALSE];
  } else {
    [menuItem setEnabled:TRUE];
  }
  if (item->checked == 1) {
    [menuItem setState:NSOnState];
  } else {
    [menuItem setState:NSOffState];
  }
}

- (void)addSeparator:(MenuItem *)item {
  NSMenuItem* menuItem;
  int existedMenuIndex = [self.menu indexOfItemWithRepresentedObject:item->menuId];
  if (existedMenuIndex == -1) {
    menuItem = [NSMenuItem separatorItem];
    [self.menu addItem:menuItem];
    [menuItem setTarget:self];
    [menuItem setRepresentedObject:item->menuId];
  }
}

- (void)addOrUpdateSubmenu:(MenuItem *)item {
  NSMenuItem* menuItem;
  int existedMenuIndex = [self.menu indexOfItemWithRepresentedObject:item->menuId];
  if (existedMenuIndex == -1) {
    menuItem = [self.menu addItemWithTitle:item->title action:nil keyEquivalent:@""];
    [menuItem setTarget:self];
    [menuItem setRepresentedObject:item->menuId];
    NSMenu *submenu = [[NSMenu alloc] initWithTitle:item->title];
    self.submenus[item->menuId] = submenu;
    [self.menu setSubmenu:submenu forItem:menuItem];
  }
}

- (void)addOrUpdateSubmenuItem:(MenuItem *)item {
  NSMenuItem* menuItem;
  NSMenu *submenu = self.submenus[item->submenuId];
  int existedMenuIndex = [submenu indexOfItemWithRepresentedObject:item->menuId];
  if (existedMenuIndex == -1) {
    menuItem = [submenu addItemWithTitle:item->title action:@selector(menuHandler:) keyEquivalent:@""];
    [menuItem setTarget:self];
    [menuItem setRepresentedObject:item->menuId];
  } else {
    menuItem = [submenu itemAtIndex:existedMenuIndex];
    [menuItem setTitle:item->title];
  }

  [menuItem setToolTip:item->tooltip];
  if (item->disabled == 1) {
    [menuItem setEnabled:FALSE];
  } else {
    [menuItem setEnabled:TRUE];
  }
  if (item->checked == 1) {
    [menuItem setState:NSOnState];
  } else {
    [menuItem setState:NSOffState];
  }
}

- (IBAction)menuHandler:(id)sender {
  NSNumber* menuId = [sender representedObject];
  cgoSystrayMenuItemSelected(menuId.intValue);
}

- (void)hang {
  sleep(100000000);
}

- (void)show:(NSString*)title tooltip:(NSString*)tooltip icon:(NSImage*)icon {
  if (self.statusItem == nil) {
    self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    [self.statusItem setMenu:self.menu];
    [self.statusItem.button setTitle:title];
    [self.statusItem.button setToolTip:tooltip];
    [icon setTemplate:YES];
    [self.statusItem.button setImage:icon];
  }
}

- (void)hide {
  if (self.statusItem != nil) {
    [[NSStatusBar systemStatusBar] removeStatusItem:self.statusItem];
    self.statusItem = nil;
  }
}

- (void)menuWillOpen:(NSMenu *)menu {
  cgoSystrayMenuOpened();
}

+ (MenuItemRegistry *)sharedRegistry {
  static dispatch_once_t pred;
  static id sharedRegistry = nil;
  dispatch_once(&pred, ^{
    sharedRegistry = [[[self class] alloc] init];
  });
  return sharedRegistry;
}

@end


int nativeLoop(void) {
  dispatch_async(dispatch_get_main_queue(), ^{
    [MenuItemRegistry sharedRegistry];
  });
  return 0;
}

void show(const char *ctitle, const char *ctooltip, const char *iconBytes, int iconLength) {
  NSString *title = [[NSString alloc] initWithCString:ctitle encoding:NSUTF8StringEncoding];
  NSString *tooltip = [[NSString alloc] initWithCString:ctooltip encoding:NSUTF8StringEncoding];
  NSData *buffer = [NSData dataWithBytes:iconBytes length:iconLength];
  NSImage *image = [[NSImage alloc] initWithData:buffer];
  dispatch_async(dispatch_get_main_queue(), ^{
    [[MenuItemRegistry sharedRegistry] show:title tooltip:tooltip icon:image];
  });
}

void hide() {
  dispatch_async(dispatch_get_main_queue(), ^{
    [[MenuItemRegistry sharedRegistry] hide];
  });
}

void add_or_update_menu_item(int menuId, char *title, char *tooltip, short disabled, short checked) {
  MenuItem *item = [[MenuItem alloc]
                     initWithId:menuId
                      withTitle:title
                     withTooltip:tooltip
                     withDisabled:disabled
                     withChecked:checked];
  free(title);
  free(tooltip);
  dispatch_async(dispatch_get_main_queue(), ^{
    [[MenuItemRegistry sharedRegistry] addOrUpdateMenuItem:item];
  });
}

void add_separator(int menuId) {
  MenuItem *item = [[MenuItem alloc] initSeparatorWithId:menuId];
  dispatch_async(dispatch_get_main_queue(), ^{
    [[MenuItemRegistry sharedRegistry] addSeparator:item];
  });
}

void add_or_update_submenu(int menuId, char *title, char *tooltip) {
  MenuItem *item = [[MenuItem alloc]
                     initSubmenuWithId:menuId
                             withTitle:title
                           withTooltip:tooltip];
  dispatch_async(dispatch_get_main_queue(), ^{
    [[MenuItemRegistry sharedRegistry] addOrUpdateSubmenu:item];
  });
}

void add_or_update_submenu_item(int submenuId, int menuId, char *title, char *tooltip, short disabled, bool checkable, short checked) {
  MenuItem *item = [[MenuItem alloc]
                     initWithId:menuId
                      withTitle:title
                     withTooltip:tooltip
                     withDisabled:disabled
                     withChecked:checked];
  item->submenuId = [NSNumber numberWithInt:submenuId];
  free(title);
  free(tooltip);
  dispatch_async(dispatch_get_main_queue(), ^{
    [[MenuItemRegistry sharedRegistry] addOrUpdateSubmenuItem:item];
  });
}

void* nativeHandle() {
  return NULL;
}

void cleanupHandle(uint64_t handle) {
  // ignore on darwin
}
