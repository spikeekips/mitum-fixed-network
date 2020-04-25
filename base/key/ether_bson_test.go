package key

import "go.mongodb.org/mongo-driver/bson"

func (t *testEtherKeypair) TestBSON() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	{
		b, err := bson.Marshal(kp)
		t.NoError(err)

		var decoded EtherPrivatekey
		t.NoError(bson.Unmarshal(b, &decoded))
		t.True(kp.Equal(decoded))
	}

	{
		pub := kp.Publickey()

		b, err := bson.Marshal(pub)
		t.NoError(err)

		var decoded EtherPublickey
		t.NoError(bson.Unmarshal(b, &decoded))
		t.True(pub.Equal(decoded))
	}
}
