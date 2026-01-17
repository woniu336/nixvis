package netparser

import (
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	suspiciousPatterns []string
)

const (
	suspiciousReasonPathScan  = "路径扫描"
	suspiciousReasonAdmin     = "尝试访问管理后台"
	suspiciousReasonSQLInject = "SQL 注入尝试"
	suspiciousReasonXSS       = "XSS 攻击尝试"
	suspiciousReasonOther     = "其他可疑行为"
	suspiciousReason403       = "大量403错误"
	suspiciousReason429       = "大量429限流"
)

var suspiciousReasonMap = map[string]string{
	suspiciousReasonPathScan:  "路径扫描",
	suspiciousReasonAdmin:     "尝试访问管理后台",
	suspiciousReasonSQLInject: "SQL 注入尝试",
	suspiciousReasonXSS:       "XSS 攻击尝试",
	suspiciousReasonOther:     "其他可疑行为",
	suspiciousReason403:       "大量403错误",
	suspiciousReason429:       "大量429限流",
}

func InitSuspiciousDetector() {
	initSuspiciousPatterns()
	logrus.Info("可疑 IP 检测器初始化完成")
}

func initSuspiciousPatterns() {
	suspiciousPatterns = []string{
		"/admin",
		"/administrator",
		"/wp-admin",
		"/phpmyadmin",
		"/myadmin",
		"/admin.php",
		"/administrator.php",
		"/login.php",
		"/wp-login.php",
		"/user/login",
		"/api/user/login",
		"/install.php",
		"/setup.php",
		"/config.php",
		"/.env",
		"/.git",
		"/.svn",
		"/web.config",
		"/.htaccess",
		"/web.xml",
		"/backup.zip",
		"/backup.sql",
		"/database.sql",
		"/dump.sql",
		"/backup.php",
		"/shell.php",
		"/cmd.php",
		"/eval.php",
		"/c99.php",
		"/r57.php",
		"/upload.php",
		"/upload",
		"/uploads",
		"/file.php",
		"/files",
		"/download.php",
		"/includes",
		"/lib",
		"/vendor",
		"/node_modules",
		"/test",
		"/tmp",
		"/temp",
		"/cache",
		"/logs",
		"/log",
		"/sql",
		"/db",
		"/database",
		"/backup",
		"/backups",
		"/bak",
		"/old",
		"/config",
		"/conf",
		"/settings",
		"/setup",
		"/install",
		"/readme",
		"/readme.txt",
		"/readme.html",
		"/changelog",
		"/license",
		"/license.txt",
		"/robots.txt",
		"/sitemap.xml",
		"/crossdomain.xml",
		"/phpinfo.php",
		"/info.php",
		"/test.php",
		"/dev",
		"/debug",
		"/trace",
		"/console",
		"/_profiler",
		"/phpunit",
		"/vendor/bin",
		"/composer.json",
		"/package.json",
		"/gulpfile.js",
		"/webpack.config.js",
		"/.bashrc",
		"/.ssh",
		"/config.bak",
		"/wp-config.php",
		"/config.php.bak",
		"/old.php",
		"/index.php.bak",
		"/index.php~",
		"/index.php.swp",
	}

	sqlInjectionPatterns := []string{
		"union select",
		"union all select",
		"or 1=1",
		"or 1=1--",
		"' or '1'='1",
		"' or 1=1--",
		"admin'--",
		"admin'#",
		"sleep(",
		"benchmark(",
		"waitfor delay",
		"concat(",
		"char(",
		"ascii(",
		"substring(",
		"mid(",
		"load_file(",
		"into outfile",
		"dumpfile",
		"information_schema",
		"pg_sleep(",
		"database(",
		"user(",
		"version(",
		"@@version",
		"xp_cmdshell",
	}

	xssPatterns := []string{
		"<script>",
		"<img src=",
		"javascript:",
		"onerror=",
		"onload=",
		"onmouseover=",
		"onclick=",
		"alert(",
		"document.cookie",
		"eval(",
		"expression(",
		"fromCharCode",
	}

	suspiciousPatterns = append(suspiciousPatterns, sqlInjectionPatterns...)
	suspiciousPatterns = append(suspiciousPatterns, xssPatterns...)
}

func DetectSuspiciousAccess(ip, url, method, userAgent string) (bool, string, string) {
	if isSpider, _, _ := DetectSpider(ip, userAgent); isSpider {
		return false, "", ""
	}

	urlLower := strings.ToLower(url)
	reasonType := suspiciousReasonOther

	for _, pattern := range suspiciousPatterns {
		patternLower := strings.ToLower(pattern)
		if strings.Contains(urlLower, patternLower) {
			if strings.Contains(urlLower, "/admin") ||
				strings.Contains(urlLower, "/wp-admin") ||
				strings.Contains(urlLower, "/administrator") {
				reasonType = suspiciousReasonAdmin
			} else if strings.Contains(urlLower, "union") ||
				strings.Contains(urlLower, "select") ||
				strings.Contains(urlLower, "sql") ||
				strings.Contains(urlLower, "database") ||
				strings.Contains(urlLower, "information_schema") {
				reasonType = suspiciousReasonSQLInject
			} else if strings.Contains(urlLower, "<script") ||
				strings.Contains(urlLower, "javascript:") ||
				strings.Contains(urlLower, "alert(") {
				reasonType = suspiciousReasonXSS
			} else {
				reasonType = suspiciousReasonPathScan
			}
			return true, reasonType, suspiciousReasonMap[reasonType]
		}
	}

	return false, "", ""
}

func GetSuspiciousReasons() []map[string]interface{} {
	list := make([]map[string]interface{}, 0, len(suspiciousReasonMap))
	for reasonType, reasonName := range suspiciousReasonMap {
		list = append(list, map[string]interface{}{
			"type": reasonType,
			"name": reasonName,
		})
	}
	return list
}

func GetSuspiciousReason403() string {
	return suspiciousReason403
}

func GetSuspiciousReason429() string {
	return suspiciousReason429
}

func GetSuspiciousReasonMap() map[string]string {
	return suspiciousReasonMap
}

func IsSuspiciousPath(url string) bool {
	urlLower := strings.ToLower(url)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(urlLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
