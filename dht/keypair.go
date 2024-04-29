package dht

import (
	"golang.org/x/crypto/ed25519"
)

type KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
}

func (k *KeyPair) Id() []byte {
	return k.PublicKey
}

func GenerateKeyPair(seed []byte) (KeyPair, error) {
	if len(seed) > 0 {
		priv := ed25519.NewKeyFromSeed(seed)
		pub := priv.Public().(ed25519.PublicKey)
		k := KeyPair{
			PrivateKey: priv,
			PublicKey:  pub,
		}
		return k, nil
	}
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	k := KeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}
	return k, err
}
