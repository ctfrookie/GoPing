# GoPing - 批量 Ping 工具

GoPing 是一个基于 Go 语言开发的批量网络扫描工具，支持多线程并发 Ping 操作，适用于快速检测 IP 地址的可达性。

## 功能特性

- **CIDR 支持**：支持通过 CIDR 格式（如 `10.0.0.0/24`）指定目标网段。
- **跨平台编译**：可一键编译生成 Windows、Linux 和 macOS 平台的二进制文件。
- **高并发扫描**：支持自定义并发线程数，充分利用硬件资源。
- **实时进度显示**：在终端中动态显示扫描进度及预计剩余时间。
- **日志记录**：将扫描结果保存到日志文件中，便于后续分析。
- **超时设置**：支持自定义 Ping 超时时间（默认 800ms）。
- **颜色输出**：使用 ANSI 颜色区分存活主机和不可达主机。

---

## 使用方法

### 命令行参数

运行程序时需要提供以下参数：

| 参数 | 描述 | 示例 |
|------|------|------|
| `-c` | CIDR 地址列表（逗号或空格分隔） | `-c 10.0.0.0/24,192.168.1.0/24` |
| `-o` | 日志文件路径（默认 `ping_log.txt`） | `-o scan.log` |
| `-t` | Ping 超时时间（毫秒，默认 800ms） | `-t 500` |
| `-n` | 并发线程数（默认 100） | `-n 200` |

#### 示例命令
```bash
goping -c 10.0.0.0/24,192.168.1.0/24 -o scan.log -t 500 -n 200
```

---

## 编译说明

### 环境要求

- 安装 [Go 编译器](https://golang.org/dl/)（建议版本 1.20 或更高）。
- 安装依赖库：
  ```bash
  go mod tidy
  ```

### 一键编译脚本

项目根目录下包含一个 `build.bat` 脚本，用于一键编译生成 Windows、Linux 和 macOS 平台的二进制文件。

#### 脚本功能
- 自动检测是否安装了 Go 编译器。
- 创建 `dist` 目录存储编译结果。
- 编译生成以下文件：
  - `dist\goping_windows_amd64.exe`（Windows）
  - `dist\goping_linux_amd64`（Linux）
  - `dist\goping_darwin_amd64`（macOS）

#### 使用方法
1. 双击运行 `build.bat`，或在命令行中执行：
   ```cmd
   build.bat
   ```
2. 编译完成后，所有生成的文件会保存在 `dist` 目录中。

---

## 注意事项

1. **管理员权限**
   - 在 Windows 上运行程序时，可能需要以管理员身份运行，以便发送 ICMP 请求。
   - 在 Linux/macOS 上，普通用户通常可以直接运行。如果遇到权限问题，可以尝试使用 `sudo`。

2. **日志文件路径**
   - 如果日志文件路径不存在，程序会尝试自动创建目录。如果创建失败，请手动创建目标目录。

3. **跨平台兼容性**
   - 编译生成的 Linux 和 macOS 可执行文件可能需要赋予权限才能运行：
     ```bash
     chmod +x goping_linux_amd64
     chmod +x goping_darwin_amd64
     ```

4. **IPv6 支持**
   - 当前版本仅支持 IPv4。如果需要支持 IPv6，请修改代码中的 `pinger.SetNetwork("ip4")`。

5. **大网段扫描**
   - 对于较大的 CIDR 网段（如 `/16`），建议适当增加超时时间或减少并发线程数，以避免网络拥塞。

---

## 示例输出

### 终端输出
```plaintext
Configuration:
  CIDRs:      10.0.0.0/24,192.168.1.0/24
  Log file:   scan.log
  Timeout:    500 ms
  Threads:    200

Processing Subnet: 10.0.0.0/24 (254 IPs)
Progress: 100.0% (254/254) | Elapsed: 5.2s | Completed
Scan completed for Subnet: 10.0.0.0/24

Alive: 12 | Dead: 242 | Total: 254 | Timeout: 500 ms
```

### 日志文件内容
```plaintext
=== Processing Subnet: 10.0.0.0/24 ===
=== Results for Subnet: 10.0.0.0/24 (Timeout: 500 ms) ===
IP Address      Status
-----------------------
10.0.0.1         UP
10.0.0.2         DOWN
...
=======================
```

---

## 贡献与反馈

如果您发现任何问题或希望添加新功能，请提交 Issue 或 Pull Request。我们欢迎任何形式的贡献！

---

## 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

---

通过以上 `README.md` 文件，您可以清晰地向用户介绍项目的功能、使用方法和编译步骤。如果有其他需求或需要进一步调整，请随时告知！