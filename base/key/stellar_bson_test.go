package key

import "go.mongodb.org/mongo-driver/bson"

func (t *testStellarKeypair) TestBSON() {
	kp, err := NewStellarPrivatekey()
	t.NoError(err)

	{
		b, err := bson.Marshal(kp)
		t.NoError(err)

		var decoded StellarPrivatekey
		t.NoError(bson.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bson.Marshal(pub)
		t.NoError(err)

		var decoded StellarPublickey
		t.NoError(bson.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
