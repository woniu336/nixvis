#  原项目

> https://github.com/BeyondXinXin/nixvis



## 功能特点

- **全面访问指标**：实时统计独立访客数 (UV)、页面浏览量 (PV) 和流量数据
- **地理位置分布**：展示国内和全球访问来源的可视化地图
- **详细访问排名**：提供 URL、引荐来源、浏览器、操作系统和设备类型的访问排名
- **时间序列分析**：支持按小时和按天查看访问趋势
- **多站点支持**：可同时监控多个网站的访问数据
- **增量日志解析**：自动扫描 Nginx 日志文件，解析并存储最新数据
- **高性能查询**：存储使用轻量级 SQLite，结合多级缓存策略实现快速响应
- **嵌入式资源**：前端资源和IP库内嵌于可执行文件中，无需额外部署静态文件
- **用户认证**：内置用户认证系统，保护敏感数据访问
- **可疑IP检测**：自动检测异常访问行为，支持IP黑名单功能
- **蜘蛛统计**：识别并统计各类搜索引擎蜘蛛的访问情况

## 快速开始

### Linux/Debian 系统部署


> 放行防火墙，例如

```
ufw allow 9523/tcp
```

**一键安装脚本**

```bash
curl -sS -O https://raw.githubusercontent.com/woniu336/nixvis/main/install.sh && chmod +x install.sh && ./install.sh
```


安装脚本会自动完成以下操作：
- 创建专用用户和目录
- 安装二进制文件到 `/opt/nixvis`
- 创建 systemd 服务
- 生成默认配置文件 `/etc/nixvis/config.json`
- 启动服务


设置正确的权限

```
sudo chown nixvis:nixvis /opt/nixvis/nixvis
sudo chmod +x /opt/nixvis/nixvis
sudo systemctl start nixvis
```



如果添加到 adm 组后仍有权限问题，可以检查：

```
  # 1. 确认 nginx 日志文件权限
  ls -la /var/log/nginx/

  # 2. 如果日志文件权限不是 640，可以修改
  sudo chmod 640 /var/log/nginx/*.log
  sudo chown www-data:adm /var/log/nginx/*.log

  # 3. 确保 nginx 日志目录可访问
  sudo chmod 755 /var/log/nginx
```



访问 Web 界面

首次访问会显示创建管理员账户的页面：
```
http://your-server-ip:9523
```

### 服务管理命令

```bash
# 查看服务状态
sudo systemctl status nixvis

# 启动服务
sudo systemctl start nixvis

# 停止服务
sudo systemctl stop nixvis

# 重启服务
sudo systemctl restart nixvis

# 查看日志
sudo journalctl -u nixvis -f

```

### 卸载

```
curl -sS -O https://raw.githubusercontent.com/woniu336/nixvis/main/uninstall.sh && chmod +x uninstall.sh && ./uninstall.sh
```

### 清空数据方式

  方法一：命令行清空（推荐）


```
  # 1. 停止服务
  sudo systemctl stop nixvis

  # 2. 删除数据库文件
  sudo rm -f /var/lib/nixvis/nixvis.db

  # 3. 删除扫描状态文件（重新从头扫描日志）
  sudo rm -f /var/lib/nixvis/nginx_scan_state.json

  # 4. 重启服务（会自动创建新的数据库）
  sudo systemctl start nixvis

  # 5. 查看服务状态
  sudo systemctl status nixvis
```

  方法二：只清空数据保留配置

  如果你想保留配置文件但清空所有数据：

```
  # 1. 停止服务
  sudo systemctl stop nixvis

  # 2. 只删除数据库和状态文件
  sudo rm -f /var/lib/nixvis/nixvis.db
  sudo rm -f /var/lib/nixvis/nginx_scan_state.json

  # 3. 保留配置文件，重启服务
  sudo systemctl start nixvis
```