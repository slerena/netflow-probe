// Command netflow-probe is a lightweight, cross-platform agent that captures
// the local host's traffic and exports NetFlow v9 to an external collector.
//
// Its primary target is the Pandora FMS NetFlow collector (nfcapd on
// UDP/9995), especially on Windows, where it can also run as a system service.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kardianos/service"

	"brutus.artica.es/slerena/netflow-probe/internal/capture"
	"brutus.artica.es/slerena/netflow-probe/internal/config"
	"brutus.artica.es/slerena/netflow-probe/internal/export"
	"brutus.artica.es/slerena/netflow-probe/internal/flow"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

// serviceName is the OS service identifier (Windows SCM, launchd, systemd).
const serviceName = "PandoraFMSNetflowProbe"

// systemdUnit is a custom systemd unit that grants CAP_NET_RAW to the service,
// so it can capture (AF_PACKET) without running as root.
const systemdUnit = `[Unit]
Description={{Description}}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart={{Path | cmdEscape}}{{range Arguments}} {{. | cmd}}{{end}}
Restart=always
RestartSec=5
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
`

func main() {
	var (
		cfgPath   = flag.String("config", "", "path to a YAML config file")
		iface     = flag.String("iface", "", "capture interface (overrides config; ignored on Windows)")
		collector = flag.String("collector", "", "NetFlow collector host (overrides config)")
		port      = flag.Int("port", 0, "NetFlow collector UDP port (overrides config; Pandora default 9995)")
		file      = flag.String("read", "", "read packets from a pcap/pcapng file instead of a live interface")
		dryRun    = flag.Bool("dry-run", false, "decode and aggregate but do not send NetFlow")
		flushInt  = flag.Duration("flush-interval", 5*time.Second, "how often to expire and export flows")
		showVer   = flag.Bool("version", false, "print version and exit")
		install   = flag.Bool("install", false, "install as a system service (needs -config)")
		uninstall = flag.Bool("uninstall", false, "uninstall the system service")
		runSvc    = flag.Bool("service", false, "run as a system service (used by the OS)")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("netflow-probe %s\n", version)
		return
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	if *install || *uninstall || *runSvc {
		if err := runService(*install, *uninstall, *cfgPath); err != nil {
			logger.Fatalf("service: %v", err)
		}
		return
	}

	cfg, err := buildConfig(*cfgPath, *iface, *collector, *port, *dryRun)
	if err != nil {
		logger.Fatalf("config: %v", err)
	}

	src, err := openSource(*file, cfg)
	if err != nil {
		logger.Fatalf("capture: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() { <-stop; cancel() }()

	err = runAgent(ctx, cfg, src, *dryRun, *flushInt, logger)
	if err != nil {
		if *file != "" {
			logger.Printf("capture ended: %v", err)
			return
		}
		logger.Fatalf("capture: %v", err)
	}
	logger.Printf("stopping")
}

// buildConfig loads the config file (or defaults) and applies CLI overrides.
func buildConfig(path, iface, collector string, port int, dryRun bool) (config.Config, error) {
	cfg, err := loadConfig(path)
	if err != nil {
		return cfg, err
	}
	if iface != "" {
		cfg.Interface = iface
	}
	if collector != "" {
		cfg.Collector = collector
	}
	if port != 0 {
		cfg.Port = port
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	if !dryRun && cfg.Collector == "" {
		return cfg, fmt.Errorf("collector is required (use -collector or -config)")
	}
	return cfg, nil
}

// runAgent captures, aggregates and exports until ctx is cancelled or the
// source ends. It is shared by the CLI and the service.
func runAgent(ctx context.Context, cfg config.Config, src capture.Source, dryRun bool, flushInt time.Duration, logger *log.Logger) error {
	defer func() { _ = src.Close() }()

	var exp *export.Exporter
	if !dryRun {
		var err error
		exp, err = export.New(cfg.CollectorAddr(), cfg.SourceID, cfg.TemplateRefresh)
		if err != nil {
			return fmt.Errorf("exporter: %w", err)
		}
		defer func() { _ = exp.Close() }()
		logger.Printf("exporting NetFlow v9 to %s (source_id=%d)", cfg.CollectorAddr(), cfg.SourceID)
	} else {
		logger.Printf("dry-run: decoding and aggregating only")
	}

	table := flow.NewTable(cfg.ActiveTimeout.Duration(), cfg.InactiveTimeout.Duration(), cfg.MaxFlows)
	link := src.LinkType()
	logger.Printf("capturing, link type %v", link)

	readErr := make(chan error, 1)
	go func() {
		for {
			data, _, err := src.Next()
			if err != nil {
				readErr <- err
				return
			}
			key, l3, flags, ok := flow.Observe(data, link)
			if ok {
				table.Observe(key, l3, flags, time.Now())
			}
		}
	}()

	exportRecords := func(recs []flow.Record) {
		if len(recs) == 0 {
			return
		}
		if dryRun {
			for _, r := range recs {
				logger.Printf("flow %s:%d -> %s:%d proto=%d tos=%d bytes=%d pkts=%d",
					r.Key.Src, r.Key.SrcPort, r.Key.Dst, r.Key.DstPort,
					r.Key.Proto, r.Key.TOS, r.Bytes, r.Packets)
			}
			return
		}
		if _, err := exp.Send(recs, time.Now()); err != nil {
			logger.Printf("export: %v", err)
		}
	}

	ticker := time.NewTicker(flushInt)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			exportRecords(table.Flush())
			return nil
		case <-ticker.C:
			exportRecords(table.Expire())
		case err := <-readErr:
			exportRecords(table.Flush())
			return err
		}
	}
}

// --- service support ---------------------------------------------------------

// agentProgram adapts the agent to the kardianos/service interface.
type agentProgram struct {
	cfgPath string
	cancel  context.CancelFunc
	done    chan struct{}
}

func (p *agentProgram) Start(s service.Service) error {
	cfg, err := loadConfig(p.cfgPath)
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	if cfg.Collector == "" {
		return fmt.Errorf("collector is required in the service config")
	}
	src, err := capture.OpenLive(cfg.Interface, cfg.Filter)
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.done = make(chan struct{})
	logger := log.New(os.Stderr, "", log.LstdFlags)
	go func() {
		defer close(p.done)
		if err := runAgent(ctx, cfg, src, false, 5*time.Second, logger); err != nil {
			logger.Printf("agent stopped: %v", err)
		}
	}()
	return nil
}

func (p *agentProgram) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.done != nil {
		<-p.done
	}
	return nil
}

func runService(install, uninstall bool, cfgPath string) error {
	args := []string{"-service"}
	if cfgPath != "" {
		args = append(args, "-config", cfgPath)
	}
	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: "Pandora FMS Netflow Probe",
		Description: "Captura el tráfico del host y lo exporta como NetFlow v9 al colector de Pandora FMS.",
		Arguments:   args,
		Option:      service.KeyValue{"SystemdScript": systemdUnit},
	}
	s, err := service.New(&agentProgram{cfgPath: cfgPath}, svcConfig)
	if err != nil {
		return err
	}
	switch {
	case install:
		if cfgPath == "" {
			return fmt.Errorf("indica -config <ruta> al instalar el servicio")
		}
		return s.Install()
	case uninstall:
		return s.Uninstall()
	default:
		return s.Run()
	}
}

// --- helpers -----------------------------------------------------------------

// loadConfig returns defaults when no path is given.
func loadConfig(path string) (config.Config, error) {
	if path == "" {
		return config.Default(), nil
	}
	return config.Load(path)
}

// openSource chooses between pcap file replay and live capture.
func openSource(file string, cfg config.Config) (capture.Source, error) {
	if file != "" {
		return capture.OpenFile(file)
	}
	return capture.OpenLive(cfg.Interface, cfg.Filter)
}
