#ifdef __cplusplus
#define SYSTRAY_EXPORT extern "C"
#else
#define SYSTRAY_EXPORT extern
#endif

#include <stdint.h>
#include <stdbool.h>

SYSTRAY_EXPORT void cgoSystrayReady();
SYSTRAY_EXPORT void cgoSystrayMenuItemSelected(int menu_id);
SYSTRAY_EXPORT void cgoSystrayMenuOpened();

SYSTRAY_EXPORT int nativeLoop(void);
SYSTRAY_EXPORT void* nativeHandle(void);
SYSTRAY_EXPORT void cleanupHandle(uint64_t handle);

SYSTRAY_EXPORT void show(const char *title, const char *tooltip, const char *iconBytes, int iconLength);
SYSTRAY_EXPORT void hide();

SYSTRAY_EXPORT void add_or_update_menu_item(int menuId, char *title, char *tooltip, short disabled, short checked);
SYSTRAY_EXPORT void add_separator(int menuId);
SYSTRAY_EXPORT void add_or_update_submenu(int menuId, char *title, char *tooltip);
SYSTRAY_EXPORT void add_or_update_submenu_item(int submenuId, int menuId, char *title, char *tooltip, short disabled, bool checkable, short checked);
