package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(405)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	if req.IP == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "ip is required"})
		return
	}

	dur, err := pingICMP(req.IP, 2*time.Second)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ip":    req.IP,
			"alive": false,
			"error": err.Error(),
		})
		return
	}

	ms := float64(dur.Microseconds()) / 1000.0
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ip":      req.IP,
		"alive":   true,
		"latency": fmt.Sprintf("%.2f ms", ms),
	})
}

func pingICMP(ip string, timeout time.Duration) (time.Duration, error) {
	// 1. Linux DGRAM ICMP socket (no root on modern kernels)
	d, err := pingICMPDgram(ip, timeout)
	if err == nil {
		return d, nil
	}
	// 2. Raw ICMP socket (needs CAP_NET_RAW or root)
	d, err = pingICMPRaw(ip, timeout)
	if err == nil {
		return d, nil
	}
	// 3. Fallback: system ping command
	return pingCmd(ip, timeout)
}

func pingICMPDgram(ip string, timeout time.Duration) (time.Duration, error) {
	addr := net.ParseIP(ip)
	if addr.To4() == nil {
		return 0, fmt.Errorf("invalid IPv4: %s", ip)
	}

	// socket(AF_INET, SOCK_DGRAM, IPPROTO_ICMP) — Linux ping socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_ICMP)
	if err != nil {
		return 0, fmt.Errorf("dgram icmp socket: %v", err)
	}
	defer syscall.Close(fd)

	id := uint16(os.Getpid() & 0xFFFF)
	seq := uint16(1)
	pkt := buildICMPEchoRequest(id, seq)

	rawAddr := &syscall.SockaddrInet4{Port: 0}
	copy(rawAddr.Addr[:], addr.To4())

	start := time.Now()
	if err := syscall.Sendto(fd, pkt, 0, rawAddr); err != nil {
		return 0, fmt.Errorf("send icmp: %v", err)
	}

	if err := syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{
		Sec:  timeout.Nanoseconds() / 1e9,
		Usec: (timeout.Nanoseconds() % 1e9) / 1e3,
	}); err != nil {
		return 0, fmt.Errorf("set timeout: %v", err)
	}

	buf := make([]byte, 1500)
	for {
		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.ETIMEDOUT {
				return 0, fmt.Errorf("ping timeout")
			}
			return 0, fmt.Errorf("recv icmp: %v", err)
		}

		// ICMP echo reply: type=0, code=0
		if n >= 8 && buf[0] == 0 && buf[1] == 0 {
			// Match our id/seq
			if binary.BigEndian.Uint16(buf[4:]) == id && binary.BigEndian.Uint16(buf[6:]) == seq {
				return time.Since(start), nil
			}
		}

		if time.Since(start) > timeout {
			return 0, fmt.Errorf("ping timeout")
		}
	}
}

func pingICMPRaw(ip string, timeout time.Duration) (time.Duration, error) {
	addr := net.ParseIP(ip)
	if addr == nil {
		return 0, fmt.Errorf("invalid IP: %s", ip)
	}

	dst := &net.IPAddr{IP: addr}

	conn, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return 0, fmt.Errorf("listen icmp: %v (may need root)", err)
	}
	defer conn.Close()

	id := uint16(os.Getpid() & 0xFFFF)
	seq := uint16(1)
	pkt := buildICMPEchoRequest(id, seq)

	start := time.Now()
	if _, err := conn.WriteTo(pkt, dst); err != nil {
		return 0, fmt.Errorf("send icmp: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1500)

	for {
		_, peer, err := conn.ReadFrom(buf)
		if err != nil {
			if os.IsTimeout(err) || strings.Contains(err.Error(), "timeout") {
				return 0, fmt.Errorf("ping timeout")
			}
			return 0, fmt.Errorf("read icmp: %v", err)
		}

		elapsed := time.Since(start)

		if peer.(*net.IPAddr).IP.Equal(addr) {
			if len(buf) >= 28 && buf[20] == 0 && buf[21] == 0 {
				return elapsed, nil
			}
		}

		if elapsed > timeout {
			return 0, fmt.Errorf("ping timeout")
		}
	}
}

func pingCmd(ip string, timeout time.Duration) (time.Duration, error) {
	sec := int(timeout.Seconds())
	if sec < 1 {
		sec = 1
	}
	out, err := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(sec), ip).Output()
	if err != nil {
		return 0, fmt.Errorf("ping command failed: %v", err)
	}
	// Parse "time=1.23 ms" from output
	re := regexp.MustCompile(`time=(\d+\.?\d*)\s*ms`)
	m := re.FindStringSubmatch(string(out))
	if len(m) < 2 {
		return 0, fmt.Errorf("no response")
	}
	t, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse latency: %v", err)
	}
	return time.Duration(t * float64(time.Millisecond)), nil
}

func buildICMPEchoRequest(id, seq uint16) []byte {
	pkt := make([]byte, 8)
	pkt[0] = 8 // Echo request
	pkt[1] = 0 // Code 0
	binary.BigEndian.PutUint16(pkt[4:], id)
	binary.BigEndian.PutUint16(pkt[6:], seq)

	// Checksum
	cs := icmpChecksum(pkt)
	binary.BigEndian.PutUint16(pkt[2:], cs)
	return pkt
}

func icmpChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i:]))
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	sum = (sum >> 16) + (sum & 0xFFFF)
	sum += sum >> 16
	return ^uint16(sum)
}
