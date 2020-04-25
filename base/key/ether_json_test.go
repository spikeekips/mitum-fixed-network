package key

import "github.com/spikeekips/mitum/util"

func (t *testEtherKeypair) TestJSON() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	{
		b, err := util.JSONMarshal(kp)
		t.NoError(err)

		var decoded EtherPrivatekey
		t.NoError(util.JSONUnmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := util.JSONMarshal(pub)
		t.NoError(err)

		var decoded EtherPublickey
		t.NoError(util.JSONUnmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
