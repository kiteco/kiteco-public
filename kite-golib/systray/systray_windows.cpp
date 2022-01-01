#include <map>

#include <SDKDDKVer.h>

#define WIN32_LEAN_AND_MEAN             // Exclude rarely-used stuff from Windows headers
#include <windows.h>
#include <stdio.h>
#include <stdlib.h>
#include <shellapi.h>

#include "systray.h"

// Message posted into message loop when Notification Icon is clicked
#define WM_SYSTRAY_MESSAGE (WM_USER + 1)
#define ICON_UID 1702127979;

static NOTIFYICONDATA nid;
static HWND hWnd;
static HMENU hTrayMenu;
static std::map<int, HMENU> hMenusById;

void reportError(const char* action) {
	LPTSTR pErrMsg = NULL;
	DWORD errCode = GetLastError();
	DWORD result = FormatMessage(FORMAT_MESSAGE_ALLOCATE_BUFFER|
			FORMAT_MESSAGE_FROM_SYSTEM|
			FORMAT_MESSAGE_ARGUMENT_ARRAY,
			NULL,
			errCode,
			LANG_NEUTRAL,
			pErrMsg,
			0,
			NULL);
	printf("%s: %d %s\n", action, errCode, pErrMsg);
}

void ShowMenu(HWND hWnd) {
	POINT p;
	if (0 == GetCursorPos(&p)) {
		reportError("Error in GetCursorPos");
		return;
	};
	cgoSystrayMenuOpened();
	SetForegroundWindow(hWnd); // Win32 bug work-around
	TrackPopupMenu(hTrayMenu, TPM_BOTTOMALIGN | TPM_LEFTALIGN, p.x, p.y, 0, hWnd, NULL);
}

int GetMenuItemId(HMENU menu, int index) {
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_DATA;
	if (0 == GetMenuItemInfo(menu, index, TRUE, &menuItemInfo)) {
		reportError("Error in GetMenuItemInfo");
		return -1;
	}
	return menuItemInfo.dwItemData;
}

LRESULT CALLBACK WndProc(HWND hWnd, UINT message, WPARAM wParam, LPARAM lParam) {
	switch (message) {
		case WM_MENUCOMMAND:
			{
				HMENU menu = (HMENU)lParam;
				int menuItemId = GetMenuItemId(menu, wParam);
				if (menuItemId != -1) {
					cgoSystrayMenuItemSelected(menuItemId);
				}
			}
			break;
		case WM_DESTROY:
			PostQuitMessage(0);
			break;
		case WM_SYSTRAY_MESSAGE:
			switch(lParam) {
				case WM_RBUTTONUP:
					ShowMenu(hWnd);
					break;
				case WM_LBUTTONUP:
					ShowMenu(hWnd);
					break;
				default:
					return DefWindowProc(hWnd, message, wParam, lParam);
			};
			break;
		default:
			return DefWindowProc(hWnd, message, wParam, lParam);
	}
	return 0;
}

void RegisterClass(HINSTANCE hInstance, TCHAR* szWindowClass) {
	WNDCLASSEX wcex;

	wcex.cbSize = sizeof(WNDCLASSEX);
	wcex.style          = CS_HREDRAW | CS_VREDRAW;
	wcex.lpfnWndProc    = WndProc;
	wcex.cbClsExtra     = 0;
	wcex.cbWndExtra     = 0;
	wcex.hInstance      = hInstance;
	wcex.hIcon          = LoadIcon(NULL, IDI_APPLICATION);
	wcex.hCursor        = LoadCursor(NULL, IDC_ARROW);
	wcex.hbrBackground  = (HBRUSH)(COLOR_WINDOW+1);
	wcex.lpszMenuName   = 0;
	wcex.lpszClassName  = szWindowClass;
	wcex.hIconSm        = LoadIcon(NULL, IDI_APPLICATION);

	RegisterClassEx(&wcex);
}

HWND InitInstance(HINSTANCE hInstance, int nCmdShow, TCHAR* szWindowClass) {
	HWND hWnd = CreateWindow(szWindowClass, TEXT(""), WS_OVERLAPPEDWINDOW,
			CW_USEDEFAULT, 0, CW_USEDEFAULT, 0, NULL, NULL, hInstance, NULL);
	if (!hWnd) {
		return 0;
	}

	ShowWindow(hWnd, nCmdShow);
	UpdateWindow(hWnd);

	return hWnd;
}

