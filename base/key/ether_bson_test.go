package key

import (
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (t *testEtherKeypair) TestBSON() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	{
		b, err := bsonencoder.Marshal(kp)
		t.NoError(err)

		var decoded EtherPrivatekey
		t.NoError(bsonencoder.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bsonencoder.Marshal(pub)
		t.NoError(err)

		var decoded EtherPublickey
		t.NoError(bsonencoder.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
