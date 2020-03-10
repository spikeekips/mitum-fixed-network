package key

import (
	"encoding/hex"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
)

func (t *testStellarKeypair) TestPrivatekeyBSONMarshal() {
	be := encoder.NewBSONEncoder()
	_ = hint.RegisterType(be.Hint().Type(), "bson-encoder")
	_ = hint.RegisterType((StellarPrivatekey{}).Hint().Type(), "stellar-privatekey")

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(be)
	_ = encs.AddHinter(StellarPrivatekey{})

	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	b, err := be.Encode(kp)
	t.NoError(err)
	t.Equal("76000000035f68696e74001c0000000574000200000000020002760006000000302e302e310000035f646174610047000000026b657900390000005343443647514d57474451543333514f434e4b594b524a5a4c33595746534c425651534c3649435657425559515a43424659555159363733000000", hex.EncodeToString(b))

	var unkp StellarPrivatekey
	t.NoError(be.Decode(b, &unkp))

	t.True(kp.Equal(unkp))
}

func (t *testStellarKeypair) TestPublickeyBSONMarshal() {
	be := encoder.NewBSONEncoder()
	_ = hint.RegisterType(be.Hint().Type(), "bson-encoder")
	_ = hint.RegisterType((StellarPrivatekey{}).Hint().Type(), "stellar-privatekey")

	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(be)
	_ = encs.AddHinter(StellarPrivatekey{})

	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)
	pb := kp.Publickey()

	b, err := be.Encode(pb)
	t.NoError(err)
	t.Equal("76000000035f68696e74001c0000000574000200000000020102760006000000302e302e310000035f646174610047000000026b65790039000000474156414f4e42455454344d5650563249594e3254374f42375a545958474e4e344246475a485559425559523647344143485a4d444f5136000000", hex.EncodeToString(b))

	var unpb StellarPublickey
	t.NoError(be.Decode(b, &unpb))
	t.True(pb.Equal(unpb))
}
