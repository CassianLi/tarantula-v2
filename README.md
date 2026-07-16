# Getting started

> **弃用说明**：Amazon 商品信息拉取功能已完全迁移至其他项目，本工程**仅支持 eBay** 商品信息拉取与截图上传 OSS。`channel=amazon` 的请求将被拒绝。
>
> 变更记录 2025-03-04: 停止 Amazon 功能开发，专注 eBay。

本工程通过访问本地的`chrome`远程调试端口进行调试，需要先启动chrome的远程调试端口，然后再启动appium，最后再启动测试脚本。

## Installation

### Prerequisites
- Chrome browser with remote debugging enabled on port 9222
- Appium server
- System user and group: tarantula:tarantula

### Deployment
1. Deploy the application to `/opt/etarantula`
2. Ensure proper permissions:
   ```bash
   sudo chown -R tarantula:tarantula /opt/etarantula
   sudo chmod -R 755 /opt/etarantula
   ```

### Service Management
The application is managed using systemd. Create a systemd service file at `/etc/systemd/system/etarantula.service`:

```ini
[Unit]
Description = Sysafri tarantula for ebay , for taking screenshots of items on the ebay website.
After = network.target syslog.target
Wants = network.target

[Service]
Type = simple
User = tarantula
Group = tarantula
WorkingDirectory=/opt/etarantula
ExecStart = /opt/etarantula/etarantula --config /opt/etarantula/.tarantula-ebay.yaml    

[Install]
WantedBy = multi-user.target
```

To manage the service:
```bash
# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable the service to start on boot
sudo systemctl enable etarantula

# Start the service
sudo systemctl start etarantula

# Check service status
sudo systemctl status etarantula
```