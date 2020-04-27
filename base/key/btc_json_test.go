package key

import (
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (t *testBTCKeypair) TestJSON() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	{
		b, err := jsonencoder.Marshal(kp)
		t.NoError(err)

		var decoded BTCPrivatekey
		t.NoError(jsonencoder.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := jsonencoder.Marshal(pub)
		t.NoError(err)

		var decoded BTCPublickey
		t.NoError(jsonencoder.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
