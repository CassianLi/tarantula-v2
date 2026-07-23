# etarantula

> **弃用说明**：Amazon 商品信息拉取功能已完全迁移至其他项目，本工程**仅支持 eBay** 商品信息拉取与截图上传 OSS。`channel=amazon` 的请求将被拒绝。
>
> 变更记录 2025-03-04: 停止 Amazon 功能开发，专注 eBay。

通过本地 Chrome 远程调试端口（默认 **9222**）打开 eBay 商品页、解析价格并截图上传 OSS。  
应用配置中须设置：

```yaml
chromedp:
  url: ws://127.0.0.1:9222   # 不要留空，否则会走本地 Exec 拉起 Chrome，易出现权限问题
  headless: false
```

---

## 启动 Chrome 调试端口（二选一）

无论用哪种方式，都建议使用**独立用户数据目录**（如 `/opt/etarantula/chrome-debug`），不要和日常浏览用的 Chrome 共用同一配置目录，否则新开的窗口往往吃不到 `--remote-debugging-port`，9222 不会打开。

启动前先确认没有占用：

```bash
pkill -f 'google-chrome|chrome-debug' || true   # 谨慎：会关掉本机 Chrome
sleep 1
ss -lntp | grep 9222 || echo "9222 空闲"
```

启动成功后验证：

```bash
curl -s http://127.0.0.1:9222/json/version
# 应返回 JSON，且含 webSocketDebuggerUrl
```

### 方式一：图形界面快捷方式 / 图标（适合本机调试）

适用于已登录桌面、用鼠标点图标启动的场景。

1. 复制系统 Chrome 桌面文件到用户目录（名称可自定）：

   ```bash
   mkdir -p ~/.local/share/applications
   cp /usr/share/applications/google-chrome.desktop \
      ~/.local/share/applications/google-chrome-debug.desktop
   ```

2. 编辑 `~/.local/share/applications/google-chrome-debug.desktop`，修改 `Name` / `Exec`，例如：

   ```ini
   [Desktop Entry]
   Name=Chrome Debug 9222
   Exec=/usr/bin/google-chrome-stable --remote-debugging-port=9222 --user-data-dir=/opt/etarantula/chrome-debug --no-first-run --no-default-browser-check %U
   Terminal=false
   Type=Application
   Icon=google-chrome
   ```

3. **参数顺序要点**（常见踩坑）：

   | 错误写法 | 问题 |
   |----------|------|
   | `...google-chrome-stable %U --remote-debugging-port=9222` | `%U` 之后的参数常被当成 URL，调试端口不生效 |
   | 不设 `--user-data-dir`，且已有日常 Chrome 在跑 | 只是附着到旧进程，**不会**开启 9222 |

   正确：调试相关参数写在 **`%U` 之前**，并指定独立 `--user-data-dir`。

4. 若用桌面「启动器属性 / 快捷方式」直接改命令，写成与上面 `Exec=` 相同即可；改完后**先退出所有 Chrome**，再只点这个 Debug 图标。

5. 目录权限（与 etarantula 部署目录一致时）：

   ```bash
   sudo mkdir -p /opt/etarantula/chrome-debug
   sudo chown -R "$USER":"$USER" /opt/etarantula/chrome-debug
   # 若服务用户是 tarantula，生产机应 chown tarantula:tarantula
   ```

### 方式二：systemd `chrome-debug.service`（适合生产 / 开机自启）

仓库单元文件：

- `deploy/systemd/chrome-debug.service` — 拉起带 9222 的 Chrome  
- `deploy/systemd/etarantula.service` — 依赖 Chrome，并在启动前等待 9222 就绪  

```bash
sudo mkdir -p /opt/etarantula/chrome-debug
sudo chown -R tarantula:tarantula /opt/etarantula

sudo cp deploy/systemd/chrome-debug.service /etc/systemd/system/
sudo cp deploy/systemd/etarantula.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable chrome-debug etarantula
sudo systemctl start etarantula    # Requires 会先拉起 chrome-debug
```

常用命令：

```bash
sudo systemctl status chrome-debug etarantula
sudo systemctl restart etarantula      # 确保 Chrome 在跑并等 9222
sudo systemctl restart chrome-debug    # 仅重启浏览器
sudo journalctl -u chrome-debug -u etarantula -f
```

**行为说明：**

- `etarantula.service` 使用 `Requires=chrome-debug.service` + `After=`：启动/重启应用时会一并保证 Chrome 起来；Chrome 起不来则应用不会「假成功」。
- 单元内已使用 `--user-data-dir=/opt/etarantula/chrome-debug` 与 `--remote-debugging-port=9222`。
- 有图形桌面时单元默认 `DISPLAY=:0`；若用户 `tarantula` 看不到该显示器，需改成实际登录桌面的用户，或配置 `XAUTHORITY` / `xhost`。
- 无桌面环境：在 `chrome-debug.service` 的 `ExecStart` 增加 `--headless=new`，并可去掉 `Environment=DISPLAY=:0`。

**不要混用：** 同一时间只保留一种调试 Chrome（图标 **或** `chrome-debug.service`），避免抢 9222 / 抢同一 `user-data-dir`。

---

## 应用部署

### Prerequisites

- Google Chrome：`/usr/bin/google-chrome-stable`
- 系统用户：`tarantula:tarantula`（可按环境调整 unit 中的 `User`/`Group`）
- 上述任一方式已使 `curl http://127.0.0.1:9222/json/version` 有输出

### 安装二进制与配置

1. 部署到 `/opt/etarantula`
2. 权限：

   ```bash
   sudo chown -R tarantula:tarantula /opt/etarantula
   sudo chmod -R 755 /opt/etarantula
   ```

3. 配置文件（如 `/opt/etarantula/.tarantula-ebay.yaml`）中保留 `chromedp.url: ws://127.0.0.1:9222`

### 仅用图形界面起浏览器时的 etarantula

若 Chrome 用方式一（图标）手动开好 9222，可暂时去掉对 `chrome-debug` 的依赖（或只用旧版仅含 etarantula 的 unit），再：

```bash
sudo systemctl start etarantula
```

生产仍推荐方式二，减少「忘记开图标 → 应用初始化失败」的情况。
