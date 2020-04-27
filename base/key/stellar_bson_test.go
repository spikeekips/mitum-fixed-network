package key

import (
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (t *testStellarKeypair) TestBSON() {
	kp, err := NewStellarPrivatekey()
	t.NoError(err)

	{
		b, err := bsonencoder.Marshal(kp)
		t.NoError(err)

		var decoded StellarPrivatekey
		t.NoError(bsonencoder.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bsonencoder.Marshal(pub)
		t.NoError(err)

		var decoded StellarPublickey
		t.NoError(bsonencoder.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
