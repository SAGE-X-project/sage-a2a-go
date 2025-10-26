module github.com/sage-x-project/sage-a2a-go

go 1.24.4

require (
	github.com/a2aproject/a2a-go v0.1.0 // A2A Protocol Go SDK
	github.com/sage-x-project/sage v1.3.1
	github.com/stretchr/testify v1.11.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/ethereum/go-ethereum v1.15.11 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Use local a2a-go for development
replace github.com/a2aproject/a2a-go => ../a2a-go
