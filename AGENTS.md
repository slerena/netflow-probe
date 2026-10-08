# AGENTS.md — instrucciones para agentes

## Qué es este repositorio

`netflow-probe` es un agente Go ligero y multiplataforma que captura el tráfico
del host y exporta **NetFlow v9** al colector de **Pandora FMS** (`nfcapd`,
UDP/9995). Target principal: **Windows**. Ver [README.md](README.md) y
[docs/decisiones.md](docs/decisiones.md).

## Reglas

- **Puro Go, `CGO_ENABLED=0`.** No introduzcas dependencias que exijan cgo,
  libpcap-dev ni SDKs C: rompería la cross-compilación a Windows.
- Todo backend de captura nuevo va en `internal/capture/capture_<os>.go` con su
  `//go:build <os>` y debe implementar `capture.Source`.
- Antes de dar por buena una entrega, compila y prueba **las tres plataformas**:
  ```sh
  go vet ./... && go test ./...
  CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
  CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build ./...
  CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build ./...
  ```
- No cambies el formato de export sin actualizar `docs/decisiones.md` (D4) y los
  tests de `internal/export`.
- **Excepción cgo**: solo `cmd/netflow-probe-gui` (la GUI de macOS) usa cgo, y va
  tras `//go:build darwin && cgo` con un *stub* para el resto. El core
  (`internal/*`, `cmd/netflow-probe`) debe seguir siendo puro Go. La GUI se
  empaqueta con `make macapp`; ver `docs/gui-macos.md`.

## Convenciones de Git

- **Conventional Commits**: `type(scope): subject` (`feat`, `fix`, `docs`,
  `chore`, `refactor`, `test`, `build`, `ci`, `perf`).
- **Nunca** añadas trailers de atribución a IA ni `Co-Authored-By`.
- Remoto: `git@brutus.artica.es:slerena/netflow-probe.git` (GitLab interno).

## Higiene de secretos

- El token de GitLab vive en `~/.config/andelain/brutus.json` (0600). **No**
  escribas tokens ni credenciales en el repositorio.
