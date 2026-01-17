package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

var (
	dataSourceName = filepath.Join(util.DataDir, "nixvis.db")
)

type NginxLogRecord struct {
	ID               int64     `json:"id"`
	IP               string    `json:"ip"`
	PageviewFlag     int       `json:"pageview_flag"`
	Timestamp        time.Time `json:"timestamp"`
	Method           string    `json:"method"`
	Url              string    `json:"url"`
	Status           int       `json:"status"`
	BytesSent        int       `json:"bytes_sent"`
	Referer          string    `json:"referer"`
	UserBrowser      string    `json:"user_browser"`
	UserOs           string    `json:"user_os"`
	UserDevice       string    `json:"user_device"`
	DomesticLocation string    `json:"domestic_location"`
	GlobalLocation   string    `json:"global_location"`
	IsSpider         int       `json:"is_spider"`
	SpiderType       string    `json:"spider_type"`
	SpiderName       string    `json:"spider_name"`
	IsSuspicious     int       `json:"is_suspicious"`
	SuspiciousType   string    `json:"suspicious_type"`
	SuspiciousReason string    `json:"suspicious_reason"`
}

type Repository struct {
	db *sql.DB
}

func NewRepository() (*Repository, error) {
	// 打开数据库
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	// 链接数据库
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// 性能优化设置
	if _, err := db.Exec(`
        PRAGMA journal_mode=WAL;
        PRAGMA synchronous=NORMAL;
        PRAGMA cache_size=32768;
        PRAGMA temp_store=MEMORY;`); err != nil {
		db.Close()
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

// 初始化数据库
func (r *Repository) Init() error {
	return r.createTables()
}

// 关闭数据库连接
func (r *Repository) Close() error {
	logrus.Info("关闭数据库")
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// 获取数据库连接
func (r *Repository) GetDB() *sql.DB {
	return r.db
}

func (r *Repository) BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	nginxTable := fmt.Sprintf("%s_nginx_logs", websiteID)

	stmtNginx, err := tx.Prepare(fmt.Sprintf(`
        INSERT INTO "%s" (
        ip, pageview_flag, timestamp, method, url,
        status_code, bytes_sent, referer,
        user_browser, user_os, user_device, domestic_location, global_location,
        is_spider, spider_type, spider_name, is_suspicious, suspicious_type, suspicious_reason)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, nginxTable))
	if err != nil {
		return err
	}
	defer stmtNginx.Close()

	for _, log := range logs {
		_, err = stmtNginx.Exec(
			log.IP, log.PageviewFlag, log.Timestamp.Unix(), log.Method, log.Url,
			log.Status, log.BytesSent, log.Referer, log.UserBrowser, log.UserOs, log.UserDevice,
			log.DomesticLocation, log.GlobalLocation,
			log.IsSpider, log.SpiderType, log.SpiderName, log.IsSuspicious, log.SuspiciousType, log.SuspiciousReason,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) RecordSuspiciousAccess(websiteID, ip string, reasonType, reasonDetail string, timestamp int64) error {
	now := timestamp
	query := `
		INSERT INTO suspicious_ips (website_id, ip, first_seen, last_seen, access_count, reason_type, reason_detail)
		VALUES (?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT(website_id, ip) DO UPDATE SET
			last_seen = ?,
			access_count = access_count + 1
	`

	_, err := r.db.Exec(query, websiteID, ip, now, now, reasonType, reasonDetail, now)
	return err
}

func (r *Repository) GetSuspiciousIPs(websiteID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			s.id,
			s.website_id,
			s.ip,
			s.access_count,
			s.reason_type,
			s.is_blocked,
			s.blocked_at
		FROM suspicious_ips s
		WHERE s.website_id = ?
		ORDER BY s.access_count DESC
		LIMIT 20
	`

	rows, err := r.db.Query(query, websiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int
		var wid, ip, reasonType string
		var blockedAt int64
		var accessCount, isBlocked int

		err := rows.Scan(&id, &wid, &ip, &accessCount, &reasonType, &isBlocked, &blockedAt)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":             id,
			"website_id":     wid,
			"ip":             ip,
			"access_count":   accessCount,
			"reason_type":    reasonType,
			"is_blocked":     isBlocked,
			"blocked_at":     blockedAt,
		})
	}

	return results, nil
}

func (r *Repository) getSpiderName(spiderType string) string {
	spiderNames := map[string]string{
		"Googlebot":            "Google",
		"Baiduspider":          "百度",
		"Bingbot":              "Bing",
		"Yandexbot":            "Yandex",
		"Sogou":                "搜狗",
		"Sosospider":           "腾讯搜搜",
		"360Spider":            "360搜索",
		"Bytespider":           "今日头条",
		"SmSpider":             "神马",
		"ClaudeBot":            "Claude",
		"GPTBot":               "ChatGPT",
		"Amazonbot":            "Amazon",
		"facebookexternalhit":  "Facebook",
		"MetaBot":              "Meta",
		"Claude-User":          "Claude",
		"ChatGPT-User":         "ChatGPT",
		"OAI-SearchBot":        "OpenAI",
		"facebookcatalog":      "Facebook",
		"meta-webindexer":      "Meta",
		"meta-externalads":     "Meta",
		"meta-externalagent":   "Meta",
		"meta-externalfetcher": "Meta",
		"DuckDuckBot":          "DuckDuckGo",
		"AhrefsBot":            "Ahrefs",
		"SemrushBot":           "Semrush",
		"PetalBot":             "华为花瓣",
		"Applebot":             "Apple",
		"Twitterbot":           "Twitter",
		"LinkedInBot":          "LinkedIn",
		"Pinterest":            "Pinterest",
		"Slurp":                "Yahoo",
		"MJ12bot":              "Majestic",
		"DotBot":               "DotBot",
		"SeznamBot":            "Seznam",
		"AspiegelBot":          "Aspiegel",
		"YisouSpider":          "宜搜",
		"Claude-SearchBot":     "Claude",
	}

	if name, ok := spiderNames[spiderType]; ok {
		return name
	}
	return "未知"
}

func (r *Repository) BlockIP(websiteID, ip string) error {
	now := time.Now().Unix()

	query := `
		UPDATE suspicious_ips
		SET is_blocked = 1, blocked_at = ?, last_seen = ?
		WHERE website_id = ? AND ip = ?
	`

	_, err := r.db.Exec(query, now, now, websiteID, ip)
	if err != nil {
		return fmt.Errorf("更新阻断状态失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"website_id": websiteID,
		"ip":         ip,
	}).Info("IP 已标记为阻断")

	return nil
}

func (r *Repository) GetSpiderStats(websiteID string, timeRange int64) ([]map[string]interface{}, error) {
	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)

	query := `
		SELECT
			spider_type,
			COUNT(*) as visits,
			COUNT(DISTINCT ip) as unique_ips
		FROM "%s"
		WHERE is_spider = 1 AND timestamp >= ?
		GROUP BY spider_type
		ORDER BY visits DESC
		LIMIT 100
	`

	rows, err := r.db.Query(fmt.Sprintf(query, tableName), timeRange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var spiderType string
		var visits, uniqueIPs int

		err := rows.Scan(&spiderType, &visits, &uniqueIPs)
		if err != nil {
			continue
		}

		spiderName := r.getSpiderName(spiderType)

		ips, err := r.getSpiderIPs(tableName, spiderType, spiderName, timeRange)
		if err != nil {
			ips = []map[string]interface{}{}
		}

		results = append(results, map[string]interface{}{
			"spider_type": spiderType,
			"spider_name": spiderName,
			"visits":      visits,
			"unique_ips":  uniqueIPs,
			"ips":         ips,
		})
	}

	return results, nil
}

func (r *Repository) getSpiderIPs(tableName, spiderType, spiderName string, timeRange int64) ([]map[string]interface{}, error) {
	query := `
		SELECT
			ip,
			COUNT(*) as visits,
			MIN(timestamp) as first_seen,
			MAX(timestamp) as last_seen
		FROM "%s"
		WHERE is_spider = 1 AND spider_type = ? AND timestamp >= ?
		GROUP BY ip
		ORDER BY visits DESC
		LIMIT 50
	`

	rows, err := r.db.Query(fmt.Sprintf(query, tableName), spiderType, timeRange)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var ip string
		var visits int
		var firstSeen, lastSeen int64

		err := rows.Scan(&ip, &visits, &firstSeen, &lastSeen)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"ip":         ip,
			"visits":     visits,
			"first_seen": firstSeen,
			"last_seen":  lastSeen,
		})
	}

	return results, nil
}

