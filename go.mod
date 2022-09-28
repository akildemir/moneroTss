module github.com/akildemir/moneroTss

go 1.17

require (
	github.com/akildemir/go-tss v0.0.0-20220928092621-60dff86ca844
	github.com/binance-chain/tss-lib v0.0.0-20201118045712-70b2cb4bf916
	github.com/blang/semver v3.5.1+incompatible
	github.com/btcsuite/btcd v0.22.1
	github.com/cosmos/cosmos-sdk v0.45.1
	github.com/deckarep/golang-set v1.7.1
	github.com/decred/dcrd/dcrec/secp256k1 v1.0.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.8.0
	github.com/ipfs/go-log v1.0.5
	github.com/libp2p/go-libp2p v0.22.0
	github.com/libp2p/go-libp2p-core v0.20.0
	github.com/libp2p/go-libp2p-discovery v0.5.0
	github.com/libp2p/go-libp2p-kad-dht v0.18.0
	github.com/libp2p/go-libp2p-peerstore v0.8.0
	github.com/libp2p/go-libp2p-testing v0.11.0
	github.com/magiconair/properties v1.8.5
	github.com/multiformats/go-multiaddr v0.6.0
	github.com/olekukonko/tablewriter v0.0.2-0.20190409134802-7e037d187b0c
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.12.1
	github.com/prometheus/client_model v0.2.0
	github.com/rs/zerolog v1.23.0
	github.com/stretchr/testify v1.8.0
	github.com/tendermint/tendermint v0.34.14
	gitlab.com/thorchain/binance-sdk v1.2.3-0.20210117202539-d569b6b9ba5d
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/text v0.3.7
	google.golang.org/protobuf v1.28.1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
)

require (
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/enigmampc/btcutil v1.0.3-0.20200723161021-e2fb6adb2a25 // indirect
	github.com/haven-protocol-org/go-haven-rpc-client v0.0.0-20220622125045-986219a60b46 // indirect
	github.com/libp2p/go-mplex v0.7.0 // indirect
	github.com/libp2p/go-sockaddr v0.1.0 // indirect
	github.com/libp2p/go-yamux v1.3.8 // indirect
)

replace (
	github.com/binance-chain/tss-lib => gitlab.com/thorchain/tss/tss-lib v0.0.0-20201118045712-70b2cb4bf916
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.2-alpha.regen.4
)
