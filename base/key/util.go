package key

func MustNewBTCPrivatekey() Privatekey {
	k, err := NewBTCPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}

func MustNewEtherPrivatekey() Privatekey {
	k, err := NewEtherPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}

func MustNewStellarPrivatekey() Privatekey {
	k, err := NewStellarPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}
