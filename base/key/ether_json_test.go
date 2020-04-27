package key

import (
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (t *testEtherKeypair) TestJSON() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	{
		b, err := jsonencoder.Marshal(kp)
		t.NoError(err)

		var decoded EtherPrivatekey
		t.NoError(jsonencoder.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := jsonencoder.Marshal(pub)
		t.NoError(err)

		var decoded EtherPublickey
		t.NoError(jsonencoder.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
