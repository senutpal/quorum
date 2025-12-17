// =============================================================================
// ACCEPTOR - The Safety Guardian of Paxos
// =============================================================================
//
// IMPLEMENTATION ORDER: File 3 of 7 in internal/paxos/
// Implement this AFTER proposal.go and message.go
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// The Acceptor is the most critical role in Paxos. Acceptors are the
// "voters" that accept or reject proposals. They provide:
//
// 1. DURABILITY: Acceptors store accepted values. Even if others fail,
//    accepted values persist.
//
// 2. SAFETY: By following strict rules about what to accept, acceptors
//    guarantee that only one value can ever be chosen.
//
// The Paxos safety property lives or dies by the Acceptor implementation.
//
// =============================================================================
// THE TWO RULES OF AN ACCEPTOR (Memorize These!)
// =============================================================================
//
// RULE 1: PROMISE RULE
//         Once you promise a proposal number N, you MUST reject any
//         Prepare or Accept with a number less than N.
//
// RULE 2: ACCEPTANCE RULE
//         Accept a value only if you haven't promised a higher number.
//         When you accept, remember both the proposal number AND the value.
//
// These two rules, when followed by a majority of acceptors, guarantee
// that at most one value is ever chosen.
//
// =============================================================================
// ACCEPTOR STATE
// =============================================================================
//
// An acceptor maintains exactly two pieces of persistent state:
//
// 1. HighestPromised: The highest proposal number this acceptor has promised.
//    - Updated when: We receive a valid Prepare with a higher number
//    - Used for: Rejecting lower-numbered Prepare and Accept requests
//    - Initial value: Zero (no promises made yet)
//
// 2. AcceptedProposal + AcceptedValue: The last proposal we accepted.
//    - Updated when: We receive a valid Accept request
//    - Used for: Returning to proposers in Promise messages
//    - Initial value: Zero/nil (nothing accepted yet)
//
// CRITICAL: This state MUST be durable (survives crashes). If an acceptor
// forgets what it promised or accepted after a crash, safety breaks.
// For now, you can use in-memory storage, but document that production
// needs disk persistence.
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define Acceptor struct
//       Fields:
//       - id string                  // Unique identifier for this acceptor
//       - highestPromised ProposalNumber
//         // The highest proposal number we have promised not to accept below
//       - acceptedProposal ProposalNumber
//         // The proposal number of the last accepted value
//       - acceptedValue []byte
//         // The last value we accepted
//       - storage Storage
//         // Interface for persistent storage (defined in storage package)
//       - mu sync.Mutex
//         // Protect concurrent access to state
//
//
// TODO: Implement NewAcceptor(id string, storage Storage) *Acceptor
//       - Initialize with zero values
//       - Load any persisted state from storage
//
//
// TODO: Implement HandlePrepare(msg Prepare) Promise
//       Algorithm:
//       1. Lock the mutex
//       2. If msg.ProposalNumber > highestPromised:
//          a. Update highestPromised = msg.ProposalNumber
//          b. Persist highestPromised to storage
//          c. Return Promise{
//               OK: true,
//               ProposalNumber: msg.ProposalNumber,
//               AcceptedProposal: acceptedProposal,
//               AcceptedValue: acceptedValue,
//               From: id,
//             }
//       3. Else (msg.ProposalNumber <= highestPromised):
//          a. Return Promise{
//               OK: false,
//               ProposalNumber: msg.ProposalNumber,
//               HighestSeen: highestPromised,
//               From: id,
//             }
//
//       WHY: The proposer needs to know what we've already accepted so it
//       can adopt that value. This prevents overwriting chosen values.
//
//
// TODO: Implement HandleAccept(msg Accept) Accepted
//       Algorithm:
//       1. Lock the mutex
//       2. If msg.ProposalNumber >= highestPromised:
//          a. Update highestPromised = msg.ProposalNumber
//          b. Update acceptedProposal = msg.ProposalNumber
//          c. Update acceptedValue = msg.Value
//          d. Persist all state to storage
//          e. Return Accepted{
//               OK: true,
//               ProposalNumber: msg.ProposalNumber,
//               Value: msg.Value,
//               From: id,
//             }
//       3. Else (msg.ProposalNumber < highestPromised):
//          a. Return Accepted{
//               OK: false,
//               ProposalNumber: msg.ProposalNumber,
//               From: id,
//             }
//
//       WHY: We only accept if we haven't promised a higher number.
//       The >= (not >) is intentional - we accept proposals at the number
//       we promised.
//
//
// TODO: Implement GetState() (ProposalNumber, ProposalNumber, []byte)
//       - Returns (highestPromised, acceptedProposal, acceptedValue)
//       - For debugging and testing
//
// =============================================================================
// THE SUBTLE COMPARISON (>= vs >)
// =============================================================================
//
// In HandlePrepare: We check msg.ProposalNumber > highestPromised
//   - Strictly greater because we only update promises for NEW higher numbers
//
// In HandleAccept: We check msg.ProposalNumber >= highestPromised
//   - "Greater or equal" because if we promised N, we should accept N
//   - This is the whole point of the promise!
//
// =============================================================================
// FAILURE SCENARIO: WHAT BREAKS IF ACCEPTOR IS WRONG
// =============================================================================
//
// If an acceptor breaks its promise (accepts a lower proposal number):
//
// 1. Proposer A sends Prepare(5), gets Promise from majority
// 2. Proposer B sends Prepare(3), gets Promise from majority (BUG!)
// 3. Proposer A sends Accept(5, "X"), gets majority accepts
// 4. Proposer B sends Accept(3, "Y"), gets majority accepts (BUG!)
// 5. DISASTER: Both X and Y are "chosen" - consensus is broken
//
// The whole system relies on acceptors keeping their promises.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Once an acceptor promises proposal number N, it will NEVER
//            accept any proposal with number less than N.
//
// This invariant is the core of Paxos safety. If any acceptor violates this,
// even once, the system can choose multiple values.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Not persisting state before responding
//
// Sequence:
// 1. Acceptor receives Accept(5, "X")
// 2. Acceptor sends Accepted(5, "X") response
// 3. Acceptor crashes BEFORE writing to disk
// 4. Acceptor restarts with no memory of accepting (5, "X")
// 5. New proposer comes in, acceptor happily promises something else
// 6. SAFETY VIOLATION: The "chosen" value X might be forgotten
//
// FIX: ALWAYS persist state to durable storage BEFORE sending the response.
// This is called "write-ahead" or "fsync-before-ack".
//
// For learning, you can skip this (use in-memory storage), but document
// that production requires durable storage with sync writes.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// In Multi-Paxos, the acceptor manages state PER SLOT:
//
//   type Acceptor struct {
//       slots map[int64]*SlotState  // State for each log slot
//   }
//
//   type SlotState struct {
//       highestPromised ProposalNumber
//       acceptedProposal ProposalNumber
//       acceptedValue []byte
//   }
//
// Each slot runs an independent Paxos instance, but they can share
// the Phase 1 promise across slots (leader optimization).
//
// =============================================================================

