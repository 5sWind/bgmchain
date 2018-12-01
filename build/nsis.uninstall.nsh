Section "Uninstall"
  # uninstall for all users
  setShellVarContext all

  # Delete (optionally) installed files
  {{range $}}Delete $INSTDIR\{{.}}
  {{end}}
  Delete $INSTDIR\uninstall.exe

  # Delete install directory
  rmDir $INSTDIR

  # Delete start menu launcher
  Delete "$SMPROGRAMS\${APPNAME}\${APPNAME}.lnk"
  Delete "$SMPROGRAMS\${APPNAME}\Attach.lnk"
  Delete "$SMPROGRAMS\${APPNAME}\Uninstall.lnk"
  rmDir "$SMPROGRAMS\${APPNAME}"

  # Firewall - remove rules if exists
  SimpleFC::AdvRemoveRule "Gbgm incoming peers (TCP:17575)"
  SimpleFC::AdvRemoveRule "Gbgm outgoing peers (TCP:17575)"
  SimpleFC::AdvRemoveRule "Gbgm UDP discovery (UDP:17575)"

  # Remove IPC endpoint (https://github.com/bgmchain/EIPs/issues/147)
  ${un.EnvVarUpdate} $0 "BGMCHAIN_SOCKET" "R" "HKLM" "\\.\pipe\gbgm.ipc"

  # Remove install directory from PATH
  Push "$INSTDIR"
  Call un.RemoveFromPath

  # Cleanup registry (deletes all sub keys)
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${GROUPNAME} ${APPNAME}"
SectionEnd
