# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.7.0] - 2025-11-02

### 🚀 Agent Framework - High-Level Abstraction Layer

This release introduces a **complete Agent Framework** that dramatically simplifies SAGE protocol agent development. The framework provides high-level abstractions over SAGE components, reducing boilerplate code by up to 94% in initialization and 78% overall.

### Added

#### Agent Framework Core 🎯
- **pkg/agent/framework/agent.go** - Main framework entry point
  - `NewAgent()` - Create agents with explicit configuration
  - `NewAgentFromEnv()` - Environment-based agent creation (zero-config deployment)
  - `CreateHPKEClient()` - Built-in HPKE client support
  - Automatic DID resolution and key management
  - Built-in session management

#### Framework Packages 📦
- **pkg/agent/framework/keys/** - Cryptographic key management
  - `LoadFromJWKFile()` - Load keys from JWK files
  - `LoadFromJWKBytes()` - Load keys from byte arrays
  - `LoadKeySet()` - Load signing and KEM keys
  - `LoadKeySetFromEnv()` - Environment-based key loading

- **pkg/agent/framework/session/** - HPKE session management
  - `NewManager()` - Create session managers
  - `GetUnderlying()` - Access SAGE session manager (prototype phase)

- **pkg/agent/framework/did/** - DID resolution and verification
  - `NewResolver()` - Create DID resolvers
  - `NewResolverFromEnv()` - Environment-based resolver creation
  - `GetDIDClient()` / `GetKeyClient()` / `GetRegistryClient()` - Access clients

- **pkg/agent/framework/middleware/** - HTTP DID authentication
  - `NewDIDAuth()` - Create DID authentication middleware
  - `ComputeContentDigest()` - RFC 9421 content digest computation
  - Support for required and optional authentication

- **pkg/agent/framework/hpke/** - HPKE client/server abstractions
  - `NewServer()` - Create HPKE servers
  - `NewClient()` - Create HPKE clients
  - Simplified configuration with automatic key management

#### Code Reduction Metrics 📊
- **94% initialization code reduction** (165 lines → 10 lines)
- **78% overall code reduction** (686 lines → 150 lines)
- **100% SAGE import elimination** (7 imports → 0 imports)
- **Simple environment configuration** - Deploy via env vars only

#### Testing 🧪
- **23 comprehensive test cases** across 5 packages
- **5 test files**: keys_test.go, session_test.go, did_test.go, middleware_test.go, hpke_test.go
- Unit tests, integration tests, and benchmark tests
- Tests skip gracefully when Ethereum node unavailable (`testing.Short()`)
- All tests passing with go1.25.2

#### Documentation 📚
- **pkg/agent/framework/README.md** - Complete framework guide
  - Quick start guide
  - Architecture overview
  - API reference
  - Environment variables
  - Code reduction examples

- **examples/framework/payment_agent.go** - Working payment agent example
  - Complete implementation using framework
  - Before/after comparison
  - Demonstrates 94% code reduction

- **examples/framework/README.md** - Example usage guide
  - Running the examples
  - Learning points
  - Code metrics

- **AGENT_FRAMEWORK_MIGRATION_GUIDE.md** - Migration guide from sage-multi-agent
  - Step-by-step migration instructions
  - Checklist for verification
  - Troubleshooting tips

- **RELEASE_NOTES_v1.6.0.md** - Detailed release documentation
  - Feature highlights
  - Package structure
  - Migration guide
  - Breaking changes (none)

### Changed

- **README.md** - Major updates for v1.7.0
  - Updated version badge to v1.7.0
  - Expanded Agent Framework section
  - Added framework quick start example
  - Added code reduction metrics
  - Added package structure overview
  - Added links to framework documentation

### Removed

- **SAGE_A2A_GO_REQUIREMENTS.md** - Obsolete requirements document
- **SAGE_A2A_USAGE_REPORT.md** - Obsolete usage report

### Architecture

The Agent Framework follows a **wrapper pattern** around SAGE components:
- Clean separation from A2A protocol implementation
- Prototype phase includes `GetUnderlying()` methods for flexibility
- Environment-based configuration for 12-factor app compliance
- Zero external SAGE imports in agent code

### Migration from sage-multi-agent

This release completes the migration of the agent framework from `sage-multi-agent/internal/agent` to `sage-a2a-go/pkg/agent/framework`. All imports updated, tests added, and documentation complete.

### Breaking Changes

**None** - This is a purely additive release. All existing APIs remain unchanged.

---

## [1.6.0] - In Development

### 🚧 Complete Wrapper Implementation

This version completes the unified API wrapper architecture started in v1.5.2, providing high-level abstractions for all SAGE features.

### Added

#### HPKE Client Wrapper 🔐
- **pkg/hpke/client.go** - Simplified HPKE client wrapper
  - `NewClient()` - Create HPKE client with simplified API
  - `InitializeSession()` - Establish end-to-end encrypted sessions
  - Automatic DID-based authentication
  - Session key management integration
  - **9 new tests** for HPKE client functionality

#### Registry Client Wrapper 📋
- **pkg/registry/client.go** - Three-phase registration wrapper
  - `CommitRegistration()` - Phase 1: Anti-front-running commitment
  - `RegisterAgent()` - Phase 2: Reveal commitment and register
  - `ActivateAgent()` - Phase 3: Time-locked activation
  - `NewRegistrationParams()` - Helper for building registration parameters
  - **6 new tests** for registry client functionality

#### Enhanced Session Manager 🔑
- **Additional session management methods**:
  - `ListByRemoteDID()` - Find all sessions for a specific peer
  - `DeleteByRemoteDID()` - Cleanup all sessions for a peer
  - `Count()` - Get total active sessions
  - `SetMetadata()` / `GetMetadata()` - Session metadata management
  - `Exists()` - Check session validity
  - `GetExpiresAt()` - Get session expiration time
  - `GetRemoteDID()` - Get peer DID for a session
  - **20 comprehensive tests** for session manager (100% coverage)

### Testing
- **35 new tests** added across v1.6.0 features
- All tests passing (session: 20, registry: 6, hpke: 9)
- Unit tests for client creation, validation, and error handling

## [1.5.2] - 2025-11-02

### 🚧 In Development

This version includes **SAGE v1.5.2 upgrade**, **unified API architecture**, **A2A Protocol v0.4.0 support**, **Server-Sent Events (SSE) streaming**, and **DID authentication**.

### Added

#### Unified API Architecture 🏗️ (v1.5.2)
- **pkg/crypto/** - Unified cryptographic operations wrapper
  - `GenerateSecp256k1KeyPair()` - Ethereum key generation
  - `GenerateEd25519KeyPair()` - Solana key generation
  - Type aliases for `KeyPair` and `KeyType`
  - Wraps SAGE crypto without exposing implementation

- **pkg/identity/** - Unified DID management wrapper
  - `AgentDID` type alias
  - `ParseDID()` - Parse DID components
  - `ValidateDID()` - DID validation
  - `MarshalPublicKey()` / `UnmarshalPublicKey()` - Key serialization
  - Wraps SAGE DID functionality

- **pkg/agent/** - High-level agent builder
  - `NewAgent()` - One-function agent creation
  - Combines DID authentication with A2A communication
  - `Agent` struct with DID, KeyPair, A2AClient, and Card
  - Complete abstraction - users only need sage-a2a-go imports

**Architecture Philosophy**: Users can now build complete A2A agents using **only** sage-a2a-go packages. No need to directly import SAGE or A2A libraries. This enables cleaner integration and better separation of concerns.

#### SSE Streaming Support ✨
- **SendStreamingMessage** - Real-time message streaming via SSE
  - W3C-compliant SSE client implementation
  - Support for all 4 A2A event types (Message, Task, TaskStatusUpdateEvent, TaskArtifactUpdateEvent)
  - Context-aware cancellation
  - Automatic DID signatures on all requests
  - Comprehensive error handling

- **ResubscribeToTask** - Reconnect to task event streams
  - Backfill support for missed events
  - Reconnection after network interruptions
  - Resume long-running task monitoring

- **New SSE Module** (`pkg/transport/sse.go`)
  - `parseSSEStream()` - W3C SSE parser
  - `parseSSEData()` - JSON-RPC response unwrapper
  - `callSSE()` - DID-authenticated SSE requests
  - Multi-line data field handling
  - Event ID tracking for resumption

#### A2A Protocol v0.4.0 Features 🚀
- **ListTasks** method with cursor-based pagination
  - Filter tasks by context, status, timestamp
  - Pagination with pageToken/nextPageToken
  - Configurable page size (1-100 results)
  - History length control
  - Artifact inclusion option
  - Custom metadata filters

#### Documentation 📚
- **SSE Streaming Guide** (`docs/SSE_STREAMING_GUIDE.md`)
  - Complete streaming tutorial
  - All 4 event types explained
  - Error handling patterns
  - Best practices
  - Advanced usage examples
  - Troubleshooting guide

- **API Reference** (`docs/API_REFERENCE.md`)
  - Complete API documentation
  - All public methods documented
  - Code examples for every API
  - Error handling reference
  - Type definitions

- **Updated README** - Roadmap and feature status

#### Testing 🧪
- **8 Comprehensive SSE Tests**
  - Success scenarios (multiple events, all event types)
  - Error handling (wrong content-type, HTTP errors, malformed data)
  - Context cancellation
  - DID signature verification
  - Unknown event types
  - Multiline data handling

- **3 ListTasks Tests**
  - Basic listing
  - Pagination
  - Empty results

- **Test Coverage**
  - pkg/client: 92.3%
  - pkg/protocol: 91.2%
  - pkg/server: 100.0%
  - pkg/signer: 92.2%
  - pkg/transport: 74.4%
  - pkg/verifier: 88.0%
  - **Overall: 85.0%**

### Changed

#### Dependency Updates
- **SAGE v1.5.2** (from v1.3.1)
  - KEM (Key Encapsulation Mechanism) naming corrections per RFC 9180
  - Security enhancements in HTTP message signing
  - Go 1.25.2 compatibility
  - HPKE client synchronization improvements
  - Updated all KME→KEM references throughout codebase

- **A2A Protocol v0.4.0** (from v0.3.0)
  - ListTasks method support
  - Enhanced pagination
  - Additional task filters

#### Code Quality & Internationalization
- Translated all Korean comments to English
  - Updated pkg/verifier/default_key_selector.go
  - Updated pkg/signer/default_a2a_signer.go
  - Updated pkg/server/middleware.go
- Improved code documentation consistency

#### Code Quality
- Removed obsolete "NotImplemented" tests
- Fixed field name compatibility with a2a-go
- Improved error messages
- Better type discrimination in SSE parsing

### Fixed

- **SSE Event Parsing** - Correct JSON-RPC unwrapping
- **Type Compatibility** - Fixed Message.ID field (was MessageID)
- **Role Constants** - Use MessageRoleUser/MessageRoleAgent
- **TaskID Type** - Proper type casting for TaskID

### Documentation Updates

- Updated `IMPLEMENTATION_STATUS.md`
  - Marked SSE streaming as complete
  - Updated test coverage statistics
  - Added SSE implementation details

- Updated `README.md`
  - SSE streaming in current features
  - Updated roadmap (v2.1.0 goals)
  - Test coverage statistics

### Performance

- Efficient SSE stream parsing with buffered I/O
- Context-aware cancellation for resource cleanup
- Minimal memory footprint for long-running streams

### Security

- All SSE requests signed with DID (RFC 9421)
- Signature verification on every request
- Context deadline enforcement
- Proper TLS support

---

## Development Notes

This project is under active development. The first stable release will be v1.0.0.

### Current Features

- ✅ DID HTTP Transport with authentication
- ✅ SSE Streaming support
- ✅ A2A Protocol v0.4.0
- ✅ Multi-key support (ECDSA, Ed25519, P-256)
- ✅ 174 tests with 91.8% average coverage

### Documentation

- See [SSE Streaming Guide](docs/SSE_STREAMING_GUIDE.md)
- See [API Reference](docs/API_REFERENCE.md)
- See [Integration Guide](docs/INTEGRATION_GUIDE.md)

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## License

LGPL-3.0 - See [LICENSE](LICENSE)
