package port

import (
	"fmt"
	"hash/fnv"
)

// Allocator handles port allocation for worktrees
type Allocator struct {
	minPort int
	maxPort int
}

// NewAllocator creates a new port allocator with the given range
func NewAllocator(minPort, maxPort int) *Allocator {
	return &Allocator{
		minPort: minPort,
		maxPort: maxPort,
	}
}

// Allocate returns a deterministic port for the given worktree name
// The port is based on a hash of the name, ensuring the same worktree
// always gets the same port
func (a *Allocator) Allocate(name string) int {
	h := fnv.New32a()
	h.Write([]byte(name))
	hash := h.Sum32()

	portRange := uint32(a.maxPort - a.minPort + 1)
	offset := hash % portRange

	return a.minPort + int(offset)
}

// AllocateWithFallback returns a port for the given worktree name,
// trying alternative ports if the primary one is in use
func (a *Allocator) AllocateWithFallback(name string, usedPorts map[int]bool) (int, error) {
	primary := a.Allocate(name)

	// Check if primary port is available
	if !usedPorts[primary] && IsAvailable(primary) {
		return primary, nil
	}

	// Try alternative ports by appending numbers to the name
	for i := 1; i <= 100; i++ {
		altName := fmt.Sprintf("%s-%d", name, i)
		altPort := a.Allocate(altName)

		if !usedPorts[altPort] && IsAvailable(altPort) {
			return altPort, nil
		}
	}

	// As a last resort, find any available port in range
	for port := a.minPort; port <= a.maxPort; port++ {
		if !usedPorts[port] && IsAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", a.minPort, a.maxPort)
}

// Range returns the port range
func (a *Allocator) Range() (int, int) {
	return a.minPort, a.maxPort
}
