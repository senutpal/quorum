// =============================================================================
// DEMO RUNNER - Single-Decree Paxos in Action
// =============================================================================
//
// IMPLEMENTATION ORDER: File 1 of 1 in cmd/demo/
// Implement this LAST, after all other files
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// This is the entry point for running a Paxos demonstration.
// It creates multiple Paxos nodes in a single process, simulates
// distributed consensus, and shows the protocol in action.
//
// Run with: go run ./cmd/demo
//
// =============================================================================
// DEMO SCENARIO
// =============================================================================
//
// The demo will:
//
// 1. Create a cluster of N nodes (e.g., 5 nodes)
// 2. Set up in-memory transport and storage
// 3. Have one node propose a value
// 4. Watch the protocol execute
// 5. Verify all nodes agree on the chosen value
//
//                     ┌─────────┐
//                     │ Client  │
//                     └────┬────┘
//                          │ Propose("hello")
//                          ▼
//   ┌─────────┬─────────┬─────────┬─────────┬─────────┐
//   │ Node 0  │ Node 1  │ Node 2  │ Node 3  │ Node 4  │
//   │ (prop)  │ (acc)   │ (acc)   │ (acc)   │ (acc)   │
//   └────┬────┴────┬────┴────┬────┴────┬────┴────┬────┘
//        │         │         │         │         │
//        └─────────┴─────────┴─────────┴─────────┘
//                   All learn "hello"
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Implement main() function
//       High-level structure:
//
//       func main() {
//           // 1. Configuration
//           numNodes := 5
//           quorumSize := (numNodes / 2) + 1  // Majority
//
//           // 2. Create shared network (in-memory transport hub)
//           network := transport.NewNetwork()
//
//           // 3. Create nodes
//           nodes := make([]*node.Node, numNodes)
//           for i := 0; i < numNodes; i++ {
//               id := fmt.Sprintf("node-%d", i)
//               storage := storage.NewMemoryStorage()
//               trans := network.AddNode(id)
//               nodes[i] = node.NewNode(id, quorumSize, trans, storage)
//           }
//
//           // 4. Start all nodes
//           for _, n := range nodes {
//               n.Start()
//           }
//
//           // 5. Have node 0 propose a value
//           value := []byte("hello, paxos!")
//           chosenValue, err := nodes[0].Propose(value)
//           if err != nil {
//               log.Fatalf("propose failed: %v", err)
//           }
//
//           // 6. Verify all learners agree
//           fmt.Printf("Chosen value: %s\n", chosenValue)
//           for i, n := range nodes {
//               v, ok := n.GetChosenValue()
//               fmt.Printf("Node %d learned: %s (ok=%v)\n", i, v, ok)
//           }
//
//           // 7. Stop all nodes
//           for _, n := range nodes {
//               n.Stop()
//           }
//       }
//
//
// TODO: Add logging to trace protocol execution
//       Add Printf statements in handlers to see:
//       - Prepare sent/received
//       - Promise sent/received
//       - Accept sent/received
//       - Accepted sent/received
//       - Value chosen
//
//       Example output:
//       [node-0] Proposing with proposal (1, node-0)
//       [node-0] → Prepare to all nodes
//       [node-1] ← Prepare from node-0, promising
//       [node-1] → Promise to node-0
//       ...
//
//
// TODO: Add demo for competing proposers
//       After basic demo works, add a test:
//
//       // Two nodes propose at the same time
//       go func() { nodes[0].Propose([]byte("value A")) }()
//       go func() { nodes[1].Propose([]byte("value B")) }()
//
//       // Only ONE value should be chosen
//       // (might be A or B, but same across all nodes)
//
//
// TODO: Add demo for crash/restart
//       Test durability:
//
//       // 1. Propose and get consensus
//       nodes[0].Propose([]byte("important"))
//
//       // 2. "Crash" a node
//       nodes[2].Stop()
//
//       // 3. "Restart" with new storage (simulating crash)
//       storage2 := storage.NewMemoryStorage()  // Lost all data!
//       trans2 := network.AddNode("node-2")
//       nodes[2] = node.NewNode("node-2", quorumSize, trans2, storage2)
//       nodes[2].Start()
//
//       // 4. New proposal should still work
//       // (but note: with in-memory storage, node-2 forgot its promises!)
//
// =============================================================================
// EXPECTED OUTPUT
// =============================================================================
//
// When you run the demo, you should see something like:
//
//   $ go run ./cmd/demo
//   Starting Paxos cluster with 5 nodes...
//   Quorum size: 3
//
//   Node-0 proposing: "hello, paxos!"
//
//   [PREPARE] node-0 → all (proposal: 1-node-0)
//   [PROMISE] node-1 → node-0 (ok=true, no prior accept)
//   [PROMISE] node-2 → node-0 (ok=true, no prior accept)
//   [PROMISE] node-3 → node-0 (ok=true, no prior accept)
//   [PROMISE] node-4 → node-0 (ok=true, no prior accept)
//
//   Received 4 promises (need 3), proceeding to Phase 2
//
//   [ACCEPT] node-0 → all (proposal: 1-node-0, value: "hello, paxos!")
//   [ACCEPTED] node-1 → node-0 (ok=true)
//   [ACCEPTED] node-2 → node-0 (ok=true)
//   [ACCEPTED] node-3 → node-0 (ok=true)
//
//   Received 3 accepts - VALUE CHOSEN!
//
//   Final state:
//   Node-0: learned "hello, paxos!" ✓
//   Node-1: learned "hello, paxos!" ✓
//   Node-2: learned "hello, paxos!" ✓
//   Node-3: learned "hello, paxos!" ✓
//   Node-4: learned "hello, paxos!" ✓
//
//   Consensus achieved! All nodes agree.
//
// =============================================================================
// TROUBLESHOOTING
// =============================================================================
//
// If the demo hangs:
// - Check that message routing is correct
// - Ensure all nodes are started
// - Check for deadlocks in channels
//
// If nodes disagree:
// - Check that proposer adopts prior accepted values
// - Check that acceptors keep promises
// - Check learner quorum counting
//
// If nothing happens:
// - Ensure messages are being sent (add logging)
// - Check transport is connected
//
// =============================================================================
// LEARNING EXERCISES
// =============================================================================
//
// After getting the basic demo working, try these:
//
// 1. COMPETING PROPOSERS
//    Have two nodes propose simultaneously.
//    Observe: Only one value is chosen.
//
// 2. MESSAGE LOSS
//    Add random message dropping to transport.
//    Observe: Protocol might stall but never chooses two values.
//
// 3. MINORITY FAILURE
//    Stop 2 out of 5 nodes mid-protocol.
//    Observe: Protocol still completes with 3 nodes.
//
// 4. MAJORITY FAILURE
//    Stop 3 out of 5 nodes.
//    Observe: Protocol cannot complete (no quorum).
//
// 5. PERFORMANCE
//    Measure how long consensus takes.
//    Observe: Two network round-trips minimum.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: The demo must verify that ALL nodes learn the SAME value.
//
// If nodes disagree, either the demo is wrong or Paxos is broken!
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Not waiting for protocol to complete
//
// WRONG:
//   go nodes[0].Propose(value)  // Fire and forget
//   for _, n := range nodes {
//       v, _ := n.GetChosenValue()  // Might be empty!
//       fmt.Printf("Learned: %s\n", v)
//   }
//
// RIGHT:
//   value, _ := nodes[0].Propose(value)  // Wait for result
//   time.Sleep(100 * time.Millisecond)   // Give learners time
//   for _, n := range nodes {
//       v, _ := n.GetChosenValue()
//       fmt.Printf("Learned: %s\n", v)
//   }
//
// Or better: Use WaitForChosen() on each learner.
//
// =============================================================================
// MULTI-PAXOS EXTENSION
// =============================================================================
//
// For Multi-Paxos, the demo would show:
//
// 1. Multiple values proposed in sequence
// 2. Values added to a replicated log
// 3. All nodes have the same log
//
//   nodes[0].Propose([]byte("cmd1"))  // Slot 0
//   nodes[0].Propose([]byte("cmd2"))  // Slot 1
//   nodes[0].Propose([]byte("cmd3"))  // Slot 2
//
//   for _, n := range nodes {
//       log := n.GetLog()
//       // All logs should be: ["cmd1", "cmd2", "cmd3"]
//   }
//
// =============================================================================

package main

// Imports needed:
// import (
//     "fmt"
//     "log"
//     "time"
//     "github.com/quorum/paxos/internal/node"
//     "github.com/quorum/paxos/internal/storage"
//     "github.com/quorum/paxos/internal/transport"
// )

// TODO: Implement the main function following the demo scenario above.
// This stub exists only to satisfy Go's requirement that package main
// must have a main function.
func main() {
	// Implementation goes here - see the TODO section above for the full algorithm.
}
