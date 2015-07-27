package goods

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

var API_URL string = "https://openapi.baidu.com/rest/2.0/lightservice/goods"

func SetApiUrl(urlStr string) {
	API_URL = urlStr
}

type Request struct {
	Method string
	Data   []interface{}
	Type   string
}

type Response struct {
	RespRaw   string        `json:"-"`
	ErrorCode int           `json:"error_code"`
	ErrorMsg  string        `json:"error_msg"`
	Data      []interface{} `json:"data"`
}

func (grt *Response) String() string {
	bs, err := json.Marshal(grt)
	if err != nil {
		return ""
	}
	return string(bs)
}

func NewRequest(method string) *Request {
	greq := &Request{
		Method: method,
		Data:   make([]interface{}, 0),
	}
	return greq
}

func (greq *Request) AddData(dataItem interface{}) {
	greq.Data = append(greq.Data, dataItem)
}

func (greq *Request) SetData(data interface{}) {
	greq.Data = data.([]interface{})
}

func (greq *Request) SetType(mytype string) {
	greq.Type = mytype
}

type Api struct {
	appInfo    *oauth2.AppInfo
	HttpClient *http.Client
}

func NewApi(appInfo *oauth2.AppInfo) (*Api, error) {
	_, err := appInfo.GetAccessToken()
	if err != nil {
		return nil, err
	}
	api := &Api{
		appInfo: appInfo,
	}
	return api, nil
}

func (api *Api) CallApi(req *Request) (*Response, error) {
	glog.V(2).Infoln("CallApi start:", req.Method)
	defer glog.V(2).Infoln("CallApi finish:", req.Method)

	if api.appInfo == nil {
		return nil, fmt.Errorf("no access_token")
	}
	token, err := api.appInfo.GetAccessToken()
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

	glog.V(2).Infoln("goods_api_url:", API_URL)
	glog.V(2).Infoln("callApi queryString:", qs)

	http_req, err := http.NewRequest("POST", API_URL, bytes.NewBufferString(qs))
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

	var respType *Response
	err = json.Unmarshal(rbs, &respType)
	if err != nil {
		glog.Warningln("json decode response failed:", err, "response:", string(bs))
		return nil, fmt.Errorf("%s,resp:%s", err, string(bs))
	}
	respType.RespRaw = string(rbs)
	return respType, nil

}
