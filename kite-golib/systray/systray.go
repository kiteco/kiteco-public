package systray

/*
#cgo linux pkg-config: gtk+-3.0 appindicator3-0.1
#cgo darwin CFLAGS: -DDARWIN -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa

#include "systray.h"
*/
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	readyCh       = make(chan interface{})
	menuItems     = make(map[int32]*MenuItem)
	menuItemsLock sync.RWMutex
	menuOpened    func()

	currentID int32
)

func addOrUpdateMenuItem(item *MenuItem) {
	var disabled C.short
	if item.disabled {
		disabled = 1
	}
	var checked C.short
	if item.checked {
		checked = 1
	}
	C.add_or_update_menu_item(
		C.int(item.ID),
		C.CString(item.title),
		C.CString(item.tooltip),
		disabled,
		checked,
	)
}

func addSeparator(item *MenuItem) {
	C.add_separator(C.int(item.ID))
}

func addOrUpdateSubmenu(item *MenuItem) {
	C.add_or_update_submenu(
		C.int(item.ID),
		C.CString(item.title),
		C.CString(item.tooltip),
	)
}

func addOrUpdateSubmenuItem(item *MenuItem) {
	var disabled C.short
	if item.disabled {
		disabled = 1
	}
	var checked C.short
	if item.checked {
		checked = 1
	}
	C.add_or_update_submenu_item(
		C.int(item.submenuID),
		C.int(item.ID),
		C.CString(item.title),
		C.CString(item.tooltip),
		disabled,
		C.bool(item.checkable),
		checked,
	)
}

//export cgoSystrayReady
func cgoSystrayReady() {
	readyCh <- nil
}

//export cgoSystrayMenuItemSelected
func cgoSystrayMenuItemSelected(cID C.int) {
	menuItemsLock.RLock()
	item := menuItems[int32(cID)]
	menuItemsLock.RUnlock()

	if item != nil && item.onclick != nil {
		item.onclick()
	}
}

//export cgoSystrayMenuOpened
func cgoSystrayMenuOpened() {
	if menuOpened != nil {
		menuOpened()
	}
}

// MenuItem is used to keep track each menu item of systray
// Don't create it directly, use the one systray.AddMenuItem() returned
type MenuItem struct {
	// onclick is the callback for when the menu item is clicked
	onclick func()

	// ID uniquely identify a menu item, not supposed to be modified
	ID int32
	// submenuID identifies the submenu this item is under
	submenuID int32
	// title is the text shown on menu item
	title string
	// tooltip is the text shown when pointing to menu item
	tooltip string
	// disabled menu item is grayed out and has no effect when clicked
	disabled bool
	// checkable menu item has a tick before the title
	checkable bool
	// checked menu item has a tick before the title
	checked bool
	// separator menu items are just visual separators
	separator bool

	// submenu designates a menu item as a submenu
	submenu bool
}

// Handle represents a paltform-specific handle to the underlying UI object.
type Handle uintptr

// Run initializes GUI and starts the event loop, then invokes the onReady
// callback.
// It blocks until systray.Quit() is called.
// Should be called at the very beginning of main() to lock at main thread.
func Run(onReady func(h Handle)) {
	go func() {
		<-readyCh
		onReady(Handle(C.nativeHandle()))
	}()

	C.nativeLoop()
}

// Show creates a tray icon with the given title, tooltip, and icon.
func Show(title, tooltip string, iconBytes []byte) {
	cstr := (*C.char)(unsafe.Pointer(&iconBytes[0]))
	C.show(C.CString(title), C.CString(tooltip), cstr, (C.int)(len(iconBytes)))
}

// Hide removes the tray icon
func Hide() {
	C.hide()
}

// SetMenuOpened sets the event handler for the menu opened event.
func SetMenuOpened(handler func()) {
	menuOpened = handler
}

// CleanupHandle removes the systray icon for a specific handle. This can be
// used on some platforms to remove the leftover tray icon for a previous
// instance of the application.
func CleanupHandle(h Handle) {
	C.cleanupHandle(C.uint64_t(h))
}

// AddMenuItem adds menu item with designated title and tooltip, returning a channel
// that notifies whenever that menu item is clicked.
//
// It can be safely invoked from different goroutines.
func AddMenuItem(title string, tooltip string, onclick func()) *MenuItem {
	item := &MenuItem{
		ID:      atomic.AddInt32(&currentID, 1),
		title:   title,
		tooltip: tooltip,
		onclick: onclick,
	}
	item.update()
	return item
}

// AddSeparator is like AddMenuItem except it adds a visual separator
func AddSeparator() *MenuItem {
	item := &MenuItem{
		ID:        atomic.AddInt32(&currentID, 1),
		separator: true,
	}
	item.update()
	return item
}

// AddSubmenu is like AddMenuItem except it adds a submenu.
// Note that clicks on submenu's do nothing and so listening
// to the clicked channel will never yield any events.
func AddSubmenu(title, tooltip string) *MenuItem {
	item := &MenuItem{
		ID:      atomic.AddInt32(&currentID, 1),
		title:   title,
		tooltip: tooltip,
		submenu: true,
	}
	item.update()
	return item
}

// AddSubmenuItem is like AddMenuItem except that it adds
// new menu item under an existing submenu.
func AddSubmenuItem(submenuID int32, title, tooltip string, checkable bool, onclick func()) *MenuItem {
	item := &MenuItem{
		ID:        atomic.AddInt32(&currentID, 1),
		submenuID: submenuID,
		title:     title,
		tooltip:   tooltip,
		onclick:   onclick,
		checkable: checkable,
		submenu:   true,
	}
	item.update()
	return item
}

// SetTitle set the text to display on a menu item
func (item *MenuItem) SetTitle(title string) {
	item.title = title
	item.update()
}

// SetTooltip set the tooltip to show when mouse hover
func (item *MenuItem) SetTooltip(tooltip string) {
	item.tooltip = tooltip
	item.update()
}

// Disabled checkes if the menu item is disabled
func (item *MenuItem) Disabled() bool {
	return item.disabled
}

// Enable a menu item regardless if it's previously enabled or not
func (item *MenuItem) Enable() {
	item.disabled = false
	item.update()
}

// Disable a menu item regardless if it's previously disabled or not
func (item *MenuItem) Disable() {
	item.disabled = true
	item.update()
}

// Checked returns if the menu item has a check mark
func (item *MenuItem) Checked() bool {
	return item.checked
}

// Check a menu item regardless if it's previously checked or not
func (item *MenuItem) Check() {
	item.checked = true
	item.update()
}

// Uncheck a menu item regardless if it's previously unchecked or not
func (item *MenuItem) Uncheck() {
	item.checked = false
	item.update()
}

// update propagates changes on a menu item to systray
func (item *MenuItem) update() {
	menuItemsLock.Lock()
	defer menuItemsLock.Unlock()
	menuItems[item.ID] = item
	if item.submenuID != 0 {
		addOrUpdateSubmenuItem(item)
	} else if item.submenu {
		addOrUpdateSubmenu(item)
	} else if item.separator {
		addSeparator(item)
	} else {
		addOrUpdateMenuItem(item)
	}
}
