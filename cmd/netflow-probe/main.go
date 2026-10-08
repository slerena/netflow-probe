// Command netflow-probe is a lightweight, cross-platform agent that captures
// the local host's traffic and exports NetFlow v9 to an external collector.
//
// Its primary target is the Pandora FMS NetFlow collector (nfcapd on
// UDP/9995), especially on Windows.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"brutus.artica.es/slerena/netflow-probe/internal/capture"
	"brutus.artica.es/slerena/netflow-probe/internal/config"
	"brutus.artica.es/slerena/netflow-probe/internal/export"
	"brutus.artica.es/slerena/netflow-probe/internal/flow"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

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
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("netflow-probe %s\n", version)
		return
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		logger.Fatalf("config: %v", err)
	}
	if *iface != "" {
		cfg.Interface = *iface
	}
	if *collector != "" {
		cfg.Collector = *collector
	}
	if *port != 0 {
		cfg.Port = *port
	}
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("config: %v", err)
	}
	if !*dryRun && cfg.Collector == "" {
		logger.Fatalf("collector is required (use -collector or -config)")
	}

	src, err := openSource(*file, cfg)
	if err != nil {
		logger.Fatalf("capture: %v", err)
	}
	defer func() { _ = src.Close() }()

	table := flow.NewTable(cfg.ActiveTimeout.Duration(), cfg.InactiveTimeout.Duration(), cfg.MaxFlows)

	var exp *export.Exporter
	if !*dryRun {
		exp, err = export.New(cfg.CollectorAddr(), cfg.SourceID, cfg.TemplateRefresh)
		if err != nil {
			logger.Fatalf("exporter: %v", err)
		}
		defer func() { _ = exp.Close() }()
		logger.Printf("exporting NetFlow v9 to %s (source_id=%d)", cfg.CollectorAddr(), cfg.SourceID)
	} else {
		logger.Printf("dry-run: decoding and aggregating only")
	}

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
			if !ok {
				continue
			}
			table.Observe(key, l3, flags, time.Now())
		}
	}()

	exportRecords := func(recs []flow.Record) {
		if len(recs) == 0 {
			return
		}
		if *dryRun {
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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(*flushInt)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			exportRecords(table.Expire())
		case err := <-readErr:
			exportRecords(table.Flush())
			if *file != "" {
				logger.Printf("capture ended: %v", err)
				return
			}
			logger.Fatalf("capture: %v", err)
		case <-stop:
			exportRecords(table.Flush())
			logger.Printf("stopping")
			return
		}
	}
}

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
