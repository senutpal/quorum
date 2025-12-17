// =============================================================================
// STORAGE INTERFACE - Abstraction for Durable State
// =============================================================================
//
// IMPLEMENTATION ORDER: File 1 of 2 in internal/storage/
// Implement this AFTER all internal/paxos/ files
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// This file defines an abstract interface for storing Paxos state durably.
// Acceptors MUST persist their state to survive crashes. This interface
// allows different storage backends:
//
// - In-memory (for testing and demos)
// - File-based (simple production)
// - Database (PostgreSQL, SQLite, etc.)
// - Distributed storage (etcd, Consul, etc.)
//
// By coding to an interface, we can swap implementations without changing
// Paxos logic.
//
// =============================================================================
// WHY DURABILITY MATTERS
// =============================================================================
//
// Recall the acceptor rules:
// 1. Once you promise N, reject all proposals < N
// 2. Once you accept (N, V), report it in future Promises
//
// If an acceptor crashes and forgets:
// - It might re-promise a lower number → Safety violation
// - It might not report prior accepts → Proposer might overwrite chosen value
//
// DURABILITY IS REQUIRED FOR SAFETY (not just liveness).
//
// In practice:
// - Demo/testing: In-memory is fine (accept that crashes lose safety)
// - Production: MUST use durable storage with fsync
//
// =============================================================================
// WHAT NEEDS TO BE STORED
// =============================================================================
//
// For Single-Decree Paxos, the acceptor needs to persist:
//
// 1. HighestPromised (ProposalNumber)
//    - The highest proposal number we promised not to accept below
//
// 2. AcceptedProposal (ProposalNumber)
//    - The proposal number of the value we accepted
//
// 3. AcceptedValue ([]byte)
//    - The value we accepted
//
// These can be stored as:
// - Three separate keys
// - One serialized struct
// - Any format that survives crashes
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define Storage interface
//       Methods:
//
//       - SavePromised(proposal ProposalNumber) error
//         // Durably store the highest promised proposal number
//         // MUST be fsync'd before returning (in production)
//
//       - LoadPromised() (ProposalNumber, error)
//         // Load the highest promised proposal number
//         // Returns zero value if never set
//
//       - SaveAccepted(proposal ProposalNumber, value []byte) error
//         // Durably store the accepted proposal and value
//         // MUST be fsync'd before returning (in production)
//
//       - LoadAccepted() (ProposalNumber, []byte, error)
//         // Load the accepted proposal and value
//         // Returns zero/nil if never accepted
//
//       - Close() error
//         // Clean up resources
//
//
// TODO: Consider a combined approach
//       Alternative design with a single state struct:
//
//       type AcceptorState struct {
//           HighestPromised  ProposalNumber
//           AcceptedProposal ProposalNumber
//           AcceptedValue    []byte
//       }
//
//       type Storage interface {
//           Save(state AcceptorState) error
//           Load() (AcceptorState, error)
//           Close() error
//       }
//
//       Pro: Atomic save of all state
//       Con: More data written on each update
//
//
// TODO: Consider error handling
//       What if storage fails? Options:
//       - Return error to caller (let them retry)
//       - Panic (crash the node, which is safe for Paxos)
//       - Log and continue (DANGEROUS - might violate safety!)
//
//       Recommendation: Crash on storage errors. A node that can't persist
//       should not participate in Paxos.
//
// =============================================================================
// FSYNC REQUIREMENT
// =============================================================================
//
// In production storage implementations:
//
// WRONG:
//   file.Write(data)
//   return nil  // Data might be in OS buffer, not on disk!
//
// RIGHT:
//   file.Write(data)
//   file.Sync()  // Force data to disk
//   return nil
//
// Without Sync(), a system crash could lose data that we thought was saved.
// Paxos safety depends on this.
//
// For learning, you can ignore this. Document it for future production use.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Storage implementations MUST guarantee that after Save returns
//            successfully, the data survives system crashes.
//
// This is the "durability" guarantee. Without it, Paxos safety fails.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Responding before persistence completes
//
// Sequence:
// 1. Acceptor receives Accept(5, X)
// 2. Acceptor calls storage.Save() - starts async write
// 3. Acceptor sends Accepted(5, X) response - BEFORE save completes
// 4. System crashes
// 5. Acceptor restarts with no record of accepting (5, X)
// 6. Safety violation possible
//
// FIX: ALWAYS wait for storage.Save() to complete AND confirm before
// sending any response to the caller.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// In Multi-Paxos, storage needs to handle multiple slots:
//
//   type Storage interface {
//       SaveSlot(slot int64, state SlotState) error
//       LoadSlot(slot int64) (SlotState, error)
//       GetHighestSlot() (int64, error)  // For recovery
//       // ... etc
//   }
//
// You might also want "SavePromised" to apply globally (leader lease),
// not per slot.
//
// =============================================================================

package storage
