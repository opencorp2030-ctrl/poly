; Poly Windows installer.
; Builds with: makensis /DVERSION=0.5.0 /DSOURCE_EXE=path\to\poly-windows-amd64.exe poly-installer.nsi
;
; Installs poly.exe into %LOCALAPPDATA%\Poly (no admin rights needed),
; adds that folder to the current user's PATH, registers itself in
; Add/Remove Programs, and writes an uninstaller.

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef SOURCE_EXE
  !error "Pass /DSOURCE_EXE=path\to\poly-windows-amd64.exe"
!endif

!define REG_UNINSTALL "Software\Microsoft\Windows\CurrentVersion\Uninstall\Poly"

!include "MUI2.nsh"
!include "WinMessages.nsh"

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

  ; Register in Add/Remove Programs so users can uninstall from Settings.
  WriteRegStr HKCU "${REG_UNINSTALL}" "DisplayName" "Poly"
  WriteRegStr HKCU "${REG_UNINSTALL}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${REG_UNINSTALL}" "Publisher" "Poly"
  WriteRegStr HKCU "${REG_UNINSTALL}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${REG_UNINSTALL}" "DisplayIcon" "$INSTDIR\poly.exe"
  WriteRegStr HKCU "${REG_UNINSTALL}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKCU "${REG_UNINSTALL}" "NoModify" 1
  WriteRegDWORD HKCU "${REG_UNINSTALL}" "NoRepair" 1

  ; Add $INSTDIR to the current user's PATH if it isn't there already.
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
  DeleteRegKey HKCU "${REG_UNINSTALL}"
  RMDir "$INSTDIR"

  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000
SectionEnd

; --- PATH helpers -----------------------------------------------------
; Adds a directory to HKCU\Environment\Path (preserving any %VAR% tokens
; by writing REG_EXPAND_SZ), skipping the add when the directory is
; already listed as its own ";"-delimited element -- so C:\Poly never
; matches inside C:\PolyExtra.

Function AddToPath
  Exch $1 ; dir to add
  Push $0
  Push $2
  Push $3

  ReadRegStr $0 HKCU "Environment" "Path"
  StrCmp $0 "" writeNew
  StrCpy $2 "$0;"
  StrCpy $3 "$1;"
  Push $2
  Push $3
  Call StrStr
  Pop $2
  StrCmp $2 "" doAdd done

  doAdd:
    StrCpy $0 "$0;$1"
    Goto write
  writeNew:
    ; No user PATH yet: write the directory itself instead of an empty string.
    StrCpy $0 "$1"
  write:
    WriteRegExpandStr HKCU "Environment" "Path" "$0"
  done:
    Pop $3
    Pop $2
    Pop $0
    Pop $1
FunctionEnd

Function un.RemoveFromPath
  Exch $0 ; dir to remove
  Push $1
  ReadRegStr $1 HKCU "Environment" "Path"
  StrCmp $1 "" done            ; nothing to clean
  StrCmp $1 $0 exactMatch      ; the dir is the whole PATH
  Push $1
  Push "$0;"
  Call un.StrRep
  Pop $1
  Push $1
  Push ";$0"
  Call un.StrRep
  Pop $1
  Goto write
  exactMatch:
    StrCpy $1 ""
  write:
    StrCmp $1 "" deleteValue
    WriteRegExpandStr HKCU "Environment" "Path" "$1"
    Goto done
  deleteValue:
    DeleteRegValue HKCU "Environment" "Path"
  done:
    Pop $1
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
  notfound:
    StrCpy $R5 ""
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
