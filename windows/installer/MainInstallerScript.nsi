Name "Kite"
VIProductVersion "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"  ; for some reason we need this, too
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "Copyright © Kite"
!ifdef WRITE_UNINSTALLER_ONLY
	VIAddVersionKey "FileDescription" "Kite Uninstaller"
	VIAddVersionKey "ProductName" "Kite Uninstaller"
	VIAddVersionKey "OriginalFilename" "KiteUninstallerGenerator.exe"
	VIAddVersionKey "InternalName" "KiteUninstallerGenerator"

	OutFile "current_build_bin\out\KiteUninstallerGenerator.exe"
!else
	VIAddVersionKey "FileDescription" "Kite Setup"
	VIAddVersionKey "ProductName" "Kite Setup"
	VIAddVersionKey "OriginalFilename" "KiteSetup.exe"
	VIAddVersionKey "InternalName" "KiteSetup"

	OutFile "current_build_bin\out\KiteSetup.exe"
!endif
Icon "..\tools\artwork\icon\app.ico"
SetCompressor /SOLID lzma
RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\Kite"
BrandingText " "
ShowInstDetails nevershow
ShowUninstDetails nevershow

Var executable_type ; e.g. "installer" "uninstaller" "updater" etc
Var skip_onboarding
Var cmdflags_start
Var cmdflags_substring

!include "MUI.nsh"
!include "LogicLib.nsh"
!include "WordFunc.nsh"
!include "StrFunc.nsh"
${StrLoc} ; must initialize this before it can be used in a Function (a nuance of StrFunc.nsh)
${UnStrLoc}
${StrRep}
${UnStrRep}
!include "FileFunc.nsh"
!include "WinVer.nsh"
!include "GetProcessInfo.nsh"
!include "servicelib.nsh"
!include "x64.nsh"
!include "CPUFeatures.nsh"
!include "NsisIncludes\Debug.nsh"
!include "NsisIncludes\GenerateMachineIDIfAppropriate.nsh"
!include "NsisIncludes\CheckInstallPrereqs.nsh"
!include "NsisIncludes\CheckAlreadyRunningInstallOrUninstall.nsh"
!include "NsisIncludes\KillAllAvailableRunningInstances.nsh"

!define MUI_ICON "..\tools\artwork\icon\app.ico"
!define MUI_UNICON "..\tools\artwork\icon\app.ico"
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_INSTFILES
!define MUI_CUSTOMFUNCTION_ABORT UserAbortedInstallCallback

Function UserAbortedInstallCallback
	${Debug} "User aborted install"
FunctionEnd

Function un.onInit
	StrCpy $executable_type "uninstaller"
	Call un.CheckAlreadyRunningInstallOrUninstall
FunctionEnd

