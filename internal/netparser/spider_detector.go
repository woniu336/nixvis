package netparser

import (
	"net"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	spiderUserAgents []string
	spiderIPRanges   []*net.IPNet
)

const (
	spiderTypeUnknown     = "unknown"
	spiderTypeGoogle      = "Googlebot"
	spiderTypeBaidu       = "Baiduspider"
	spiderTypeBing        = "Bingbot"
	spiderTypeYandex      = "Yandexbot"
	spiderTypeSogou       = "Sogou"
	spiderTypeSoso        = "Sosospider"
	spiderType360         = "360Spider"
	spiderTypeBytespider  = "Bytespider"
	spiderTypeYisouspider = "YisouSpider"
	spiderTypeClaudeBot   = "ClaudeBot"
	spiderTypeOpenAI      = "GPTBot"
	spiderTypeAmazonbot   = "Amazonbot"
	spiderTypeFacebook    = "facebookexternalhit"
	spiderTypeMeta        = "MetaBot"
	spiderTypeDuckDuckBot = "DuckDuckBot"
	spiderTypeAhrefsBot   = "AhrefsBot"
	spiderTypeSemrushBot  = "SemrushBot"
	spiderTypePetalBot    = "PetalBot"
	spiderTypeApplebot    = "Applebot"
	spiderTypeTwitterbot  = "Twitterbot"
	spiderTypeLinkedInBot = "LinkedInBot"
	spiderTypePinterest   = "Pinterest"
	spiderTypeSlurp       = "Slurp"
	spiderTypeMJ12bot     = "MJ12bot"
	spiderTypeDotBot      = "DotBot"
	spiderTypeSeznamBot   = "SeznamBot"
	spiderTypeAspiegelBot = "AspiegelBot"
)

var spiderInfoMap = map[string]string{
	spiderTypeGoogle:       "Google",
	spiderTypeBaidu:        "百度",
	spiderTypeBing:         "Bing",
	spiderTypeYandex:       "Yandex",
	spiderTypeSogou:        "搜狗",
	spiderTypeSoso:         "腾讯搜搜",
	spiderType360:          "360搜索",
	spiderTypeBytespider:   "今日头条",
	"SmSpider":             "神马",
	spiderTypeClaudeBot:    "Claude",
	spiderTypeOpenAI:       "ChatGPT",
	spiderTypeAmazonbot:    "Amazon",
	spiderTypeFacebook:     "Facebook",
	spiderTypeMeta:         "Meta",
	"Claude-User":          "Claude",
	"ChatGPT-User":         "ChatGPT",
	"OAI-SearchBot":        "OpenAI",
	"facebookcatalog":      "Facebook",
	"meta-webindexer":      "Meta",
	"meta-externalads":     "Meta",
	"meta-externalagent":   "Meta",
	"meta-externalfetcher": "Meta",
	spiderTypeDuckDuckBot:  "DuckDuckGo",
	spiderTypeAhrefsBot:    "Ahrefs",
	spiderTypeSemrushBot:   "Semrush",
	spiderTypePetalBot:     "华为花瓣",
	spiderTypeApplebot:     "Apple",
	spiderTypeTwitterbot:   "Twitter",
	spiderTypeLinkedInBot:  "LinkedIn",
	spiderTypePinterest:    "Pinterest",
	spiderTypeSlurp:        "Yahoo",
	spiderTypeMJ12bot:      "Majestic",
	spiderTypeDotBot:       "DotBot",
	spiderTypeSeznamBot:    "Seznam",
	spiderTypeAspiegelBot:  "Aspiegel",
	"YisouSpider":          "宜搜",
	"Claude-SearchBot":     "Claude",
}

func InitSpiderDetector() {
	initSpiderUserAgents()
	initSpiderIPRanges()
	logrus.Info("蜘蛛检测器初始化完成")
}

