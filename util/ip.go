package util

import (
	"encoding/json"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const localhost = "127.0.0.1"

var dbFile = filepath.Join(AppPath(), "file/ip2region.xdb")
var searcher *xdb.Searcher

var regionFile = filepath.Join(AppPath(), "file/regions.json")
var regionMap map[string]Region

type IpRegion struct {
	// 默认的 region 信息都固定了格式：国家|区域|省份|城市|ISP，缺省的地域信息默认是0。
	Country  string
	Region   string
	Province string
	City     string
	Isp      string
	Ip       string
}

type Region struct {
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	Children []Region          `json:"childs,omitempty"`
	ChildMap map[string]Region `json:"childMap,omitempty"`
}

func init() {
	// 1、从 dbPath 加载 VectorIndex 缓存，把下述 vIndex 变量全局到内存里面。
	vIndex, err := xdb.LoadVectorIndexFromFile(dbFile)
	if err != nil {
		log.Println("[加载IP数据库失败]", err.Error())
		return
	}
	// 2、用全局的 vIndex 创建带 VectorIndex 缓存的查询对象。
	searcher, err = xdb.NewWithVectorIndex(dbFile, vIndex)
	if err != nil {
		log.Println("[创建VectorIndex失败]", err.Error())
		return
	}

	buff, err := os.ReadFile(regionFile)
	if err != nil {
		log.Println("[读取地区数据错误]", err.Error())
		return
	}
	var regions []Region
	if err = json.Unmarshal(buff, &regions); err != nil {
		log.Println("[地区数据解析失败]", err.Error())
		return
	}

	var tmpProvinceMap = make(map[string]Region)
	for _, province := range regions {
		var tmpCityMap = make(map[string]Region)
		for _, city := range province.Children {
			var tmpDistinctMap = make(map[string]Region)
			for _, distinct := range city.Children {
				tmpDistinctMap[distinct.Name] = distinct
			}
			city.ChildMap = tmpDistinctMap
			city.Children = nil
			tmpCityMap[city.Name] = city
		}
		province.ChildMap = tmpCityMap
		province.Children = nil
		tmpProvinceMap[province.Name] = province
	}
	regionMap = tmpProvinceMap
}

func IpAddress(ip string) IpRegion {
	var ipRegion = IpRegion{Ip: ip}
	region, err := searcher.SearchByStr(ip)
	if err != nil {
		return ipRegion
	}
	for i, s := range strings.Split(region, "|") {
		switch i {
		case 0:
			ipRegion.Country = parseZeroValue(s)
		case 1:
			ipRegion.Region = parseZeroValue(s)
		case 2:
			ipRegion.Province = parseZeroValue(s)
		case 3:
			ipRegion.City = parseZeroValue(s)
		case 4:
			ipRegion.Isp = parseZeroValue(s)
		}
	}
	return ipRegion
}

func parseZeroValue(v string) string {
	if v == "0" {
		return ""
	}
	return v
}

func IpChina(ip string) bool {
	if ip == localhost {
		return true
	}
	var address = IpAddress(ip)
	if address.Country == "" || address.Country == "中国" {
		return true
	}
	return false
}

//{
//	"Country": "",
//	"Region": "",
//	"Province": "",
//	"City": "内网IP",
//	"Isp": "内网IP",
//	"Ip": "127.0.0.1"
//}

//{
//	"Country": "中国",
//	"Region": "",
//	"Province": "北京",
//	"City": "北京市",
//	"Isp": "腾讯",
//	"Ip": "101.32.215.12"
//}

//{
//	"Country": "中国",
//	"Region": "",
//	"Province": "江苏省",
//	"City": "南京市",
//	"Isp": "电信",
//	"Ip": "58.213.40.156"
//}

//{
//	"Country": "美国",
//	"Region": "",
//	"Province": "佐治亚",
//	"City": "亚特兰大",
//	"Isp": "",
//	"Ip": "216.180.236.102"
//}

func InternetIp(ip string) bool {
	var _ip = net.ParseIP(ip)
	if _ip.IsPrivate() {
		return true
	}
	if _ip.IsLoopback() {
		return true
	}
	return false
}
