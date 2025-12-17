// =============================================================================
// LEARNER - The Observer of Paxos Consensus
// =============================================================================
//
// IMPLEMENTATION ORDER: File 5 of 7 in internal/paxos/
// Implement this AFTER proposer.go
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// The Learner discovers which value has been chosen. Unlike proposers and
// acceptors who participate in choosing a value, learners simply observe
// what was chosen.
//
// Why separate this role?
//
// 1. NOT ALL NODES NEED TO BE ACCEPTORS
//    - Acceptors must be careful about durability and promises
//    - Some nodes just want to know the result, not vote on it
//
// 2. CLIENTS ARE LEARNERS
//    - When a client submits a request, it ultimately wants to learn
//      whether its value was chosen (or what value was chosen instead)
//
// 3. REPLICAS ARE LEARNERS
//    - In a replicated state machine, each replica learns commands
//      and applies them to its state
//
// =============================================================================
// HOW A LEARNER LEARNS
// =============================================================================
//
// A value is "chosen" when a majority of acceptors have accepted it.
// There are several ways a learner can find out:
//
// OPTION 1: ACCEPTORS NOTIFY LEARNERS
//   - When an acceptor accepts a value, it sends Accepted to all learners
//   - Learner counts: if same (proposal, value) from majority → CHOSEN
//   - Pro: Learners find out quickly
//   - Con: N acceptors × M learners = O(N×M) messages
//
// OPTION 2: PROPOSER NOTIFIES LEARNERS
//   - Proposer gets Accepted from majority, then sends Learn message
//   - Pro: Only O(M) messages from proposer to learners
//   - Con: If proposer crashes after getting accepts but before notifying,
//          learners don't find out
//
// OPTION 3: DISTINGUISHED LEARNER
//   - Only one "main" learner listens to acceptors
//   - That learner then broadcasts to others
//   - Pro: O(N) + O(M) instead of O(N×M)
//   - Con: Single point of failure
//
// FOR THIS IMPLEMENTATION: Use Option 1 for learning, since it's most
// straightforward to implement.
//
// =============================================================================
// LEARNER STATE
// =============================================================================
//
// The learner tracks accepted messages to detect when a majority exists.
//
// Key state:
//
// 1. acceptedCounts: Map from (ProposalNumber, Value) → Set of acceptor IDs
//    - When size of set >= quorum, value is chosen
//
// 2. chosenValue: The value that has been chosen (nil if not yet chosen)
//
// 3. quorumSize: How many acceptors form a majority
//
// Note: Learner state doesn't need to be durable. If a learner restarts,
// it can ask acceptors what they've accepted, or wait for new messages.
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define AcceptedRecord struct (for tracking)
//       Fields:
//       - proposalNumber ProposalNumber
//       - value []byte
//       Key: This is what we count - (proposal, value) pairs
//
//
// TODO: Define Learner struct
//       Fields:
//       - id string
//         // Identifier for this learner
//       - quorumSize int
//         // Number of acceptors that form a majority
//       - accepted map[AcceptedKey]map[string]bool
//         // AcceptedKey → set of acceptor IDs who sent this
//         // AcceptedKey = (ProposalNumber, Value hash) or similar
//       - chosenValue []byte
//         // The chosen value, once we detect it
//       - chosenProposal ProposalNumber
//         // The proposal number of the chosen value
//       - isChosen bool
//         // True once we've detected a chosen value
//       - mu sync.Mutex
//         // Protect concurrent access
//       - chosenChan chan []byte
//         // Optional: channel to notify when value is chosen
//
//
// TODO: Implement NewLearner(id string, quorumSize int) *Learner
//
//
// TODO: Implement HandleAccepted(msg Accepted)
//       Algorithm:
//       1. Lock the mutex
//       2. If we already have a chosen value, ignore (or verify consistency)
//       3. Create key from (msg.ProposalNumber, msg.Value)
//       4. Add msg.From to the set for this key
//       5. If set size >= quorumSize:
//          a. Set chosenValue = msg.Value
//          b. Set chosenProposal = msg.ProposalNumber
//          c. Set isChosen = true
//          d. If using channel: send msg.Value to chosenChan
//
//       NOTE: The same acceptor might send multiple Accepted messages
//       (e.g., for different proposals). Only count each acceptor once
//       per (proposal, value) pair.
//
//
// TODO: Implement HandleLearn(msg Learn)
//       Algorithm:
//       1. Lock the mutex
//       2. If isChosen: verify or ignore
//       3. Set chosenValue = msg.Value
//       4. Set isChosen = true
//
//       This is used when the proposer directly tells us the result.
//       We trust the proposer (in a real system, you might verify).
//
//
// TODO: Implement GetChosenValue() ([]byte, bool)
//       Returns (chosenValue, isChosen)
//
//
// TODO: Implement WaitForChosen() []byte
//       Blocks until a value is chosen, then returns it.
//       Implementation options:
//       - Busy poll GetChosenValue() (simple but wasteful)
//       - Wait on chosenChan (better)
//       - Use sync.Cond (also good)
//
// =============================================================================
// DISTINGUISHING ACCEPTED MESSAGES
// =============================================================================
//
// The learner needs to group Accepted messages by (proposal, value).
// But how do we make a map key from these?
//
// Option 1: Stringify
//   key := fmt.Sprintf("%d-%s-%x", prop.Round, prop.ProposerID, value)
//   Pro: Simple
//   Con: String allocation, slow
//
// Option 2: Struct key (if value is small)
//   type AcceptedKey struct {
//       Round      int64
//       ProposerID string
//       Value      string  // Only works if value is string-like
//   }
//   Pro: Efficient, type-safe
//   Con: Only works for fixed-size or string values
//
// Option 3: Hash the value
//   type AcceptedKey struct {
//       Round      int64
//       ProposerID string
//       ValueHash  [32]byte  // SHA256 of value
//   }
//   Pro: Works for any value
//   Con: Hash collision (extremely unlikely but possible)
//
// For learning: Start with Option 1 (stringify). Optimize later.
//
// =============================================================================
// CONSISTENCY CHECK
// =============================================================================
//
// Once a value is chosen, the learner might still receive Accepted messages.
// These should all have the same (or higher) proposal number with the same
// value.
//
// If we receive Accepted with a DIFFERENT value after marking chosen,
// something is wrong:
// - Either our quorum counting is buggy
// - Or safety was violated elsewhere
//
// For debugging: Log a warning if this happens. In production, this should
// trigger an alert.
//
// =============================================================================
// FAILURE SCENARIO: WHAT BREAKS IF LEARNER IS WRONG
// =============================================================================
//
// Unlike proposers and acceptors, learner bugs don't break safety!
// They break liveness:
//
// - If learner miscounts and thinks a value is chosen when it isn't:
//   Client gets wrong answer, but the real Paxos state is fine.
//   Eventually, the real chosen value will be discovered.
//
// - If learner fails to recognize a chosen value:
//   Client blocks waiting. Liveness is violated.
//
// Safety is maintained by acceptors. Learners just observe.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: The learner only reports a value as chosen when it has received
//            Accepted messages from a quorum of acceptors for the same
//            (ProposalNumber, Value) pair.
//
// Premature reporting leads to clients seeing inconsistent results.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Counting acceptors, not (proposal, value) pairs
//
// Scenario:
// - Acceptor A: Accepted (5, X)
// - Acceptor B: Accepted (7, Y)
// - Acceptor C: Accepted (5, X)
//
// Wrong: "3 acceptors accepted something, quorum reached!"
// Right: Only 2 acceptors (A, C) accepted (5, X). Not a quorum.
//        Only 1 acceptor (B) accepted (7, Y). Not a quorum.
//        No value is chosen yet.
//
// Always group by (proposal, value) before counting.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// In Multi-Paxos, the learner tracks chosen values for each SLOT:
//
//   type Learner struct {
//       slots map[int64]*SlotLearner  // Learner state per slot
//   }
//
// Additionally, learners often maintain a "log" of chosen values:
//   log [][]byte  // log[i] = chosen value for slot i
//
// Gaps in the log indicate slots where the value isn't known yet.
// The learner can query those slots specifically.
//
// =============================================================================

package paxos
