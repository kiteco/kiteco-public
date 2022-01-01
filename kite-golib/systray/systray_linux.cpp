#include <map>

#include <stdlib.h>
#include <string.h>
#include <errno.h>
#include <limits.h>
#include <libappindicator/app-indicator.h>
#include "systray.h"

static AppIndicator *global_app_indicator;
static GtkWidget *root_menu = NULL;
static std::map<int, GtkWidget*> menus;
static std::map<int, GtkWidget*> menu_items;
static bool ignore_events = false;

typedef struct {
	int id;
	int parent_id;
	char* title;
	char* tooltip;
	bool disabled;
	bool checkable;  // whether this item can be checked/unchecked
	bool checked;    // whether this item is currently checked
	bool separator;
	bool submenu;
} MenuItemInfo;

int nativeLoop(void) {
	// initialize GTK
	gtk_init(0, NULL);
	global_app_indicator = app_indicator_new("kite", "", APP_INDICATOR_CATEGORY_APPLICATION_STATUS);
	app_indicator_set_status(global_app_indicator, APP_INDICATOR_STATUS_PASSIVE);

	// create the root menu
	root_menu = gtk_menu_new();
	menus[-1] = root_menu;

	// install the root menu into the indicator area
	app_indicator_set_menu(global_app_indicator, GTK_MENU(root_menu));
	cgoSystrayReady();

	// loop forever
	gtk_main();
	return 0;
}

// runs in main thread, should always return FALSE to prevent gtk to execute it again
gboolean do_show(gpointer data) {
	GBytes* bytes = (GBytes*)data;
	char* temp_file_name = new char[PATH_MAX];
	strcpy(temp_file_name, "/tmp/systray_XXXXXX");
	int fd = mkstemp(temp_file_name);
	if (fd == -1) {
		printf("failed to create temp icon file %s: %s\n", temp_file_name, strerror(errno));
		return FALSE;
	}
	gsize size = 0;
	gconstpointer icon_data = g_bytes_get_data(bytes, &size);
	ssize_t written = write(fd, icon_data, size);
	close(fd);
	if (written != size) {
		printf("failed to write temp icon file %s: %s\n", temp_file_name, strerror(errno));
		return FALSE;
	}
	app_indicator_set_icon_full(global_app_indicator, temp_file_name, "");
	app_indicator_set_attention_icon_full(global_app_indicator, temp_file_name, "");
	app_indicator_set_status(global_app_indicator, APP_INDICATOR_STATUS_ACTIVE);
	g_bytes_unref(bytes);
	return FALSE;
}

void _systray_menu_item_selected(int *id) {
	if (!ignore_events) {
		cgoSystrayMenuItemSelected(*id);
	}
}

// runs in main thread, should always return FALSE to prevent gtk to execute it again
gboolean do_add_or_update_menu_item(gpointer data) {
	MenuItemInfo *mii = (MenuItemInfo*)data;

	// first look up the parent menu
	std::map<int, GtkWidget*>::iterator parent_it = menus.find(mii->parent_id);
	if (parent_it == menus.end()) {
		printf("parent menu does not exist (parent_id=%d) for item: %s\n", mii->parent_id, mii->title);
		return FALSE;
	}
	GtkWidget* parent = parent_it->second;

	GtkWidget *menu_item;
	std::map<int, GtkWidget*>::iterator it = menu_items.find(mii->id);
	if (it == menu_items.end()) {
		// create the menu item
		if (mii->separator) {
			menu_item = gtk_separator_menu_item_new();
		} else if (mii->checkable) {
			menu_item = gtk_check_menu_item_new_with_label(mii->title);
		} else {
			menu_item = gtk_menu_item_new_with_label(mii->title);
		}

		// create submenu if requested
		if (mii->submenu) {
			GtkWidget* submenu = gtk_menu_new();
			gtk_menu_item_set_submenu(GTK_MENU_ITEM(menu_item), submenu);
			menus[mii->id] = submenu;
		}

		// register callback
		int *id = new int;
		*id = mii->id;
		g_signal_connect_swapped(G_OBJECT(menu_item), "activate", G_CALLBACK(_systray_menu_item_selected), id);

		// append the menu item to the parent menu
		gtk_menu_shell_append(GTK_MENU_SHELL(parent), menu_item);

		// store the menu item to the global map
		menu_items[mii->id] = menu_item;
	} else {
		menu_item = it->second;
        gtk_menu_item_set_label(GTK_MENU_ITEM(menu_item), mii->title);
	}

	// update state
	gtk_widget_set_sensitive(menu_item, mii->disabled == 1 ? FALSE : TRUE);
	if (mii->checkable) {
		// changing the menu state will emit the "activate" event, which we must ignore
		ignore_events = true;
		gtk_check_menu_item_set_active(GTK_CHECK_MENU_ITEM(menu_item), mii->checked);
		ignore_events = false;
	}

	// render the menu item
	gtk_widget_show_all(root_menu);

	free(mii->title);
	free(mii->tooltip);
	free(mii);
	return FALSE;
}

// runs in main thread, should always return FALSE to prevent gtk to execute it again
gboolean do_hide(gpointer data) {
	// app indicator doesn't provide a way to remove it, hide it as a workaround
	app_indicator_set_status(global_app_indicator, APP_INDICATOR_STATUS_PASSIVE);
	return FALSE;
}

void show(const char* title, const char* tooltip, const char* iconBytes, int iconLength) {
	// currently title and tooltip are ignored
	GBytes* bytes = g_bytes_new_static(iconBytes, iconLength);
	g_idle_add(do_show, bytes);
}

void add_or_update_menu_item(int id, char* title, char* tooltip, short disabled, short checked) {
	MenuItemInfo *mii = new MenuItemInfo;
	mii->id = id;
	mii->parent_id = -1;  // resolves to the root menu
	mii->title = title;
	mii->tooltip = tooltip;
	mii->disabled = disabled;
	mii->checkable = false;
	mii->checked = checked;
	mii->separator = false;
	mii->submenu = false;
	g_idle_add(do_add_or_update_menu_item, mii);
}

void add_separator(int id) {
	MenuItemInfo *mii = new MenuItemInfo;
	mii->id = id;
	mii->parent_id = -1;  // resolves to the root menu
	mii->title = NULL;
	mii->tooltip = NULL;
	mii->disabled = false;
	mii->checkable = false;
	mii->checked = false;
	mii->separator = true;
	mii->submenu = false;
	g_idle_add(do_add_or_update_menu_item, mii);
}

void add_or_update_submenu(int id, char *title, char *tooltip) {
	MenuItemInfo *mii = new MenuItemInfo;
	mii->id = id;
	mii->parent_id = -1;  // resolves to the root menu
	mii->title = title;
	mii->tooltip = tooltip;
	mii->disabled = false;
	mii->checkable = false;
	mii->checked = false;
	mii->submenu = true;
	g_idle_add(do_add_or_update_menu_item, mii);
}

void add_or_update_submenu_item(int parent_id, int id, char *title, char *tooltip, short disabled, bool checkable, short checked) {
	MenuItemInfo *mii = new MenuItemInfo;
	mii->id = id;
	mii->parent_id = parent_id;
	mii->title = title;
	mii->tooltip = tooltip;
	mii->disabled = disabled;
	mii->checkable = checkable;
	mii->checked = checked;
	mii->separator = false;
	mii->submenu = false;
	g_idle_add(do_add_or_update_menu_item, mii);
}

void hide() {
	g_idle_add(do_hide, NULL);
}

void* nativeHandle() {
	return NULL;
}

void cleanupHandle(uint64_t handle) {
	// ignore on linux
}
