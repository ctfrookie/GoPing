package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
)

// ANSI颜色编码（用于终端输出）
const (
	ColorReset  = "\033[0m"
	ColorGreen  = "\033[32m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

// 默认值
const (
	defaultConcurrency = 100
	defaultTimeout     = 800 // 毫秒
)

// 存储ping结果的结构体
type PingResult struct {
	IP    string
	Alive bool
}

func main() {
	// 定义命令行参数
	var (
		cidrs   string
		logFile string
		timeout int
		threads int
	)

	flag.StringVar(&cidrs, "c", "", "CIDR地址列表（逗号分隔）")
	flag.StringVar(&logFile, "o", "ping_log.txt", "日志文件路径")
	flag.IntVar(&timeout, "t", defaultTimeout, "Ping超时时间（毫秒）")
	flag.IntVar(&threads, "n", defaultConcurrency, "并发线程数")

	// 自定义帮助信息
	flag.Usage = func() {
		fmt.Printf("%sUsage: goping -c <CIDRs> [options]%s\n", ColorYellow, ColorReset)
		fmt.Printf("Options:\n")
		fmt.Printf("  -c string   CIDR地址列表（逗号分隔） (e.g. \"10.0.0.0/24,192.168.1.0/24\")\n")
		fmt.Printf("  -o string   日志文件路径 (default \"ping_log.txt\")\n")
		fmt.Printf("  -t int      Ping超时时间（毫秒） (default %d)\n", defaultTimeout)
		fmt.Printf("  -n int      并发线程数 (default %d)\n", defaultConcurrency)
		fmt.Printf("\n%sExample:%s\n", ColorCyan, ColorReset)
		fmt.Printf("  goping -c 10.0.0.0/24,192.168.1.0/24 -o scan.log -t 500 -n 200\n")
	}

	// 解析命令行参数
	flag.Parse()

	// 检查必需的CIDR参数
	if cidrs == "" {
		fmt.Printf("%sError: Missing required -c parameter%s\n", ColorRed, ColorReset)
		flag.Usage()
		return
	}

	// 确保日志文件目录存在
	dir := filepath.Dir(logFile)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Printf("%sError creating directory: %s%s\n", ColorRed, err, ColorReset)
		return
	}

	// 打开或创建日志文件
	logFileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("%sError opening log file: %s%s\n", ColorRed, err, ColorReset)
		return
	}
	defer logFileHandle.Close()

	// 验证参数
	if timeout <= 0 {
		fmt.Printf("%sWarning: Invalid timeout value (%d), using default %d ms%s\n",
			ColorYellow, timeout, defaultTimeout, ColorReset)
		timeout = defaultTimeout
	}

	if threads <= 0 {
		fmt.Printf("%sWarning: Invalid thread count (%d), using default %d%s\n",
			ColorYellow, threads, defaultConcurrency, ColorReset)
		threads = defaultConcurrency
	}

	// 显示当前配置
	fmt.Printf("%sConfiguration:%s\n", ColorCyan, ColorReset)
	fmt.Printf("  CIDRs:      %s\n", cidrs)
	fmt.Printf("  Log file:   %s\n", logFile)
	fmt.Printf("  Timeout:    %d ms\n", timeout)
	fmt.Printf("  Threads:    %d\n", threads)
	fmt.Println()

	// 处理每个CIDR
	cidrList := splitCIDRs(cidrs)
	if len(cidrList) == 0 {
		fmt.Printf("%sError: No valid CIDRs provided%s\n", ColorRed, ColorReset)
		return
	}

	for _, cidr := range cidrList {
		subnets := splitIntoSubnets(cidr)
		if len(subnets) == 0 {
			fmt.Printf("%sWarning: No subnets found for CIDR: %s%s\n", ColorYellow, cidr, ColorReset)
			continue
		}
		for _, subnet := range subnets {
			processCIDR(subnet, logFileHandle, timeout, threads)
			fmt.Println() // 每个子网之间增加一个空行
		}
	}
}

// 分割CIDR地址列表（支持逗号或空格分隔）
func splitCIDRs(input string) []string {
	var cidrs []string
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' '
	})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && isValidCIDR(part) {
			cidrs = append(cidrs, part)
		}
	}
	return cidrs
}

