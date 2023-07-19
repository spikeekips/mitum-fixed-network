module github.com/spikeekips/mitum-fixed-network

go 1.16

require (
	github.com/alecthomas/kong v0.2.20
	github.com/beevik/ntp v0.3.0
	github.com/bluele/gcache v0.0.2
	github.com/btcsuite/btcd v0.22.0-beta
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce
	github.com/go-redis/redis/v8 v8.11.4
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d
	github.com/hashicorp/memberlist v0.3.0
	github.com/json-iterator/go v1.1.12
	github.com/justinas/alice v1.2.0
	github.com/lucas-clemente/quic-go v0.24.0
	github.com/mattn/go-isatty v0.0.14
	github.com/oklog/ulid v1.3.1
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.26.0
	github.com/satori/go.uuid v1.2.0
	github.com/spikeekips/mitum v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	github.com/ulule/limiter/v3 v3.9.0
	github.com/zeebo/blake3 v0.2.1
	go.mongodb.org/mongo-driver v1.8.0
	go.uber.org/automaxprocs v1.4.0
	go.uber.org/goleak v1.1.10
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e
	golang.org/x/mod v0.5.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/spikeekips/mitum => ./
