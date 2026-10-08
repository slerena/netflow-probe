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

### Known gaps (roadmap)
- Tray UI for non-technical users (Wails or Fyne).
- Windows service + installer (elevation, auto-start, bundled WinDivert driver).
- Kernel BPF filters on Linux/macOS (currently capture-all + user-space accounting).
- sFlow export and IPFIX (not needed for the Pandora nfcapd target).
