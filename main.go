package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"text/tabwriter"
)

var configFile = func() string {
	if dir := os.Getenv("UGAPP_DATA_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir + "/devices.json"
		}
	}
	return "devices.json"
}()

type Device struct {
	Name      string `json:"name"`
	MAC       string `json:"mac"`
	Interface string `json:"interface"`
}

type NetIfaceInfo struct {
	Name string   `json:"name"`
	IPs  []string `json:"ips"`
}

var (
	devices []Device
	mu      sync.Mutex
)

func main() {
	loadDevices()

	if len(os.Args) < 2 {
		runServer(nil)
		return
	}

	switch os.Args[1] {
	case "interfaces", "ifaces":
		cmdInterfaces()
	case "list", "ls":
		cmdList()
	case "add":
		cmdAdd(os.Args[2:])
	case "delete", "rm":
		cmdDelete(os.Args[2:])
	case "wake":
		cmdWake(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		runServer(os.Args[1:])
	}
}

func printUsage() {
	fmt.Print(`Usage: wakeonlan_serv [command|flags]

Commands:
  interfaces, ifaces   List available network interfaces
  list, ls             List configured devices
  add                  Add a new device
  delete, rm           Delete a device
  wake                 Send a Wake-on-LAN magic packet
  help                 Show this help

Flags for add:
  -name     Device hostname
  -mac      MAC address (AA:BB:CC:DD:EE:FF)
  -iface    Network interface name

Flags for delete:
  -name     Device hostname
  -mac      MAC address
  -iface    Network interface name

Flags for wake:
  -name     Device name (auto-fill MAC and interface from config)
  -mac      Target MAC address (manual mode)
  -iface    Network interface (manual mode)

Server flags:
  -port     HTTP server port (default: 21010)

Run without arguments or flags to start the HTTP server on port 21010.
`)
}

func getInterfaces() []NetIfaceInfo {
	ifaces, _ := net.Interfaces()
	var out []NetIfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		var ips []string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ips = append(ips, ipnet.IP.String())
			}
		}
		if len(ips) == 0 {
			continue
		}
		out = append(out, NetIfaceInfo{Name: iface.Name, IPs: ips})
	}
	return out
}

func cmdInterfaces() {
	ifaces := getInterfaces()
	if len(ifaces) == 0 {
		fmt.Println("No network interfaces found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tIP ADDRESSES")
	for _, i := range ifaces {
		fmt.Fprintf(w, "%s\t%s\n", i.Name, strings.Join(i.IPs, ", "))
	}
	w.Flush()
}

func cmdList() {
	if len(devices) == 0 {
		fmt.Printf("No devices configured. (%s is empty)\n", configFile)
		return
	}
	ifaces := getInterfaces()
	ifaceIPs := make(map[string]string)
	for _, i := range ifaces {
		ifaceIPs[i.Name] = strings.Join(i.IPs, ", ")
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "#\tNAME\tMAC\tINTERFACE")
	for i, d := range devices {
		ifaceLabel := d.Interface
		if ips := ifaceIPs[d.Interface]; ips != "" {
			ifaceLabel = fmt.Sprintf("%s (%s)", d.Interface, ips)
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", i+1, d.Name, d.MAC, ifaceLabel)
	}
	w.Flush()
	fmt.Printf("\n%d device(s) in %s\n", len(devices), configFile)
}

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	name := fs.String("name", "", "Device hostname")
	mac := fs.String("mac", "", "MAC address")
	iface := fs.String("iface", "", "Network interface")
	fs.Parse(args)

	if *name == "" || *mac == "" || *iface == "" {
		fmt.Println("Error: -name, -mac and -iface are required")
		fs.Usage()
		os.Exit(1)
	}

	d := Device{
		Name:      strings.TrimSpace(*name),
		MAC:       strings.TrimSpace(*mac),
		Interface: strings.TrimSpace(*iface),
	}
	if _, err := net.ParseMAC(d.MAC); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid MAC address: %s\n", d.MAC)
		os.Exit(1)
	}

	mu.Lock()
	devices = append(devices, d)
	saveDevices()
	mu.Unlock()

	fmt.Printf("Added: %s (%s) via %s\n", d.Name, d.MAC, d.Interface)
}