package paxos

import (
	"sync"
	"quorum/internal/storage"
)

type Storage interface {
	SavePromised(proposal storage.ProposalNumber) error
	LoadPromised() (storage.ProposalNumber, error)
	SaveAccepted(proposal storage.ProposalNumber, value []byte) error
	LoadAccepted() (storage.ProposalNumber, []byte, error)
	Close() error
}

type Acceptor struct {
	id               string
	highestPromised  ProposalNumber
	acceptedProposal ProposalNumber
	acceptedValue    []byte
	storage          Storage
	mu               sync.Mutex
}

func NewAcceptor(id string, s Storage) *Acceptor {
	a := &Acceptor{
		id:      id,
		storage: s,
	}
	if promised, err := s.LoadPromised(); err == nil {
		a.highestPromised = ProposalNumber{
			Round:      promised.Round,
			ProposerID: promised.ProposerID,
		}
	}
	if accepted, value, err := s.LoadAccepted(); err == nil {
		a.acceptedProposal = ProposalNumber{
			Round:      accepted.Round,
			ProposerID: accepted.ProposerID,
		}
		a.acceptedValue = value
	}
	return a
}

func toStorageProposal(p ProposalNumber) storage.ProposalNumber {
	return storage.ProposalNumber{
		Round:      p.Round,
		ProposerID: p.ProposerID,
	}
}

func (a *Acceptor) HandlePrepare(msg Prepare) Promise {
	a.mu.Lock()
	defer a.mu.Unlock()

	if msg.ProposalNumber.GreaterThan(a.highestPromised) {
		a.highestPromised = msg.ProposalNumber
		a.storage.SavePromised(toStorageProposal(a.highestPromised))
		return Promise{
			OK:               true,
			ProposalNumber:   msg.ProposalNumber,
			AcceptedProposal: a.acceptedProposal,
			AcceptedValue:    a.acceptedValue,
			From:             a.id,
		}
	}
	return Promise{
		OK:               false,
		ProposalNumber:   msg.ProposalNumber,
		AcceptedProposal: a.highestPromised,
		From:             a.id,
	}
}

func (a *Acceptor) HandleAccept(msg Accept) Accepted {
	a.mu.Lock()
	defer a.mu.Unlock()

	if msg.ProposalNumber.GreaterThan(a.highestPromised) || msg.ProposalNumber.Equal(a.highestPromised) {
		a.highestPromised = msg.ProposalNumber
		a.acceptedProposal = msg.ProposalNumber
		a.acceptedValue = msg.Value
		a.storage.SavePromised(toStorageProposal(a.highestPromised))
		a.storage.SaveAccepted(toStorageProposal(a.acceptedProposal), a.acceptedValue)
		return Accepted{
			OK:             true,
			ProposalNumber: msg.ProposalNumber,
			Value:          msg.Value,
			From:           a.id,
		}
	}
	return Accepted{
		OK:             false,
		ProposalNumber: msg.ProposalNumber,
		From:           a.id,
	}
}

func (a *Acceptor) GetState() (ProposalNumber, ProposalNumber, []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.highestPromised, a.acceptedProposal, a.acceptedValue
}