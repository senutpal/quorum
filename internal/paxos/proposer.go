// =============================================================================
// PROPOSER - The Driver of Paxos Consensus
// =============================================================================
//
// IMPLEMENTATION ORDER: File 4 of 7 in internal/paxos/
// Implement this AFTER acceptor.go
//
// =============================================================================
// WHAT THIS FILE REPRESENTS
// =============================================================================
//
// The Proposer is the "active" role in Paxos. While acceptors wait passively
// for messages, proposers drive the protocol forward by:
//
// 1. Choosing proposal numbers
// 2. Sending Prepare requests (Phase 1)
// 3. Collecting Promise responses
// 4. Sending Accept requests (Phase 2)
// 5. Collecting Accepted responses
// 6. Notifying learners when consensus is reached
//
// Think of the proposer as the "project manager" - it coordinates the
// acceptors to reach agreement.
//
// =============================================================================
// HIGH-LEVEL ALGORITHM
// =============================================================================
//
// PHASE 1: PREPARE PHASE
// ┌─────────────────────────────────────────────────────────────────────────┐
// │ 1. Pick a proposal number N higher than any I've used before           │
// │ 2. Send Prepare(N) to ALL acceptors (or at least a majority)           │
// │ 3. Wait for Promise responses from a MAJORITY of acceptors             │
// │    - If any acceptor rejects, their response tells me what number      │
// │      to try next. Go back to step 1 with a higher number.              │
// │ 4. Look at all the Promise responses. If any acceptor already          │
// │    accepted a value, I MUST use the value from the highest-numbered    │
// │    accepted proposal.                                                  │
// │    - This is the KEY SAFETY RULE for proposers!                        │
// └─────────────────────────────────────────────────────────────────────────┘
//
// PHASE 2: ACCEPT PHASE
// ┌─────────────────────────────────────────────────────────────────────────┐
// │ 5. Send Accept(N, V) to ALL acceptors                                  │
// │    - V is either: the value I wanted to propose (if no prior accepts)  │
// │                   OR: the value from step 4 (if prior accepts exist)   │
// │ 6. Wait for Accepted responses from a MAJORITY of acceptors            │
// │    - If any acceptor rejects, someone else got a higher prepare in.    │
// │      Go back to Phase 1 with a higher number.                          │
// │ 7. If a majority accepted: VALUE V IS CHOSEN!                          │
// │    - Notify learners                                                   │
// │    - We're done!                                                       │
// └─────────────────────────────────────────────────────────────────────────┘
//
// =============================================================================
// PROPOSER STATE
// =============================================================================
//
// Unlike acceptors, proposer state doesn't need to be durable (persist
// across crashes). If a proposer crashes mid-protocol, it just restarts.
//
// Required state:
//
// 1. proposerID: A unique identifier for this proposer
//    - Used to create unique proposal numbers
//
// 2. highestProposalNumber: The highest proposal number I've used
//    - Start from 0, increment before each new proposal attempt
//    - Must be higher than any proposal number I've received in a rejection
//
// 3. currentPhase: Where we are in the protocol
//    - "idle", "preparing", "accepting", "done"
//
// 4. collectedPromises: Promises received in the current prepare phase
//    - Need a majority to proceed
//
// 5. currentValue: The value we're trying to propose
//    - May be overwritten if we see a prior accepted value
//
// =============================================================================
// TODO: IMPLEMENTATION TASKS
// =============================================================================
//
// TODO: Define Proposer struct
//       Fields:
//       - id string
//         // Unique identifier for this proposer
//       - highestRound int64
//         // Highest round number we've used (incremented each attempt)
//       - currentProposal ProposalNumber
//         // The proposal number we're currently using
//       - originalValue []byte
//         // The value the client wants us to propose
//       - valueToPropose []byte
//         // The value we'll actually propose (might be from prior accept)
//       - promises []Promise
//         // Collected promise responses
//       - quorumSize int
//         // How many acceptors form a majority
//       - transport Transport
//         // How we send messages
//       - mu sync.Mutex
//         // Protect concurrent access
//
//
// TODO: Implement NewProposer(id string, quorumSize int, transport Transport) *Proposer
//
//
// TODO: Implement Propose(value []byte) ([]byte, error)
//       This is the main entry point. Algorithm:
//       1. Store value as originalValue and valueToPropose
//       2. Generate a new proposal number (increment round, combine with id)
//       3. Call runPhase1()
//       4. Call runPhase2()
//       5. Return the chosen value (might differ from originalValue!)
//
//
// TODO: Implement runPhase1() error
//       Algorithm:
//       1. Create Prepare message with currentProposal
//       2. Send to all known acceptors via transport
//       3. Wait for quorumSize Promise responses (with timeout)
//       4. If any rejection: update highestRound, return error to retry
//       5. Find the highest accepted proposal among promises
//       6. If any accepted value exists: set valueToPropose = that value
//       7. Return nil (success)
//
//       KEY INSIGHT: Step 6 is what preserves safety! If some value might
//       already be chosen, we adopt it instead of proposing our own.
//
//
// TODO: Implement runPhase2() error
//       Algorithm:
//       1. Create Accept message with currentProposal and valueToPropose
//       2. Send to all known acceptors via transport
//       3. Wait for quorumSize Accepted responses (with timeout)
//       4. If any rejection: return error (caller will retry Phase 1)
//       5. Notify learners that value is chosen
//       6. Return nil (success)
//
//
// TODO: Implement generateProposalNumber() ProposalNumber
//       Algorithm:
//       1. Increment highestRound
//       2. Return ProposalNumber{Round: highestRound, ProposerID: id}
//
//       NOTE: If we received a rejection with a higher round, we must
//       update highestRound first to skip past it.
//
//
// TODO: Implement handleRejection(highestSeen ProposalNumber)
//       Algorithm:
//       1. If highestSeen.Round > highestRound:
//          highestRound = highestSeen.Round
//       2. This ensures our next proposal number will be higher
//
// =============================================================================
// THE CRITICAL SAFETY RULE
// =============================================================================
//
// When a proposer receives Promise responses, if ANY acceptor reports it
// has already accepted a proposal, the proposer MUST:
//
//   1. Find the accepted proposal with the HIGHEST proposal number
//   2. Use that value instead of its own
//
// Why? Because if a value was accepted by even one acceptor, it might
// have been accepted by a majority (and thus chosen). We must not
// propose a different value.
//
// Example:
//   - Proposer A proposed value X at proposal 5, got some accepts
//   - We don't know if X was chosen or not
//   - Proposer B starts proposal 7, learns about (5, X) in promises
//   - B MUST propose X at proposal 7, not its own value
//   - This ensures if X was chosen, it stays chosen
//
// =============================================================================
// FAILURE SCENARIO: WHAT BREAKS IF PROPOSER IS WRONG
// =============================================================================
//
// If a proposer ignores prior accepted values:
//
// 1. Proposer A proposes X at proposal 5, gets majority accepts
// 2. Value X is CHOSEN (we just don't know it yet)
// 3. Proposer B starts proposal 7, gets promises that mention (5, X)
// 4. B ignores X and proposes its own value Y at proposal 7 (BUG!)
// 5. If B gets a majority, Y is also "chosen"
// 6. DISASTER: Two values are chosen
//
// =============================================================================
// LIVENESS CONSIDERATIONS
// =============================================================================
//
// Paxos guarantees safety but NOT liveness under all conditions.
//
// Problem: Dueling Proposers
//   - Proposer A does Phase 1 with proposal 5
//   - Before A can do Phase 2, Proposer B does Phase 1 with proposal 7
//   - A's Phase 2 is rejected (7 > 5)
//   - A tries Phase 1 with proposal 9
//   - Before A can finish, B does Phase 1 with proposal 11
//   - And so on forever...
//
// Solutions (implement later):
//   1. Randomized backoff before retrying
//   2. Leader election (one proposer at a time)
//   3. Lease-based leadership (Multi-Paxos optimization)
//
// For now, don't worry about this. Just document it.
//
// =============================================================================
// INVARIANT THIS FILE MUST UPHOLD
// =============================================================================
//
// INVARIANT: Before proposing a value in Phase 2, the proposer MUST adopt
//            the value from the highest-numbered accepted proposal seen
//            in any Promise from Phase 1.
//
// This is the proposer's contribution to safety. Violating this allows
// multiple values to be chosen.
//
// =============================================================================
// COMMON BUG TO AVOID
// =============================================================================
//
// BUG: Not checking ALL promises for accepted values
//
// Scenario:
// - We need promises from 3 acceptors (out of 5)
// - Acceptor 1: No prior accept
// - Acceptor 2: Accepted (5, X)
// - Acceptor 3: Accepted (3, Y)
//
// Wrong: Propose our own value because "2 out of 3 have no accept"
// Right: Propose X because proposal 5 > proposal 3, so X wins
//
// You must find the MAXIMUM accepted proposal number and use THAT value.
//
// =============================================================================
// MULTI-PAXOS EXTENSION POINT
// =============================================================================
//
// In Multi-Paxos:
//
// 1. The proposer becomes a "leader" that handles many slots
// 2. Phase 1 can be done ONCE and reused for many slots (leader lease)
// 3. Only Phase 2 is needed per slot (much faster!)
// 4. Leader election determines who is the active proposer
//
// The structure here supports this by keeping Phase 1 and Phase 2 separate.
//
// =============================================================================

