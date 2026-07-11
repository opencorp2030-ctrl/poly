; Poly Windows installer.
; Builds with: makensis /DVERSION=0.5.0 /DSOURCE_EXE=path\to\poly-windows-amd64.exe poly-installer.nsi
;
; Installs poly.exe into %LOCALAPPDATA%\Poly (no admin rights needed),
; adds that folder to the current user's PATH, and writes an uninstaller.

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef SOURCE_EXE
  !error "Pass /DSOURCE_EXE=path\to\poly-windows-amd64.exe"
!endif

!include "MUI2.nsh"

Name "Poly"
OutFile "poly-setup-${VERSION}.exe"
InstallDir "$LOCALAPPDATA\Poly"
InstallDirRegKey HKCU "Software\Poly" "InstallDir"
RequestExecutionLevel user

!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "French"

Section "Poly" SecPoly
  SetOutPath "$INSTDIR"
  File /oname=poly.exe "${SOURCE_EXE}"

  WriteRegStr HKCU "Software\Poly" "InstallDir" "$INSTDIR"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Add $INSTDIR to the current user's PATH if it isn't there already.
  ReadRegStr $0 HKCU "Environment" "Path"
  Push "$0"
  Push "$INSTDIR"
  Call AddToPath

  ; Tell the shell PATH changed so new terminals pick it up without a reboot.
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\poly.exe"
  Delete "$INSTDIR\uninstall.exe"

  Push "$INSTDIR"
  Call un.RemoveFromPath

  DeleteRegKey HKCU "Software\Poly"
  RMDir "$INSTDIR"

  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd

; --- PATH helpers -----------------------------------------------------
; Adapted from the standard NSIS "add to PATH" recipe: appends a
; directory to HKCU\Environment\Path if it isn't already a substring.

Function AddToPath
  Exch $1 ; dir to add
  Exch
  Exch $0 ; current PATH value
  Push $2
  Push $3

  StrCpy $2 "$0;"
  StrCpy $3 "$1;"
  Push $2
  Push $3
  Call StrStr
  Pop $2
  StrCmp $2 "" doAdd alreadyThere

  doAdd:
    StrCmp $0 "" writeNew
    StrCpy $0 "$0;$1"
    Goto writeNew
  writeNew:
    WriteRegExpandStr HKCU "Environment" "Path" "$0"
    Goto done
  alreadyThere:
  done:
    Pop $3
    Pop $2
    Pop $0
    Pop $1
FunctionEnd

Function un.RemoveFromPath
  Exch $0 ; dir to remove
  ReadRegStr $1 HKCU "Environment" "Path"
  Push $1
  Push "$0;"
  Call un.StrRep
  Pop $1
  Push $1
  Push ";$0"
  Call un.StrRep
  Pop $1
  WriteRegExpandStr HKCU "Environment" "Path" "$1"
  Pop $0
FunctionEnd

Function StrStr
  Exch $R1 ; needle
  Exch
  Exch $R2 ; haystack
  Push $R3
  Push $R4
  Push $R5
  StrLen $R3 $R1
  StrCpy $R4 0
  loop:
    StrCpy $R5 $R2 $R3 $R4
    StrCmp $R5 $R1 found
    StrCmp $R5 "" notfound
    IntOp $R4 $R4 + 1
    Goto loop
  found:
    StrCpy $R1 $R2 "" $R4
    Goto end
  notfound:
    StrCpy $R1 ""
  end:
    Pop $R5
    Pop $R4
    Pop $R3
    Pop $R2
    Exch $R1
FunctionEnd

Function un.StrRep
  Exch $R0 ; substring to remove
  Exch
  Exch $R1 ; input string
  Push $R2
  Push $R3
  StrCpy $R3 ""
  StrLen $R2 $R0
  loop:
    StrCpy $R4 $R1 $R2
    StrCmp $R4 $R0 found
    StrCmp $R1 "" done
    StrCpy $R4 $R1 1
    StrCpy $R3 "$R3$R4"
    StrCpy $R1 $R1 "" 1
    Goto loop
  found:
    StrCpy $R1 $R1 "" $R2
    Goto loop
  done:
    StrCpy $R1 $R3
    Pop $R3
    Pop $R2
    Pop $R0
    Exch $R1
FunctionEnd
