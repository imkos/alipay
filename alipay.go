package alipay

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/imkos/alipay/encoding"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

var (
	RSA  = &RSA_sign{sign_type: K_SIGN_TYPE_RSA, hash: crypto.SHA1}
	RSA2 = &RSA_sign{sign_type: K_SIGN_TYPE_RSA2, hash: crypto.SHA256}
)

type AliPay struct {
	appId     string
	apiDomain string
	partnerId string
	client    *http.Client
	Signer    AliSign
}

//此方法一般一个开发者对象只需调用一次，调用后全局的RSA,RSA2的sig都会同样初始
func NewAliPay(s_appId, s_partnerId string, sg *encoding.SignPKCS, isProduction bool) (*AliPay, error) {
	if sg == nil {
		return nil, errors.New("*SignPKCS is nil")
	}
	cli := &AliPay{
		appId:     s_appId,
		partnerId: s_partnerId,
		client:    &http.Client{},
	}
	if isProduction {
		cli.apiDomain = K_ALI_PAY_PRODUCTION_API_URL
	} else {
		cli.apiDomain = K_ALI_PAY_SANDBOX_API_URL
	}
	RSA.sig = sg
	RSA2.sig = sg
	//默认采用RSA2, 如需使用RSA,在New..后面重新指定RSA
	cli.Signer = RSA2
	return cli, nil
}

func (ap *AliPay) URLValues(param AliPayParam) (value url.Values, err error) {
	p := url.Values{}
	p.Add("app_id", ap.appId)
	p.Add("method", param.APIName())
	p.Add("format", K_FORMAT)
	p.Add("charset", K_CHARSET)
	p.Add("sign_type", ap.Signer.Get_Signtype())
	p.Add("timestamp", time.Now().Format(K_TIME_FORMAT))
	p.Add("version", K_VERSION)
	if len(param.ExtJSONParamName()) > 0 {
		p.Add(param.ExtJSONParamName(), param.ExtJSONParamValue())
	}
	ps := param.Params()
	if ps != nil {
		for key, value := range ps {
			p.Add(key, value)
		}
	}
	s_keys := make([]string, 0)
	for key := range p {
		s_keys = append(s_keys, key)
	}
	sort.Strings(s_keys)
	sign, err := ap.Signer.Sign(s_keys, p)
	if err != nil {
		return nil, err
	}
	p.Add("sign", sign)
	return p, nil
}

func (ap *AliPay) doRequest(method string, param AliPayParam, results interface{}) (err error) {
	if param == nil {
		return errors.New("AliPayParam is nil!")
	}
	p, err := ap.URLValues(param)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, ap.apiDomain, strings.NewReader(p.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := ap.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if ap.Signer.CanVerify() {
		rootNodeName := strings.Replace(param.APIName(), ".", "_", -1) + k_RESPONSE_SUFFIX
		gj := gjson.ParseBytes(data)
		if err := ap.Signer.VerifyResponseData([]byte(gj.Get(rootNodeName).Raw), gj.Get("sign").Str); err != nil {
			return err
		}
	}
	return json.Unmarshal(data, results)
}

func (ap *AliPay) DoRequest(method string, param AliPayParam, results interface{}) (err error) {
	return ap.doRequest(method, param, results)
}

//AliPay签名
type AliSign interface {
	Get_Signtype() string
	Sign(keys []string, param url.Values) (string, error)
	CanVerify() bool
	VerifyResponseData(data []byte, sign string) error
}

type RSA_sign struct {
	sig       *encoding.SignPKCS
	sign_type string
	hash      crypto.Hash
}

func (r *RSA_sign) Get_Signtype() string {
	return r.sign_type
}

func (r *RSA_sign) Sign(keys []string, param url.Values) (string, error) {
	if r.sig == nil {
		return "", errors.New("*SignPKCS is nil!")
	}
	//如两个参数出现空值直接返回
	if keys == nil || param == nil {
		return "", nil
	}
	pList := make([]string, 0, 0)
	for _, key := range keys {
		value := strings.TrimSpace(param.Get(key))
		if len(value) > 0 {
			pList = append(pList, key+"="+value)
		}
	}
	src := strings.Join(pList, "&")
	sig, err := r.sig.SignPKCS1v15([]byte(src), r.hash)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func (p *RSA_sign) CanVerify() bool {
	if p.sig == nil {
		return false
	}
	return p.sig.CanVerify()
}

func (r *RSA_sign) VerifyResponseData(data []byte, sign string) error {
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return err
	}
	return r.sig.VerifyPKCS1v15(data, signBytes, r.hash)
}

func verifySign(req *http.Request) (ok bool, err error) {
	sign, err := base64.StdEncoding.DecodeString(req.PostForm.Get("sign"))
	signType := req.PostForm.Get("sign_type")
	if err != nil {
		return false, err
	}
	keys := make([]string, 0)
	for key, value := range req.PostForm {
		if key == "sign" || key == "sign_type" {
			continue
		}
		if len(value) > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	pList := make([]string, 0)
	for _, key := range keys {
		value := strings.TrimSpace(req.PostForm.Get(key))
		if len(value) > 0 {
			pList = append(pList, key+"="+value)
		}
	}
	s := strings.Join(pList, "&")
	if signType == K_SIGN_TYPE_RSA {
		if RSA.CanVerify() {
			err = RSA.sig.VerifyPKCS1v15([]byte(s), sign, crypto.SHA1)
		}
	} else {
		if RSA2.CanVerify() {
			err = RSA2.sig.VerifyPKCS1v15([]byte(s), sign, crypto.SHA256)
		}
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
