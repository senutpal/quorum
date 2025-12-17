// =============================================================================
// NODE - Wiring All Paxos Roles Together
// =============================================================================
//
// IMPLEMENTATION ORDER: File 1 of 1 in internal/node/
// Implement this AFTER all storage and transport files
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// A "Node" in Paxos is a single participant that can play ALL THREE roles:
//
//   ┌─────────────────────────────────────────────────────────┐
//   │                         NODE                            │
//   │  ┌───────────┐  ┌───────────┐  ┌───────────┐           │
//   │  │ PROPOSER  │  │ ACCEPTOR  │  │  LEARNER  │           │
//   │  │           │  │           │  │           │           │
//   │  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘           │
//   │        │              │              │                  │
//   │        └──────────────┼──────────────┘                  │
//   │                       │                                 │
//   │                 ┌─────┴─────┐                          │
//   │                 │ TRANSPORT │                          │
//   │                 └───────────┘                          │
//   │                       │                                 │
//   │                 ┌─────┴─────┐                          │
//   │                 │  STORAGE  │                          │
//   │                 └───────────┘                          │
//   └─────────────────────────────────────────────────────────┘
//
// This file wires everything together:
// - Creates Proposer, Acceptor, Learner instances
// - Connects them to Transport and Storage
// - Routes incoming messages to the right handler
// - Provides a clean API for clients
//
// =============================================================================
// WHY COMBINE ROLES IN ONE NODE
// =============================================================================
//
// In theory, Paxos roles are separate. In practice:
//
// 1. EFFICIENCY: Each server plays all roles → fewer network hops
// 2. SIMPLICITY: One process to deploy, monitor, and manage
// 3. FAULT TOLERANCE: N servers, each is proposer + acceptor + learner
//
// When a client wants to propose a value:
// 1. Client contacts any node
// 2. That node's Proposer runs the protocol
// 3. That node's Acceptor participates in voting
// 4. That node's Learner discovers the result
// 5. Node returns result to client
//
// =============================================================================
// MESSAGE ROUTING
// =============================================================================
//
// When a message arrives, the node must route it to the right handler:
//
//   switch msg.Type {
//   case Prepare:
//       response := node.acceptor.HandlePrepare(msg)
//       node.transport.Send(msg.From, response)
//
//   case Promise:
//       node.proposer.HandlePromise(msg)  // Proposer collects these
//
//   case Accept:
//       response := node.acceptor.HandleAccept(msg)
//       node.transport.Send(msg.From, response)
//       // Also notify learners!
//       node.learner.HandleAccepted(response)
//
//   case Accepted:
//       node.proposer.HandleAccepted(msg)  // Proposer collects these
//       node.learner.HandleAccepted(msg)   // Learner tracks these
//   }
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define Node struct
//       Fields:
//       - id string
//         // Unique identifier for this node
//       - proposer *Proposer
//         // This node's proposer instance
//       - acceptor *Acceptor
//         // This node's acceptor instance
//       - learner *Learner
//         // This node's learner instance
//       - transport Transport
//         // Network layer
//       - storage Storage
//         // Persistent storage
//       - quorumSize int
//         // How many nodes form a majority
//       - mu sync.Mutex
//         // Protect concurrent access
//       - running bool
//         // True if the node is processing messages
//       - stopCh chan struct{}
//         // Signal to stop the message loop
//
//
// TODO: Implement NewNode(id string, quorumSize int, transport Transport, storage Storage) *Node
//       1. Create storage instance (or use provided)
//       2. Create acceptor with storage
//       3. Create proposer with transport and quorumSize
//       4. Create learner with quorumSize
//       5. Wire them together
//
//
// TODO: Implement Start() error
//       Start the message handling loop:
//       1. Set running = true
//       2. Start goroutine for handleMessages()
//       3. Return immediately (non-blocking)
//
//
// TODO: Implement Stop() error
//       1. Set running = false
//       2. Close stopCh to signal handleMessages to stop
//       3. Wait for goroutine to finish (optional)
//
//
// TODO: Implement handleMessages()
//       Main message loop (runs in goroutine):
//
//       for {
//           select {
//           case <-n.stopCh:
//               return
//           default:
//               msg, err := n.transport.ReceiveTimeout(100 * time.Millisecond)
//               if err == ErrTimeout {
//                   continue  // Check stopCh again
//               }
//               if err != nil {
//                   log.Printf("receive error: %v", err)
//                   continue
//               }
//               n.routeMessage(msg)
//           }
//       }
//
//
// TODO: Implement routeMessage(msg Message)
//       Route message to appropriate handler:
//
//       switch m := msg.(type) {
//       case *Prepare:
//           response := n.acceptor.HandlePrepare(m)
//           n.transport.Send(m.From, response)
//       case *Promise:
//           // Proposer handles this internally during Propose()
//           // May need a channel or callback mechanism
//       case *Accept:
//           response := n.acceptor.HandleAccept(m)
//           n.transport.Send(m.From, response)
//           // Notify our local learner
//           if response.OK {
//               n.learner.HandleAccepted(response)
//           }
//       case *Accepted:
//           n.learner.HandleAccepted(m)
//       case *Learn:
//           n.learner.HandleLearn(m)
//       }
//
//
// TODO: Implement Propose(value []byte) ([]byte, error)
//       Client-facing API to propose a value:
//       1. Call proposer.Propose(value)
//       2. Return the chosen value (might differ from input!)
//
//
// TODO: Implement GetChosenValue() ([]byte, bool)
//       Returns what the learner knows:
//       return n.learner.GetChosenValue()
//
// =============================================================================
// CONCURRENCY MODEL
// =============================================================================
//
// The node needs to handle:
// 1. Incoming messages (from other nodes)
// 2. Client requests (Propose calls)
// 3. Internal state updates
//
// Options:
//
// OPTION A: Single-threaded with message queue
//   - All operations go through a single goroutine
//   - No locks needed (all access is serialized)
//   - Simple but potentially slow
//
// OPTION B: Multi-threaded with locks
//   - Multiple goroutines handle messages
//   - Use mutexes to protect shared state
//   - Better throughput, more complex
//
// OPTION C: Actor model
//   - Each role runs in its own goroutine
//   - Communication via channels
//   - Clean separation, good for learning
//
// For learning: Start with OPTION A or C. Add complexity as needed.
//
// =============================================================================
// ERROR HANDLING
// =============================================================================
//
// What if things go wrong?
//
// 1. Storage error: Crash the node (safety-critical)
// 2. Transport error: Log and continue (network is unreliable anyway)
// 3. Invalid message: Log and ignore (don't crash on bad input)
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: The node correctly routes each message type to its handler
//            and ensures responses are sent to the original sender.
//
// Misrouting or dropping responses breaks the protocol.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Forgetting to notify learners of local accepts
//
// When OUR acceptor accepts a value, our learner should also know:
//
// WRONG:
//   response := n.acceptor.HandleAccept(m)
//   n.transport.Send(m.From, response)  // Only sends to proposer
//
// RIGHT:
//   response := n.acceptor.HandleAccept(m)
//   n.transport.Send(m.From, response)
//   n.learner.HandleAccepted(response)  // ALSO notify our learner!
//
// Otherwise, our learner might miss accepts and never learn the value.
//
// =============================================================================
// CLIENT INTERACTION
// =============================================================================
//
// How do clients interact with the Paxos cluster?
//
// OPTION 1: Any node can handle proposals
//   - Client sends request to any node
//   - That node's proposer runs the protocol
//   - Returns result to client
//   - Pro: Simple client, automatic load balancing
//   - Con: Multiple concurrent proposers might compete
//
// OPTION 2: Leader-based (Multi-Paxos)
//   - Clients always talk to the "leader"
//   - Leader handles all proposals
//   - Other nodes forward requests to leader
//   - Pro: No dueling proposers, better performance
//   - Con: Need leader election
//
// For Single-Decree Paxos: Use OPTION 1.
// For Multi-Paxos: Use OPTION 2.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// For Multi-Paxos, the node gains:
//
// 1. Slot management: Track which slots are filled, which are pending
// 2. Log: Ordered sequence of chosen values
// 3. Leader election: Determine who is the active proposer
// 4. Lease renewal: Keep leadership without repeating Phase 1
//
// The node structure expands:
//
//   type Node struct {
//       // ... existing fields ...
//       log        [][]byte           // Chosen values by slot
//       pendingSlots map[int64]*SlotState
//       isLeader   bool
//       leaderID   string
//   }
//
// =============================================================================

