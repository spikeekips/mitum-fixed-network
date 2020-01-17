package key

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
)

func (t *testStellarKeypair) TestPrivatekeyJSONMarshal() {
	je := encoder.NewJSONEncoder()
	_ = hint.RegisterType(je.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((StellarPrivatekey{}).Hint().Type(), "stellar-privatekey")

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(je)
	_ = encs.AddHinter(StellarPrivatekey{})

	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	b, err := je.Encode(kp)
	t.NoError(err)
	t.Equal(`{"_hint":{"type":{"name":"stellar-privatekey","code":"0200"},"version":"0.1"},"key":"SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"}`, string(b))

	var unkp StellarPrivatekey
	t.NoError(je.Decode(b, &unkp))
	t.True(kp.Equal(unkp))
}

func (t *testStellarKeypair) TestPublickeyJSONMarshal() {
	_ = hint.RegisterType((StellarPrivatekey{}).Hint().Type(), "stellar-privatekey")
	je := encoder.NewJSONEncoder()
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(je)
	_ = encs.AddHinter(StellarPrivatekey{})

	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)
	pb := kp.Publickey()

	b, err := je.Encode(pb)
	t.NoError(err)
	t.Equal(`{"_hint":{"type":{"name":"stellar-publickey","code":"0201"},"version":"0.1"},"key":"GAVAONBETT4MVPV2IYN2T7OB7ZTYXGNN4BFGZHUYBUYR6G4ACHZMDOQ6"}`, string(b))

	var unpb StellarPublickey
	t.NoError(je.Decode(b, &unpb))
	t.True(pb.Equal(unpb))
}
