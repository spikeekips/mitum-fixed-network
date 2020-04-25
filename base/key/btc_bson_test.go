package key

import "go.mongodb.org/mongo-driver/bson"

func (t *testBTCKeypair) TestBSON() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	{
		b, err := bson.Marshal(kp)
		t.NoError(err)

		var decoded BTCPrivatekey
		t.NoError(bson.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bson.Marshal(pub)
		t.NoError(err)

		var decoded BTCPublickey
		t.NoError(bson.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