// 将大CIDR拆分为/24子网
func splitIntoSubnets(cidr string) []string {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Printf("%sError parsing CIDR %s: %s%s\n", ColorRed, cidr, err, ColorReset)
		return []string{}
	}

	// 如果已经是/24或更小，直接返回
	ones, bits := ipNet.Mask.Size()
	if ones >= 24 || bits != 32 {
		return []string{cidr}
	}

	// 拆分为/24子网
	var subnets []string
	step := 1 << (32 - 24) // 每个子网的大小（256）
	start := ip.Mask(ipNet.Mask).To4()
	if start == nil {
		return []string{cidr}
	}

	current := make(net.IP, len(start))
	copy(current, start)

	for {
		// 创建当前子网
		subnet := &net.IPNet{
			IP:   current,
			Mask: net.CIDRMask(24, 32),
		}
		subnets = append(subnets, subnet.String())

		// 计算下一个子网起始地址：当前地址+256
		ipInt := uint32(current[0])<<24 | uint32(current[1])<<16 |
			uint32(current[2])<<8 | uint32(current[3])
		ipInt += uint32(step)

		// 转换为IP地址
		current[0] = byte(ipInt >> 24)
		current[1] = byte(ipInt >> 16)
		current[2] = byte(ipInt >> 8)
		current[3] = byte(ipInt)

		// 检查是否超出原始CIDR范围
		if !ipNet.Contains(current) {
			break
		}
	}
	return subnets
}

// 处理单个CIDR网段
func processCIDR(cidr string, logFile *os.File, timeoutMs, maxThreads int) {
	ips, err := parseCIDR(cidr)
	if err != nil {
		fmt.Printf("%sError parsing CIDR %s: %s%s\n", ColorRed, cidr, err, ColorReset)
		return
	}

	totalIPs := len(ips)
	if totalIPs == 0 {
		fmt.Printf("%sWarning: No IPs to scan for CIDR: %s%s\n", ColorYellow, cidr, ColorReset)
		return
	}

	fmt.Printf("\n%sProcessing Subnet: %s (%d IPs)%s\n", ColorYellow, cidr, totalIPs, ColorReset)
	logToFile(logFile, fmt.Sprintf("\n=== Processing Subnet: %s ===", cidr))

	// 添加进度显示
	progress := make(chan int, totalIPs)
	done := make(chan struct{})
	go showProgress(totalIPs, progress, done)

	results := pingAllWithConcurrency(ips, progress, timeoutMs, maxThreads)
	close(done) // 通知进度显示完成

	// 格式化输出结果
	fmt.Printf("\n%sScan completed for Subnet: %s%s\n", ColorYellow, cidr, ColorReset)
	printResults(results, timeoutMs)

	// 保存排序后的结果到日志文件
	saveSortedResultsToLog(logFile, results, cidr, timeoutMs)
	logToFile(logFile, fmt.Sprintf("=== Completed Subnet: %s ===\n", cidr))
}

// 显示实时进度
func showProgress(total int, progress <-chan int, done <-chan struct{}) {
	var count int
	startTime := time.Now()

	for {
		select {
		case _, ok := <-progress:
			if !ok {
				return
			}
			count++
			percent := float64(count) / float64(total) * 100

			// 计算预计剩余时间
			elapsed := time.Since(startTime)
			remaining := time.Duration(float64(elapsed) / float64(count) * float64(total-count))

			fmt.Printf("\r%sProgress: %.1f%% (%d/%d) | Elapsed: %s | ETA: %s%s",
				ColorYellow,
				percent,
				count,
				total,
				formatDuration(elapsed),
				formatDuration(remaining),
				ColorReset)
		case <-done:
			fmt.Printf("\r%sProgress: 100.0%% (%d/%d) | Elapsed: %s | Completed%s\n",
				ColorGreen,
				total,
				total,
				formatDuration(time.Since(startTime)),
				ColorReset)
			return
		}
	}
}

// 格式化时间显示
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}

	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}

	return fmt.Sprintf("%.1fmin", d.Minutes())
}

// 解析CIDR网段
func parseCIDR(cidr string) ([]string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR format: %w", err)
	}

	// 检查IPv4
	if ip.To4() == nil {
		return nil, fmt.Errorf("IPv6 not supported, please use IPv4 CIDR")
	}

	var ips []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}

	// 排除网络地址和广播地址
	if len(ips) < 2 {
		return nil, fmt.Errorf("network too small for scanning")
	}
	return ips[1 : len(ips)-1], nil
}