BOOL createMenu() {
	hTrayMenu = CreatePopupMenu();
	hMenusById[-1] = hTrayMenu;

	MENUINFO menuInfo;
	menuInfo.cbSize = sizeof(MENUINFO);
	menuInfo.fMask = MIM_APPLYTOSUBMENUS | MIM_STYLE;
	menuInfo.dwStyle = MNS_NOTIFYBYPOS;
	return SetMenuInfo(hTrayMenu, &menuInfo);
}

int nativeLoop() {
	HINSTANCE hInstance = GetModuleHandle(NULL);
	TCHAR* szWindowClass = TEXT(const_cast<char*>("SystrayClass"));
	RegisterClass(hInstance, szWindowClass);
	hWnd = InitInstance(hInstance, FALSE, szWindowClass); // Don't show window
	if (!hWnd) {
		return EXIT_FAILURE;
	}
	if (!createMenu()) {
		return EXIT_FAILURE;
	}
	cgoSystrayReady();

	MSG msg;
	while (GetMessage(&msg, NULL, 0, 0)) {
		TranslateMessage(&msg);
		DispatchMessage(&msg);
	}   
	return EXIT_SUCCESS;
}

void show(const char* title, const char* tooltip, const char *iconBytes, int length) {
	// Initialize the tray item
	nid.cbSize = sizeof(NOTIFYICONDATA);
	nid.hWnd = hWnd;
	nid.uID = ICON_UID;
	nid.uCallbackMessage = WM_SYSTRAY_MESSAGE;
	nid.uFlags = NIF_MESSAGE;
	Shell_NotifyIcon(NIM_ADD, &nid);

	// Set the tooltip
	strncpy(nid.szTip, tooltip, sizeof(nid.szTip)/sizeof(char));
	nid.uFlags = NIF_TIP;
	Shell_NotifyIcon(NIM_MODIFY, &nid);

	// Get a temp path env string (no guarantee it's a valid path).
	TCHAR lpTempPathBuffer[MAX_PATH];
	DWORD dwRetVal = GetTempPath(MAX_PATH, lpTempPathBuffer);
	if (dwRetVal > MAX_PATH || (dwRetVal == 0)) {
		reportError("Error in GetTempPath");
		return;
	}

	// Generate a temporary file name. 
	TCHAR szTempFileName[MAX_PATH];  
	UINT uRetVal = GetTempFileName(lpTempPathBuffer, // directory for tmp files
								   TEXT("trayicon"), // temp file name prefix 
								   0,                // create unique name 
								   szTempFileName);  // buffer for name 
	if (uRetVal == 0) {
		reportError("Error in GetTempFileName");
		return;
	}

	// Change the extension to .ico
	strncpy(szTempFileName+strlen(szTempFileName)-3, "ico", 3);

	// Create the new file
	HANDLE hTempFile = INVALID_HANDLE_VALUE; 
	hTempFile = CreateFile((LPTSTR)szTempFileName, // file name 
						   GENERIC_WRITE,        // open for write 
						   0,                    // do not share 
						   NULL,                 // default security 
						   CREATE_ALWAYS,        // overwrite existing
						   FILE_ATTRIBUTE_NORMAL,// normal file 
						   NULL);                // no template 

	if (hTempFile == INVALID_HANDLE_VALUE) {
		reportError("Error creating temporary file for tray icon");
		return;
	}

	// Write to file
	DWORD bytesWritten;
	WriteFile(hTempFile, iconBytes, length, &bytesWritten, NULL);
	if (bytesWritten != length) {
		reportError("Error writing temporary file for tray icon");
		return;
	}

	// Close the file
	CloseHandle(hTempFile);

	// Load the icon
	HICON hIcon = (HICON)LoadImage(NULL, szTempFileName, IMAGE_ICON, 64, 64, LR_LOADFROMFILE);
	if (hIcon == NULL) {
		reportError("Error loading icon");
		return;
	}

	// Create the tray item
	nid.hIcon = hIcon;
	nid.uFlags = NIF_ICON;
	Shell_NotifyIcon(NIM_MODIFY, &nid);
}

