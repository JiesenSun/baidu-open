package oauth2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

var OAUTH_SERVER_URL string = "https://openapi.baidu.com/oauth/2.0/token"

const SYS_ERROR_CODE_110 string = `"error_code":110`

func SetOauthServerUrl(urlStr string) {
	OAUTH_SERVER_URL = urlStr
}

type AppInfo struct {
	AppId       int                    `json:"appid"`
	ApiKey      string                 `json:"api_key"`
	SecretKey   string                 `json:"secret_key"`
	Scope       []string               `json:"scope"`
	GrantType   string                 `json:"-"`
	AccessToken *AccessTokenType       `json:"token"`
	HttpClient  *http.Client           `json:"-"`
	ConfigPath  string                 `json:"-"`
	Attrs       map[string]interface{} `json:"attrs"`
	mu          sync.RWMutex
}

type SysResponse struct {
	RespRaw   string `json:"-"`
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

func NewApp(appid int, apiKey string, secretKey string) *AppInfo {
	return &AppInfo{
		AppId:      appid,
		ApiKey:     apiKey,
		SecretKey:  secretKey,
		GrantType:  "client_credentials",
		Scope:      make([]string, 0),
		HttpClient: http.DefaultClient,
	}
}

func NewAppByJsonFile(jsonPath string) (*AppInfo, error) {
	bs, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		glog.Warningf("read jsonFile[%s] failed,err:%s", jsonPath, err.Error())
		return nil, err
	}
	var at *AppInfo
	err = json.Unmarshal(bs, &at)
	if err != nil {
		glog.Warningf("json decode jsonFile[%s] failed,err:%s", jsonPath, err.Error())
		return nil, err
	}
	at.GrantType = "client_credentials"
	at.ConfigPath = jsonPath
	at.HttpClient = http.DefaultClient
	return at, nil
}

type AccessTokenType struct {
	TokenGetTime  time.Time `json:"token_get_time"`
	AccessToken   string    `json:"access_token"`
	ExpiresIn     int64     `json:"expires_in"`
	RefreshToken  string    `json:"refresh_token"`
	Scope         string    `json:"scope"`
	SessionKey    string    `json:"session_key"`
	SessionSecret string    `json:"session_secret"`
}

func (p *AppInfo) AddScope(scope string) {
	p.Scope = append(p.Scope, scope)
}

func (p *AppInfo) GetNewAccessToken() (*AccessTokenType, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	glog.V(2).Infoln("GetNewAccessToken start")
	defer glog.V(2).Infoln("GetNewAccessToken finish")

	glog.V(2).Infoln("server_url:", OAUTH_SERVER_URL)

	vs := make(url.Values)
	vs.Add("grant_type", p.GrantType)
	vs.Add("client_id", p.ApiKey)
	vs.Add("client_secret", p.SecretKey)
	vs.Add("scope", strings.Join(p.Scope, ","))

	qs := vs.Encode()
	glog.V(2).Infoln("get access_token querystring:", qs)

	body := bytes.NewBufferString(qs)
	req, err := http.NewRequest("POST", OAUTH_SERVER_URL, body)
	if err != nil {
		glog.Warningln("build request failed:", err)
		return nil, err
	}

	if glog.V(2) {
		dump, _ := httputil.DumpRequest(req, true)
		glog.Infoln("request_dump,", string(dump))
	}

	resp, err := p.HttpClient.Do(req)
	if err != nil {
		glog.Warningln("send request failed:", err)
		return nil, err
	}
	glog.V(2).Infoln("resp status:", resp.Status)
	defer resp.Body.Close()
	dataBs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warningln("read response failed:", err)
		return nil, err
	}
	glog.V(2).Info("response:", string(dataBs))
	var token *AccessTokenType
	err = json.Unmarshal(dataBs, &token)
	if err != nil {
		return nil, fmt.Errorf("%s,resp:%s", err.Error(), string(dataBs))
	}
	token.TokenGetTime = time.Now()
	p.AccessToken = token
	err = p.Save2File()
	return token, err
}

func (p *AppInfo) GetAccessToken() (*AccessTokenType, error) {
	if p.AccessToken == nil || p.AccessToken.ExpiresIn < 100 {
		return p.GetNewAccessToken()
	}

	if time.Now().Unix()-p.AccessToken.TokenGetTime.Unix() >= p.AccessToken.ExpiresIn {
		return p.RefreshAccessToken()
	}
	return p.AccessToken, nil
}

func (p *AppInfo) RefreshAccessToken() (*AccessTokenType, error) {
	return p.GetNewAccessToken()
}

func (p *AppInfo) Save2File() error {
	if p.ConfigPath == "" {
		return nil
	}
	bs, _ := json.MarshalIndent(p, "", "  ")
	return ioutil.WriteFile(p.ConfigPath, bs, 0644)
}

func (token *AccessTokenType) String() string {
	bs, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		log.Println("encoding faild:", err)
		return ""
	}
	return string(bs)
}

func (p *AppInfo) CleanAccessToken() {
	p.AccessToken = nil
	p.Save2File()
}

func (p *AppInfo) ExecuteApi(req *http.Request, res interface{}) error {
	apiUrl := req.URL.String()
	glog.V(2).Infoln("CallApi start:", apiUrl)
	defer glog.V(2).Infoln("CallApi finish:", apiUrl)

	tryTimes := 0

callApi:
	tryTimes++
	if tryTimes > 1 {
		return fmt.Errorf("out off tryTimes")
	}

	token, err := p.GetAccessToken()
	if err != nil {
		return err
	}
	var urlNew string
	if req.URL.RawQuery == "" {
		urlNew = apiUrl + "?access_token=" + token.AccessToken
	} else {
		urlNew = apiUrl + "&access_token=" + token.AccessToken
	}
	req.URL, err = req.URL.Parse(urlNew)
	if err != nil {
		return err
	}

	if glog.V(2) {
		dump, _ := httputil.DumpRequest(req, true)
		glog.Infoln("request_dump\n", string(dump))
	}

	resp, err := p.HttpClient.Do(req)
	if err != nil {
		glog.Warningln("send request failed:", err)
		return err
	}
	glog.V(2).Infoln("resp status:", resp.Status)

	defer resp.Body.Close()
	rbs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warningln("read response failed:", err)
		return err
	}
	glog.V(2).Infoln("response:", string(rbs))

	if bytes.Contains(rbs, []byte(SYS_ERROR_CODE_110)) {
		glog.V(2).Infoln("access_token error,retry,response:", string(rbs))
		p.CleanAccessToken()
		_, err := p.GetNewAccessToken()
		if err != nil {
			glog.Warningln()
			return err
		}
		goto callApi
	}

	err = json.Unmarshal(rbs, &res)
	if err != nil {
		glog.Warningln("json decode response failed:", err, "response:", string(rbs))
		return fmt.Errorf("%s,resp:%s", err, string(rbs))
	}

	return nil
}
