package handler

const (
	sourceTypeAuto = "auto"
	sourceTypeHLS  = "hls"
	useragent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
)

const apiM3U8ProxyUrl = "https://airplay-api.artools.cc/api/m3u8p"
const ApiM3U8ProxyUrl = apiM3U8ProxyUrl

const (
	czzyHost      = "https://www.czzymovie.com"
	czzyTagUrl    = "https://www.czzymovie.com/%s/page/%d"
	czzySearchUrl = "https://www.czzymovie.com/xsss1O1?q=%s&f=_all&p=%d"
	czzyDetailUrl = "https://www.czzymovie.com/movie/%s.html"
	czzyPlayUrl   = "https://www.czzymovie.com/v_play/%s.html"
)

const (
	subb          = "https://subaibai.vip"
	subbHost      = "https://www.subaibai.com"
	subbTagUrl    = "https://www.subaibai.com/%s/page/%d"
	subbSearchUrl = "https://www.subaibai.com/search?q=%s&p=%d"
	subbDetailUrl = "https://www.subaibai.com/movie/%s.html"
	subbPlayUrl   = "https://www.subaibai.com/v_play/%s.html"
)

const (
	mayiHost      = "https://www.mayiyingshi.tv"
	mayiTagUrl    = "https://www.mayiyingshi.tv/vodtype/%s-%d.html"
	mayiSearchUrl = "https://www.mayiyingshi.tv/vodsearch/%s----------%d---.html"
	mayiDetailUrl = "https://www.mayiyingshi.tv/voddetail/%s.html"
	mayiPlayUrl   = "https://www.mayiyingshi.tv/vodplay/%s.html"
	mayiParseUrl  = "https://zj.sp-flv.com:8443/?url=%s" // 云解析
)

const (
	yingshiHost      = "https://yingshi.tv"
	yingshiTagUrl    = "https://api.yingshi.tv/vod/v1/vod/list?order=desc&limit=30&tid=%s&by=time&page=%d"
	yingshiSearchUrl = "https://api.yingshi.tv/vod/v1/search?wd=%s&limit=20&page=%d"
	yingshiDetailUrl = "https://api.yingshi.tv/vod/v1/info?id=%s&tid=%s"
	yingshiPlayUrl   = "https://api.yingshi.tv/vod/v1/info?id=%s&tid=%s"
)

const (
	netflixgcHost        = "https://www.netflixgc.com"
	netflixgcTagUrl      = "https://www.netflixgc.com/index.php/api/vod"
	netflixgcSearchUrl   = "https://www.netflixgc.com/vodsearch/%s----------%d---.html"
	netflixgcDetailUrl   = "https://www.netflixgc.com/detail/%s.html"
	netflixgcPlayUrl     = "https://www.netflixgc.com/play/%s.html"
	netflixgcEcScriptUrl = "https://www.netflixgc.com/static/Streamlab/js/ecscript.js"
)

const (
	meiyidaHost      = "https://www.mydys1.com"
	meiyidaTagUrl    = "https://www.mydys1.com/vodshow/%s--------%d---.html"
	meiyidaSearchUrl = "https://www.mydys1.com/vodsearch/%s----------%d---.html"
	meiyidaDetailUrl = "https://www.mydys1.com/voddetail/%s.html"
	meiyidaPlayUrl   = "https://www.mydys1.com/vodplay/%s.html"
)

const (
	huawei8Host      = "https://huaweiba.live/"
	huawei8TagUrl    = "https://huaweiba.live/index.php/vod/type/id/%s/page/%d.html"
	huawei8SearchUrl = "https://huaweiba.live/index.php/vod/search/page/%d/wd/%s.html"
	huawei8DetailUrl = "https://huaweiba.live/index.php/vod/detail/id/%s.html"
	huawei8PlayUrl   = "https://huaweiba.live/index.php/vod/detail/id/%s.html"
)
