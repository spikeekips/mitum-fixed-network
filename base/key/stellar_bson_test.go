package key

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (t *testStellarKeypair) TestBSON() {
	kp, err := NewStellarPrivatekey()
	t.NoError(err)

	{
		b, err := bsonenc.Marshal(kp)
		t.NoError(err)

		var decoded StellarPrivatekey
		t.NoError(bsonenc.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bsonenc.Marshal(pub)
		t.NoError(err)

		var decoded StellarPublickey
		t.NoError(bsonenc.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
