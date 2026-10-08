// Package config holds the agent configuration and its on-disk (YAML) form.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that (un)marshals as a human string such as
// "30m" or "15s", so the config file stays editable by hand.
type Duration time.Duration

// Duration returns the value as a standard time.Duration.
func (d Duration) Duration() time.Duration { return time.Duration(d) }

// MarshalYAML implements yaml.Marshaler.
func (d Duration) MarshalYAML() (interface{}, error) { return time.Duration(d).String(), nil }

// UnmarshalYAML implements yaml.Unmarshaler.
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Config is the agent configuration. It mirrors the settings a non-technical
// user would fill in through the (future) tray UI.
type Config struct {
	// Interface is the capture interface/device. On Windows it is ignored
	// (WinDivert captures the whole host); on Linux/macOS it is required.
	Interface string `yaml:"interface"`
	// Collector is the NetFlow collector host. For Pandora FMS this is the
	// host running nfcapd, usually the Pandora server itself.
	Collector string `yaml:"collector"`
	// Port is the collector UDP port. Pandora FMS default: 9995.
	Port int `yaml:"port"`
	// ActiveTimeout flushes a flow after this long even if still active.
	ActiveTimeout Duration `yaml:"active_timeout"`
	// InactiveTimeout flushes a flow after this long without packets.
	InactiveTimeout Duration `yaml:"inactive_timeout"`
	// MaxFlows caps the flow table size to bound memory usage.
	MaxFlows int `yaml:"max_flows"`
	// SourceID is the NetFlow v9 observation domain / source id.
	SourceID uint32 `yaml:"source_id"`
	// TemplateRefresh is how many export packets may pass between template
	// refreshes (NetFlow v9 templates must be re-sent periodically).
	TemplateRefresh int `yaml:"template_refresh"`
	// Filter is a capture filter. On Windows it is passed to WinDivert
	// ("ip or ipv6", "tcp or udp", ...). On Linux/macOS it is currently
	// ignored (capture-all + user-space accounting) — see docs/arquitectura.md.
	Filter string `yaml:"filter"`
	// LogLevel is "info" or "debug".
	LogLevel string `yaml:"log_level"`
}

// Default returns a Config with sensible defaults (Pandora FMS oriented).
func Default() Config {
	return Config{
		Port:            9995,
		ActiveTimeout:   Duration(30 * time.Minute),
		InactiveTimeout: Duration(15 * time.Second),
		MaxFlows:        65536,
		SourceID:        1,
		TemplateRefresh: 20,
		LogLevel:        "info",
	}
}

// Load reads and parses a YAML config file, starting from the defaults.
func Load(path string) (Config, error) {
	c := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse %s: %w", path, err)
	}
	return c, c.Validate()
}

// Save writes the config as YAML.
func (c Config) Save(path string) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Validate checks the invariants that must hold for the agent to run.
func (c Config) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	if c.ActiveTimeout <= 0 || c.InactiveTimeout <= 0 {
		return fmt.Errorf("timeouts must be positive")
	}
	if c.MaxFlows <= 0 {
		return fmt.Errorf("max_flows must be positive")
	}
	if c.TemplateRefresh <= 0 {
		return fmt.Errorf("template_refresh must be positive")
	}
	return nil
}

// CollectorAddr returns the host:port address of the NetFlow collector.
func (c Config) CollectorAddr() string {
	return net.JoinHostPort(c.Collector, strconv.Itoa(c.Port))
}
