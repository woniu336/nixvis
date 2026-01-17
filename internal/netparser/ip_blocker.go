package netparser

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/sirupsen/logrus"
)

const (
	ipsetName     = "nixvis_blocked"
	iptablesChain = "INPUT"
)

type BlockResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Command string `json:"command,omitempty"`
}

func BlockIP(ip string) BlockResult {
	if runtime.GOOS == "windows" {
		return BlockResult{
			Success: false,
			Message: "Windows 系统暂不支持直接屏蔽 IP，请使用防火墙规则",
		}
	}

	result := BlockResult{}

	if err := addToIpset(ip); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("添加到 ipset 失败: %v", err)
		return result
	}

	if err := addToIptables(); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("添加到 iptables 失败: %v", err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("IP %s 已成功屏蔽", ip)
	return result
}

func addToIpset(ip string) error {
	cmd := exec.Command("ipset", "test", ipsetName, ip)
	if err := cmd.Run(); err == nil {
		logrus.Infof("IP %s 已在 ipset 中，跳过添加", ip)
		return nil
	}

	cmd = exec.Command("ipset", "add", ipsetName, ip)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 ipset add 失败: %v", err)
	}

	logrus.Infof("IP %s 已添加到 ipset %s", ip, ipsetName)
	return nil
}

func addToIptables() error {
	cmd := exec.Command("iptables", "-C", iptablesChain, "-m", "set",
		"--match-set", ipsetName, "src", "-j", "DROP")
	if err := cmd.Run(); err == nil {
		logrus.Infof("iptables 规则已存在，跳过添加")
		return nil
	}

	cmd = exec.Command("iptables", "-A", iptablesChain, "-m", "set",
		"--match-set", ipsetName, "src", "-j", "DROP")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行 iptables 命令失败: %v", err)
	}

	logrus.Infof("iptables 规则已添加，匹配 ipset %s", ipsetName)
	return nil
}

func UnblockIP(ip string) BlockResult {
	result := BlockResult{}

	cmd := exec.Command("ipset", "del", ipsetName, ip)
	if err := cmd.Run(); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("从 ipset 删除失败: %v", err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("IP %s 已成功解除屏蔽", ip)
	return result
}

func InitIpset() error {
	if runtime.GOOS == "windows" {
		logrus.Warn("Windows 系统跳过 ipset 初始化")
		return nil
	}

	cmd := exec.Command("ipset", "list", ipsetName)
	if err := cmd.Run(); err == nil {
		logrus.Infof("ipset %s 已存在，跳过创建", ipsetName)
		return nil
	}

	cmd = exec.Command("ipset", "create", ipsetName, "hash:ip")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("创建 ipset 失败: %v", err)
	}

	logrus.Infof("ipset %s 创建成功", ipsetName)
	return nil
}

func ListBlockedIPs() ([]string, error) {
	cmd := exec.Command("ipset", "list", ipsetName)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := string(output)
	var ips []string

	start := false
	for _, line := range splitLines(lines) {
		if contains(line, "Members:") {
			start = true
			continue
		}
		if start && line != "" {
			ips = append(ips, line)
		}
	}

	return ips, nil
}

func splitLines(s string) []string {
	var lines []string
	line := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(c)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func GenerateBlockScript(ips []string) string {
	script := "#!/bin/bash\n\n"
	script += fmt.Sprintf("# NixVis IP 屏蔽脚本\n")
	script += fmt.Sprintf("# 生成时间: %s\n\n", getTimestamp())

	script += fmt.Sprintf("# 创建 ipset\n")
	script += fmt.Sprintf("ipset create %s hash:ip 2>/dev/null || true\n\n", ipsetName)

	script += fmt.Sprintf("# 清空现有规则\n")
	script += fmt.Sprintf("ipset flush %s\n\n", ipsetName)

	script += fmt.Sprintf("# 添加屏蔽 IP\n")
	for _, ip := range ips {
		script += fmt.Sprintf("ipset add %s %s\n", ipsetName, ip)
	}

	script += "\n"
	script += "# 添加 iptables 规则\n"
	script += fmt.Sprintf("iptables -C %s -m set --match-set %s src -j DROP 2>/dev/null || \\\n", iptablesChain, ipsetName)
	script += fmt.Sprintf("    iptables -A %s -m set --match-set %s src -j DROP\n", iptablesChain, ipsetName)

	return script
}

func getTimestamp() string {
	return fmt.Sprintf("%d", 0)
}
