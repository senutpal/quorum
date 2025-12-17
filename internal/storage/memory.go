// =============================================================================
// IN-MEMORY STORAGE - Testing/Demo Implementation
// =============================================================================
//
// IMPLEMENTATION ORDER: File 2 of 2 in internal/storage/
// Implement this AFTER storage.go
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// This is a simple in-memory implementation of the Storage interface.
// It stores data in Go variables - nothing is persisted to disk.
//
// USE CASES:
// - Unit testing (fast, no cleanup needed)
// - Demos and learning
// - Development (quick iteration)
//
// NOT FOR PRODUCTION: Data is lost on restart!
//
// =============================================================================
// WHY WE NEED THIS
// =============================================================================
//
// Even though in-memory storage doesn't provide real durability, we need
// SOME storage implementation to:
//
// 1. Test the Paxos algorithm logic
// 2. Run demos without setting up a database
// 3. Develop and debug quickly
//
// The Paxos code itself doesn't know or care that this is in-memory.
// It just calls Save/Load. This is the power of interfaces.
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define MemoryStorage struct
//       Fields:
//       - highestPromised ProposalNumber
//         // In-memory copy of promised proposal
//       - acceptedProposal ProposalNumber
//         // In-memory copy of accepted proposal
//       - acceptedValue []byte
//         // In-memory copy of accepted value
//       - mu sync.RWMutex
//         // Protect concurrent access
//
//
// TODO: Implement NewMemoryStorage() *MemoryStorage
//       - Initialize with zero values
//       - Return pointer to new storage
//
//
// TODO: Implement SavePromised(proposal ProposalNumber) error
//       1. Lock the mutex
//       2. Set highestPromised = proposal
//       3. Return nil
//
//       NOTE: In a real implementation, we'd fsync here.
//       In-memory just returns immediately.
//
//
// TODO: Implement LoadPromised() (ProposalNumber, error)
//       1. RLock the mutex
//       2. Return highestPromised
//
//
// TODO: Implement SaveAccepted(proposal ProposalNumber, value []byte) error
//       1. Lock the mutex
//       2. Set acceptedProposal = proposal
//       3. Set acceptedValue = copy of value (defensive copy!)
//       4. Return nil
//
//       WHY COPY: If caller modifies the value slice after calling Save,
//       we don't want our stored value to change.
//
//
// TODO: Implement LoadAccepted() (ProposalNumber, []byte, error)
//       1. RLock the mutex
//       2. Return (acceptedProposal, copy of acceptedValue)
//
//       WHY COPY: Don't let caller modify our internal state.
//
//
// TODO: Implement Close() error
//       - Clear all state (optional)
//       - Return nil
//
//
// TODO: Optional - Implement Reset() for testing
//       - Clear all state
//       - Useful for test isolation
//
// =============================================================================
// THREAD SAFETY
// =============================================================================
//
// The storage implementation MUST be thread-safe because:
//
// 1. Multiple goroutines might access storage concurrently
// 2. The Paxos node might handle multiple requests in parallel
//
// Use sync.RWMutex:
// - RLock for Load operations (allow concurrent reads)
// - Lock for Save operations (exclusive access)
//
// =============================================================================
// DEFENSIVE COPYING
// =============================================================================
//
// Always copy []byte values in and out:
//
// WRONG (SaveAccepted):
//   m.acceptedValue = value  // Stores the same slice!
//
// RIGHT:
//   m.acceptedValue = make([]byte, len(value))
//   copy(m.acceptedValue, value)
//
// WRONG (LoadAccepted):
//   return m.acceptedProposal, m.acceptedValue, nil  // Exposes internal!
//
// RIGHT:
//   result := make([]byte, len(m.acceptedValue))
//   copy(result, m.acceptedValue)
//   return m.acceptedProposal, result, nil
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Load always returns the most recent Save value (or zero if
//            never saved).
//
// This seems obvious but is easy to break with concurrency bugs.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Forgetting to copy byte slices
//
// Scenario:
// 1. Save(proposal, value) - stores reference to caller's slice
// 2. Caller modifies value slice
// 3. Our stored value is now corrupted!
//
// This is subtle because it might work in tests but fail in production
// where slices are reused.
//
// FIX: Always copy []byte on Save and Load.
//
// =============================================================================
// TESTING WITH IN-MEMORY STORAGE
// =============================================================================
//
// Example test pattern:
//
//   func TestAcceptor(t *testing.T) {
//       storage := NewMemoryStorage()
//       acceptor := NewAcceptor("test-1", storage)
//
//       // ... test acceptor behavior ...
//   }
//
// Each test gets a fresh storage - no cleanup needed, no interference.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// For Multi-Paxos, extend to store per-slot state:
//
//   type MemoryStorage struct {
//       globalPromise ProposalNumber     // Leader promise
//       slots map[int64]*SlotState       // Per-slot state
//       mu sync.RWMutex
//   }
//
// =============================================================================

package storage

import "sync"

type MemoryStorage struct {
	highestPromised  ProposalNumber
	acceptedProposal ProposalNumber
	acceptedValue    []byte
	mu               sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{}
}

func (m *MemoryStorage) SavePromised(proposal ProposalNumber) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.highestPromised = proposal
	return nil
}

func (m *MemoryStorage) LoadPromised() (ProposalNumber, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.highestPromised, nil
}

func (m *MemoryStorage) SaveAccepted(proposal ProposalNumber, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acceptedProposal = proposal
	m.acceptedValue = make([]byte, len(value))
	copy(m.acceptedValue, value)
	return nil
}

func (m *MemoryStorage) LoadAccepted() (ProposalNumber, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]byte, len(m.acceptedValue))
	copy(result, m.acceptedValue)
	return m.acceptedProposal, result, nil
}

func (m *MemoryStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.highestPromised = ProposalNumber{}
	m.acceptedProposal = ProposalNumber{}
	m.acceptedValue = nil
	return nil
}

func (m *MemoryStorage) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.highestPromised = ProposalNumber{}
	m.acceptedProposal = ProposalNumber{}
	m.acceptedValue = nil
}