void add_or_update(int menuId, int menuItemId, MENUITEMINFO* menuItemInfo) {
	HMENU menu = hMenusById[menuId];
	int itemCount = GetMenuItemCount(menu);
	for (int i = 0; i < itemCount; i++) {
		int id = GetMenuItemId(menu, i);
		if (-1 == id) {
			continue;
		}
		if (menuItemId == id) {
			SetMenuItemInfo(menu, i, TRUE, menuItemInfo);
			return;
		}
	}
	InsertMenuItem(menu, -1, TRUE, menuItemInfo);
}

void add_or_update_menu_item(int menuItemId, char* title, char* tooltip, short disabled, short checked) {
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_FTYPE | MIIM_STRING | MIIM_DATA | MIIM_STATE;
	menuItemInfo.fType = MFT_STRING;
	menuItemInfo.dwTypeData = title;
	menuItemInfo.cch = strlen(title) + 1;
	menuItemInfo.dwItemData = (ULONG_PTR)menuItemId;
	menuItemInfo.fState = 0;
	if (disabled == 1) {
		menuItemInfo.fState |= MFS_DISABLED;
	}
	if (checked == 1) {
		menuItemInfo.fState |= MFS_CHECKED;
	}

	add_or_update(-1, menuItemId, &menuItemInfo);
}

void add_separator(int menuItemId) {
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_FTYPE | MIIM_DATA | MIIM_STATE;
	menuItemInfo.fType = MFT_SEPARATOR;
	menuItemInfo.dwItemData = (ULONG_PTR)menuItemId;
	menuItemInfo.fState = 0;

	add_or_update(-1, menuItemId, &menuItemInfo);
}

void add_or_update_submenu(int menuItemId, char *title, char *tooltip) {
	HMENU submenu = CreatePopupMenu();
	hMenusById[menuItemId] = submenu;

	MENUINFO menuInfo;
	menuInfo.cbSize = sizeof(MENUINFO);
	menuInfo.fMask = MIM_APPLYTOSUBMENUS | MIM_STYLE;
	menuInfo.dwStyle = MNS_NOTIFYBYPOS;
	SetMenuInfo(submenu, &menuInfo);
	
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_FTYPE | MIIM_STRING | MIIM_DATA | MIIM_STATE | MIIM_SUBMENU;
	menuItemInfo.fType = MFT_STRING;
	menuItemInfo.dwTypeData = title;
	menuItemInfo.cch = strlen(title) + 1;
	menuItemInfo.dwItemData = (ULONG_PTR)menuItemId;
	menuItemInfo.fState = 0;
	menuItemInfo.hSubMenu = submenu;

	add_or_update(-1, menuItemId, &menuItemInfo);
}

void add_or_update_submenu_item(int submenuId, int menuItemId, char *title, char *tooltip, short disabled, bool checkable, short checked) {
	MENUITEMINFO menuItemInfo;
	menuItemInfo.cbSize = sizeof(MENUITEMINFO);
	menuItemInfo.fMask = MIIM_FTYPE | MIIM_STRING | MIIM_DATA | MIIM_STATE;
	menuItemInfo.fType = MFT_STRING;
	menuItemInfo.dwTypeData = title;
	menuItemInfo.cch = strlen(title) + 1;
	menuItemInfo.dwItemData = (ULONG_PTR)menuItemId;
	menuItemInfo.fState = 0;
	if (disabled) {
		menuItemInfo.fState |= MFS_DISABLED;
	}
	if (checkable && checked) {
		menuItemInfo.fState |= MFS_CHECKED;
	}

	add_or_update(submenuId, menuItemId, &menuItemInfo);
}

void hide() {
	Shell_NotifyIcon(NIM_DELETE, &nid);
}

void* nativeHandle() {
	return hWnd;
}

void cleanupHandle(uint64_t handle) {
	NOTIFYICONDATA icondata;
	icondata.cbSize = sizeof(NOTIFYICONDATA);
	icondata.hWnd = (HWND)handle;
	icondata.uID = ICON_UID;
	icondata.uCallbackMessage = WM_SYSTRAY_MESSAGE;
	icondata.uFlags = NIF_MESSAGE;
	Shell_NotifyIcon(NIM_DELETE, &icondata);
}
