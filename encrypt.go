package steam

import (
	"crypto/rsa"
	"math/big"
	"encoding/base64"
	"crypto/rand"
	"errors"
)

// TODO Tidy up encryptPassword function
// encryptPassword encrypts a given string using a modulus and exponent.
// This is required for Steam passwords when logging in.
func encryptPassword(password, modulus, exponent string) (string, error) {
	// Set encryption variables
	var privateKey *rsa.PrivateKey
	var publicKey rsa.PublicKey
	var plain_text, encrypted []byte

	plain_text = []byte(password)

	// Generate Private Key
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024);
	if err != nil {
		return "", err
	}

	privateKey.Precompute()

	if err := privateKey.Validate(); err != nil {
		return "", err
	}

	mod, success := new(big.Int).SetString(modulus, 16)
	if !success {
		return "", errors.New("Unable to set modulus.")
	}


	exp, success := new(big.Int).SetString(exponent, 16)
	if !success {
		return "", errors.New("Unable to set modulus.")
	}

	publicKey.N = mod
	publicKey.E = int(exp.Int64())

	encrypted, err = rsa.EncryptPKCS1v15(rand.Reader, &publicKey, plain_text)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted[0:len(encrypted)]), nil
}

/*func encryptPasswordV2(password, modulus string, exponent int64) string {
	// Set encryption variables
	var privateKey *rsa.PrivateKey
	var publicKey rsa.PublicKey
	var encrypted []byte

	// Generate Private Key
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return ""
	}

	privateKey.Precompute()

	if err := privateKey.Validate(); err != nil {
		return ""
	}

	mod, success := new(big.Int).SetString(modulus, 16)
	if !success {
		return ""
	}
	exp := new(big.Int).SetInt64(exponent)

	publicKey.N = mod
	publicKey.E = int(exp.Int64())

	encrypted, err = rsa.EncryptPKCS1v15(rand.Reader, &publicKey, []byte(password))
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(encrypted[0:len(encrypted)])
}*/