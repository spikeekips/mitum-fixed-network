package key

import "github.com/spikeekips/mitum/util"

func (t *testBTCKeypair) TestJSON() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	{
		b, err := util.JSONMarshal(kp)
		t.NoError(err)

		var decoded BTCPrivatekey
		t.NoError(util.JSONUnmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := util.JSONMarshal(pub)
		t.NoError(err)

		var decoded BTCPublickey
		t.NoError(util.JSONUnmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
