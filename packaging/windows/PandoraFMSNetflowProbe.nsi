; ---------------------------------------------------------------------------
; Pandora FMS Netflow Probe — instalador para Windows (64 bits), es/en.
;
; Basado en el instalador de Pandora MINI. Se compila desde macOS/Linux con
; makensis (ver make_installer.sh); no hace falta Windows.
;
; Instala en Archivos de programa la GUI, el agente CLI, WinDivert (DLL+driver)
; y, opcionalmente, el "modo compatible" con Mesa para equipos sin OpenGL 2.x
; (máquinas virtuales, RDP, drivers antiguos).
;
; La configuración del usuario (%USERPROFILE%\.netflow-probe.yaml) NUNCA se toca.
;
; Parámetros (los pasa make_installer.sh con -D):
;   VERSION, EXE_SRC, CLI_SRC, WINDIVERT_DLL, WINDIVERT_SYS, OUT_FILE,
;   ICO, WELCOME_BMP, HEADER_BMP, LICENSE_ES, LICENSE_EN,
;   WITH_MESA, MESA_DIR, COMPAT_BAT, LEEME
; ---------------------------------------------------------------------------
Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
!ifndef EXE_SRC
  !define EXE_SRC "packaging/windows/payload/PandoraFMSNetflowProbe.exe"
!endif
!ifndef CLI_SRC
  !define CLI_SRC "packaging/windows/payload/netflow-probe.exe"
!endif
!ifndef WINDIVERT_DLL
  !define WINDIVERT_DLL "packaging/windows/payload/WinDivert.dll"
!endif
!ifndef WINDIVERT_SYS
  !define WINDIVERT_SYS "packaging/windows/payload/WinDivert64.sys"
!endif
!ifndef OUT_FILE
  !define OUT_FILE "Pandora FMS Netflow Probe Setup.exe"
!endif
!ifndef ICO
  !define ICO "packaging/windows/AppIcon.ico"
!endif
!ifndef WELCOME_BMP
  !define WELCOME_BMP "packaging/windows/welcome.bmp"
!endif
!ifndef HEADER_BMP
  !define HEADER_BMP "packaging/windows/header.bmp"
!endif
!ifndef LICENSE_ES
  !define LICENSE_ES "packaging/macos/dmg-resources/legal/es.txt"
!endif
!ifndef LICENSE_EN
  !define LICENSE_EN "packaging/macos/dmg-resources/legal/en.txt"
!endif
!ifndef LEEME
  !define LEEME "packaging/windows/LEEME.txt"
!endif

!define LANG_EN 1033
!define LANG_ES 1034

!define APP_NAME "Pandora FMS Netflow Probe"
!define APP_EXE "PandoraFMSNetflowProbe.exe"
; tasklist trunca el nombre de imagen (~24 chars) en su salida, así que el
; chequeo de "app en ejecución" busca un prefijo corto, no APP_EXE completo.
!define APP_IMAGE_PREFIX "PandoraFMSNetflow"
!define APP_ICO "AppIcon.ico"
!define PUBLISHER "Pandora FMS"
!define APP_KEY "Software\PandoraFMS\NetflowProbe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\PandoraFMSNetflowProbe"

Name "${APP_NAME} ${VERSION}"
OutFile "${OUT_FILE}"
InstallDir "$PROGRAMFILES64\${APP_NAME}"
InstallDirRegKey HKLM "${APP_KEY}" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APP_NAME}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "CompanyName" "${PUBLISHER}"
VIAddVersionKey "LegalCopyright" "(c) Pandora FMS 2026"
VIAddVersionKey "FileDescription" "${APP_NAME} ${VERSION} - Instalador"

; --- interfaz ---------------------------------------------------------------
!define MUI_ICON "${ICO}"
!define MUI_UNICON "${ICO}"
!define MUI_ABORTWARNING
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "${HEADER_BMP}"
!define MUI_WELCOMEFINISHPAGE_BITMAP "${WELCOME_BMP}"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "${WELCOME_BMP}"
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_SHOWREADME "$INSTDIR\LEEME.txt"

