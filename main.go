package main

import (
	"fmt"
	"time"

	"github.com/sensu/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/core/v2"
	"github.com/shirou/gopsutil/v3/cpu"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	Critical float64
	Warning  float64
	Interval int
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-cpu-usage",
			Short:    "Check CPU usage and provide metrics",
			Keyspace: "sensu.io/plugins/check-cpu-usage/config",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[float64]{
			Path:      "critical",
			Argument:  "critical",
			Shorthand: "c",
			Default:   float64(90),
			Usage:     "Critical threshold for overall CPU usage",
			Value:     &plugin.Critical,
		},
		&sensu.PluginConfigOption[float64]{
			Path:      "warning",
			Argument:  "warning",
			Shorthand: "w",
			Default:   float64(75),
			Usage:     "Warning threshold for overall CPU usage",
			Value:     &plugin.Warning,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "sample-interval",
			Argument:  "sample-interval",
			Shorthand: "s",
			Default:   2,
			Usage:     "Length of sample interval in seconds",
			Value:     &plugin.Interval,
		},
	}
)

func main() {
	check := sensu.NewCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	if plugin.Critical == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--critical is required")
	}
	if plugin.Warning == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--warning is required")
	}
	if plugin.Warning > plugin.Critical {
		return sensu.CheckStateWarning, fmt.Errorf("--warning cannot be greater than --critical")
	}
	if plugin.Interval == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--interval is required")
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	start, err := cpu.Times(false)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Error obtaining CPU timings: %v", err)
	}

	startTotal := start[0].User + start[0].System + start[0].Idle + start[0].Nice + start[0].Iowait + start[0].Irq + start[0].Softirq + start[0].Steal + start[0].Guest + start[0].GuestNice

	duration, err := time.ParseDuration(fmt.Sprintf("%ds", plugin.Interval))
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Error parsing duration: %v", err)
	}

	time.Sleep(duration)

	end, err := cpu.Times(false)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("Error obtaining CPU timings: %v", err)
	}

	endTotal := end[0].User + end[0].System + end[0].Idle + end[0].Nice + end[0].Iowait + end[0].Irq + end[0].Softirq + end[0].Steal + end[0].Guest + end[0].GuestNice

	diff := endTotal - startTotal
	idlePct := ((end[0].Idle - start[0].Idle) / diff) * 100
	usedPct := 100 - idlePct

	userPct := ((end[0].User - start[0].User) / diff) * 100
	sysPct := ((end[0].System - start[0].System) / diff) * 100
	nicePct := ((end[0].Nice - start[0].Nice) / diff) * 100
	iowaitPct := ((end[0].Iowait - start[0].Iowait) / diff) * 100
	irqPct := ((end[0].Irq - start[0].Irq) / diff) * 100
	softirqPct := ((end[0].Softirq - start[0].Softirq) / diff) * 100
	stealPct := ((end[0].Steal - start[0].Steal) / diff) * 100
	guestPct := ((end[0].Guest - start[0].Guest) / diff) * 100
	guestnicePct := ((end[0].GuestNice - start[0].GuestNice) / diff) * 100
	perfData := fmt.Sprintf("cpu_idle=%.2f, cpu_system=%.2f, cpu_user=%.2f, cpu_nice=%.2f, cpu_iowait=%.2f, cpu_irq=%.2f, cpu_softirq=%.2f, cpu_steal=%.2f, cpu_guest=%.2f, cpu_guestnice=%.2f", idlePct, sysPct, userPct, nicePct, iowaitPct, irqPct, softirqPct, stealPct, guestPct, guestnicePct)
	if usedPct > plugin.Critical {
		fmt.Printf("%s Critical: %.2f%% CPU usage | %s\n", plugin.PluginConfig.Name, usedPct, perfData)

		return sensu.CheckStateCritical, nil
	} else if usedPct > plugin.Warning {
		fmt.Printf("%s Warning: %.2f%% CPU usage | %s\n", plugin.PluginConfig.Name, usedPct, perfData)
		return sensu.CheckStateWarning, nil
	}

	fmt.Printf("%s OK: %.2f%% CPU usage | %s\n", plugin.PluginConfig.Name, usedPct, perfData)
	return sensu.CheckStateOK, nil
}