func initSpiderUserAgents() {
	spiderUserAgents = []string{
		"Googlebot",
		"Googlebot-Image",
		"Googlebot-Mobile",
		"Baiduspider",
		"Baiduspider-image",
		"Baiduspider-mobile",
		"Bingbot",
		"MSNBot",
		"Yandexbot",
		"YandexImages",
		"Sogou web spider",
		"Sogou inst spider",
		"Sosospider",
		"360Spider",
		"360Search",
		"Bytespider",
		"SmSpider",
		"Slurp",
		"DuckDuckBot",
		"AhrefsBot",
		"SemrushBot",
		"MJ12bot",
		"DotBot",
		"SeznamBot",
		"PetalBot",
		"AspiegelBot",
		"Amazonbot",
		"facebookexternalhit",
		"facebookcatalog",
		"meta-webindexer",
		"meta-externalads",
		"meta-externalagent",
		"meta-externalfetcher",
		"Twitterbot",
		"LinkedInBot",
		"Pinterest",
		"Applebot",
		"ClaudeBot",
		"Claude-User",
		"Claude-SearchBot",
		"OAI-SearchBot",
		"ChatGPT-User",
		"GPTBot",
	}
}

func initSpiderIPRanges() {
	cidrs := []string{
		"66.249.64.0/19",
		"66.249.88.0/24",
		"66.249.92.0/24",
		"203.208.60.0/24",
		"210.242.125.0/24",
		"220.181.38.0/24",
		"123.125.71.0/24",
		"40.77.167.0/24",
		"52.167.144.0/20",
		"77.88.0.0/18",
		"87.250.0.0/16",
		"37.9.0.0/20",
		"37.140.128.0/18",
		"5.10.69.0/24",
		"5.10.70.0/24",
		"106.11.0.0/16",
		"110.242.68.0/24",
		"220.181.108.0/24",
	}

	spiderIPRanges = make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			logrus.WithError(err).Warnf("解析蜘蛛 IP 范围失败: %s", cidr)
			continue
		}
		spiderIPRanges = append(spiderIPRanges, ipnet)
	}
}

func DetectSpider(ip, userAgent string) (bool, string, string) {
	spiderType := spiderTypeUnknown

	if spiderType = detectByIP(ip); spiderType != spiderTypeUnknown {
		return true, spiderType, spiderInfoMap[spiderType]
	}

	if spiderType = detectByUserAgent(userAgent); spiderType != spiderTypeUnknown {
		return true, spiderType, spiderInfoMap[spiderType]
	}

	return false, spiderTypeUnknown, "未知"
}

func detectByIP(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return spiderTypeUnknown
	}

	for _, ipnet := range spiderIPRanges {
		if ipnet.Contains(ip) {
			return identifySpiderByIP(ipStr)
		}
	}

	return spiderTypeUnknown
}

func identifySpiderByIP(ip string) string {
	ipNum := ipToNumber(ip)

	googleRanges := []struct {
		start string
		end   string
	}{
		{"66.249.64.0", "66.249.95.255"},
		{"203.208.60.0", "203.208.60.255"},
	}

	baiduRanges := []struct {
		start string
		end   string
	}{
		{"123.125.71.0", "123.125.71.255"},
		{"210.242.125.0", "210.242.125.255"},
		{"220.181.38.0", "220.181.38.255"},
	}

	for _, r := range googleRanges {
		if ipNum >= ipToNumber(r.start) && ipNum <= ipToNumber(r.end) {
			return spiderTypeGoogle
		}
	}

	for _, r := range baiduRanges {
		if ipNum >= ipToNumber(r.start) && ipNum <= ipToNumber(r.end) {
			return spiderTypeBaidu
		}
	}

	return spiderTypeUnknown
}

func ipToNumber(ip string) uint32 {
	ipBytes := net.ParseIP(ip).To4()
	if ipBytes == nil {
		return 0
	}
	return uint32(ipBytes[0])<<24 | uint32(ipBytes[1])<<16 | uint32(ipBytes[2])<<8 | uint32(ipBytes[3])
}

func detectByUserAgent(userAgent string) string {
	uaLower := strings.ToLower(userAgent)

	for _, spiderUA := range spiderUserAgents {
		if strings.Contains(uaLower, strings.ToLower(spiderUA)) {
			return identifySpiderByUA(spiderUA)
		}
	}

	return spiderTypeUnknown
}