package paxos

import (
	"errors"
	"sync"
)

type Proposer struct {
	id string
	highestRound int64
	currentProposal ProposalNumber
	originalValue []byte
	valueToPropose []byte
	promise []Promise
	quorumSize int
	transport Transport
	mu sync.Mutex
}

func NewProposer(id string, quorumSize int, transport Transport) *Proposer {
	return &Proposer{
		id:        id,
		quorumSize: quorumSize,
		transport: transport,
	}
}

func (p *Proposer) Propose(value []byte) ([]byte, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.originalValue = value
	p.valueToPropose = value
	for {
		p.currentProposal = p.generateProposalNumber()
		p.promise = nil 
		err := p.runPhase1()
		if err != nil {
			continue
		}
		err = p.runPhase2()
		if err != nil {
			continue
		}
		return p.valueToPropose, nil
	}
}

func (p *Proposer) runPhase1() error {
	prepareMsg := Prepare{
		ProposalNumber: p.currentProposal,
		From:           p.id,
	}
	p.transport.Broadcast(prepareMsg)
	promiseCount := 0
	for promiseCount < p.quorumSize {
		msg, err := p.transport.Receive()
		if err != nil {
			return err
		}
		promise, ok := msg.(Promise)
		if !ok {
			continue 
		}
		if !promise.ProposalNumber.Equal(p.currentProposal) {
			continue
		}
		if !promise.OK {
			p.handleRejection(promise.AcceptedProposal)
			return ErrRejected
		}
		p.promise = append(p.promise, promise)
		promiseCount++
	}
	var highestAccepted ProposalNumber
	for _, promise := range p.promise {
		if !promise.AcceptedProposal.IsZero() {
			if promise.AcceptedProposal.GreaterThan(highestAccepted) {
				highestAccepted = promise.AcceptedProposal
				p.valueToPropose = promise.AcceptedValue
			}
		}
	}
	return nil
}

func (p *Proposer) runPhase2() error {
	acceptMsg := Accept{
		ProposalNumber: p.currentProposal,
		Value:          p.valueToPropose,
		From:           p.id,
	}
	p.transport.Broadcast(acceptMsg)
	acceptedCount := 0
	for acceptedCount < p.quorumSize {
		msg, err := p.transport.Receive()
		if err != nil {
			return err
		}
		accepted, ok := msg.(Accepted)
		if !ok {
			continue
		}
		if !accepted.ProposalNumber.Equal(p.currentProposal) {
			continue
		}
		if !accepted.OK {
			return ErrRejected
		}
		acceptedCount++
	}
	learnMsg := Learn{
		ProposalNumber: p.currentProposal,
		Value:          p.valueToPropose,
		From:           p.id,
	}
	p.transport.Broadcast(learnMsg)
	return nil
}

func (p *Proposer) generateProposalNumber() ProposalNumber {
	p.highestRound++
	return ProposalNumber{
		Round:      p.highestRound,
		ProposerID: p.id,
	}
}

func (p *Proposer) handleRejection(highestSeen ProposalNumber) {
	if highestSeen.Round > p.highestRound {
		p.highestRound = highestSeen.Round
	}
}
var ErrRejected = errors.New("proposal rejected")

