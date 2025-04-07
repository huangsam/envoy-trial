package main

import (
	"log"
	"net"
)

func main() {
	// SNMP listener configuration
	listenAddress := "0.0.0.0:161"
	upstreamAddress := "et-sim:161" // Replace with your target SNMP agent

	// Create UDP listener
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddress)
	if err != nil {
		log.Fatalf("Error resolving UDP address: %v", err)
	}
	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Error listening on UDP: %v", err)
	}
	defer listener.Close()

	log.Printf("SNMP proxy listening on %s, forwarding to %s", listenAddress, upstreamAddress)

	buffer := make([]byte, 65535) // Maximum UDP packet size

	for {
		n, addr, err := listener.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}
		go handleRequest(buffer[:n], addr, listener, upstreamAddress)
	}
}

func handleRequest(request []byte, clientAddr *net.UDPAddr, listener *net.UDPConn, upstreamAddress string) {
	// Forward request to upstream SNMP agent
	upstreamConn, err := net.Dial("udp", upstreamAddress)
	if err != nil {
		log.Printf("Error dialing upstream: %v", err)
		return
	}
	defer upstreamConn.Close()

	_, err = upstreamConn.Write(request)
	if err != nil {
		log.Printf("Error writing to upstream: %v", err)
		return
	}

	// Read response from upstream
	responseBuffer := make([]byte, 65535)
	n, err := upstreamConn.Read(responseBuffer)
	if err != nil {
		log.Printf("Error reading from upstream: %v", err)
		return
	}

	// Forward response to client
	_, err = listener.WriteToUDP(responseBuffer[:n], clientAddr)
	if err != nil {
		log.Printf("Error writing to client: %v", err)
	}
}
