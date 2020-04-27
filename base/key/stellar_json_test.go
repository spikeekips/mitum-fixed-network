package key

import (
	"github.com/spikeekips/mitum/util/encoder"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

func (t *testStellarKeypair) TestPrivatekeyJSONMarshal() {
	je := jsonencoder.NewEncoder()

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(je)
	_ = encs.AddHinter(StellarPrivatekey{})

	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	b, err := jsonencoder.Marshal(kp)
	t.NoError(err)
	t.Equal(`{"_hint":{"type":{"name":"stellar-privatekey","code":"0200"},"version":"0.0.1"},"key":"SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"}`, string(b))

	var unkp StellarPrivatekey
	t.NoError(je.Decode(b, &unkp))
	t.True(kp.Equal(unkp))
}

func (t *testStellarKeypair) TestPrivatekeyNativeJSONMarshal() {
	kp, _ := NewStellarPrivatekeyFromString("SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673")

	b, err := jsonencoder.Marshal(kp)
	t.NoError(err)

	var ukp StellarPrivatekey
	t.NoError(jsonencoder.Unmarshal(b, &ukp))
	t.True(kp.Equal(ukp))
}

func (t *testStellarKeypair) TestPublickeyJSONMarshal() {
	je := jsonencoder.NewEncoder()
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(je)
	_ = encs.AddHinter(StellarPrivatekey{})

	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)
	pb := kp.Publickey()

	b, err := jsonencoder.Marshal(pb)
	t.NoError(err)
	t.Equal(`{"_hint":{"type":{"name":"stellar-publickey","code":"0201"},"version":"0.0.1"},"key":"GAVAONBETT4MVPV2IYN2T7OB7ZTYXGNN4BFGZHUYBUYR6G4ACHZMDOQ6"}`, string(b))

	var unpb StellarPublickey
	t.NoError(je.Decode(b, &unpb))
	t.True(pb.Equal(unpb))
}

func (t *testStellarKeypair) TestPublickeyNativeJSONMarshal() {
	kp, _ := NewStellarPrivatekeyFromString("SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673")

	pb := kp.Publickey()

	b, err := jsonencoder.Marshal(pb)
	t.NoError(err)

	var upb StellarPublickey
	t.NoError(jsonencoder.Unmarshal(b, &upb))
	t.True(pb.Equal(upb))
}
