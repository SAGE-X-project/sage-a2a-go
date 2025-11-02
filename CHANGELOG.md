# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### 🚧 In Development

This version is under active development with **SAGE v1.5.2 upgrade**, **unified API architecture**, **A2A Protocol v0.4.0 support**, **Server-Sent Events (SSE) streaming**, and **DID authentication**.

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
