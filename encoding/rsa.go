package encoding

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

func packageData(originalData []byte, packageSize int) (r [][]byte) {
	var src = make([]byte, len(originalData))
	copy(src, originalData)

	r = make([][]byte, 0)
	if len(src) <= packageSize {
		return append(r, src)
	}
	for len(src) > 0 {
		var p = src[:packageSize]
		r = append(r, p)
		src = src[packageSize:]
		if len(src) <= packageSize {
			r = append(r, src)
			break
		}
	}
	return r
}

func RSAEncrypt(plaintext, key []byte) ([]byte, error) {
	var err error
	var block *pem.Block
	block, _ = pem.Decode(key)
	if block == nil {
		return nil, errors.New("public key error")
	}

	var pubInterface interface{}
	pubInterface, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	var pub = pubInterface.(*rsa.PublicKey)

	var data = packageData(plaintext, pub.N.BitLen()/8-11)
	var cipherData []byte = make([]byte, 0, 0)

	for _, d := range data {
		var c, e = rsa.EncryptPKCS1v15(rand.Reader, pub, d)
		if e != nil {
			return nil, e
		}
		cipherData = append(cipherData, c...)
	}

	return cipherData, nil
}

func RSADecrypt(ciphertext, key []byte) ([]byte, error) {
	var err error
	var block *pem.Block
	block, _ = pem.Decode(key)
	if block == nil {
		return nil, errors.New("private key error")
	}

	var pri *rsa.PrivateKey
	pri, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	var data = packageData(ciphertext, pri.PublicKey.N.BitLen()/8)
	var plainData []byte = make([]byte, 0, 0)

	for _, d := range data {
		var p, e = rsa.DecryptPKCS1v15(rand.Reader, pri, d)
		if e != nil {
			return nil, e
		}
		plainData = append(plainData, p...)
	}
	return plainData, nil
}

type SignPKCS struct {
	pri *rsa.PrivateKey
	pub *rsa.PublicKey
}

//私钥负责签名，公钥负责验证
//pri_key: 用于请求alipay的请求中的sign签名计算，而把配对的公钥要填到开发者帐号中去给阿里校检请求
//b_AliPayPubKey: 用于校检来自于请求返回值与回调的请求内容校检,此公钥来自于支付宝公钥,非自己生成的
//key_mod: 密钥格式 PKCS8 = 8, 其他为PKCS1
func NewSignPKCS(pri_key, b_AliPayPubKey []byte, pri_mode uint8) (*SignPKCS, error) {
	//解析私钥
	pri_block, _ := pem.Decode(pri_key)
	if pri_block == nil {
		return nil, errors.New("pem.Decode fail!")
	}
	var pri *rsa.PrivateKey
	if pri_mode == 8 {
		pcs8, err := x509.ParsePKCS8PrivateKey(pri_block.Bytes)
		if err != nil {
			return nil, err
		}
		pri = pcs8.(*rsa.PrivateKey)
	} else {
		var e2 error
		pri, e2 = x509.ParsePKCS1PrivateKey(pri_block.Bytes)
		if e2 != nil {
			return nil, e2
		}
	}
	//解析公钥并组合SignPKCS
	if b_AliPayPubKey != nil {
		pub_block, _ := pem.Decode(b_AliPayPubKey)
		if pub_block == nil {
			return nil, errors.New("public key error")
		}
		pubIntf, e1 := x509.ParsePKIXPublicKey(pub_block.Bytes)
		if e1 != nil {
			return nil, e1
		}
		return &SignPKCS{pri, pubIntf.(*rsa.PublicKey)}, nil
	} else {
		return &SignPKCS{pri, nil}, nil
	}
}

func (p *SignPKCS) SignPKCS1v15(src []byte, hash crypto.Hash) ([]byte, error) {
	h := hash.New()
	h.Write(src)
	hashed := h.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, p.pri, hash, hashed)
}

func (p *SignPKCS) CanVerify() bool {
	return p.pub != nil
}

func (p *SignPKCS) VerifyPKCS1v15(src, sig []byte, hash crypto.Hash) error {
	h := hash.New()
	h.Write(src)
	hashed := h.Sum(nil)
	return rsa.VerifyPKCS1v15(p.pub, hash, hashed, sig)
}
