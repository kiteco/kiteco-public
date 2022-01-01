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

#include "sidebar_windows.h"

// isRunning returns whether we fine a process running with the same executable name
bool isRunning(char* name) {
    // convert char* path to wchar_t
    wchar_t nameW[MAX_PATH];
    mbstowcs(nameW, name, MAX_PATH);

    bool exists = false;
    PROCESSENTRY32W entry;
    entry.dwSize = sizeof(PROCESSENTRY32W);

    HANDLE snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, (DWORD)0);

    if (Process32FirstW(snapshot, &entry)) {
        while (Process32NextW(snapshot, &entry)) {
            if (wcsicmp(entry.szExeFile, nameW) == 0) {
                exists = true;
                break;
            }
        }
    }

    CloseHandle(snapshot);
    return exists;
}

// terminateAppEnum is the callback used for EnumWindows in killIfRunning
// to attempt to gracefully shutdown. It posts WC_CLOSE to all windows
BOOL CALLBACK terminateAppEnum(HWND hWnd, LPARAM lParam) {
   DWORD processID;
   GetWindowThreadProcessId(hWnd, &processID);
   if(processID == (DWORD)lParam) {
      PostMessage(hWnd, WM_CLOSE, 0, 0);
   }

   return true;
}

// killIfRunning will kill processes with the provided executable name if they are running
void killIfRunning(char *name) {
    // convert char* path to wchar_t
    wchar_t nameW[MAX_PATH];
    mbstowcs(nameW, name, MAX_PATH);

    DWORD timeout = 5000; // 5 second timeout

    PROCESSENTRY32W entry;
    entry.dwSize = sizeof(PROCESSENTRY32W);

    HANDLE snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, (DWORD)0);

    if (Process32FirstW(snapshot, &entry)) {
        while (Process32NextW(snapshot, &entry)) {
            if (!wcsicmp(entry.szExeFile, nameW)) {
                HANDLE hProcess = OpenProcess(SYNCHRONIZE|PROCESS_TERMINATE, FALSE, entry.th32ProcessID);
                if (hProcess == NULL) {
                    continue;
                }

                // terminateAppEnum posts WM_CLOSE to all windows for this process
                // based on https://support.microsoft.com/en-us/help/178893/how-to-terminate-an-application-cleanly-in-win32
                EnumWindows(terminateAppEnum, (LPARAM)entry.th32ProcessID);
                if (WaitForSingleObject(hProcess, timeout) != WAIT_OBJECT_0) {
                    // Terminate if it doesn't close in time
                    TerminateProcess(hProcess, 0);
                }

                CloseHandle(hProcess);
            }
        }
    }

    CloseHandle(snapshot);
}

// focusEnumWindows is the callback method passed into EnumWindows in void focus()
BOOL CALLBACK focusEnumWindows(HWND hWnd, LPARAM lParam) {
    DWORD processID;
    GetWindowThreadProcessId(hWnd, &processID);
    if (processID == 0) {
        return true;
    }

    DWORD currentID = GetCurrentThreadId();
    if (currentID == 0) {
        return true;
    }

    HANDLE hProcess = OpenProcess(PROCESS_QUERY_INFORMATION | PROCESS_VM_READ, FALSE, processID);
    if (hProcess == NULL) {
        return true;
    }

    wchar_t nameProcW[MAX_PATH];
    if (!GetProcessImageFileNameW(hProcess, nameProcW, MAX_PATH)) {
        return true;
    }

    // Convert exe name argument to wchar_t
    wchar_t nameW[MAX_PATH];
    mbstowcs(nameW, (char*)lParam, MAX_PATH);

    std::wstring exeName(nameW);
    std::wstring fullPath(nameProcW);

    // Is this the process we are looking for? Do some string things to check...
    if (fullPath.length() >= exeName.length()) {
        if (fullPath.compare(fullPath.length() - exeName.length(), exeName.length(), exeName) == 0) {
            // If the window is minimize, bring it back up
            if (IsIconic(hWnd)) {
                ShowWindow(hWnd, SW_RESTORE);
            }

            SetForegroundWindow(hWnd);
            SetActiveWindow(hWnd);

            if (currentID != processID) {
                AttachThreadInput(currentID, processID, true);
                SetFocus(hWnd);
                AttachThreadInput(currentID, processID, false);
            }
        }
    }

    CloseHandle(hProcess);

    return true;
}

// focus will focus the app with the provided executable name
void focus(char *name) {
    EnumWindows(focusEnumWindows, (LPARAM)name);
}