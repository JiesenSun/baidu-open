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

type Oauth2Type struct {
	AppId       int
	ApiKey      string
	SecretKey   string
	Scope       []string
	GrantType   string
	AccessToken *AccessTokenType
	HttpClient  *http.Client
}

func NewOauth2(appid int, apiKey string, secretKey string) *Oauth2Type {
	return &Oauth2Type{
		AppId:     appid,
		ApiKey:    apiKey,
		SecretKey: secretKey,
		GrantType: "client_credentials",
		Scope:     make([]string, 0),
	}
}

type AccessTokenType struct {
	TokenGetTime  time.Time `json:"-"`
	AccessToken   string    `json:"access_token"`
	ExpiresIn     int64     `json:"expires_in"`
	RefreshToken  string    `json:"refresh_token"`
	Scope         string    `json:"scope"`
	SessionKey    string    `json:"session_key"`
	SessionSecret string    `json:"session_secret"`
}

func (p *Oauth2Type) AddScope(scope string) {
	p.Scope = append(p.Scope, scope)
}

func (p *Oauth2Type) GetNewAccessToken() (*AccessTokenType, error) {

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
	glog.V(2).Info("response:",string(dataBs))
	var token *AccessTokenType
	err = json.Unmarshal(dataBs, &token)
	if err != nil {
		return nil, fmt.Errorf("%s,resp:%s", err.Error(), string(dataBs))
	}
	token.TokenGetTime = time.Now()
	p.AccessToken = token
	return token, nil
}

func (p *Oauth2Type) GetAccessToken() (*AccessTokenType, error) {
	if p.AccessToken == nil {
		return p.GetNewAccessToken()
	}

	if time.Now().Unix()-p.AccessToken.TokenGetTime.Unix() >= p.AccessToken.ExpiresIn {
		return p.RefreshAccessToken()
	}
	return p.AccessToken, nil
}

func (p *Oauth2Type) RefreshAccessToken() (*AccessTokenType, error) {
	return p.GetNewAccessToken()
}

func (token *AccessTokenType) String() string {
	bs, err := json.Marshal(token)
	if err != nil {
		log.Println("encoding faild:", err)
		return ""
	}
	return string(bs)
}
