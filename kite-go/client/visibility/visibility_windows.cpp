#define WIN32_LEAN_AND_MEAN             // Exclude rarely-used stuff from Windows headers

#include <cstdio>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <iostream>
#include <string>

#include <windows.h>
#include <tlhelp32.h>
#include <Psapi.h>

#include "visibility_windows.h"

std::string pathFromWindow(HWND hWnd) {
    DWORD processID;
    GetWindowThreadProcessId(hWnd, &processID);

    HANDLE hProcess = OpenProcess(PROCESS_QUERY_INFORMATION | PROCESS_VM_READ, FALSE, processID);
    if (hProcess == NULL) {
        return "";
    }

    TCHAR nameProc[MAX_PATH];
    if (!GetProcessImageFileName(hProcess, nameProc, MAX_PATH)) {
        return "";
    }

    CloseHandle(hProcess);
    return std::string(nameProc);
}

struct paramData {
    char* name;
    bool visible;
};

BOOL CALLBACK visibleEnumWindows(HWND hWnd, LPARAM lParam) {
    // Ignore minimized windows
    if (IsIconic(hWnd)) {
        return true;
    }

    paramData* data = (paramData*)lParam;

    // TODO(tarak): sorry, strings are easier to work with ...
    std::string exeName((char*)data->name);
    std::string fullPath = pathFromWindow(hWnd);

    // Is this the process we are looking for? Do some string things to check...
    if (fullPath.length() < exeName.length()) {
        return true;
    }

    // Actually compare the exeName...
    if (fullPath.compare(fullPath.length() - exeName.length(), exeName.length(), exeName)) {
        return true;
    }

    RECT rect;
    if (!GetWindowRect(hWnd, &rect)) {
        return true;
    }

    // Ignore obviously weird windows
    if (rect.left == 0 && rect.top == 0 && rect.right == 0 && rect.bottom == 0) {
        return true;
    }

    // Compute center (NOTE: origin is top-left of main window, and we are ok with integer math)
    POINT center;
    center.x = rect.left + (rect.right - rect.left)/2;
    center.y = rect.top + (rect.bottom - rect.top)/2;

    // Point is not contained in any display monitor
    HMONITOR mon = MonitorFromPoint(center, MONITOR_DEFAULTTONULL);
    if (mon == NULL) {
        return true;
    }

    CloseHandle(mon);

    // Compare image names
    HWND cWnd = WindowFromPoint(center);
    if (pathFromWindow(cWnd) == fullPath) {
        data->visible = true;
    }

    CloseHandle(cWnd);

    return true;
}

bool windowVisible(char* name) {
    struct paramData data = {0};
    data.name = name;
    EnumWindows(visibleEnumWindows, (LPARAM)&data);
    return data.visible;
}
