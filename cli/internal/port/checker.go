package port

import (
	"fmt"
	"net"
	"time"
)

// IsAvailable checks if a port is available for binding
func IsAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// IsListening checks if something is listening on the given port.
// Tries both IPv4 (127.0.0.1) and IPv6 (::1) since servers may bind to either.
func IsListening(port int) bool {
	// Try IPv4 first
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	// Try IPv6
	addr = fmt.Sprintf("[::1]:%d", port)
	conn, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}

	return false
}

// WaitForPort waits for a port to become available (listening)
func WaitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if IsListening(port) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for port %d to become available", port)
}

// WaitForPortFree waits for a port to become free (not listening)
func WaitForPortFree(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if IsAvailable(port) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for port %d to become free", port)
}

// FindAvailablePort finds an available port in the given range
func FindAvailablePort(minPort, maxPort int) (int, error) {
	for port := minPort; port <= maxPort; port++ {
		if IsAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", minPort, maxPort)
}
