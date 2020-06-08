package key

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (t *testEtherKeypair) TestJSON() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	{
		b, err := jsonenc.Marshal(kp)
		t.NoError(err)

		var decoded EtherPrivatekey
		t.NoError(jsonenc.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := jsonenc.Marshal(pub)
		t.NoError(err)

		var decoded EtherPublickey
		t.NoError(jsonenc.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
