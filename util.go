package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/lonng/nano/session"
)

// 通过id获取Role
func GetRoleById(id int64) (*Role, bool) {
	role, ok := RoomMgr.Members[id]
	if ok {
		return role, true
	} else {
		return nil, false
	}
}

// 检查是否已登录
func CheckLogin(s *session.Session) bool {
	if s.UID() == 0 {
		err := s.Push("notice", NoticeMessage{Content: "请先登录"})
		if err != nil {
			log.Println("提示失败==", err)
		}
		return false
	}
	return true
}

// 返回
func Response(s *session.Session, data interface{}) {
	if err := s.Response(data); err != nil {
		log.Println("session response err:", err)
	}
}

// 获取accesstoken

// 获取 access_token 成功返回数据
type response struct {
	Errcode     int           `json:"errcode"`
	Errmsg      string        `json:"errmsg"`
	AccessToken string        `json:"access_token"`
	ExpireIn    time.Duration `json:"expires_in"`
}

func AccessToken(appID, secret string) (string, time.Duration, error) {
	urlP, err := url.Parse("https://api.weixin.qq.com/cgi-bin/token")
	if err != nil {
		return "", 0, err
	}

	query := urlP.Query()

	query.Set("appid", appID)
	query.Set("secret", secret)
	query.Set("grant_type", "client_credential")

	urlP.RawQuery = query.Encode()

	res, err := http.Get(urlP.String())
	if err != nil {
		return "", 0, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Println("====util 68 error==", err)
		}
	}()

	if res.StatusCode != 200 {
		return "", 0, errors.New("获取accesstoken失败")
	}

	var data response
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return "", 0, err
	}

	if data.Errcode != 0 {
		return "", 0, errors.New(data.Errmsg)
	}

	return data.AccessToken, time.Second * data.ExpireIn, nil
}

// 检验登录code
type Code2SessionRes struct {
	Openid     string `json:"openid"`
	SessionKey string `json:"session_key"`
	Unionid    string `json:"unionid"`
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
}

func CheckCode(code *string) (*Code2SessionRes, error) {
	urlP, err := url.Parse("https://api.weixin.qq.com/sns/jscode2session")
	if err != nil {
		return nil, err
	}

	query := urlP.Query()

	query.Set("appid", APPID)
	query.Set("secret", APPSecret)
	query.Set("js_code", *code)
	query.Set("grant_type", "authorization_code")

	urlP.RawQuery = query.Encode()

	res, err := http.Get(urlP.String())
	if err != nil {
		return nil, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Println("====util 117 error==", err)
		}
	}()

	if res.StatusCode != 200 {
		return nil, errors.New("校验js_code失败")
	}

	var data Code2SessionRes
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Errcode != 0 {
		return nil, errors.New(data.Errmsg)
	}

	return &data, nil
}