package node

import (
	"log"
	"sync"
	"time"

	"quorum/internal/paxos"
	"quorum/internal/storage"
	"quorum/internal/transport"
)

type Node struct {
	id         string
	proposer   *paxos.Proposer
	acceptor   *paxos.Acceptor
	learner    *paxos.Learner
	transport  transport.Transport
	storage    storage.Storage
	quorumSize int
	mu         sync.Mutex
	running    bool
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func NewNode(id string, quorumSize int, t transport.Transport, s storage.Storage) *Node {
	acceptor := paxos.NewAcceptor(id, s)
	learner := paxos.NewLearner(id, quorumSize)
	proposerTransport := &proposerTransportAdapter{transport: t}
	proposer := paxos.NewProposer(id, quorumSize, proposerTransport)
	return &Node{
		id:         id,
		proposer:   proposer,
		acceptor:   acceptor,
		learner:    learner,
		transport:  t,
		storage:    s,
		quorumSize: quorumSize,
		stopCh:     make(chan struct{}),
	}
}

func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.running {
		return nil
	}
	n.running = true
	n.stopCh = make(chan struct{})
	n.wg.Add(1)
	go n.handleMessages()
	return nil
}

func (n *Node) Stop() error {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return nil
	}
	n.running = false
	close(n.stopCh)
	n.mu.Unlock()
	n.wg.Wait()
	return nil
}

