package steam

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
)

// TODO Tidy up encryptPassword function
// encryptPassword encrypts a given string using a modulus and exponent.
func encryptPassword(password, modulus, exponent string) string {
	// Set encryption variables
	var privateKey *rsa.PrivateKey
	var publicKey rsa.PublicKey
	var plain_text, encrypted []byte

	plain_text = []byte(password)

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

	exp, success := new(big.Int).SetString(exponent, 16)
	if !success {
		return ""
	}

	publicKey.N = mod
	publicKey.E = int(exp.Int64())

	encrypted, err = rsa.EncryptPKCS1v15(rand.Reader, &publicKey, plain_text)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(encrypted[0:len(encrypted)])
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