; --- textos ----------------------------------------------------------------
LangString TXT_SEC_APP     ${LANG_ES} "Pandora FMS Netflow Probe (obligatorio)"
LangString TXT_SEC_APP     ${LANG_EN} "Pandora FMS Netflow Probe (required)"
LangString TXT_SEC_APP_D   ${LANG_ES} "La aplicación, el agente CLI y WinDivert (driver de captura). Se instala en Archivos de programa."
LangString TXT_SEC_APP_D   ${LANG_EN} "The app, the CLI agent and WinDivert (capture driver). Installed under Program Files."

LangString TXT_SEC_DSK     ${LANG_ES} "Acceso directo en el escritorio"
LangString TXT_SEC_DSK     ${LANG_EN} "Desktop shortcut"
LangString TXT_SEC_DSK_D   ${LANG_ES} "Crea un icono en el escritorio."
LangString TXT_SEC_DSK_D   ${LANG_EN} "Creates a desktop icon."

LangString TXT_SEC_SWGL    ${LANG_ES} "Modo compatible (equipos sin OpenGL 2.x)"
LangString TXT_SEC_SWGL    ${LANG_EN} "Compatible mode (computers without OpenGL 2.x)"
LangString TXT_SEC_SWGL_D  ${LANG_ES} "Copia del programa con Mesa que dibuja por software. Necesario en máquinas virtuales, escritorio remoto (RDP) o drivers genéricos. Ocupa unos 60 MB más."
LangString TXT_SEC_SWGL_D  ${LANG_EN} "A copy of the program with Mesa that renders in software. Needed on virtual machines, remote desktop (RDP) or generic drivers. Takes about 60 MB more."
LangString TXT_SWGL_SH      ${LANG_ES} "Pandora FMS Netflow Probe (modo compatible)"
LangString TXT_SWGL_SH      ${LANG_EN} "Pandora FMS Netflow Probe (compatible mode)"

LangString TXT_UNINST_SH   ${LANG_ES} "Desinstalar Pandora FMS Netflow Probe"
LangString TXT_UNINST_SH   ${LANG_EN} "Uninstall Pandora FMS Netflow Probe"

LangString TXT_UPGRADE     ${LANG_ES} "Se ha detectado una instalacion anterior. Se actualizara el programa; tu configuracion (%USERPROFILE%\.netflow-probe.yaml) se conserva intacta."
LangString TXT_UPGRADE     ${LANG_EN} "An earlier installation was found. The program will be updated; your configuration (%USERPROFILE%\.netflow-probe.yaml) is left untouched."
LangString TXT_RUNNING     ${LANG_ES} "La aplicacion esta en ejecucion. Cierrala y pulsa Reintentar para continuar."
LangString TXT_RUNNING     ${LANG_EN} "The application is running. Close it and click Retry to continue."
LangString TXT_DATA_KEPT   ${LANG_ES} "El programa se ha desinstalado. Tu configuracion se conserva en %USERPROFILE%\.netflow-probe.yaml; borrala a mano si ya no la quieres."
LangString TXT_DATA_KEPT   ${LANG_EN} "The program has been uninstalled. Your configuration is kept in %USERPROFILE%\.netflow-probe.yaml; delete it by hand if you no longer want it."

LicenseLangString LICENSETEXT ${LANG_ES} "${LICENSE_ES}"
LicenseLangString LICENSETEXT ${LANG_EN} "${LICENSE_EN}"

; --- páginas ---------------------------------------------------------------
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "$(LICENSETEXT)"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_RESERVEFILE_LANGDLL
!insertmacro MUI_LANGUAGE "Spanish"
!insertmacro MUI_LANGUAGE "English"

; --- arranque --------------------------------------------------------------
Function .onInit
  !insertmacro MUI_LANGDLL_DISPLAY

  ReadRegStr $0 HKLM "${UNINST_KEY}" "DisplayVersion"
  StrCmp $0 "" checkRunning
  MessageBox MB_OKCANCEL|MB_ICONINFORMATION "$(TXT_UPGRADE)$\r$\n$\r$\n$0  ->  ${VERSION}" \
    /SD IDOK IDOK checkRunning
  Abort