// IP地址递增
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// 使用协程池Ping所有IP
func pingAllWithConcurrency(ips []string, progress chan<- int, timeoutMs, maxThreads int) []PingResult {
	var (
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, maxThreads)
		results   = make(chan PingResult, len(ips))
	)

	// 预分配切片空间
	pingResults := make([]PingResult, 0, len(ips))

	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			semaphore <- struct{}{} // 占用一个协程槽

			result := pingIP(ip, timeoutMs)
			results <- result

			<-semaphore   // 释放一个协程槽
			progress <- 1 // 更新进度
		}(ip)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	for res := range results {
		pingResults = append(pingResults, res)
	}

	// 返回未排序的结果
	return pingResults
}

// Ping单个IP
func pingIP(ip string, timeoutMs int) PingResult {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return PingResult{IP: ip, Alive: false}
	}

	// 在Windows上需要管理员权限
	pinger.SetPrivileged(true)

	pinger.Count = 1
	pinger.Timeout = time.Duration(timeoutMs) * time.Millisecond
	pinger.SetNetwork("ip4") // 强制使用IPv4

	err = pinger.Run()
	if err != nil {
		return PingResult{IP: ip, Alive: false}
	}

	stats := pinger.Statistics()
	return PingResult{IP: ip, Alive: stats.PacketsRecv > 0}
}

// 获取IP地址的尾号
func getLastOctet(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 0 {
		return ip
	}
	return parts[len(parts)-1]
}

// 写入日志文件
func logToFile(file *os.File, message string) {
	_, err := file.WriteString(message + "\n")
	if err != nil {
		fmt.Printf("%sError writing to log file: %s%s\n", ColorRed, err, ColorReset)
	}
}

// 格式化输出结果
func printResults(results []PingResult, timeoutMs int) {
	const columns = 20

	// 按尾号数字排序
	sort.Slice(results, func(i, j int) bool {
		numI, _ := strconv.Atoi(getLastOctet(results[i].IP))
		numJ, _ := strconv.Atoi(getLastOctet(results[j].IP))
		return numI < numJ
	})

	// 转换为彩色输出
	output := make([]string, len(results))
	for i, res := range results {
		color := ColorRed
		if res.Alive {
			color = ColorGreen
		}
		output[i] = fmt.Sprintf("%s%s%s", color, getLastOctet(res.IP), ColorReset)
	}

	// 打印结果
	for i := 0; i < len(output); i += columns {
		end := i + columns
		if end > len(output) {
			end = len(output)
		}

		line := ""
		for _, result := range output[i:end] {
			line += fmt.Sprintf("%-15s", result)
		}
		fmt.Println(line)
	}

	// 统计结果
	aliveCount := 0
	for _, r := range results {
		if r.Alive {
			aliveCount++
		}
	}

	fmt.Printf("\n%sAlive: %s%d%s | Dead: %s%d%s | Total: %d | Timeout: %d ms%s\n",
		ColorYellow,
		ColorGreen, aliveCount, ColorYellow,
		ColorRed, len(results)-aliveCount, ColorYellow,
		len(results),
		timeoutMs,
		ColorReset)
}

// 保存排序后的结果到日志文件
func saveSortedResultsToLog(logFile *os.File, results []PingResult, cidr string, timeoutMs int) {
	// 按尾号数字排序
	sort.Slice(results, func(i, j int) bool {
		numI, _ := strconv.Atoi(getLastOctet(results[i].IP))
		numJ, _ := strconv.Atoi(getLastOctet(results[j].IP))
		return numI < numJ
	})

	// 写入标题
	logToFile(logFile, fmt.Sprintf("=== Results for Subnet: %s (Timeout: %d ms) ===", cidr, timeoutMs))
	logToFile(logFile, "IP Address      Status")
	logToFile(logFile, "-----------------------")

	// 写入结果
	for _, res := range results {
		status := "DOWN"
		if res.Alive {
			status = "UP"
		}
		logToFile(logFile, fmt.Sprintf("%-15s %s", res.IP, status))
	}

	logToFile(logFile, "=======================")
}

// 验证CIDR格式是否有效
func isValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}
