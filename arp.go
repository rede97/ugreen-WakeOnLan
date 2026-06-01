package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

type ArpEntry struct {
	IP  string `json:"ip"`
	MAC string `json:"mac"`
	Dev string `json:"dev,omitempty"`
}

func cmdArp() {
	entries, err := readArpTable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Println("No ARP entries found.")
		return
	}

	mu.Lock()
	macToName := make(map[string]string)
	for _, d := range devices {
		macToName[strings.ToUpper(d.MAC)] = d.Name
	}
	mu.Unlock()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IP\tMAC\tIFACE\tDEVICE")
	for _, e := range entries {
		name := macToName[strings.ToUpper(e.MAC)]
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.IP, e.MAC, e.Dev, name)
	}
	w.Flush()
}

func cmdScan() {
	entries, err := readArpTable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		fmt.Println("No ARP entries found.")
		return
	}

	mu.Lock()
	macToName := make(map[string]string)
	for _, d := range devices {
		macToName[strings.ToUpper(d.MAC)] = d.Name
	}
	mu.Unlock()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IP\tMAC\tIFACE\tDEVICE\tLATENCY")
	for _, e := range entries {
		name := macToName[strings.ToUpper(e.MAC)]
		if name == "" {
			name = "-"
		}
		latency := "-"
		if dur, err := pingICMP(e.IP, 2*time.Second); err == nil {
			ms := float64(dur.Microseconds()) / 1000.0
			latency = fmt.Sprintf("%.2f ms", ms)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.IP, e.MAC, e.Dev, name, latency)
	}
	w.Flush()
}

func handleArp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	entries, err := readArpTable()
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  err.Error(),
			"arp_ok": false,
		})
		return
	}

	mu.Lock()
	macToName := make(map[string]string)
	for _, d := range devices {
		macToName[strings.ToUpper(d.MAC)] = d.Name
	}
	mu.Unlock()

	type out struct {
		IP    string `json:"ip"`
		MAC   string `json:"mac"`
		Iface string `json:"iface,omitempty"`
		Name  string `json:"name,omitempty"`
	}

	var list []out
	for _, e := range entries {
		name := macToName[strings.ToUpper(e.MAC)]
		list = append(list, out{IP: e.IP, MAC: e.MAC, Iface: e.Dev, Name: name})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": list,
		"ping_ok": pingCapable,
		"arp_ok":  true,
	})
}

func readArpTable() ([]ArpEntry, error) {
	data, err := os.ReadFile("/proc/net/arp")
	if err != nil {
		return nil, fmt.Errorf("cannot read /proc/net/arp: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, nil
	}

	var entries []ArpEntry
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		ip := fields[0]
		hwType := fields[1]
		flags := fields[2]
		mac := fields[3]

		if hwType != "0x1" || flags != "0x2" {
			continue
		}

		iface := ""
		if len(fields) >= 6 {
			iface = fields[5]
		}
		entries = append(entries, ArpEntry{IP: ip, MAC: mac, Dev: iface})
	}
	return entries, nil
}