checkRunning:
  nsExec::ExecToStack 'cmd /C tasklist /FI "IMAGENAME eq ${APP_EXE}" /NH | find /I "${APP_IMAGE_PREFIX}"'
  Pop $0
  Pop $1
  StrCmp $0 "0" 0 done
  MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "$(TXT_RUNNING)" /SD IDCANCEL IDRETRY checkRunning
  Abort
done:
FunctionEnd

; --- instalación -----------------------------------------------------------
Section "$(TXT_SEC_APP)" SEC_APP
  SectionIn RO
  SetRegView 64
  SetOutPath "$INSTDIR"
  File "/oname=${APP_EXE}" "${EXE_SRC}"
  File "/oname=netflow-probe.exe" "${CLI_SRC}"
  File "/oname=WinDivert.dll" "${WINDIVERT_DLL}"
  File "/oname=WinDivert64.sys" "${WINDIVERT_SYS}"
  File "/oname=${APP_ICO}" "${ICO}"
  File "/oname=LEEME.txt" "${LEEME}"

  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}" "" "$INSTDIR\${APP_ICO}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\$(TXT_UNINST_SH).lnk" "$INSTDIR\Uninstall.exe"

  WriteRegStr HKLM "${APP_KEY}" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${PUBLISHER}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${APP_ICO}"
  WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKLM "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

Section "$(TXT_SEC_DSK)" SEC_DESKTOP
  CreateShortCut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}" "" "$INSTDIR\${APP_ICO}"
SectionEnd

!ifdef WITH_MESA
Section "$(TXT_SEC_SWGL)" SEC_SWGL
  SetRegView 64
  SetOutPath "$INSTDIR\swgl"
  File "/oname=${APP_EXE}" "${EXE_SRC}"
  File "/oname=WinDivert.dll" "${WINDIVERT_DLL}"
  File "/oname=WinDivert64.sys" "${WINDIVERT_SYS}"
  File "${MESA_DIR}/opengl32.dll"
  File "${MESA_DIR}/libgallium_wgl.dll"
  SetOutPath "$INSTDIR"
  File "/oname=Modo compatible.bat" "${COMPAT_BAT}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\$(TXT_SWGL_SH).lnk" "$INSTDIR\Modo compatible.bat" "" "$INSTDIR\${APP_ICO}"
SectionEnd
!endif

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_APP} $(TXT_SEC_APP_D)
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_DESKTOP} $(TXT_SEC_DSK_D)
!ifdef WITH_MESA
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_SWGL} $(TXT_SEC_SWGL_D)
!endif
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; --- desinstalación (NO toca la configuración del usuario) ------------------
Section "Uninstall"
  SetRegView 64
  nsExec::ExecToLog 'taskkill /IM "${APP_EXE}" /F'
  Pop $0

  Delete "$INSTDIR\swgl\${APP_EXE}"
  Delete "$INSTDIR\swgl\WinDivert.dll"
  Delete "$INSTDIR\swgl\WinDivert64.sys"
  Delete "$INSTDIR\swgl\opengl32.dll"
  Delete "$INSTDIR\swgl\libgallium_wgl.dll"
  RMDir "$INSTDIR\swgl"

  Delete "$INSTDIR\Modo compatible.bat"
  Delete "$INSTDIR\LEEME.txt"
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\netflow-probe.exe"
  Delete "$INSTDIR\WinDivert.dll"
  Delete "$INSTDIR\WinDivert64.sys"
  Delete "$INSTDIR\${APP_ICO}"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Desinstalar Pandora FMS Netflow Probe.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall Pandora FMS Netflow Probe.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Pandora FMS Netflow Probe (modo compatible).lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Pandora FMS Netflow Probe (compatible mode).lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"
  Delete "$DESKTOP\${APP_NAME}.lnk"

  DeleteRegKey HKLM "${UNINST_KEY}"
  DeleteRegKey HKLM "${APP_KEY}"

  IfSilent +2 0
  MessageBox MB_OK|MB_ICONINFORMATION "$(TXT_DATA_KEPT)"
SectionEnd
