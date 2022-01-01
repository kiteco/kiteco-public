!include "NsisIncludes\DetermineIfDotNetThreeOrFourInstalled.nsh"

; using the pattern from http://nsis.sourceforge.net/Sharing_functions_between_Installer_and_Uninstaller
; so this function can be called during uninstalls, too.
!macro CIP_FUNC_MACRO un

; Typical Usage:
;
; Call CheckInstallPrereqs
; Pop $0  ; long, readable error message, or "ok" if no error
; Pop $1  ; short, machine_like_error_message, or "ok" if there was no error
; ${If} $0 != "ok"
;		MessageBox MB_OK|MB_ICONINFORMATION $0
;		Quit
; ${EndIf}
;
; NOTE: clobbers $0 thru $7
Function ${un}CheckInstallPrereqs

	; Prerequisites --
	;   Kite can't be already installed or have been installed.
	;   CPU must support the AVX instruction set (required for tensorflow.dll, currently)
	;   64-bit Windows
	;   Running as admin
	;   Windows 7 or higher
	;   Is a .NET CLR installed that can run .NET 3.5 apps?


	; Are we already installed?
	${DirState} "$PROGRAMFILES64\Kite" $1
	${If} "$1" != "-1" ; -1 means the directory was not found
		Push "already_installed_directory"
		Push "It looks like Kite is already installed.$\n$\nIf you wish to reinstall Kite then first uninstall it via Control Panel, and make sure $PROGRAMFILES64\Kite is deleted before re-running setup.$\n$\nSetup will now exit."
		Return
	${EndIf}

	; Does CPU support the AVX instruction set?
	${IfNot} ${CPUSupports} "AVX1"
		Push "avx1_not_supported"
		Push "Kite cannot run on your computer at the moment, unfortunately. We use a library called Tensorflow which requires that your CPU support the AVX instruction set. This instruction set is supported on most, but not all, computers built after 2012.$\n$\nYou can discuss or follow this issue at https://github.com/kiteco/plugins/issues/118.$\n$\nWe're sorry you can't install Kite for now. Setup will now exit."
		Return
	${EndIf}

	; 64-bit Windows?
	${IfNot} ${RunningX64}
		Push "not_64_bit_windows"
		Push "Kite requires 64-bit Windows. Setup will now exit."
		Return
	${EndIf}

	; Running as admin?
	UserInfo::GetAccountType
	Pop $0
	${If} $0 != "Admin"
		Push "not_admin_user"
		Push "Kite setup must be ran under an account with Administrator privileges. Setup will now exit."
		Return
	${EndIf}

	; Are we running an okay version of Windows?
	${IfNot} ${AtLeastWin7}
		Push "incompatible_windows_version"
		Push "Kite did not detect a compatible version of Microsoft Windows. Setup will now exit."
		Return
	${EndIf}

	; Check for .NET 3.5+ Framework
	Call ${un}DetermineIfDotNetThreeOrFourInstalled
	Pop $0
	${If} $0 == ""
		Push "incompatible_dotnet_version"
		Push "Kite requires the Microsoft .NET Framework 3.5 or higher. Setup will now exit."
		Return
	${EndIf}

  Push "ok"
  Push "ok"

FunctionEnd

!macroend

!insertmacro CIP_FUNC_MACRO ""
!insertmacro CIP_FUNC_MACRO "un."
