package oauth2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var OAUTH_SERVER_URL string = "https://openapi.baidu.com/oauth/2.0/token"

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
}

func NewApp(appid int, apiKey string, secretKey string) *AppInfo {
	return &AppInfo{
		AppId:     appid,
		ApiKey:    apiKey,
		SecretKey: secretKey,
		GrantType: "client_credentials",
		Scope:     make([]string, 0),
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
	glog.V(2).Infoln("GetNewAccessToken start")
	defer glog.V(2).Infoln("GetNewAccessToken finish")

	if p.AccessToken != nil && p.AccessToken.ExpiresIn > 100 {
		life := time.Now().Sub(p.AccessToken.TokenGetTime).Seconds()
		if life < float64(p.AccessToken.ExpiresIn) {
			glog.V(2).Infoln("access_token is not expired,life:", life)
			return p.AccessToken, nil
		}
	}

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
	client := p.HttpClient
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
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
	if p.AccessToken == nil {
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