// CleanOldLogs 清理45天前的日志数据
func (r *Repository) CleanOldLogs() error {
	cutoffTime := time.Now().AddDate(0, 0, -45).Unix()

	deletedCount := 0

	rows, err := r.db.Query(`
        SELECT name FROM sqlite_master 
        WHERE type='table' AND name LIKE '%_nginx_logs'
    `)
	if err != nil {
		return fmt.Errorf("查询表名失败: %v", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			logrus.WithError(err).Error("扫描表名失败")
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	for _, tableName := range tableNames {
		result, err := r.db.Exec(
			fmt.Sprintf(`DELETE FROM "%s" WHERE timestamp < ?`, tableName), cutoffTime,
		)
		if err != nil {
			logrus.WithError(err).Errorf("清理表 %s 的旧日志失败", tableName)
			continue
		}

		count, _ := result.RowsAffected()
		deletedCount += int(count)
	}

	if deletedCount > 0 {
		logrus.Infof("删除了 %d 条45天前的日志记录", deletedCount)
		if _, err := r.db.Exec("VACUUM"); err != nil {
			logrus.WithError(err).Error("数据库压缩失败")
		}
	}

	return nil
}

func (r *Repository) createTables() error {
	common := `id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip TEXT NOT NULL,
	pageview_flag INTEGER NOT NULL DEFAULT 0,
	timestamp INTEGER NOT NULL,
	method TEXT NOT NULL,
	url TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	bytes_sent INTEGER NOT NULL,
	referer TEXT NOT NULL,
	user_browser TEXT NOT NULL,
	user_os TEXT NOT NULL,
	user_device TEXT NOT NULL,
	domestic_location TEXT NOT NULL,
	global_location TEXT NOT NULL,
	is_spider INTEGER NOT NULL DEFAULT 0,
	spider_type TEXT NOT NULL DEFAULT '',
	spider_name TEXT NOT NULL DEFAULT '',
	is_suspicious INTEGER NOT NULL DEFAULT 0,
	suspicious_type TEXT NOT NULL DEFAULT '',
	suspicious_reason TEXT NOT NULL DEFAULT ''`

	for _, id := range util.GetAllWebsiteIDs() {
		tableName := fmt.Sprintf("%s_nginx_logs", id)

		// 创建表
		q := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s_nginx_logs" (%s);`, id, common)
		if _, err := r.db.Exec(q); err != nil {
			logrus.WithError(err).Errorf("创建表 %s 失败", tableName)
			continue
		}

		// 尝试添加新字段（如果已存在会报错，忽略）
		alterQueries := []string{
			fmt.Sprintf(`ALTER TABLE "%s_nginx_logs" ADD COLUMN is_spider INTEGER NOT NULL DEFAULT 0;`, id),
			fmt.Sprintf(`ALTER TABLE "%s_nginx_logs" ADD COLUMN spider_type TEXT NOT NULL DEFAULT '';`, id),
			fmt.Sprintf(`ALTER TABLE "%s_nginx_logs" ADD COLUMN spider_name TEXT NOT NULL DEFAULT '';`, id),
			fmt.Sprintf(`ALTER TABLE "%s_nginx_logs" ADD COLUMN is_suspicious INTEGER NOT NULL DEFAULT 0;`, id),
			fmt.Sprintf(`ALTER TABLE "%s_nginx_logs" ADD COLUMN suspicious_type TEXT NOT NULL DEFAULT '';`, id),
			fmt.Sprintf(`ALTER TABLE "%s_nginx_logs" ADD COLUMN suspicious_reason TEXT NOT NULL DEFAULT '';`, id),
		}
		for _, alterQ := range alterQueries {
			if _, err := r.db.Exec(alterQ); err != nil {
				// 字段可能已存在，忽略错误
				logrus.WithError(err).Debugf("字段可能已存在，跳过 ALTER TABLE")
			}
		}

		// 创建单列索引
		indexQueries := []string{
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON "%s_nginx_logs"(timestamp);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_url ON "%s_nginx_logs"(url);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_ip ON "%s_nginx_logs"(ip);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_referer ON "%s_nginx_logs"(referer);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_browser ON "%s_nginx_logs"(user_browser);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_os ON "%s_nginx_logs"(user_os);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_device ON "%s_nginx_logs"(user_device);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_domestic_location ON "%s_nginx_logs"(domestic_location);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_global_location ON "%s_nginx_logs"(global_location);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_is_spider ON "%s_nginx_logs"(is_spider);`, id, id),
			fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_is_suspicious ON "%s_nginx_logs"(is_suspicious);`, id, id),
		}
		for _, idxQ := range indexQueries {
			if _, err := r.db.Exec(idxQ); err != nil {
				logrus.WithError(err).Warnf("创建单列索引失败: %s", idxQ)
			}
		}

		// 创建复合索引
		_, err := r.db.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_pv_ts_ip ON "%s_nginx_logs" (pageview_flag, timestamp, ip);`, id, id))
		if err != nil {
			logrus.WithError(err).Warnf("创建复合索引失败 [%s]", tableName)
		}
	}

	if err := r.createSuspiciousIPTable(); err != nil {
		return err
	}

	return nil
}

func (r *Repository) createSuspiciousIPTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS suspicious_ips (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			website_id TEXT NOT NULL,
			ip TEXT NOT NULL,
			first_seen INTEGER NOT NULL,
			last_seen INTEGER NOT NULL,
			access_count INTEGER NOT NULL DEFAULT 1,
			reason_type TEXT NOT NULL,
			reason_detail TEXT NOT NULL,
			is_blocked INTEGER NOT NULL DEFAULT 0,
			blocked_at INTEGER NOT NULL DEFAULT 0,
			UNIQUE(website_id, ip)
		)
	`)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_suspicious_ips_website ON suspicious_ips(website_id);
		CREATE INDEX IF NOT EXISTS idx_suspicious_ips_ip ON suspicious_ips(ip);
		CREATE INDEX IF NOT EXISTS idx_suspicious_ips_is_blocked ON suspicious_ips(is_blocked);
		CREATE INDEX IF NOT EXISTS idx_suspicious_ips_last_seen ON suspicious_ips(last_seen);
	`)
	return err
}

// CreateTableForWebsite 为指定站点创建数据库表
func (r *Repository) CreateTableForWebsite(websiteID string) error {
	common := `id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip TEXT NOT NULL,
	pageview_flag INTEGER NOT NULL DEFAULT 0,
	timestamp INTEGER NOT NULL,
	method TEXT NOT NULL,
	url TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	bytes_sent INTEGER NOT NULL,
	referer TEXT NOT NULL,
	user_browser TEXT NOT NULL,
	user_os TEXT NOT NULL,
	user_device TEXT NOT NULL,
	domestic_location TEXT NOT NULL,
	global_location TEXT NOT NULL,
	is_spider INTEGER NOT NULL DEFAULT 0,
	spider_type TEXT NOT NULL DEFAULT '',
	spider_name TEXT NOT NULL DEFAULT '',
	is_suspicious INTEGER NOT NULL DEFAULT 0,
	suspicious_type TEXT NOT NULL DEFAULT '',
	suspicious_reason TEXT NOT NULL DEFAULT ''`

	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)

	// 创建表
	q := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s_nginx_logs" (%s);`, websiteID, common)
	if _, err := r.db.Exec(q); err != nil {
		return fmt.Errorf("创建表 %s 失败: %v", tableName, err)
	}

	// 创建单列索引
	indexQueries := []string{
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON "%s_nginx_logs"(timestamp);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_url ON "%s_nginx_logs"(url);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_ip ON "%s_nginx_logs"(ip);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_referer ON "%s_nginx_logs"(referer);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_browser ON "%s_nginx_logs"(user_browser);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_os ON "%s_nginx_logs"(user_os);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_user_device ON "%s_nginx_logs"(user_device);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_domestic_location ON "%s_nginx_logs"(domestic_location);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_global_location ON "%s_nginx_logs"(global_location);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_is_spider ON "%s_nginx_logs"(is_spider);`, websiteID, websiteID),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_is_suspicious ON "%s_nginx_logs"(is_suspicious);`, websiteID, websiteID),
	}
	for _, idxQ := range indexQueries {
		if _, err := r.db.Exec(idxQ); err != nil {
			logrus.WithError(err).Warnf("创建单列索引失败: %s", idxQ)
		}
	}

	// 创建复合索引
	_, err := r.db.Exec(fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_pv_ts_ip ON "%s_nginx_logs" (pageview_flag, timestamp, ip);`, websiteID, websiteID))
	if err != nil {
		logrus.WithError(err).Warnf("创建复合索引失败 [%s]", tableName)
	}

	logrus.Infof("站点 %s 的数据库表创建成功", websiteID)
	return nil
}