func (n *Node) handleMessages() {
	defer n.wg.Done()
	for {
		select {
		case <-n.stopCh:
			return
		default:
			msg, err := n.transport.ReceiveTimeout(100 * time.Millisecond)
			if err == transport.ErrTimeout {
				continue
			}
			if err != nil {
				log.Printf("[%s] receive error: %v", n.id, err)
				continue
			}
			n.routeMessage(msg)
		}
	}
}

func (n *Node) routeMessage(msg transport.Message) {
	switch m := msg.(type) {
	case paxos.Prepare:
		response := n.acceptor.HandlePrepare(m)
		n.transport.Send(m.From, response)

	case *paxos.Prepare:
		response := n.acceptor.HandlePrepare(*m)
		n.transport.Send(m.From, response)
	case paxos.Accept:
		response := n.acceptor.HandleAccept(m)
		n.transport.Send(m.From, response)
		if response.OK {
			n.learner.HandleAccepted(response)
		}
	case *paxos.Accept:
		response := n.acceptor.HandleAccept(*m)
		n.transport.Send(m.From, response)
		if response.OK {
			n.learner.HandleAccepted(response)
		}
	case paxos.Accepted:
		n.learner.HandleAccepted(m)

	case *paxos.Accepted:
		n.learner.HandleAccepted(*m)

	case paxos.Learn:
		n.learner.HandleLearn(m)

	case *paxos.Learn:
		n.learner.HandleLearn(*m)
	default:
		log.Printf("[%s] unknown message type: %T", n.id, msg)
	}
}

func (n *Node) Propose(value []byte) ([]byte, error) {
	return n.proposer.Propose(value)
}

func (n *Node) GetChosenValue() ([]byte, bool) {
	return n.learner.GetChosenValue()
}

func (n *Node) ID() string {
	return n.id
}

type proposerTransportAdapter struct {
	transport transport.Transport
}

func (a *proposerTransportAdapter) Broadcast(msg interface{}) error {
	if m, ok := msg.(transport.Message); ok {
		return a.transport.Broadcast(m)
	}
	return a.transport.Broadcast(&messageWrapper{msg: msg})
}

func (a *proposerTransportAdapter) Receive() (interface{}, error) {
	msg, err := a.transport.Receive()
	if err != nil {
		return nil, err
	}
	return msg, nil
}

type messageWrapper struct {
	msg  interface{}
	from string
}

func (w *messageWrapper) GetFrom() string {
	return w.from
}
