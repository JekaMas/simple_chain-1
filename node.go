package bc

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"sort"
)

func MyNode(peer string) (*Node, error) {
	key := ed25519.PrivateKey{}
	err := ReadFromJSON(fmt.Sprintf("Keys/%s.json", peer), &key)
	if err != nil {
		return nil, err
	}

	genesis, err := LoadGenesis()
	if err != nil {
		return nil, err
	}

	return NewNode(key, genesis)
}

func SortValidators(validators []crypto.PublicKey) []ed25519.PublicKey {
	validatorsTmp := make([]ed25519.PublicKey, len(validators))

	for i, val := range validators {
		validatorsTmp[i] = val.(ed25519.PublicKey)
	}

	sort.Slice(validatorsTmp, func(i, j int) bool {
		return string(validatorsTmp[i]) < string(validatorsTmp[j])
	})

	return validatorsTmp
}