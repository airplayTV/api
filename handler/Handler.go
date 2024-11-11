package handler

import (
	"errors"
	"fmt"
	"github.com/airplayTV/api/model"
	"github.com/airplayTV/api/util"
	"github.com/bytecodealliance/wasmtime-go/v25"
	"github.com/tidwall/gjson"
	"github.com/zc310/headers"
	"log"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Handler struct {
	httpClient util.HttpClient
}

func (x *Handler) parsePageNumber(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	if n <= 0 {
		return 1
	}
	return n
}

func (x *Handler) simpleRegEx(plainText, regex string) string {
	//regEx := regexp.MustCompile(`(\d+)`)
	regEx := regexp.MustCompile(regex)
	tmpList := regEx.FindStringSubmatch(plainText)
	if len(tmpList) < 2 {
		return ""
	}
	return tmpList[1]
}

func (x *Handler) simpleRegExList(plainText, regex string) []string {
	regEx := regexp.MustCompile(regex)
	tmpList := regEx.FindStringSubmatch(plainText)
	if len(tmpList) < 2 {
		return nil
	}
	return tmpList
}

func (x *Handler) parseVideoType(sourceUrl string) string {
	if strings.Contains(sourceUrl, ".m3u8") {
		return sourceTypeHLS
	}
	if strings.HasSuffix(sourceUrl, "m3u8") {
		return sourceTypeHLS
	}
	if strings.HasSuffix(sourceUrl, ".mp4") {
		return sourceTypeAuto
	}

	return ""
}

func (x *Handler) requestUrlBypassSafeLineChallenge(requestUrl string) ([]byte, error) {
	header, buff, err := x.httpClient.GetResponse(requestUrl)
	if err != nil {
		return nil, err
	}
	var clientId = x.simpleRegEx(string(buff), `SafeLineChallenge\("(\S+)",`)
	if len(clientId) == 0 {
		return buff, nil
	}
	log.Println("[SafeLineChallenge]", clientId)

	var tmpSession = x.simpleRegEx(header.Get(headers.SetCookie), `sl-session=(\S+);`)

	var httpClient = util.HttpClient{}
	httpClient.SetHeaders(x.httpClient.GetHeaders())

	var postIssueForm = map[string]interface{}{
		"client_id": clientId,
		"level":     1,
	}
	httpClient.AddHeader("Content-Type", "application/json")
	buff, err = httpClient.Post("https://challenge.rivers.chaitin.cn/challenge/v2/api/issue", util.ToString(postIssueForm))
	if err != nil {
		//log.Println("[发生数据失败]", err.Error())
		return nil, err
	}

	var jsonResult = gjson.ParseBytes(buff)
	var intList = make([]int, 0)
	jsonResult.Get("data").Get("data").ForEach(func(key, value gjson.Result) bool {
		intList = append(intList, int(value.Int()))
		return true
	})

	challenge, err := x.SafeLineChallengeWasmCalc(intList)
	if err != nil {
		//log.Println("[ChallengeError]", err.Error())
		return nil, err
	}

	var postIssueForm2 = map[string]interface{}{
		"issue_id": jsonResult.Get("data").Get("issue_id").String(),
		"result":   challenge,
		"serials":  make([]string, 0),
		"client": map[string]interface{}{
			"userAgent": useragent,
			"platform":  "Win32",
			"language":  "zh,en-GB,en-US,en,zh-CN",
			"vendor":    "Google Inc.",
			"screen":    []int{2560, 1440},
			"score":     0,
		},
	}

	httpClient.AddHeader("Content-Type", "application/json")
	buff, err = httpClient.Post("https://challenge.rivers.chaitin.cn/challenge/v2/api/verify", util.ToString(postIssueForm2))
	if err != nil {
		log.Println("[发生数据失败]", err.Error())
		return nil, err
	}

	var jsonVerify = gjson.ParseBytes(buff)

	if !jsonVerify.Get("data").Get("verified").Bool() {
		//log.Println("【解析失败】")
		return nil, errors.New("SafeLine挑战验证失败")
	}

	httpClient.AddHeader(headers.Cookie, fmt.Sprintf("sl-session=%s; sl-challenge-jwt=%s;", tmpSession, jsonVerify.Get("data").Get("jwt")))
	header, buff, err = httpClient.GetResponse(requestUrl)
	if err != nil {
		log.Println("[请求失败]", err.Error())
		return nil, err
	}

	// 这个cookie是持久的
	x.httpClient.AddHeader(headers.Cookie, header.Get(headers.SetCookie))

	return buff, nil
}

func (x *Handler) SafeLineChallengeWasmCalc(data []int) ([]int, error) {
	config := wasmtime.NewConfig()
	config.SetConsumeFuel(true)
	engine := wasmtime.NewEngineWithConfig(config)
	store := wasmtime.NewStore(engine)
	err := store.SetFuel(10000)
	if err != nil {
		return nil, err
	}
	module, err := wasmtime.NewModuleFromFile(engine, path.Join(util.AppPath(), "file", "calc.wasm"))
	if err != nil {
		log.Println("[获取长亭 WAF wasm 文件失败]")
		return nil, err
	}
	instance, err := wasmtime.NewInstance(store, module, []wasmtime.AsExtern{})
	if err != nil {
		return nil, err
	}
	fnReset := instance.GetFunc(store, "reset")
	if fnReset == nil {
		return nil, err
	}
	fnArg := instance.GetFunc(store, "arg")
	if fnArg == nil {
		return nil, err
	}
	fnCalc := instance.GetFunc(store, "calc")
	if fnCalc == nil {
		return nil, err
	}
	fnRet := instance.GetFunc(store, "ret")
	if fnRet == nil {
		return nil, err
	}
	output, err := fnReset.Call(store)
	if err != nil {
		return nil, err
	}
	for _, v := range data {
		_, err = fnArg.Call(store, v)
		if err != nil {
			return nil, err
		}
	}
	output, err = fnCalc.Call(store)
	if err != nil {
		return nil, err
	}

	switch output.(type) {
	case int32:
		var resp = make([]int, 0)
		for i := 0; i < int(output.(int32)); i++ {
			output7, err := fnRet.Call(store)
			if err != nil {
				return nil, err
			}
			resp = append(resp, int(output7.(int32)))
		}

		return resp, nil
	default:
		return nil, errors.New("wasm调用返回异常")
	}
}

func (x *Handler) handleM3u8pUrl(tmpUrl string) string {
	parsed, err := url.Parse(tmpUrl)
	if err != nil {
		return tmpUrl
	}
	if !slices.Contains(model.M3u8ProxyHosts, parsed.Host) {
		return tmpUrl
	}
	return fmt.Sprintf("https://airplay-tv.lixiang4u.xyz/api/m3u8p?url=%s", util.EncodeComponentUrl(tmpUrl))
}
