# Changelog

All notable changes to this project are documented here.
This project follows [Conventional Commits](https://www.conventionalcommits.org/).

## [1.0.33] - 2026-10-18

### Changed
- Windows installer: the Mesa `swgl/` fallback is installed **silently** as part
  of the app. Removed the "compatible mode" component, its Start Menu shortcut
  and `Modo compatible.bat` from the installer and portable ZIP, and removed all
  OpenGL/Mesa references from the user-facing LEEME. It is an internal detail:
  the normal executable relaunches itself when needed.

## [1.0.32] - 2026-10-18

### Added
- Windows **portable ZIP** and **NSIS installer** (`make windows-zip`,
  `make windows-installer`) built from macOS/Linux with `makensis`, bundling
  WinDivert and a Mesa `swgl/` compatible mode.
- Windows **service**: `netflow-probe.exe -install -config …` /
  `-uninstall`, via `github.com/kardianos/service`.
- Windows **automatic Mesa fallback**: if the machine cannot provide OpenGL 2.1
  and `swgl/` exists, the app relaunches itself with llvmpipe; otherwise it
  shows a native warning instead of failing silently.
- Windows child processes (`powershell`, etc.) are launched with
  `CREATE_NO_WINDOW` + `HideWindow`: no console flashing.
- Mesa pinned by SHA-256 (`packaging/windows/fetch_mesa.sh`).
- Linux: GUI builds with Fyne/cgo, a `CAP_NET_RAW` permission flow (`setcap`)
  and a systemd unit with `AmbientCapabilities=CAP_NET_RAW`. Prepared, not
  deployed.
- macOS DMG installer (`make dmg`) with background and GPL-2.0 notice; `make
  macapp` also produces a ZIP.
- Version is shown in the window title; `VERSION` + `scripts/bump_version.py`
  keep every place in sync.

### Fixed
- GUI **Detener** button was a no-op (it shared the start handler); now it is a
  real toggle and the capture goroutine is unblocked on close.
- macOS capture now uses a direct `/dev/bpf*` reader instead of
  `gopacket/bsdbpf` (EINTR retry, 18-byte header, 4-byte alignment).
- Windows installer detects a running app (`tasklist` truncates image names).
- DMG license text no longer has hard line breaks.

## [1.0.0] - 2026-10-17

### Added
- Configuration model (YAML) with Pandora FMS defaults (`collector` + `port: 9995`).
- Packet decoding into unidirectional 5-tuple flow keys (IPv4/IPv6, TCP/UDP).
- Flow aggregation table with active/inactive timeouts and LRU overflow eviction.
- NetFlow v9 exporter: IPv4 and IPv6 templates, template refresh, UDP transport.
- Capture backends, all pure Go (`CGO_ENABLED=0`):
  - Linux: AF_PACKET (`gopacket/afpacket`).
  - macOS: `/dev/bpf` (direct BPF reader).
  - Windows: WinDivert (`imgk/divert-go`).
  - All platforms: pcap/pcapng replay (for tests and dry runs).
- CLI: `-read`, `-dry-run`, `-collector`, `-port`, `-iface`, `-flush-interval`,
  `-version`, `-install`/`-uninstall`/`-service`.
- Unit tests for decoding, aggregation and NetFlow v9 serialisation.
- Cross-platform Makefile and GitLab CI.
- Documentation: decisions, architecture, research notes, Windows/Linux install.
- **macOS GUI (PoC)**: Fyne desktop app with a Pandora icon, Start/Stop,
  status/log and the capture-permission flow (persistent `/dev/bpf` LaunchDaemon).
- GUI real-time feedback: live counters (packets, bytes, active flows, exported
  flows) plus the green `kittBar` spinner.
- GUI menu-bar (tray) icon: grey Pandora glyph with Show/Hide, Start, Stop, Quit.

### Known gaps (roadmap)
- Kernel BPF filters on Linux/macOS (currently capture-all + user-space accounting).
- sFlow export and IPFIX (not needed for the Pandora nfcapd target).
- Python/app bundling for Linux (.deb/.rpm/AppImage).
