# quorum

paxos consensus algorithm in go.

## what is this

implementation of single-decree paxos. consensus means getting multiple computers to agree on a single value even when some fail or messages get lost.

## project structure

```
quorum/
├── cmd/demo/main.go              # demo runner
├── internal/paxos/
│   ├── proposal.go               # proposal numbers
│   ├── message.go                # message types
│   ├── acceptor.go               # acceptor role
│   ├── proposer.go               # proposer role
│   └── learner.go                # learner role
├── internal/node/node.go         # combines all roles
├── internal/storage/             # storage interface + memory impl
├── internal/transport/           # transport interface + memory impl
└── go.mod
```

## quick start

```bash
go run ./cmd/demo
```

## the roles

- **proposer**: suggests values, runs the protocol
- **acceptor**: votes on proposals, stores state
- **learner**: observes when consensus is reached

each node runs all three roles.

## the two phases

**phase 1 (prepare)**:
```
proposer -> acceptors: prepare(n)
acceptors -> proposer: promise(n) [or reject]
```

**phase 2 (accept)**:
```
proposer -> acceptors: accept(n, v)
acceptors -> proposer: accepted [or reject]
```

a value is chosen when a majority of acceptors accept it.

## key rules

1. acceptors only accept if the proposal number is >= their highest promised
2. proposers must adopt the highest-value from any prior accepts
3. once chosen, a value cannot be changed

## files in order

1. `proposal.go` - proposal numbers
2. `message.go` - message types
3. `acceptor.go` - acceptor logic (safety lives here)
4. `proposer.go` - proposer logic
5. `learner.go` - learner logic
6. `storage.go` / `memory.go` - storage interface
7. `transport.go` / `memory.go` - transport interface
8. `node.go` - wire everything together
9. `main.go` - run demo

## resources

- [paxos made simple](https://lamport.azurewebsites.net/pubs/paxos-simple.pdf)
- [raft paper](https://raft.github.io/raft.pdf) - similar but more understandable