func identifySpiderByUA(ua string) string {
	uaLower := strings.ToLower(ua)

	switch {
	case strings.Contains(uaLower, "googlebot"):
		return spiderTypeGoogle
	case strings.Contains(uaLower, "baiduspider"):
		return spiderTypeBaidu
	case strings.Contains(uaLower, "bingbot") || strings.Contains(uaLower, "msnbot"):
		return spiderTypeBing
	case strings.Contains(uaLower, "yandex"):
		return spiderTypeYandex
	case strings.Contains(uaLower, "sogou"):
		return spiderTypeSogou
	case strings.Contains(uaLower, "sosospider"):
		return spiderTypeSoso
	case strings.Contains(uaLower, "360spider") || strings.Contains(uaLower, "360search"):
		return spiderType360
	case strings.Contains(uaLower, "bytespider"):
		return spiderTypeBytespider
	case strings.Contains(uaLower, "yisouspider"):
		return "SmSpider"
	case strings.Contains(uaLower, "duckduckbot"):
		return "DuckDuckBot"
	case strings.Contains(uaLower, "ahrefsbot"):
		return "AhrefsBot"
	case strings.Contains(uaLower, "semrushbot"):
		return "SemrushBot"
	case strings.Contains(uaLower, "petalbot"):
		return "PetalBot"
	case strings.Contains(uaLower, "applebot"):
		return "Applebot"
	case strings.Contains(uaLower, "amazonbot"):
		return spiderTypeAmazonbot
	case strings.Contains(uaLower, "facebookexternalhit"):
		return spiderTypeFacebook
	case strings.Contains(uaLower, "facebookcatalog"):
		return "facebookcatalog"
	case strings.Contains(uaLower, "meta-webindexer"):
		return "meta-webindexer"
	case strings.Contains(uaLower, "meta-externalads"):
		return "meta-externalads"
	case strings.Contains(uaLower, "meta-externalagent"):
		return "meta-externalagent"
	case strings.Contains(uaLower, "meta-externalfetcher"):
		return "meta-externalfetcher"
	case strings.Contains(uaLower, "claudebot"):
		return spiderTypeClaudeBot
	case strings.Contains(uaLower, "claude-user"):
		return "Claude-User"
	case strings.Contains(uaLower, "claude-searchbot"):
		return "Claude-SearchBot"
	case strings.Contains(uaLower, "oai-searchbot"):
		return "OAI-SearchBot"
	case strings.Contains(uaLower, "chatgpt-user"):
		return "ChatGPT-User"
	case strings.Contains(uaLower, "gptbot"):
		return spiderTypeOpenAI
	case strings.Contains(uaLower, "slurp"):
		return spiderTypeSlurp
	case strings.Contains(uaLower, "duckduckbot"):
		return spiderTypeDuckDuckBot
	case strings.Contains(uaLower, "ahrefsbot"):
		return spiderTypeAhrefsBot
	case strings.Contains(uaLower, "semrushbot"):
		return spiderTypeSemrushBot
	case strings.Contains(uaLower, "mj12bot"):
		return spiderTypeMJ12bot
	case strings.Contains(uaLower, "dotbot"):
		return spiderTypeDotBot
	case strings.Contains(uaLower, "seznambot"):
		return spiderTypeSeznamBot
	case strings.Contains(uaLower, "aspiegelbot"):
		return spiderTypeAspiegelBot
	case strings.Contains(uaLower, "petalbot"):
		return spiderTypePetalBot
	case strings.Contains(uaLower, "applebot"):
		return spiderTypeApplebot
	case strings.Contains(uaLower, "twitterbot"):
		return spiderTypeTwitterbot
	case strings.Contains(uaLower, "linkedinbot"):
		return spiderTypeLinkedInBot
	case strings.Contains(uaLower, "pinterest"):
		return spiderTypePinterest
	default:
		return spiderTypeUnknown
	}
}

func GetSpiderList() []map[string]interface{} {
	list := make([]map[string]interface{}, 0, len(spiderInfoMap))
	for spiderType, spiderName := range spiderInfoMap {
		list = append(list, map[string]interface{}{
			"type": spiderType,
			"name": spiderName,
		})
	}
	return list
}
