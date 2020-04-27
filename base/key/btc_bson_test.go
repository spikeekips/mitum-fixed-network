package key

import (
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (t *testBTCKeypair) TestBSON() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	{
		b, err := bsonencoder.Marshal(kp)
		t.NoError(err)

		var decoded BTCPrivatekey
		t.NoError(bsonencoder.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bsonencoder.Marshal(pub)
		t.NoError(err)

		var decoded BTCPublickey
		t.NoError(bsonencoder.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
