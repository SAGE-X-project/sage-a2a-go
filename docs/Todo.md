# SAGE-A2A-GO Implementation Todo List

**Project**: sage-a2a-go
**SAGE Version**: v1.1.0
**A2A Protocol**: JSON-RPC 2.0 over HTTP(S)
**Development Approach**: TDD with 90%+ test coverage
**Branch**: feature/did-rfc9421-integration
**Date**: 2025-10-18

---

## Overview

This project integrates SAGE DID system with A2A (Agent-to-Agent) Protocol, providing:
- DID-based authentication for A2A agents
- RFC9421 HTTP Message Signatures with SAGE DIDs
- Multi-key support (ECDSA for Ethereum, Ed25519 for Solana)
- Agent Card signing and verification
- Integration with a2a-go SDK

---

## Phase 1: Project Setup ✅

- [x] 1.1 Create feature branch (feature/did-rfc9421-integration)
- [x] 1.2 Create project directory structure
- [x] 1.3 Create .gitignore for Go project
- [x] 1.4 Create docs/Todo.md (this file)
- [ ] 1.5 Initialize go.mod with SAGE v1.1.0
- [ ] 1.6 Create README.md with project overview
- [ ] 1.7 Create design documentation (docs/design.md)

---

## Phase 2: KeySelector Implementation (TDD)

**Purpose**: Select appropriate cryptographic key based on protocol/chain

- [ ] 2.1 Design KeySelector interface
- [ ] 2.2 Write test cases for KeySelector
  - [ ] Test Ethereum protocol selects ECDSA key
  - [ ] Test Solana protocol selects Ed25519 key
  - [ ] Test fallback to first available key
  - [ ] Test error handling (no keys found)
  - [ ] Test multiple keys scenario
- [ ] 2.3 Implement KeySelector to pass all tests
- [ ] 2.4 Verify test coverage ≥ 90%
- [ ] 2.5 Refactor and document

---

## Phase 3: DIDVerifier Implementation (TDD)

**Purpose**: Verify HTTP signatures using SAGE DIDs

- [ ] 3.1 Design DIDVerifier interface
- [ ] 3.2 Write test cases for DIDVerifier
  - [ ] Test DID public key resolution
  - [ ] Test HTTP signature verification with ECDSA
  - [ ] Test HTTP signature verification with Ed25519
  - [ ] Test invalid DID handling
  - [ ] Test expired signature handling
  - [ ] Test replay attack prevention
- [ ] 3.3 Implement DIDVerifier to pass all tests
- [ ] 3.4 Verify test coverage ≥ 90%
- [ ] 3.5 Refactor and document

---

## Phase 4: A2ASigner Implementation (TDD)

**Purpose**: Sign HTTP messages for A2A communication with DID

- [ ] 4.1 Design A2ASigner interface
- [ ] 4.2 Write test cases for A2ASigner
  - [ ] Test request signing with ECDSA key
  - [ ] Test request signing with Ed25519 key
  - [ ] Test DID inclusion in signature
  - [ ] Test signature format (RFC9421 compliance)
  - [ ] Test timestamp inclusion
- [ ] 4.3 Implement A2ASigner to pass all tests
- [ ] 4.4 Verify test coverage ≥ 90%
- [ ] 4.5 Refactor and document

---

## Phase 5: RFC9421 DID Adapter (TDD)

**Purpose**: Integrate SAGE RFC9421 verifier with DID resolution

- [ ] 5.1 Design RFC9421DIDAdapter
- [ ] 5.2 Write test cases for adapter
  - [ ] Test keyid (DID) extraction from headers
  - [ ] Test DID to public key resolution
  - [ ] Test RFC9421 signature verification
  - [ ] Test integration with KeySelector
- [ ] 5.3 Implement RFC9421DIDAdapter to pass all tests
- [ ] 5.4 Verify test coverage ≥ 90%
- [ ] 5.5 Refactor and document

---

## Phase 6: Agent Card Support (TDD)

**Purpose**: Sign and verify A2A Agent Cards with DID

- [ ] 6.1 Design AgentCardSigner interface
- [ ] 6.2 Write test cases for Agent Card
  - [ ] Test Agent Card creation
  - [ ] Test Agent Card signing (JWS)
  - [ ] Test Agent Card verification
  - [ ] Test DID inclusion in Agent Card
- [ ] 6.3 Implement AgentCardSigner to pass all tests
- [ ] 6.4 Verify test coverage ≥ 90%
- [ ] 6.5 Refactor and document