func cmdDelete(args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	name := fs.String("name", "", "Device hostname")
	mac := fs.String("mac", "", "MAC address")
	iface := fs.String("iface", "", "Network interface")
	fs.Parse(args)

	if *name == "" || *mac == "" || *iface == "" {
		fmt.Println("Error: -name, -mac and -iface are required")
		fs.Usage()
		os.Exit(1)
	}

	mu.Lock()
	defer mu.Unlock()
	for i, d := range devices {
		if d.Name == *name && d.MAC == *mac && d.Interface == *iface {
			devices = append(devices[:i], devices[i+1:]...)
			saveDevices()
			fmt.Printf("Deleted: %s (%s)\n", d.Name, d.MAC)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Error: device not found\n")
	os.Exit(1)
}

func cmdWake(args []string) {
	fs := flag.NewFlagSet("wake", flag.ExitOnError)
	name := fs.String("name", "", "Device name (auto-fill MAC and interface from config)")
	mac := fs.String("mac", "", "Target MAC address")
	iface := fs.String("iface", "", "Network interface")
	fs.Parse(args)

	if *name != "" {
		mu.Lock()
		for _, d := range devices {
			if d.Name == *name {
				*mac = d.MAC
				*iface = d.Interface
				break
			}
		}
		mu.Unlock()
		if *mac == "" {
			fmt.Fprintf(os.Stderr, "Error: device '%s' not found\n", *name)
			os.Exit(1)
		}
	}

	if *mac == "" || *iface == "" {
		fmt.Println("Error: use -name to wake a configured device, or -mac and -iface manually")
		fs.Usage()
		os.Exit(1)
	}

	if err := sendMagicPacket(*mac, *iface); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Magic packet sent to %s via %s\n", *mac, *iface)
}

// --- HTTP Server ---

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	port := fs.Int("port", 21010, "HTTP server port")
	fs.Parse(args)

	debug.SetMemoryLimit(32 << 20) // 32 MiB soft cap
	debug.SetGCPercent(50)         // trade CPU for lower memory

	cors := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(200)
				return
			}
			next(w, r)
		}
	}

	http.HandleFunc("/api/devices", cors(handleDevices))
	http.HandleFunc("/api/wake", cors(handleWake))
	http.HandleFunc("/api/interfaces", cors(handleInterfaces))

	webDir := "www"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		webDir = "rootfs_common/www"
	}
	http.Handle("/", http.FileServer(http.Dir(webDir)))

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("WakeOnLan server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// --- Persistence ---

func loadDevices() {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("WARN: cannot read %s: %v", configFile, err)
		}
		return
	}
	if err := json.Unmarshal(data, &devices); err != nil {
		log.Printf("WARN: cannot parse %s: %v", configFile, err)
	}
	log.Printf("Loaded %d device(s) from %s", len(devices), configFile)
}

func saveDevices() {
	data, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		log.Printf("WARN: marshal devices: %v", err)
		return
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		log.Printf("WARN: write %s: %v", configFile, err)
	}
}

// --- HTTP Handlers ---

func handleInterfaces(w http.ResponseWriter, r *http.Request) {
	out := getInterfaces()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"interfaces": out})
}

func handleDevices(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(map[string]interface{}{"devices": devices})

	case http.MethodPost:
		var d Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
			return
		}
		d.Name = strings.TrimSpace(d.Name)
		d.MAC = strings.TrimSpace(d.MAC)
		d.Interface = strings.TrimSpace(d.Interface)
		if d.Name == "" || d.MAC == "" || d.Interface == "" {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "name, mac and interface are required"})
			return
		}
		if _, err := net.ParseMAC(d.MAC); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid MAC address"})
			return
		}
		devices = append(devices, d)
		saveDevices()
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(d)

	case http.MethodDelete:
		var d Device
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
			return
		}
		for i, dev := range devices {
			if dev.Name == d.Name && dev.MAC == d.MAC && dev.Interface == d.Interface {
				devices = append(devices[:i], devices[i+1:]...)
				saveDevices()
				json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
				return
			}
		}
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(map[string]string{"error": "device not found"})

	default:
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}
}

func handleWake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		MAC       string `json:"mac"`
		Interface string `json:"interface"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if err := sendMagicPacket(req.MAC, req.Interface); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "mac": req.MAC})
}

// --- Magic Packet ---

func sendMagicPacket(macStr, ifaceName string) error {
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return err
	}

	packet := make([]byte, 102)
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	for i := 1; i <= 16; i++ {
		copy(packet[i*6:], mac)
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("interface not found: %s", ifaceName)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("cannot get addresses for %s: %v", ifaceName, err)
	}

	var localIP net.IP
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
			localIP = ipnet.IP.To4()
			break
		}
	}
	if localIP == nil {
		return fmt.Errorf("no IPv4 address on interface %s", ifaceName)
	}

	conn, err := net.DialUDP("udp", &net.UDPAddr{IP: localIP}, &net.UDPAddr{IP: net.IPv4bcast, Port: 9})
	if err != nil {
		return fmt.Errorf("cannot bind to %s: %v", ifaceName, err)
	}
	defer conn.Close()

	_, err = conn.Write(packet)
	return err
}
