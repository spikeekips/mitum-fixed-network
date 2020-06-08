package key

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (t *testBTCKeypair) TestBSON() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	{
		b, err := bsonenc.Marshal(kp)
		t.NoError(err)

		var decoded BTCPrivatekey
		t.NoError(bsonenc.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bsonenc.Marshal(pub)
		t.NoError(err)

		var decoded BTCPublickey
		t.NoError(bsonenc.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
