package lightservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/hidu/baidu-open/oauth2"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var GOODS_API_URL string = "https://openapi.baidu.com/rest/2.0/lightservice/goods"

const (
	GOODS_M_SHOP_LIST_GET           string = "shop.list.get"
	GOODS_M_SKU_PRICE_UPDATE               = "sku.price.update"
	GOODS_M_SKU_STOCK_UPDATE               = "sku.stock.update"
	GOODS_M_SPU_ITEM_BASE_UPDATE           = "spu.item.base.update"
	GOODS_M_SPU_ITEM_CONTENT_UPDATE        = "spu.item.content.update"
	GOODS_M_SPU_ITEM_COUNT                 = "spu.item.count"
	GOODS_M_SPU_ITEM_DELETE                = "spu.item.delete"
	GOODS_M_SPU_ITEM_DELISTING             = "spu.item.delisting"
	GOODS_M_SPU_ITEM_GET                   = "spu.item.get"
	GOODS_M_SPU_ITEM_IMAGES_UPDATE         = "spu.item.images.update"
	GOODS_M_SPU_ITEM_LIST                  = "spu.item.list"
	GOODS_M_SPU_ITEM_LISTING               = "spu.item.listing"
	GOODS_M_SPU_ITEM_PRICE_UPDATE          = "spu.item.price.update"
	GOODS_M_SPU_ITEM_SAVE                  = "spu.item.save"
	GOODS_M_SPU_ITEM_STOCK_UPDATE          = "spu.item.stock.update"
	GOODS_M_SPU_ITEM_TAGS_UPDATE           = "spu.item.tags.update"
	GOODS_M_SPU_SCHEMA_GET                 = "spu.schema.get"
	GOODS_M_TAG_ITEM_ADD                   = "tag.item.add"
	GOODS_M_TAG_ITEM_DELETE                = "tag.item.delete"
	GOODS_M_TAG_ITEM_UPDATE                = "tag.item.update"
	GOODS_M_TAG_LIST_GET                   = "tag.list.get"
)

func SetGoodsApiUrl(urlStr string) {
	GOODS_API_URL = urlStr
}

type GoodsReqType struct {
	Method string
	Data   []interface{}
	Type   string
}

type GoodsRespType struct {
	RespRaw   string        `json:"-"`
	ErrorCode int           `json:"error_code"`
	ErrorMsg  string        `json:"error_msg"`
	Data      []interface{} `json:"data"`
}

func (grt *GoodsRespType) String() string {
	bs, err := json.Marshal(grt)
	if err != nil {
		return ""
	}
	return string(bs)
}

func NewGoodsReq(method string) *GoodsReqType {
	greq := &GoodsReqType{
		Method: method,
		Data:   make([]interface{}, 0),
	}
	return greq
}

func (greq *GoodsReqType) AddData(dataItem interface{}) {
	greq.Data = append(greq.Data, dataItem)
}

func (greq *GoodsReqType) SetType(mytype string) {
	greq.Type = mytype
}

type GoodsApiType struct {
	oauth2Req  *oauth2.Oauth2Type
	HttpClient *http.Client
}

func NewGoodsApi(oauthReq *oauth2.Oauth2Type) (*GoodsApiType, error) {
	_, err := oauthReq.GetAccessToken()
	if err != nil {
		return nil, err
	}
	api := &GoodsApiType{
		oauth2Req: oauthReq,
	}
	return api, nil
}

func (api *GoodsApiType) CallApi(req *GoodsReqType) (*GoodsRespType, error) {
	glog.V(2).Infoln("CallApi start:",req.Method)
	defer glog.V(2).Infoln("CallApi finish:",req.Method)

	if api.oauth2Req == nil {
		return nil, fmt.Errorf("no access_token")
	}
	token, err := api.oauth2Req.GetAccessToken()
	if err != nil {
		return nil, err
	}
	vs := make(url.Values)
	vs.Add("method_name", req.Method)

	bs, err := json.Marshal(req.Data)
	if err != nil {
		return nil, err
	}
	vs.Add("data", string(bs))
	vs.Add("time", fmt.Sprintf("%d", time.Now().Unix()))
	if req.Type != "" {
		vs.Add("type", req.Type)
	}

	client := api.HttpClient
	if client == nil {
		client = &http.Client{}
	}
	vs.Add("access_token", token.AccessToken)
	qs := vs.Encode()

	glog.V(2).Infoln("goods_api_url:",GOODS_API_URL)
	glog.V(2).Infoln("callApi queryString:", qs)

	http_req, err := http.NewRequest("POST", GOODS_API_URL, bytes.NewBufferString(qs))
	if err != nil {
		glog.Warningln("build request failed:", err)
		return nil, err
	}

	resp, err := client.Do(http_req)
	if err != nil {
		glog.Warningln("send request failed:", err)
		return nil, err
	}
	glog.V(2).Infoln("resp status:", resp.Status)

	defer resp.Body.Close()
	rbs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warningln("read response failed:", err)
		return nil, err
	}
	glog.V(2).Infoln("response:", string(rbs))

	var respType *GoodsRespType
	err = json.Unmarshal(rbs, &respType)
	if err != nil {
		glog.Warningln("json decode response failed:", err, "response:", string(bs))
		return nil, fmt.Errorf("%s,resp:%s", err, string(bs))
	}
	respType.RespRaw = string(rbs)
	return respType, nil

}
