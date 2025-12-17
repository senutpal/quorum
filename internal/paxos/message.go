// =============================================================================
// PAXOS MESSAGE TYPES
// =============================================================================
//
// IMPLEMENTATION ORDER: File 2 of 7 in internal/paxos/
// Implement this AFTER proposal.go
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// This file defines all the messages that flow between Paxos nodes.
// These are the "packets" of the Paxos protocol.
//
// Paxos is fundamentally a message-passing protocol. Understanding the
// messages is understanding the protocol itself.
//
// =============================================================================
// THE FOUR MESSAGE TYPES (Two Phases, Two Directions Each)
// =============================================================================
//
// PHASE 1: PREPARE PHASE (Establishes leadership/priority)
// ─────────────────────────────────────────────────────────
//
// ┌──────────────┐   Prepare(N)    ┌──────────────┐
// │   PROPOSER   │ ───────────────▶│   ACCEPTOR   │
// │              │                 │              │
// │              │◀─────────────── │              │
// └──────────────┘   Promise(N)    └──────────────┘
//
// Prepare: "I want to propose with number N"
// Promise: "OK, I won't accept anything lower than N. Here's what I've
//           already accepted (if anything)."
//
//
// PHASE 2: ACCEPT PHASE (Actually proposes a value)
// ──────────────────────────────────────────────────
//
// ┌──────────────┐  Accept(N, V)   ┌──────────────┐
// │   PROPOSER   │ ───────────────▶│   ACCEPTOR   │
// │              │                 │              │
// │              │◀─────────────── │              │
// └──────────────┘   Accepted(N)   └──────────────┘
//
// Accept: "Please accept value V at proposal number N"
// Accepted: "I have accepted (N, V)"
//
//
// PHASE 3: LEARNING (Optional but necessary)
// ───────────────────────────────────────────
//
// When an acceptor accepts a value, it can notify learners.
// Alternatively, the proposer can notify learners after getting a majority.
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define Prepare message
//       Fields:
//       - ProposalNumber  // The proposal number the proposer wants to use
//       - From string     // ID of the proposer sending this
//
//
// TODO: Define Promise message (response to Prepare)
//       Fields:
//       - ProposalNumber         // The proposal number being promised
//       - AcceptedProposal       // Highest proposal number previously accepted
//                                // (zero value if nothing accepted)
//       - AcceptedValue          // Value that was accepted at AcceptedProposal
//                                // (nil/empty if nothing accepted)
//       - From string            // ID of the acceptor sending this
//       - OK bool                // true if promise granted, false if rejected
//
//       WHY AcceptedProposal/AcceptedValue:
//       This is CRITICAL for safety! The proposer MUST adopt the value
//       from the highest-numbered proposal among all promises received.
//       This prevents overwriting a potentially-chosen value.
//
//
// TODO: Define Reject message (alternative response to Prepare)
//       Fields:
//       - ProposalNumber         // The proposal number being rejected
//       - HighestSeen            // The highest proposal number this acceptor
//                                // has seen (helps proposer pick a higher one)
//       - From string            // ID of the acceptor sending this
//
//       Design choice: You could merge this with Promise using an OK field,
//       or keep them separate. Both approaches are valid.
//
//
// TODO: Define Accept message
//       Fields:
//       - ProposalNumber  // The proposal number
//       - Value           // The value to accept (could be []byte or interface{})
//       - From string     // ID of the proposer sending this
//
//
// TODO: Define Accepted message (response to Accept)
//       Fields:
//       - ProposalNumber  // The proposal number that was accepted
//       - Value           // The value that was accepted
//       - From string     // ID of the acceptor sending this
//       - OK bool         // true if accepted, false if rejected
//
//
// TODO: Define Learn message (notification to learners)
//       Fields:
//       - ProposalNumber  // The proposal number of the chosen value
//       - Value           // The chosen value
//       - From string     // ID of the sender (proposer or acceptor)
//
// =============================================================================
// VALUE TYPE DECISION
// =============================================================================
//
// The "Value" in Paxos is opaque to the protocol - Paxos doesn't care what
// it is. Common choices:
//
// Option A: []byte
//   Pros: Simple, can encode anything
//   Cons: Caller must serialize/deserialize
//
// Option B: interface{}
//   Pros: Flexible, can hold any Go type
//   Cons: Type assertions needed, harder to serialize for network
//
// Option C: Define a Value interface
//   type Value interface {
//       Bytes() []byte
//   }
//   Pros: Clean abstraction
//   Cons: More boilerplate
//
// Recommendation for learning: Start with []byte. It's simple and makes
// serialization obvious.
//
// =============================================================================
// MESSAGE ROUTING
// =============================================================================
//
// Each message needs to know:
// 1. Where it came from (From field)
// 2. Where it's going (typically handled by transport layer)
//
// The transport layer (defined elsewhere) handles actual delivery.
// This file just defines the message structure.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Promise messages MUST include previously accepted proposals
//
// When an acceptor sends a Promise, it MUST include:
// - The highest-numbered proposal it has accepted
// - The value from that proposal
//
// If this information is missing or incorrect, the proposer might propose
// a new value that conflicts with a potentially-chosen value, violating
// safety.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Forgetting to return accepted value in Promise
//
// Scenario:
// 1. Proposer A proposes value X at proposal 5, gets majority accepts
// 2. Proposer B starts proposal 7, sends Prepare
// 3. Acceptors promise but DON'T tell B about "accepted X at 5"
// 4. B proposes its own value Y at proposal 7
// 5. SAFETY VIOLATION: Both X and Y might appear "chosen"
//
// FIX: Always include (AcceptedProposal, AcceptedValue) in Promise messages.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// For Multi-Paxos, add a SlotNumber/Index field to EVERY message:
//
//   type Prepare struct {
//       Slot           int64          // NEW: Which log slot this is for
//       ProposalNumber ProposalNumber
//       From           string
//   }
//
// Each slot runs an independent Paxos instance. The slot number tells
// nodes which instance this message belongs to.
//
// =============================================================================

package paxos

type Prepare struct {
	ProposalNumber ProposalNumber
	From string
}

func (p Prepare) GetFrom() string { return p.From }

type Promise struct {
	ProposalNumber ProposalNumber
	AcceptedProposal ProposalNumber
	AcceptedValue []byte
	From string
	OK bool
}

func (p Promise) GetFrom() string { return p.From }

type Reject struct {
	ProposalNumber ProposalNumber
	HighestSeen ProposalNumber
	From string
}

func (r Reject) GetFrom() string { return r.From }

type Accept struct {
	ProposalNumber ProposalNumber
	Value []byte
	From string
}

func (a Accept) GetFrom() string { return a.From }

type Accepted struct {
	ProposalNumber ProposalNumber
	Value []byte
	From string
	OK bool
}

func (a Accepted) GetFrom() string { return a.From }

type Learn struct {
	ProposalNumber ProposalNumber
	Value []byte
	From string
}

func (l Learn) GetFrom() string { return l.From }
