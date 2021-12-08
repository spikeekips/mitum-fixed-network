package deploy

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type baseDeployKeyHandler struct {
	suite.Suite
	cache cache.Cache
	enc   encoder.Encoder
}

func (t *baseDeployKeyHandler) SetupTest() {
	c, err := cache.NewGCache("lru", 100*100, time.Minute)
	t.NoError(err)

	t.cache = c

	encs := encoder.NewEncoders()

	t.enc = jsonenc.NewEncoder()
	encs.AddEncoder(t.enc)
	_ = encs.TestAddHinter(key.BasePublickey{})
}
