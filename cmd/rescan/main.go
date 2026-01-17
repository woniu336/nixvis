package main

import (
	"bufio"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beyondxinxin/nixvis/internal/netparser"
	"github.com/beyondxinxin/nixvis/internal/util"
	_ "modernc.org/sqlite"
)

var (
	nginxLogPattern = regexp.MustCompile(`^(\S+) - (\S+) \[([^\]]+)\] "(\S+) ([^"]+) HTTP\/\d\.\d" (\d+) (\d+) "([^"]*)" "([^"]*)"`)
)

func main() {
	fmt.Println("开始重新扫描日志并标记蜘蛛和可疑 IP...")

	err := netparser.InitIPGeoLocation()
	if err != nil {
		fmt.Printf("初始化 IP 地理位置失败: %v\n", err)
		return
	}

	netparser.InitPVFilters()
	netparser.InitSpiderDetector()
	netparser.InitSuspiciousDetector()

	dbPath := filepath.Join(util.DataDir, "nixvis.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("打开数据库失败: %v\n", err)
		return
	}
	defer db.Close()

	cfg := util.ReadConfig()
	totalProcessed := 0
	totalUpdated := 0

	for _, website := range cfg.Websites {
		websiteID := generateID(website.Name)

		fmt.Printf("\n处理网站: %s (%s)\n", website.Name, websiteID)

		processed, updated, err := processWebsite(db, websiteID, website.LogPath)
		if err != nil {
			fmt.Printf("处理失败: %v\n", err)
			continue
		}

		totalProcessed += processed
		totalUpdated += updated
	}

	fmt.Printf("\n处理完成!\n")
	fmt.Printf("总处理记录数: %d\n", totalProcessed)
	fmt.Printf("更新记录数: %d\n", totalUpdated)
}

func processWebsite(db *sql.DB, websiteID, logPath string) (int, int, error) {
	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)

	paths, err := getLogPaths(logPath)
	if err != nil {
		return 0, 0, err
	}

	processed := 0
	updated := 0

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("无法打开文件 %s: %v\n", path, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			processed++

			matches := nginxLogPattern.FindStringSubmatch(line)
			if len(matches) < 10 {
				continue
			}

			timestamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[3])
			if err != nil {
				continue
			}

			ip := matches[1]
			userAgent := matches[9]
			statusCode, _ := strconv.Atoi(matches[6])
			method := matches[4]

			url := matches[5]
			statusCodeStr := strconv.Itoa(statusCode)

			isSpider, spiderType, spiderName := netparser.DetectSpider(ip, userAgent)
			isSuspicious, susType, susReason := netparser.DetectSuspiciousAccess(ip, url, method, userAgent)

			spiderTypeValue := ""
			spiderNameValue := ""
			suspiciousTypeValue := ""
			suspiciousReasonValue := ""

			if isSpider {
				spiderTypeValue = spiderType
				spiderNameValue = spiderName
			}

			if isSuspicious {
				suspiciousTypeValue = susType
				suspiciousReasonValue = susReason
			}

			updateSQL := fmt.Sprintf(`
				UPDATE "%s"
				SET is_spider = ?, spider_type = ?, spider_name = ?,
				    is_suspicious = ?, suspicious_type = ?, suspicious_reason = ?
				WHERE ip = ? AND timestamp = ? AND method = ? AND url = ? AND status_code = ?
			`, tableName)

			result, err := db.Exec(updateSQL,
				boolToInt(isSpider), spiderTypeValue, spiderNameValue,
				boolToInt(isSuspicious), suspiciousTypeValue, suspiciousReasonValue,
				ip, timestamp.Unix(), method, url, statusCodeStr,
			)

			if err != nil {
				fmt.Printf("更新失败: %v\n", err)
				continue
			}

			rowsAffected, _ := result.RowsAffected()
			updated += int(rowsAffected)

			if processed%1000 == 0 {
				fmt.Printf("已处理 %d 条记录，更新 %d 条\n", processed, updated)
			}
		}
	}

	return processed, updated, nil
}

func getLogPaths(logPath string) ([]string, error) {
	if strings.Contains(logPath, "*") {
		return filepath.Glob(logPath)
	}
	return []string{logPath}, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func generateID(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:2])
}