Function .onInit
	!ifdef WRITE_UNINSTALLER_ONLY
		; this is a build of the setup that is only meant to emit the uninstaller (so we can subsequently sign it)
		System::Call "kernel32::GetCurrentDirectoryW(i ${NSIS_MAX_STRLEN}, t .r0)"
		WriteUninstaller "$0\Uninstaller.exe"
		SetErrorLevel 0
		Quit
	!endif

	; we are a 32 bit installer, uninstaller, and updater, but the main Kite binaries are 64-bit, so we
	;   try to standardize on the 64-bit view where possible.
	SetRegView 64

	${StrLoc} $0 $CMDLINE "testprereqsonly" ">"
	${If} $0 != ""
		${Debug} "testprereqsonly command arg set; testing prereqs only.."

		Call CheckInstallPrereqs
		Pop $0
		Pop $1
		${If} $0 == "ok"
			${Debug} "prereqs checked out ok"
			SetErrorLevel 0 ; just pick some rare values
		${Else}
			${Debug} "prereqs checked failed"
			SetErrorLevel 14
		${EndIf}

		Quit
	${EndIf}

	StrCpy $executable_type "installer"

	Call CheckAlreadyRunningInstallOrUninstall

	Call ReadMachineIDOrGenerateIfAppropriate

	; Check for installation prereq's
	${StrLoc} $0 $CMDLINE "skipprereqs" ">"
	${If} $0 == "" ; no match
		Call CheckInstallPrereqs
		Pop $0
		Pop $1
		${If} $0 != "ok"
			${Debug} "Prereq fail reason: $1"

			MessageBox MB_OK|MB_ICONINFORMATION $0
			SetErrorLevel 21
			Quit
		${EndIf}
	${Else}
		${Debug} "Skipping prereqs check due to command line argument..."
	${EndIf}

	; this will make the installer not show any UI (other than MessageBox's)
	; note we don't set this on the uninstaller
	SetSilent silent
FunctionEnd

Section ""
!ifndef WRITE_UNINSTALLER_ONLY ; otherwise don't include an installer section
	; Do this especially before launching Kite or any of the executables.
	Call WriteTentativeOrActualMachineIDToRegistry

	; Let the fun begin!
	${Debug} "Copying files..."
	SetOutPath "$INSTDIR"

	; Launch the splash screen!
	File "current_build_bin\in\KiteSetupSplashScreen.exe"
	File "current_build_bin\in\KiteSetupSplashScreen.exe.config"
	${StrLoc} $0 $CMDLINE "--plugin-launch" ">"
	${If} $0 == "" ; command line flag NOT present -> show the splash screen
		Exec '"$INSTDIR\KiteSetupSplashScreen.exe"'
	${EndIf}

	ClearErrors
 	ReadRegDword $0 HKLM "SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64" "Installed"
	IfErrors 0 redist_found
		File "current_build_bin\in\vc_redist.x64.exe"
		ExecWait '"$INSTDIR\vc_redist.x64.exe" /install /passive /quiet /norestart'
		Delete /REBOOTOK "$INSTDIR\vc_redist.x64.exe"

	redist_found:
	File /r "current_build_bin\in\win-unpacked"
	File "current_build_bin\in\KiteService.exe"
	File "current_build_bin\in\KiteService.exe.config"
	File "current_build_bin\in\tensorflow.dll"
	File "current_build_bin\in\kited.exe"
	File "current_build_bin\in\kite-lsp.exe"

	WriteRegStr HKLM "Software\Kite\AppData" "InstallPath" "$INSTDIR"

	; Set 'Run' key in registry
	;
	; Note: This is updated by client/internal/autostart/autostart_windows.go.
	; The user has the option to disable autostart through the copilot settings so this
	; ensures the 'Run' key is only set when autostart is enabled.
	WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Kite" '"$INSTDIR\kited.exe" --system-boot'

	; Set protocol handler for electron application
	WriteRegStr HKLM "Software\Classes\kite" "" "URL:kite"
	WriteRegStr HKLM "Software\Classes\kite" "URL Protocol" ""
	WriteRegStr HKLM "Software\Classes\kite\shell\open\command" "" '"$INSTDIR\win-unpacked\Kite.exe" "%1"'

	; Add 'Program Files' shortcut.  This is particularly (well, somewhat) important for users who disable
	; auto-start.
	; The last argument on Kite Local Settings points to kited.exe for the "icon" parameter.
	SetShellVarContext all ; install shortcut for all users
	CreateDirectory "$SMPROGRAMS\Kite"
	CreateShortCut "$SMPROGRAMS\Kite\Kite.lnk" "$INSTDIR\kited.exe"

	; Install service
	; don't forget the trailing ';' in the param list
	!insertmacro SERVICE "create" "KiteService" "path=$INSTDIR\KiteService.exe;autostart=1;interact=0;display=KiteService;description=Kite Service maintains your installation of Kite to ensure it is always up to date.;"

	; Setup uninstaller
	File "current_build_bin\out\Uninstaller.exe"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Kite" "DisplayName" "Kite"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Kite" "Publisher" "Manhattan Engineering Inc"
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Kite" "DisplayIcon" "$\"$INSTDIR\KiteService.exe$\""
	WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Kite" "UninstallString" "$\"$INSTDIR\Uninstaller.exe$\""

	; Launch service and kited
	!insertmacro SERVICE "start" "KiteService" ""

	${StrLoc} $skip_onboarding $CMDLINE "--skip-onboarding" ">"
	${If} $skip_onboarding != ""
		${Debug} "skip-onboarding flag specified; creating env var KITE_SKIP_ONBOARDING for kited child process"
		System::Call 'Kernel32::SetEnvironmentVariable(t, t)i ("KITE_SKIP_ONBOARDING", "1").r0'
	${EndIf}

	; This logic takes everything starting at "--" and appends it to kited.exe. The goal is forward along any
	; commandline flags passed into the installer. We first look for `KiteSetup.exe` to exclude any occurances
	; of "--" in the path, then we find find the first `--` and forward everything along to `kited.exe`
	${StrLoc} $cmdflags_start $CMDLINE "KiteSetup.exe" ">"
	${If} $cmdflags_start != ""
		StrCpy $cmdflags_substring $CMDLINE "" $cmdflags_start
		${StrLoc} $cmdflags_start $cmdflags_substring "--" ">"
		${If} $cmdflags_start != ""
			StrCpy $cmdflags_substring $cmdflags_substring "" $cmdflags_start
		${EndIf}
	${EndIf}

	Exec '"$INSTDIR\kited.exe" $cmdflags_substring'

	${Debug} "Install completed."
!endif
SectionEnd

Section "Uninstall"
	; kill all possible running instances
	Call un.KillAllAvailableRunningInstances
	Sleep 2000  ; This used to not be here, but despite best efforts seems like the RMDir still sometimes needs reboot

	; remove old tray icon if appropriate / possible
	SetRegView 64
	ReadRegDWORD $0 HKCU "Software\Kite\AppData" "LastTrayHwnd"
	${If} $0 > 0
		System::Call '*(&l4, i, i, i, i, i, &t64) i(, $0, 1702127979, 0, 0, 0, "") .r0'
		System::Call 'Shell32::Shell_NotifyIcon(i 2, i r0) i.r1'
		System::Free $0
	${EndIf}

	; Note that kited.exe could still be running in other user sessions
	; Thus we'll specify /REBOOTOK when deleting files, and let the user know if they need to reboot

	; stop and uninstall service
	!insertmacro SERVICE "stop" "KiteService" ""
	Sleep 2000
	FindProcDLL::WaitProcEnd "KiteService.exe" 20000
	Sleep 2000  ; This used to not be here, but despite best efforts seems like the RMDir still sometimes needs reboot
	!insertmacro SERVICE "delete" "KiteService" ""

	RMDir /r /REBOOTOK "$INSTDIR"  ; This will delete all of the files, including the uninstaller.

	; the line below is commented out, so that $LOCALAPPDATA\Kite will be left behind.
	; this is important so that the editors (at least Atom) know Kite has been installed previously
	;   -> they can differentiate an uninstall vs the plugin was installed and it needs to show
	;   the installation wizard.
	; this also mirrors the uninstallation behavior on macOS of leaving behind ~/.kite
	; RMDir /r /REBOOTOK "$LOCALAPPDATA\Kite"  ; Log files, etc.  We might leave behind ones for other users.

	; delete the 'Program Files' shortcut
	SetShellVarContext all ; uninstall shortcut for all users
	Delete /REBOOTOK "$SMPROGRAMS\Kite\Kite.lnk"
	Delete /REBOOTOK "$SMPROGRAMS\Kite\Kite Local Settings.lnk"
	RMDir "$SMPROGRAMS\Kite"

	; there might be other AppData's for other users, but we'll unfortunately have
	;   to leave those behind.
	DeleteRegKey HKCU "Software\Kite\AppData" ; Don't delete the MachineID
	DeleteRegKey HKLM "Software\Kite\AppData" ; Don't delete the MachineID
	SetRegView 32
	DeleteRegKey HKCU "Software\Kite\AppData" ; Don't delete the MachineID
	DeleteRegKey HKLM "Software\Kite\AppData" ; Don't delete the MachineID
	SetRegView 64

	DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Kite"  ; note: HKLM!

	; Delete the 'Run' key if it exists
	DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "Kite"  ; note: HKCU!

	; Delete protocol handler for electron application
	DeleteRegKey HKLM "Software\Classes\kite"

	; It gets added here too for some reason; have to to delete it in HKCR as well
	DeleteRegKey HKCR "kite"

	IfRebootFlag 0 noreboot
		MessageBox MB_YESNO|MB_ICONINFORMATION "There are some files that will not be deleted until you reboot your computer, probably because another user is running Kite.  Would you like to reboot now?" IDNO noreboot
		Reboot
	noreboot:

	${un.Debug} "Uninstall completed."
SectionEnd
