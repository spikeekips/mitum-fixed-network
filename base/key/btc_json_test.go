package key

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (t *testBTCKeypair) TestJSON() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	{
		b, err := jsonenc.Marshal(kp)
		t.NoError(err)

		var decoded BTCPrivatekey
		t.NoError(jsonenc.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := jsonenc.Marshal(pub)
		t.NoError(err)

		var decoded BTCPublickey
		t.NoError(jsonenc.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
