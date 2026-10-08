# Changelog

All notable changes to this project are documented here.
This project follows [Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added
- Configuration model (YAML) with Pandora FMS defaults (`collector` + `port: 9995`).
- Packet decoding into unidirectional 5-tuple flow keys (IPv4/IPv6, TCP/UDP).
- Flow aggregation table with active/inactive timeouts and LRU overflow eviction.
- NetFlow v9 exporter: IPv4 and IPv6 templates, template refresh, UDP transport.
- Capture backends, all pure Go (`CGO_ENABLED=0`):
  - Linux: AF_PACKET (`gopacket/afpacket`).
  - macOS: `/dev/bpf` (`gopacket/bsdbpf`).
  - Windows: WinDivert (`imgk/divert-go`).
  - All platforms: pcap/pcapng replay (for tests and dry runs).
- CLI: `-read`, `-dry-run`, `-collector`, `-port`, `-iface`, `-flush-interval`, `-version`.
- Unit tests for decoding, aggregation and NetFlow v9 serialisation.
- Cross-platform Makefile (`make dist`) and GitLab CI.
- Documentation: decisions, architecture, research notes, Windows install.
- **macOS GUI (PoC)**: Fyne desktop app (`make macapp`) with a Pandora icon,
  Start/Stop, status/log, and a macOS capture-permission flow (opens System
  Settings and can grant `/dev/bpf` access with an admin prompt).
- GUI real-time feedback: live counters (packets, bytes, active flows,
  exported flows) refreshed every second, plus the green `kittBar` spinner
  (ported from pandora-mini) shown while capturing.
- GUI menu-bar (tray) icon: grey Pandora glyph with Show/Hide, Start, Stop and
  Quit; closing the window hides it to the tray.
- Capture permission on macOS: installs a persistent ChmodBPF `LaunchDaemon`
  (`/Library/LaunchDaemons/es.artica.netflowprobe.chmodbpf.plist`) and retries
  automatically. Removed the misleading link to System Settings — macOS has no
  Privacy toggle for packet capture.

### Fixed
- macOS capture is now implemented directly on `/dev/bpf*` instead of
  `gopacket/bsdbpf`, which had three defects: it mutated a package-level default
  (a second open used `fd=0` → `operation not supported by device`), it did not
  retry reads interrupted by signals (`interrupted system call`), and it aligned
  BPF records to 8 bytes while the kernel uses 4, corrupting multi-record
  buffers (`corrupted BpfHdr struct`). The new reader retries on `EINTR`, uses
  the correct 18-byte header and 4-byte alignment, and resynchronises on
  malformed records.

### Known gaps (roadmap)
- Tray UI for non-technical users (Wails or Fyne).
- Windows service + installer (elevation, auto-start, bundled WinDivert driver).
- Kernel BPF filters on Linux/macOS (currently capture-all + user-space accounting).
- sFlow export and IPFIX (not needed for the Pandora nfcapd target).
