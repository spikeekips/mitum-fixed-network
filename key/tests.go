// +build test

package key

func MustNewBTCPrivatekey() BTCPrivatekey {
	k, err := NewBTCPrivatekey()
	if err != nil {
		panic(err)
	}

	return k
}