---

## Phase 7: Integration Tests

**Requirements**: Local Ethereum testnet, SAGE Registry V4 deployed

- [ ] 7.1 Set up test environment configuration
- [ ] 7.2 Write end-to-end integration tests
  - [ ] Agent registration with multi-key
  - [ ] Agent A signs message → Agent B verifies
  - [ ] Ethereum agent (ECDSA) ↔ Solana agent (Ed25519)
  - [ ] Agent Card signing and verification flow
- [ ] 7.3 Test multi-key selection scenarios
- [ ] 7.4 Test cross-protocol communication
- [ ] 7.5 Verify all integration tests pass

---

## Phase 8: A2A Protocol Integration

**Purpose**: Integrate with a2a-go SDK

- [ ] 8.1 Research a2a-go SDK structure
- [ ] 8.2 Design A2A client with DID auth
- [ ] 8.3 Design A2A server with DID verification
- [ ] 8.4 Implement A2A client (TDD)
- [ ] 8.5 Implement A2A server (TDD)
- [ ] 8.6 Test complete A2A flow with DID auth

---

## Phase 9: Documentation and Examples

- [ ] 9.1 Write comprehensive README.md
- [ ] 9.2 Write API documentation (GoDoc)
- [ ] 9.3 Create example programs
  - [ ] Simple agent with DID registration
  - [ ] Agent-to-agent communication
  - [ ] Multi-key agent example
- [ ] 9.4 Write integration guide
- [ ] 9.5 Create architectural diagrams

---

## Phase 10: Testing and Quality

- [ ] 10.1 Run full test suite
- [ ] 10.2 Generate coverage report (must be ≥ 90%)
- [ ] 10.3 Run linters (golangci-lint)
- [ ] 10.4 Run security checks (gosec)
- [ ] 10.5 Performance benchmarks
- [ ] 10.6 Fix all issues found

---

## Phase 11: PR and Review

- [ ] 11.1 Update CHANGELOG.md
- [ ] 11.2 Ensure all commits follow convention
- [ ] 11.3 Squash/clean commit history if needed
- [ ] 11.4 Create PR to main branch
- [ ] 11.5 Address review feedback
- [ ] 11.6 Merge PR after approval

---

## Test Coverage Requirements

- **Target**: ≥ 90% for all packages
- **Command**: `go test -cover -coverprofile=coverage.out ./...`
- **Report**: `go tool cover -html=coverage.out -o coverage.html`
- **CI Integration**: Coverage checked in PR pipeline

---

## Branch Strategy

- **Development Branch**: `feature/did-rfc9421-integration`
- **Main Branch**: `main`
- **PR Required**: Yes (no direct commits to main)
- **Review**: At least 1 approval required

---

## Commit Guidelines

- **Language**: English only
- **Format**: `<type>: <subject>`
- **Types**:
  - `feat`: New feature
  - `fix`: Bug fix
  - `test`: Add or update tests
  - `refactor`: Code refactoring
  - `docs`: Documentation changes
  - `chore`: Maintenance tasks
- **No co-author tags**: Remove all `Co-Authored-By` lines
- **Examples**:
  - `feat: implement DIDVerifier with RFC9421 support`
  - `test: add KeySelector test cases for multi-key scenarios`
  - `fix: handle missing Ed25519 key gracefully`

---

## Dependencies

- **SAGE**: `github.com/SAGE-X-project/sage@v1.1.0`
- **A2A**: `github.com/a2aproject/a2a-go` (latest)
- **Testing**: `github.com/stretchr/testify`
- **Mocking**: `github.com/golang/mock`

---

## Notes

- All tests must pass before moving to next phase
- Follow TDD strictly: Write tests first, then implementation
- Refactor after tests pass
- Document all public interfaces
- Keep functions small and focused
- Use meaningful variable names
- Add comments for complex logic

---

## Progress Tracking

**Overall Progress**: 🟢 Phase 1 in progress

- Phase 1: 🟢 In Progress (4/7 completed)
- Phase 2: ⚪ Not Started
- Phase 3: ⚪ Not Started
- Phase 4: ⚪ Not Started
- Phase 5: ⚪ Not Started
- Phase 6: ⚪ Not Started
- Phase 7: ⚪ Not Started
- Phase 8: ⚪ Not Started
- Phase 9: ⚪ Not Started
- Phase 10: ⚪ Not Started
- Phase 11: ⚪ Not Started

---

**Last Updated**: 2025-10-18
**Maintainer**: SAGE Development Team
