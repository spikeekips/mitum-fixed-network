module github.com/spikeekips/mitum

go 1.14

require (
	github.com/alecthomas/kong v0.2.2
	github.com/beevik/ntp v0.2.0
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v1.0.1
	github.com/ethereum/go-ethereum v1.9.11
	github.com/golang/protobuf v1.3.4 // indirect
	github.com/gorilla/mux v1.7.4
	github.com/hashicorp/golang-lru v0.5.4
	github.com/json-iterator/go v1.1.9
	github.com/justinas/alice v1.2.0
	github.com/lib/pq v1.3.0 // indirect
	github.com/lucas-clemente/quic-go v0.14.4
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rs/zerolog v1.18.0
	github.com/satori/go.uuid v1.2.0
	github.com/spikeekips/avl v0.0.0-20191218024138-87d874c0c032
	github.com/stellar/go v0.0.0-20200226233544-b4fe8efa472b
	github.com/stellar/go-xdr v0.0.0-20191118232123-3aa35463cbdb // indirect
	github.com/stretchr/testify v1.5.1
	github.com/syndtr/goleveldb v1.0.1-0.20190923125748-758128399b1d
	go.mongodb.org/mongo-driver v1.3.0
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/mod v0.2.0
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b // indirect
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

// NOTE TODO to prevent race condition of quic.Config; check the new release
replace github.com/lucas-clemente/quic-go v0.14.4 => github.com/spikeekips/quic-go v0.14.4-race-quicconfig
