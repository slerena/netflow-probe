Name:           netflow-probe
Version:        __VERSION__
Release:        1%{?dist}
Summary:        Pandora FMS NetFlow Probe
License:        GPL-2.0-only
URL:            https://github.com/slerena/netflow-probe
Source0:        netflow-probe-gui
Source1:        netflow-probe
Source2:        netflow-probe.desktop
Source3:        netflow-probe.png
Source4:        LICENSE
Source5:        netflow-probe.yaml
BuildArch:      x86_64
Requires:       glibc
Requires:       libglvnd-glx
Requires:       libX11
Requires:       libwayland-client
Requires:       libcap

%description
Ligero agente que captura el trafico del propio equipo y lo exporta como
NetFlow v9 al colector de Pandora FMS (nfcapd, UDP/9995).

Incluye GUI (Fyne), CLI/servicio, entrada de menu e icono. El post-install
aplica CAP_NET_RAW para capturar sin ejecutar como root.

%install
mkdir -p %{buildroot}%{_bindir}
mkdir -p %{buildroot}%{_datadir}/applications
mkdir -p %{buildroot}%{_datadir}/icons/hicolor/512x512/apps
mkdir -p %{buildroot}%{_datadir}/doc/netflow-probe
mkdir -p %{buildroot}%{_datadir}/netflow-probe
install -m755 %{SOURCE0} %{buildroot}%{_bindir}/netflow-probe-gui
install -m755 %{SOURCE1} %{buildroot}%{_bindir}/netflow-probe
install -m644 %{SOURCE2} %{buildroot}%{_datadir}/applications/netflow-probe.desktop
install -m644 %{SOURCE3} %{buildroot}%{_datadir}/icons/hicolor/512x512/apps/netflow-probe.png
install -m644 %{SOURCE4} %{buildroot}%{_datadir}/doc/netflow-probe/copyright
install -m644 %{SOURCE5} %{buildroot}%{_datadir}/netflow-probe/netflow-probe.yaml.example

%post
if command -v setcap >/dev/null 2>&1; then
  setcap cap_net_raw+ep %{_bindir}/netflow-probe-gui || :
  setcap cap_net_raw+ep %{_bindir}/netflow-probe || :
fi
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q %{_datadir}/applications || :
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t -f %{_datadir}/icons/hicolor || :
fi

%postun
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q %{_datadir}/applications || :
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t -f %{_datadir}/icons/hicolor || :
fi

%files
%{_bindir}/netflow-probe-gui
%{_bindir}/netflow-probe
%{_datadir}/applications/netflow-probe.desktop
%{_datadir}/icons/hicolor/512x512/apps/netflow-probe.png
%{_datadir}/doc/netflow-probe/copyright
%{_datadir}/netflow-probe/netflow-probe.yaml.example

%changelog
* Sun Oct 18 2026 Pandora FMS <info@pandorafms.com> - __VERSION__-1
- Release __VERSION__
