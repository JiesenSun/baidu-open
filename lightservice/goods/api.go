package goods

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/hidu/baidu-open/oauth2"
	"net/http"
	"net/url"
	"time"
)

var API_URL string = "https://openapi.baidu.com/rest/2.0/lightservice/goods"

func SetApiUrl(urlStr string) {
	API_URL = urlStr
}

type Api struct {
	Method string
	Data   []interface{}
	Type   string
	app    *oauth2.AppInfo
}

type Response struct {
	RespRaw   string        `json:"-"`
	ErrorCode int           `json:"error_code"`
	ErrorMsg  string        `json:"error_msg"`
	Data      []interface{} `json:"data"`
	RequestID string        `json:"request_id"`
}

func (grt *Response) String() string {
	bs, err := json.Marshal(grt)
	if err != nil {
		return ""
	}
	return string(bs)
}

func NewApi(app *oauth2.AppInfo, method string) *Api {
	greq := &Api{
		Method: method,
		Data:   make([]interface{}, 0),
		app:    app,
	}
	return greq
}

func (greq *Api) AddData(dataItem interface{}) {
	greq.Data = append(greq.Data, dataItem)
}

func (greq *Api) SetData(data interface{}) {
	greq.Data = data.([]interface{})
}

func (greq *Api) SetType(mytype string) {
	greq.Type = mytype
}

func (api *Api) BuildRequest() (*http.Request, error) {

	vs := make(url.Values)
	vs.Add("method_name", api.Method)

	bs, err := json.Marshal(api.Data)
	if err != nil {
		return nil, err
	}
	vs.Add("data", string(bs))
	vs.Add("time", fmt.Sprintf("%d", time.Now().Unix()))
	if api.Type != "" {
		vs.Add("type", api.Type)
	}
	qs := vs.Encode()

	glog.V(2).Infoln("goods_api_url:", API_URL)
	glog.V(2).Infoln("callApi queryString:", qs)

	http_req, err := http.NewRequest("POST", API_URL, bytes.NewBufferString(qs))
	if err != nil {
		glog.Warningln("build ApiRequest failed:", err)
		return nil, err
	}

	return http_req, err
}

func (api *Api) Execute() (resp *Response, err error) {
	if api.app == nil {
		return nil, fmt.Errorf("no app info")
	}
	req, err := api.BuildRequest()
	if err != nil {
		return nil, err
	}
	err = api.app.ExecuteApi(req, &resp)
	return resp, err
}
