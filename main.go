package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// External configuration
type Config struct {
	PS5Interface      string `json:"ps5_interface"`
	InternetInterface string `json:"internet_interface"`
	ExitLagPath       string `json:"exitlag_path"`
	LogFile           string `json:"log_file"`
	MaxLogSizeBytes   int64  `json:"max_log_size_bytes"`
	LogLevel          string `json:"log_level"`
}

var (
	connMap         = make(map[string]*net.UDPAddr)
	connMutex       sync.Mutex
	config          *Config
	logger          *log.Logger
	levelMap        = map[string]int{"info": 1, "warn": 2, "error": 3}
	currentLogLevel int
)

func main() {
	var err error
	config, err = loadConfig("config.json")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	currentLogLevel = levelMap[config.LogLevel]
	logger = initLogger(config.LogFile, config.MaxLogSizeBytes)
	info("Application started.")

	listInterfaces() // show available interfaces

	cmd := exec.Command(config.ExitLagPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	err = cmd.Start()
	if err != nil {
		errorLog("Failed to start ExitLag: %v", err)
		os.Exit(1)
	}
	info("ExitLag started.")

	go captureAndForward(config.PS5Interface)
	go captureResponses(config.InternetInterface)

	go func() {
		cmd.Wait()
		info("ExitLag closed. Shutting down proxy.")
		os.Exit(0)
	}()

	select {}
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func initLogger(logFile string, maxSize int64) *log.Logger {
	checkAndRotateLog(logFile, maxSize)

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	logger := log.New(f, "", log.LstdFlags)
	return logger
}

func checkAndRotateLog(logFile string, maxSize int64) {
	info, err := os.Stat(logFile)
	if err == nil && info.Size() > maxSize {
		timestamp := time.Now().Format("20060102_150405")
		backupName := fmt.Sprintf("%s.%s.bak", logFile, timestamp)
		os.Rename(logFile, backupName)
	}
}

func info(format string, v ...interface{}) {
	if currentLogLevel <= 1 {
		logger.Printf("[INFO] "+format, v...)
	}
}

func warn(format string, v ...interface{}) {
	if currentLogLevel <= 2 {
		logger.Printf("[WARN] "+format, v...)
	}
}

func errorLog(format string, v ...interface{}) {
	if currentLogLevel <= 3 {
		logger.Printf("[ERROR] "+format, v...)
	}
}

func listInterfaces() {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		errorLog("Failed to list interfaces: %v", err)
		return
	}
	info("Available network interfaces:")
	for _, dev := range devices {
		info("%s (%s)", dev.Name, dev.Description)
	}
}

func captureAndForward(iface string) {
	handle, err := pcap.OpenLive(iface, 65536, true, pcap.BlockForever)
	if err != nil {
		errorLog("Failed to open interface %s: %v", iface, err)
		listInterfaces()
		os.Exit(1)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	info("Listening for PS5 on: %s", iface)

	for packet := range packetSource.Packets() {
		udpLayer := packet.Layer(layers.LayerTypeUDP)
		ipLayer := packet.Layer(layers.LayerTypeIPv4)

		if udpLayer == nil || ipLayer == nil {
			continue
		}

		fmt.Println("[DEBUG] Packet captured from PS5 interface:", packet.String())

		udp := udpLayer.(*layers.UDP)
		ip := ipLayer.(*layers.IPv4)

		payload := udp.Payload
		destAddr := fmt.Sprintf("%s:%d", ip.DstIP, udp.DstPort)
		ps5Addr := &net.UDPAddr{IP: ip.SrcIP, Port: int(udp.SrcPort)}

		connMutex.Lock()
		connMap[destAddr] = ps5Addr
		connMutex.Unlock()

		go func(dest string, data []byte) {
			conn, err := net.Dial("udp", dest)
			if err != nil {
				warn("Failed to forward to %s: %v", dest, err)
				return
			}
			defer conn.Close()

			_, err = conn.Write(data)
			if err != nil {
				warn("Failed to send data to %s: %v", dest, err)
			}
		}(destAddr, payload)
	}
}

func captureResponses(iface string) {
	handle, err := pcap.OpenLive(iface, 65536, true, pcap.BlockForever)
	if err != nil {
		errorLog("Failed to open interface %s: %v", iface, err)
		listInterfaces()
		os.Exit(1)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	info("Listening for responses on: %s", iface)

	for packet := range packetSource.Packets() {
		udpLayer := packet.Layer(layers.LayerTypeUDP)
		ipLayer := packet.Layer(layers.LayerTypeIPv4)

		if udpLayer == nil || ipLayer == nil {
			continue
		}

		udp := udpLayer.(*layers.UDP)
		ip := ipLayer.(*layers.IPv4)

		srcKey := fmt.Sprintf("%s:%d", ip.SrcIP, udp.SrcPort)

		connMutex.Lock()
		ps5Addr, exists := connMap[srcKey]
		connMutex.Unlock()

		if !exists {
			continue
		}

		conn, err := net.DialUDP("udp", nil, ps5Addr)
		if err != nil {
			warn("Failed to send response to PS5 %s: %v", ps5Addr.String(), err)
			continue
		}

		_, err = conn.Write(udp.Payload)
		if err != nil {
			warn("Failed to write response to PS5 %s: %v", ps5Addr.String(), err)
		}
		conn.Close()
	}
}
