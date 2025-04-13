package main

import (
	"log/slog"
	"net"
	"os"
	"unicode/utf8"

	"github.com/gosnmp/gosnmp"
)

const (
	listenAddress   = "0.0.0.0:161"        // listen for traffic
	upstreamAddress = "snmp-simulator:161" // route traffic to the simulator
	maxUDPSize      = 65535                // max UDP packet size
)

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddress)
	if err != nil {
		slog.Error("Error resolving UDP address", "error", err)
		os.Exit(1)
	}
	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		slog.Error("Error listening on UDP", "error", err)
		os.Exit(1)
	}
	defer func() { _ = listener.Close() }()

	slog.Info("Started UDP listener", "listenAddress", listenAddress, "upstreamAddress", upstreamAddress)

	buffer := make([]byte, maxUDPSize)
	for {
		n, addr, err := listener.ReadFromUDP(buffer)
		if err != nil {
			slog.Error("Error reading from UDP", "error", err)
			continue
		}
		go processRequest(addr, listener, upstreamAddress, buffer[:n]) // concurrent
	}
}

// processRequest handles all incoming requests and routes them accordingly, based on their contents.
func processRequest(clientAddr *net.UDPAddr, listener *net.UDPConn, upstreamAddress string, request []byte) {
	if utf8.Valid(request) {
		processHealthRequest(clientAddr, listener)
	} else {
		processSNMPRequest(clientAddr, listener, upstreamAddress, request)
	}
}

// processHealthRequest processes a health request.
func processHealthRequest(clientAddr *net.UDPAddr, listener *net.UDPConn) {
	if _, err := listener.WriteToUDP([]byte("up\n"), clientAddr); err != nil {
		slog.Warn("Error sending health check", "error", err)
	}
	slog.Info("Sent health response", "clientAddr", clientAddr)
}

// processSNMPRequest processes an SNMP request.
func processSNMPRequest(clientAddr *net.UDPAddr, listener *net.UDPConn, upstreamAddress string, request []byte) {
	// Parse the incoming SNMP packet
	gsnmp := &gosnmp.GoSNMP{}
	pkt, err := gsnmp.SnmpDecodePacket(request)
	if err != nil {
		slog.Warn("Error decoding SNMP packet", "error", err)
		return
	}

	// Re-encode the packet to send to the upstream SNMP agent
	encodedPacket, err := pkt.MarshalMsg()
	if err != nil {
		slog.Warn("Error encoding SNMP packet", "error", err)
		return
	}

	// Resolve the upstream address
	upstreamAddr, err := net.ResolveUDPAddr("udp", upstreamAddress)
	if err != nil {
		slog.Warn("Error resolving upstream address", "error", err)
		return
	}

	// Dial the upstream SNMP agent
	conn, err := net.DialUDP("udp", nil, upstreamAddr)
	if err != nil {
		slog.Warn("Error dialing upstream SNMP agent", "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// Write the encoded packet to the upstream SNMP agent
	_, err = conn.Write(encodedPacket)
	if err != nil {
		slog.Warn("Error writing to upstream SNMP agent", "error", err)
		return
	}

	// Read the response from the upstream SNMP agent
	responseBuffer := make([]byte, 65535)
	n, _, err := conn.ReadFromUDP(responseBuffer)
	if err != nil {
		slog.Warn("Error reading response from upstream SNMP agent", "error", err)
		return
	}

	// Send the response back to the client
	_, err = listener.WriteToUDP(responseBuffer[:n], clientAddr)
	if err != nil {
		slog.Warn("Error writing SNMP response to client", "error", err)
		return
	}

	slog.Info("Sent SNMP response", "clientAddr", clientAddr)
}
