package key

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (t *testEtherKeypair) TestBSON() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	{
		b, err := bsonenc.Marshal(kp)
		t.NoError(err)

		var decoded EtherPrivatekey
		t.NoError(bsonenc.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bsonenc.Marshal(pub)
		t.NoError(err)

		var decoded EtherPublickey
		t.NoError(bsonenc.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
