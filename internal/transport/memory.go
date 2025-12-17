// =============================================================================
// IN-MEMORY TRANSPORT - Testing/Demo Implementation
// =============================================================================
//
// IMPLEMENTATION ORDER: File 2 of 2 in internal/transport/
// Implement this AFTER transport.go
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// This is an in-memory implementation of the Transport interface.
// All "nodes" run in the same Go process and communicate via channels.
//
// USE CASES:
// - Unit testing (no network setup)
// - Demos (run 5 Paxos nodes in one process)
// - Development (fast iteration)
// - Learning (see message flow with logging)
//
// NOT FOR PRODUCTION: Only works within a single process!
//
// =============================================================================
// HOW IT WORKS
// =============================================================================
//
// The in-memory transport uses Go channels for message passing:
//
//   ┌─────────┐     channel      ┌─────────┐
//   │  Node A │ ───────────────▶ │  Node B │
//   │         │                  │         │
//   │  inbox  │ ◀─────────────── │  Send() │
//   └─────────┘     channel      └─────────┘
//
// Each node has an "inbox" channel. When A sends to B:
// 1. A looks up B's inbox channel in a shared registry
// 2. A writes the message to B's channel
// 3. B receives from its inbox channel
//
// This simulates async network communication without actual networking.
//
// =============================================================================
// NETWORK SIMULATION FEATURES
// =============================================================================
//
// For realistic testing, you might want to simulate:
//
// 1. MESSAGE DELAY
//    - Add random delay before delivery
//    - Helps catch race conditions
//
// 2. MESSAGE LOSS
//    - Randomly drop messages
//    - Tests Paxos resilience
//
// 3. MESSAGE REORDERING
//    - Deliver messages out of order
//    - Tests that Paxos doesn't assume ordering
//
// 4. NETWORK PARTITIONS
//    - Block messages between certain nodes
//    - Tests split-brain scenarios
//
// Start simple (no simulation), add these features for advanced testing.
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define MemoryTransport struct
//       Fields:
//       - nodeID string
//         // This transport's node ID
//       - inbox chan Message
//         // Channel to receive messages
//       - nodes map[string]chan Message
//         // Registry: node ID → inbox channel
//         // Shared between all transports in the network
//       - mu sync.RWMutex
//         // Protect the shared registry
//       - closed bool
//         // True if transport is closed
//
//
// TODO: Define Network struct (factory/registry)
//       A shared object that all transports reference:
//
//       type Network struct {
//           channels map[string]chan Message
//           mu       sync.RWMutex
//       }
//
//       func NewNetwork() *Network
//       func (n *Network) AddNode(id string) *MemoryTransport
//         // Creates a new transport for this node
//         // Registers its inbox in the shared channels map
//
//
// TODO: Implement NewMemoryTransport(nodeID string, network *Network) *MemoryTransport
//       1. Create inbox channel (buffered, e.g., 100 messages)
//       2. Register in network's channel map
//       3. Store reference to network for Send
//
//
// TODO: Implement Send(to string, msg Message) error
//       1. Look up destination's inbox channel
//       2. If not found, return error (unknown node)
//       3. Send message to channel (non-blocking)
//          - If channel full, either block or drop (your choice)
//       4. Return nil
//
//       NOTE: Use select with default for non-blocking send:
//         select {
//         case destInbox <- msg:
//             return nil
//         default:
//             return ErrInboxFull  // Or drop silently
//         }
//
//
// TODO: Implement Broadcast(msg Message) error
//       For each node in the network (except self):
//         Send(nodeID, msg)
//       Return first error or nil
//
//
// TODO: Implement Receive() (Message, error)
//       1. If closed, return error
//       2. Block on inbox channel: msg := <-inbox
//       3. Return msg
//
//
// TODO: Implement ReceiveTimeout(timeout time.Duration) (Message, error)
//       select {
//       case msg := <-inbox:
//           return msg, nil
//       case <-time.After(timeout):
//           return nil, ErrTimeout
//       }
//
//
// TODO: Implement Close() error
//       1. Set closed = true
//       2. Close inbox channel
//       3. Remove from network registry
//
// =============================================================================
// BUFFERED VS UNBUFFERED CHANNELS
// =============================================================================
//
// UNBUFFERED:
//   inbox := make(chan Message)
//   - Sender blocks until receiver reads
//   - Simulates synchronous RPC
//   - Can cause deadlocks if not careful
//
// BUFFERED:
//   inbox := make(chan Message, 100)
//   - Sender doesn't block (until buffer full)
//   - Simulates async network
//   - Recommended for Paxos
//
// Paxos assumes async network, so BUFFERED is more realistic.
//
// =============================================================================
// LOGGING FOR LEARNING
// =============================================================================
//
// Add logging to see message flow:
//
//   func (t *MemoryTransport) Send(to string, msg Message) error {
//       log.Printf("[%s] → [%s]: %T", t.nodeID, to, msg)
//       // ... send logic ...
//   }
//
//   func (t *MemoryTransport) Receive() (Message, error) {
//       msg := <-t.inbox
//       log.Printf("[%s] ← received: %T", t.nodeID, msg)
//       return msg, nil
//   }
//
// This makes it easy to trace the Paxos protocol in action.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Messages sent to node X are only received by node X.
//
// This seems trivial for in-memory (each node has its own channel), but
// bugs in the registry lookup could cause misdelivery.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Deadlock from unbuffered channel + same-goroutine call
//
// Scenario:
// 1. Node A handles a message in goroutine G
// 2. While handling, A sends a message to itself (A → A)
// 3. Send blocks because A's inbox (unbuffered) is not being read
// 4. But A can't read because G is blocked in Send
// 5. DEADLOCK
//
// FIX:
// - Use buffered channels
// - OR never send to self
// - OR use separate goroutines for send/receive
//
// =============================================================================
// ADVANCED: SIMULATING FAILURES
// =============================================================================
//
// For advanced testing, add methods like:
//
//   func (n *Network) Partition(nodeA, nodeB string)
//     // Block messages between A and B
//
//   func (n *Network) Heal(nodeA, nodeB string)
//     // Restore communication
//
//   func (n *Network) SetMessageLoss(probability float64)
//     // Randomly drop messages
//
// These help test Paxos behavior under failure conditions.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// The transport doesn't change for Multi-Paxos. It just delivers messages.
// The messages themselves contain slot numbers, but the transport doesn't
// care about that.
//
// =============================================================================

package transport

// Import sync for mutex.
// Uncomment when implementing:
// import "sync"
