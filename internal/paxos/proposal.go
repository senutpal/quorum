// =============================================================================
// PROPOSAL NUMBERS - The Foundation of Paxos Ordering
// =============================================================================
//
// IMPLEMENTATION ORDER: File 1 of 7 in internal/paxos/
// Implement this FIRST before any other Paxos file.
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// A proposal number is a unique identifier that totally orders all proposals
// across all proposers in the system. It's the foundation upon which Paxos
// builds its safety guarantees.
//
// Think of it like a "priority ticket" - higher numbers win, and no two
// proposers can ever have the same number.
//
// =============================================================================
// WHY PROPOSAL NUMBERS EXIST
// =============================================================================
//
// Problem: Multiple proposers might try to propose different values at the
// same time. How do we decide which one "wins"?
//
// Solution: Each proposal carries a unique number. Acceptors always prefer
// higher numbers. This creates a total ordering of proposals.
//
// Key insight: Proposal numbers are NOT sequence numbers of values.
// They're competition markers - "my proposal is newer/higher priority than
// yours".
//
// =============================================================================
// STRUCTURE REQUIREMENTS
// =============================================================================
//
// A proposal number typically has two parts:
//
// 1. Round/Sequence Number (the "counter"):
//    - Incremented each time this proposer starts a new proposal
//    - Example: 1, 2, 3, 4, ...
//
// 2. Proposer ID (the "tiebreaker"):
//    - A unique identifier for this proposer
//    - Could be: IP:Port, UUID, or any unique string
//    - Ensures no two proposers generate the same proposal number
//
// Comparison rules:
//    - First compare round number (higher wins)
//    - If equal, compare proposer ID (lexicographically or numerically)
//
// Example ordering (ascending):
//    (1, "node-a") < (1, "node-b") < (2, "node-a") < (3, "node-a")
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define ProposalNumber struct
//       Fields:
//       - Round int64      // Monotonically increasing counter
//       - ProposerID string // Unique ID of the proposer who created this
//
// TODO: Implement comparison methods
//       - LessThan(other ProposalNumber) bool
//       - GreaterThan(other ProposalNumber) bool
//       - Equal(other ProposalNumber) bool
//
// TODO: Implement IsZero() method
//       - Returns true if this is the "zero value" proposal number
//       - Zero proposal number is less than all real proposal numbers
//
// TODO: Implement String() method
//       - For debugging: "(round=5, proposer=node-a)"
//
// TODO: Consider implementing a constructor
//       - NewProposalNumber(round int64, proposerID string) ProposalNumber
//
// =============================================================================
// USAGE IN PAXOS
// =============================================================================
//
// Phase 1 (Prepare):
//   Proposer sends: "I want to propose with number N"
//   Acceptor checks: Is N > my highest promised number?
//
// Phase 2 (Accept):
//   Proposer sends: "Accept value V at proposal number N"
//   Acceptor checks: Is N >= my highest promised number?
//
// The proposal number is used everywhere - it's how we know which
// proposal "wins" when there are conflicts.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Proposal numbers are GLOBALLY UNIQUE
//
// No two proposal numbers anywhere in the system may be identical.
// This is achieved by combining a local counter with a globally unique
// proposer ID.
//
// Why: If two proposers use the same proposal number, acceptors can't
// distinguish between them, and both might get accepted with different
// values, breaking Paxos safety.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Using only a counter without proposer ID
//
// If Proposer-A uses proposal number 5, and Proposer-B also uses 5,
// and they propose different values, some acceptors might accept A's value
// while others accept B's value - BOTH at "proposal 5".
//
// The system would think the same proposal has two different values.
// This violates safety.
//
// FIX: Always include a unique proposer ID as the second component.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// In Single-Decree Paxos, we just have one consensus instance.
// In Multi-Paxos, we have many slots/instances, each choosing one value.
//
// The proposal number structure stays the same - it's about ordering
// proposals within a single slot. The "slot number" is a separate concept
// added to messages, not to proposal numbers.
//
// =============================================================================

package paxos
