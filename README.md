AliPay SDK for Golang


## 鸣谢

感谢下列人员对本项目的支持：

[@wusphinx](https://github.com/wusphinx)

[@clearluo](https://github.com/clearluo)

[@zwh8800](https://github.com/zwh8800) 

## 已实现接口

#### 手机网站支付API

* **手机网站支付接口**

  alipay.trade.wap.pay

* **电脑网站支付**

  alipay.trade.page.pay

* **统一收单线下交易查询**

  alipay.trade.query

* **统一收单交易支付接口**

  alipay.trade.pay

* **统一收单交易创建接口**

  alipay.trade.create

* **统一收单线下交易预创建**

  alipay.trade.precreate

* **统一收单交易撤销接口**

  alipay.trade.cancel

* **统一收单交易关闭接口**

  alipay.trade.close

* **统一收单交易退款接口**

  alipay.trade.refund

* **App支付接口**

  alipay.trade.app.pay

* **统一收单交易退款查询**

  alipay.trade.fastpay.refund.query

* **单笔转账到支付宝账户接口**

  alipay.fund.trans.toaccount.transfer

* **查询转账订单接口**

  alipay.fund.trans.order.query 

#### 通知

* **通知内容转换及签名验证**

  将支付宝的通知内容转换为 Golang 的结构体，并且验证其合法性。

## 集成流程

从[支付宝开放平台](https://open.alipay.com/)申请创建相关的应用，使用自己的支付宝账号登录即可。

#### 沙箱环境

支付宝开放平台为每一个应用提供了沙箱环境，供开发人员开发测试使用。

沙箱环境是独立的，每一个应用都会有一个商家账号和买家账号。

#### 应用信息配置

参考[官网文档](https://doc.open.alipay.com/docs/doc.htm?spm=a219a.7629140.0.0.5pgfxp&treeId=200&articleId=105894&docType=1) 进行应用的配置。

本 SDK 中的签名方法为 RSA2，所以请注意配置 **RSA2(SHA256)密钥**。

请参考 [如何生成 RSA 密钥](https://doc.open.alipay.com/docs/doc.htm?treeId=291&articleId=105971&docType=1)。

#### Example:

``` go
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/imkos/alipay"
	"github.com/imkos/alipay/encoding"
	"os"
)

const (
	PRI_BEGIN = "-----BEGIN RSA PRIVATE KEY-----"
	PRI_END   = "-----END RSA PRIVATE KEY-----"
	//
	PUB_BEGIN = "-----BEGIN RSA PUBLIC KEY-----"
	PUB_END   = "-----END RSA PUBLIC KEY-----"
)

//判断文件或文件夹是否存在
func Exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func loadfromfile(s_file string, b_bstr, b_estr []byte) []byte {
	b_file, e1 := os.Open(s_file)
	if e1 != nil {
		return nil
	}
	defer b_file.Close()
	br := bufio.NewReader(b_file)
	buf := bytes.NewBuffer(b_bstr)
	buf.WriteByte(10)
	buf_line := make([]byte, 64)
	for {
		n, err := br.Read(buf_line)
		if err != nil {
			break
		}
		if n < 64 {
			buf.Write(buf_line[:n])
		} else {
			buf.Write(buf_line)
		}
		buf.WriteByte(10)
	}
	buf.Write(b_estr)
	return buf.Bytes()
}

func prikfromfile(s_file string) []byte {
	if !Exist(s_file) {
		return nil
	}
	return loadfromfile(s_file, []byte(PRI_BEGIN), []byte(PRI_END))
}

func pubkfromfile(s_file string) []byte {
	if !Exist(s_file) {
		return nil
	}
	return loadfromfile(s_file, []byte(PUB_BEGIN), []byte(PUB_END))
}

func main() {
	sig, e0 := encoding.NewSignPKCS(prikfromfile("prik.txt"), pubkfromfile("alipubk.txt"), 8)
	if e0 != nil {
		fmt.Println(e0)
		return
	}
	ap, e1 := alipay.NewAliPay("2017121800******", "", sig, true)
	if e1 != nil {
		fmt.Println(e1)
		return
	}
	gdi := &alipay.GoodsDetailItem{
		GoodsId:   "10002001",
		GoodsName: "文具",
		Quantity:  1,
		Price:     0.01,
	}
	pay_param := &alipay.AliPayTradePay{
		OutTradeNo:     "20171218010101001",
		Scene:          "bar_code",
		AuthCode:       "2871131954465*****",
		Subject:        "自营收款",
		Body:           "trade pay test",
		TimeoutExpress: "90m",
		GoodsDetail:    []*alipay.GoodsDetailItem{gdi},
	}
	var pay_resp alipay.AliPayTradePayResponse
	if e2 := ap.DoRequest("POST", pay_param, &pay_resp); e2 != nil {
		fmt.Println(e2)
		return
	}
	fmt.Println(pay_resp)
}
```

#### 同步返回验签

支持自动对支付宝返回的数据进行签名验证，详细信息请参考[自行实现验签](https://doc.open.alipay.com/docs/doc.htm?docType=1&articleId=106120).

如果需要开启自动验签，只需要在初始化 AliPay 对象之后给 **AliPayPublicKey** 属性设置从支付宝管理后台获取到的支付宝公钥即可，如下：

``` Golang
encoding.NewSignPKCS（...） 初始时可以对公钥进行赋nil,表示不验回调或返回的sign
```

#### 验证支付结果

有支付或者其它动作发生后，支付宝服务器会调用我们提供的 Notify URL，并向其传递会相关的信息。参考[手机网站支付结果异步通知](https://doc.open.alipay.com/docs/doc.htm?spm=a219a.7629140.0.0.XM5C4a&treeId=203&articleId=105286&docType=1)。

我们需要在提供的 Notify URL 服务中获取相关的参数并进行验证:

```Golang

var client = alipay.New(appId, partnerId, publickKey, privateKey, false)
client.AliPayPublicKey = xxx // 从支付宝管理后台获取支付宝提供的公钥
 
http.HandleFunc("/alipay", func(rep http.ResponseWriter, req *http.Request) {
	var noti, _ = client.GetTradeNotification(req)
	if noti != nil {
		fmt.Println("支付成功")
	} else {
		fmt.Println("支付失败")
	}
})
```

此验证方法适用于支付宝所有情况下发送的 Notify，不管是手机 App 支付还是 Wap 支付。

**需要特别注意，从支付宝后台获取到支付宝的公钥之后，需要将其转换成标准的公钥格式，如下所示：**

```
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2MhEVUp+rRRyAD9HZfiS
g8LLxRAX18XOMJE8/MNnlSSTWCCoHnM+FIU+AfB+8FE+gGIJYXJlpTIyWn4VUMte
wh/4C8uwzBWod/3ilw9Uy7lFblXDBd8En8a59AxC6c9YL1nWD7/sh1szqej31VRI
2OXQSYgvhWNGjzw2/KS1GdrWmdsVP2hOiKVy6TNtH7XnCSRfBBCQ+LgqO1tE0NHD
DswRwBLAFmIlfZ//qZ+a8FvMc//sUm+CV78pQba4nnzsmh10fzVVFIWiKw3VDsxX
PRrAtOJCwNsBwbvMuI/ictvxxjUl4nBZDw4lXt5eWWqBrnTSzogFNOk06aNmEBTU
hwIDAQAB
-----END PUBLIC KEY-----
```

#### 支持 RSA 签名及验证
默认采用的是 RSA2 签名，如果需要使用 RSA 签名，只需要在初始化 AliPay 的时候，将其 Signer设置为 alipay.RSA 即可:

```Golang
var client = alipay.NewAliPay(....)
client.Signer = alipay.RSA
```

当然，相关的 Key 也要注意替换。