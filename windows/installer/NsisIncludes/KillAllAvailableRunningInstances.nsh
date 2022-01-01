
; using the pattern from http://nsis.sourceforge.net/Sharing_functions_between_Installer_and_Uninstaller
!macro KILL_PROC_MACRO un

Function ${un}KillAllAvailableRunningInstances

	; NOTE: a "polite" close isn't easy at all since we can't PostMessage() to kited.exe
	; since it's running from the user's desktop and we're sometimes running from Session0.  So
	; we just have to TerminateProcess().
	;
	; WARNING: FindProc and KillProc *might* not work if there are multiple users running the
	;   executable.  More specifically, if this code is running from Session0 as SYSTEM then
	;   we'll be able to kill all running instances (e.g. during an update).  If it's running
	;   as a normal user we'll only be able to find and terminate the instances in the current
	;   session/desktop.

	Push $R0

	FindProcDLL::FindProc "kited.exe"
	${If} $R0 == 1
		!insertmacro ${un}Debug "kited is currently running.  Killing it now..."

		FindProcDLL::KillProc "kited.exe"
		${If} $R0 != 0  ; 0 = the process was not found.
		${AndIf} $R0 != 1  ; 1 = at least one process, maybe more, were successfully terminated.
			!insertmacro ${un}Debug "Error killing kited.exe."
		${EndIf}
		${If} $R0 == 1
			; at this point every matching process was killed
			; use WaitProcEnd to wait for all of them to exit
			; this should happen very quickly, because KillProc uses TerminateProcess(), which doesn't
			;   give the process the chance to do anything before ending, but we check just in case.
			; we use a non-infinite timeout because it's possible that another instance of the app was
			;   launched in the interim, so we don't want to hang.
			!insertmacro ${un}Debug "Waiting for kited.exe to really be gone..."
			FindProcDLL::WaitProcEnd "kited.exe" 4000
			${If} $R0 == 100
				!insertmacro ${un}Debug "Timed out waiting for process to terminate"
			${Else}
				!insertmacro ${un}Debug "Process terminated successfully"
			${EndIf}
		${EndIf}
	${EndIf}

	; below is a copy-paste for Kite.exe.
	; sorry about that!

	FindProcDLL::FindProc "Kite.exe"
	${If} $R0 == 1
		!insertmacro ${un}Debug "Kite is currently running.  Killing it now..."

		FindProcDLL::KillProc "Kite.exe"
		${If} $R0 != 0  ; 0 = the process was not found.
		${AndIf} $R0 != 1  ; 1 = at least one process, maybe more, were successfully terminated.
			!insertmacro ${un}Debug "Error killing Kite.exe."
		${EndIf}
		${If} $R0 == 1
			; at this point every matching process was killed
			; use WaitProcEnd to wait for all of them to exit
			; this should happen very quickly, because KillProc uses TerminateProcess(), which doesn't
			;   give the process the chance to do anything before ending, but we check just in case.
			; we use a non-infinite timeout because it's possible that another instance of the app was
			;   launched in the interim, so we don't want to hang.
			!insertmacro ${un}Debug "Waiting for Kite.exe to really be gone..."
			FindProcDLL::WaitProcEnd "Kite.exe" 4000
			${If} $R0 == 100
				!insertmacro ${un}Debug "Timed out waiting for process to terminate"
			${Else}
				!insertmacro ${un}Debug "Process terminated successfully"
			${EndIf}
		${EndIf}
	${EndIf}

	; below is a copy-paste for KiteSetupSplashScreen.exe.
	; sorry about that!

	FindProcDLL::FindProc "KiteSetupSplashScreen.exe"
	${If} $R0 == 1
		!insertmacro ${un}Debug "KiteSetupSplashScreen is currently running.  Killing it now..."

		FindProcDLL::KillProc "KiteSetupSplashScreen.exe"
		${If} $R0 != 0  ; 0 = the process was not found.
		${AndIf} $R0 != 1  ; 1 = at least one process, maybe more, were successfully terminated.
			!insertmacro ${un}Debug "Error killing KiteSetupSplashScreen.exe."
		${EndIf}
		${If} $R0 == 1
			; at this point every matching process was killed
			; use WaitProcEnd to wait for all of them to exit
			; this should happen very quickly, because KillProc uses TerminateProcess(), which doesn't
			;   give the process the chance to do anything before ending, but we check just in case.
			; we use a non-infinite timeout because it's possible that another instance of the app was
			;   launched in the interim, so we don't want to hang.
			!insertmacro ${un}Debug "Waiting for KiteSetupSplashScreen.exe to really be gone..."
			FindProcDLL::WaitProcEnd "KiteSetupSplashScreen.exe" 4000
			${If} $R0 == 100
				!insertmacro ${un}Debug "Timed out waiting for process to terminate"
			${Else}
				!insertmacro ${un}Debug "Process terminated successfully"
			${EndIf}
		${EndIf}
	${EndIf}

	Pop $R0
FunctionEnd

!macroend

!insertmacro KILL_PROC_MACRO ""
!insertmacro KILL_PROC_MACRO "un."