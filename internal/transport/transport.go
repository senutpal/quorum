// =============================================================================
// TRANSPORT INTERFACE - Abstraction for Message Passing
// =============================================================================
//
// IMPLEMENTATION ORDER: File 1 of 2 in internal/transport/
// Implement this AFTER internal/storage/ files
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// This file defines an abstract interface for sending and receiving messages
// between Paxos nodes. Like Storage, different implementations allow:
//
// - In-memory (single process, for testing)
// - TCP/UDP (real network)
// - gRPC (production-ready RPC)
// - Unix sockets (local multi-process)
//
// Paxos is a message-passing protocol. This interface is how messages flow.
//
// =============================================================================
// WHY ABSTRACT TRANSPORT
// =============================================================================
//
// Real distributed systems need networks. But for learning and testing:
//
// 1. Network code is complex (connections, serialization, errors)
// 2. Network tests are slow and flaky
// 3. We want to test Paxos logic, not networking
//
// Solution: Define an interface. Implement simple in-memory version first.
// Add real network later without changing Paxos code.
//
// =============================================================================
// TRANSPORT SEMANTICS
// =============================================================================
//
// Paxos assumes an ASYNCHRONOUS network:
//
// - Messages can be delayed arbitrarily
// - Messages can be lost
// - Messages can be reordered
// - Messages are NOT duplicated (or duplicates can be detected)
// - Messages are NOT corrupted (or corruption can be detected)
//
// The transport interface should reflect these semantics:
//
// - Send is "fire and forget" (no guarantee of delivery)
// - Receive blocks until a message arrives (or timeout)
// - No ordering guarantees between different senders
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define message envelope
//       The transport needs to know:
//       - Where to send the message (destination node ID)
//       - What type of message it is (Prepare, Promise, Accept, Accepted)
//       - The message content
//
//       type Envelope struct {
//           To      string      // Destination node ID
//           From    string      // Sender node ID
//           Type    MessageType // Enum: Prepare, Promise, Accept, Accepted
//           Payload []byte      // Serialized message
//       }
//
//       OR use an interface:
//       type Message interface {
//           GetFrom() string
//       }
//
//
// TODO: Define MessageType enum
//       const (
//           MessageTypePrepare MessageType = iota
//           MessageTypePromise
//           MessageTypeAccept
//           MessageTypeAccepted
//           MessageTypeLearn
//       )
//
//
// TODO: Define Transport interface
//       Methods:
//
//       - Send(to string, msg Message) error
//         // Send a message to the specified node
//         // May return error if node is unknown
//         // Does NOT guarantee delivery (async network)
//
//       - Broadcast(msg Message) error
//         // Send message to all known nodes
//         // Convenience for proposer sending to all acceptors
//
//       - Receive() (Message, error)
//         // Block until a message is received
//         // Returns error if transport is closed
//
//       - ReceiveTimeout(timeout time.Duration) (Message, error)
//         // Same as Receive but with timeout
//         // Returns error on timeout (important for liveness!)
//
//       - RegisterHandler(msgType MessageType, handler func(Message))
//         // Alternative: callback-based message handling
//         // When message of msgType arrives, call handler
//
//       - Close() error
//         // Shut down the transport
//
//
// TODO: Choose between Receive vs RegisterHandler patterns
//
//       Pattern 1: Polling (Receive)
//         for {
//             msg, _ := transport.Receive()
//             switch m := msg.(type) {
//             case *Prepare: handlePrepare(m)
//             case *Accept: handleAccept(m)
//             }
//         }
//
//       Pattern 2: Callbacks (RegisterHandler)
//         transport.RegisterHandler(MessageTypePrepare, func(m Message) {
//             handlePrepare(m.(*Prepare))
//         })
//         transport.Start()  // Starts receiving and calling handlers
//
//       For learning: Polling is simpler to understand.
//       For production: Callbacks are more flexible.
//
// =============================================================================
// SERIALIZATION
// =============================================================================
//
// Messages need to be serialized for network transport. Options:
//
// 1. encoding/gob (Go-native, easy)
// 2. encoding/json (human-readable, slow)
// 3. Protocol Buffers (fast, cross-language)
// 4. Custom binary (fastest, most complex)
//
// For learning: Use encoding/gob or encoding/json.
// For production: Use Protocol Buffers.
//
// The transport layer handles serialization. Paxos code works with
// Go structs, not bytes.
//
// =============================================================================
// NODE DISCOVERY
// =============================================================================
//
// How does a node know about other nodes?
//
// Static configuration:
//   nodes := []string{"node-1", "node-2", "node-3"}
//   transport := NewTCPTransport(nodes)
//
// Dynamic discovery:
//   transport := NewTransport()
//   transport.AddNode("node-1", "192.168.1.1:8080")
//   transport.AddNode("node-2", "192.168.1.2:8080")
//
// For learning: Use static configuration.
// For production: Consider service discovery (Consul, DNS, etc.)
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Messages are delivered at most once, uncorrupted, and to the
//            correct destination.
//
// - "At most once": OK to lose messages (Paxos handles this), but don't
//   duplicate (or deduplicate at receiver)
// - "Uncorrupted": Use checksums or let TCP handle it
// - "Correct destination": Don't deliver node-1's messages to node-2
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Blocking Send when destination is down
//
// If Send blocks waiting for acknowledgment and the destination is down,
// the sender hangs and can't make progress.
//
// WRONG:
//   func (t *TCPTransport) Send(to string, msg Message) error {
//       conn := t.connect(to)  // Blocks if down!
//       conn.Write(serialize(msg))
//       ack := conn.Read()  // Blocks waiting for ack!
//       return nil
//   }
//
// RIGHT:
//   func (t *TCPTransport) Send(to string, msg Message) error {
//       conn := t.getConnection(to)  // Use existing or fail fast
//       if conn == nil {
//           return ErrNodeDown  // Return immediately
//       }
//       go func() {
//           conn.Write(serialize(msg))  // Fire and forget
//       }()
//       return nil
//   }
//
// Paxos is designed for unreliable networks. Let the protocol handle
// lost messages rather than blocking forever.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// For Multi-Paxos, the transport remains largely the same. Messages just
// include a slot number:
//
//   type Prepare struct {
//       Slot           int64
//       ProposalNumber ProposalNumber
//       From           string
//   }
//
// The transport doesn't care about slot numbers - it just delivers messages.
//
// =============================================================================

package transport

// Import the paxos package for message types.
// Uncomment when implementing:
// import "github.com/quorum/paxos/internal/paxos"
