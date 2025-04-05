package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	shutdownWait = 5 * time.Second        // for forceful termination
	sendTimeout  = 1 * time.Second        // send duration
	sendInterval = 100 * time.Millisecond // send frequency
	handlerCount = 10                     // worker count
	handlerSize  = 4 * 1024               // worker buffer
)

type workerIdKey struct{}

type workerId int

func (id workerId) String() string {
	return fmt.Sprintf("work-%d", id)
}

func main() {
	addr := "0.0.0.0:161" // envoy will forward to this port
	conn := setupConnection(addr)

	slog.Info("UDP server listening", "address", addr)

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	for i := range handlerCount { // start handlers
		wg.Add(1)
		go func(id int) {
			ctx = context.WithValue(ctx, workerIdKey{}, workerId(id))
			defer wg.Done()
			handleConnection(ctx, conn)
		}(i)
	}

	quit := make(chan os.Signal, 1) // wait for interrupt
	signal.Notify(quit, os.Interrupt)
	<-quit
	closeConnection(conn)
	cancel() // stop handlers
	slog.Info("Initiate shutdown", "threshold", shutdownWait)
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()
	select {
	case <-time.After(shutdownWait): // forceful termination
		slog.Warn("Shutdown forcefully")
	case <-done:
		slog.Info("Shutdown gracefully")
	}
}

// setupConnection initializes a UDP connection on the specified address.
func setupConnection(addr string) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		slog.Error("Error resolving UDP address", "error", err)
		os.Exit(1)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		slog.Error("Error listening for UDP", "error", err)
		os.Exit(1)
	}
	return conn
}

// closeConnection safely closes the given UDP connection and logs any errors.
func closeConnection(conn *net.UDPConn) {
	if err := conn.Close(); err != nil {
		slog.Warn("Error closing the connection", "error", err)
	} else {
		slog.Info("Closed connection")
	}
}

// handleConnection processes incoming UDP packets and sends periodic responses.
func handleConnection(ctx context.Context, conn *net.UDPConn) {
	id := ctx.Value(workerIdKey{}).(workerId)
	slog.Info("Start handler", "id", id)
	buffer := make([]byte, handlerSize) // buffer to hold incoming UDP packets
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if errors.Is(err, net.ErrClosed) {
			slog.Info("Connection closed", "id", id)
			return
		}
		if err != nil {
			slog.Error("Error reading from UDP", "id", id, "error", err)
			continue
		}
		message := buffer[:n]
		slog.Info("Receive data", "id", id, "bytes", len(message), "remoteAddr", remoteAddr)
		sendPeriodicResponses(ctx, conn, remoteAddr)
	}
}

// sendPeriodicResponses sends periodic responses to the specified remote address for a limited time.
func sendPeriodicResponses(ctx context.Context, conn *net.UDPConn, remoteAddr *net.UDPAddr) {
	id := ctx.Value(workerIdKey{}).(workerId)
	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()
	timeout := time.After(sendTimeout)
	for {
		select {
		case <-ctx.Done():
			slog.Warn("Abort sending data", "id", id)
			return
		case <-timeout:
			slog.Info("Stop sending data", "id", id)
			return
		case <-ticker.C:
			message := []byte("echo\n")
			slog.Info("Send data", "id", id, "bytes", len(message), "remoteAddr", remoteAddr)
			if _, err := conn.WriteToUDP(message, remoteAddr); err != nil {
				slog.Error("Error writing to UDP", "id", id, "error", err)
				return
			}
		}
	}
}
