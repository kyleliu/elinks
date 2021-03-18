package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/coredhcp/coredhcp/config"
	"github.com/coredhcp/coredhcp/logger"
	"github.com/coredhcp/coredhcp/server"
	"github.com/sirupsen/logrus"

	"github.com/coredhcp/coredhcp/plugins"
	pl_dns "github.com/coredhcp/coredhcp/plugins/dns"
	pl_file "github.com/coredhcp/coredhcp/plugins/file"
	pl_leasetime "github.com/coredhcp/coredhcp/plugins/leasetime"
	pl_nbp "github.com/coredhcp/coredhcp/plugins/nbp"
	pl_netmask "github.com/coredhcp/coredhcp/plugins/netmask"
	pl_prefix "github.com/coredhcp/coredhcp/plugins/prefix"
	pl_range "github.com/coredhcp/coredhcp/plugins/range"
	pl_router "github.com/coredhcp/coredhcp/plugins/router"
	pl_searchdomains "github.com/coredhcp/coredhcp/plugins/searchdomains"
	pl_serverid "github.com/coredhcp/coredhcp/plugins/serverid"
	pl_sleep "github.com/coredhcp/coredhcp/plugins/sleep"
)

var (
	flagHost        = flag.String("host", "", "侦听地址")
	flagPort        = flag.String("port", "32768", "侦听端口")
	flagFile        = flag.String("file", "TestQueue.txt", "测试队列文件名")
	flagTmac        = flag.String("tmac", "", "测试手机的MAC地址")
	flagLogFile     = flag.String("logfile", "", "Name of the log file to append to. Default: stdout/stderr only")
	flagLogNoStdout = flag.Bool("nostdout", false, "Disable logging to stdout/stderr")
	flagLogLevel    = flag.String("loglevel", "info", fmt.Sprintf("Log level. One of %v", getLogLevels()))
	flagConfig      = flag.String("conf", "dhcp.yml", "Use this configuration file instead of the default location")
	flagPlugins     = flag.Bool("plugins", false, "list plugins")
)

var logLevels = map[string]func(*logrus.Logger){
	"none":    func(l *logrus.Logger) { l.SetOutput(ioutil.Discard) },
	"debug":   func(l *logrus.Logger) { l.SetLevel(logrus.DebugLevel) },
	"info":    func(l *logrus.Logger) { l.SetLevel(logrus.InfoLevel) },
	"warning": func(l *logrus.Logger) { l.SetLevel(logrus.WarnLevel) },
	"error":   func(l *logrus.Logger) { l.SetLevel(logrus.ErrorLevel) },
	"fatal":   func(l *logrus.Logger) { l.SetLevel(logrus.FatalLevel) },
}

func getLogLevels() []string {
	var levels []string
	for k := range logLevels {
		levels = append(levels, k)
	}
	return levels
}

var desiredPlugins = []*plugins.Plugin{
	&pl_dns.Plugin,
	&pl_file.Plugin,
	&pl_leasetime.Plugin,
	&pl_nbp.Plugin,
	&pl_netmask.Plugin,
	&pl_prefix.Plugin,
	&pl_range.Plugin,
	&pl_router.Plugin,
	&pl_searchdomains.Plugin,
	&pl_serverid.Plugin,
	&pl_sleep.Plugin,
}

func main() {
	flag.Parse()

	// 只是输出插件信息
	if *flagPlugins {
		for _, p := range desiredPlugins {
			fmt.Println(p.Name)
		}
		os.Exit(0)
	}

	// 设置LOG
	log := logger.GetLogger("main")
	fn, ok := logLevels[*flagLogLevel]
	if !ok {
		log.Fatalf("Invalid log level '%s'. Valid log levels are %v", *flagLogLevel, getLogLevels())
	}
	fn(log.Logger)
	log.Infof("Setting log level to '%s'", *flagLogLevel)

	// LOG到文件
	if *flagLogFile != "" {
		log.Infof("Logging to file %s", *flagLogFile)
		logger.WithFile(log, *flagLogFile)
	}
	if *flagLogNoStdout {
		log.Infof("Disabling logging to stdout/stderr")
		logger.WithNoStdOutErr(log)
	}

	// DHCP配置文件
	config, err := config.Load(*flagConfig)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 注册DHCP插件
	for _, plugin := range desiredPlugins {
		if err := plugins.RegisterPlugin(plugin); err != nil {
			log.Fatalf("Failed to register plugin '%s': %v", plugin.Name, err)
		}
	}

	// 测试手机地址必须指定
	testMAC := strings.ToUpper(strings.Replace(*flagTmac, ":", "", -1))
	if len(testMAC) != 12 {
		flag.Usage()
		LogPrintln("E", "无效的测试手机MAC地址[", testMAC, "]")
		os.Exit(1)
	}

	// Init test queue
	queue, err := CreateTestQueueFromFile(*flagFile, testMAC)
	if queue == nil {
		flag.Usage()
		LogPrintln("E", "解析TestQueue错误：", err)
		os.Exit(1)
	}

	// Start dhcp server
	srv, err := server.Start(config)
	if err != nil {
		log.Fatal(err)
	}

	// Start to listen
	cli := NewClient()
	go handleListen(&cli)

	// Wait client ready
	cli.WaitReady()

	// Do tests
	for _, q := range queue {
		LogPrintln("[T]", "-----------------------------------------------------------------------------------------------")
		LogPrintln("[T]", "测试名称:", q.Name)
		if q.Interface != "" {
			LogPrintln("[T]", "接口名称:", q.Interface)
		}
		LogPrintln("[T]", "超时时间:", q.RecTimeOut, "秒")
		LogPrintln("[T]", "词语匹配:", q.ResponseKeyWord)

		// Prompt user to press key
		if q.MessageBox != "" {
			LogPrintln("[T]", "---", q.MessageBox, "---")
			LogPrintln("[T]", "---", "按回车键继续", ">>>>>>>>")
			LogEnable(false)
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
			LogEnable(true)
		}

		// Send request
		testBegin := time.Now()
		LogPrintln("[T]", "打印开始:", "vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv")
		cli.SendRequest(q.Request)

		// Wait response and check keywords
		q.Pass = cli.WaitAndCheckResponse(
			q.RecTimeOut, q.ResponseKeyWord)

		LogPrintln("[T]", "打印结束:", "^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")

		// Show result
		LogPrintln("[T]", "花费时间:", time.Now().Sub(testBegin).Seconds(), "秒")
		LogPrintln("[T]", "测试结果:", strconv.FormatBool(q.Pass))
		LogPrintln("[T]", "-----------------------------------------------------------------------------------------------")
	}

	// Auto get tester's name
	username := "nobody"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	// Dump test result
	count := 0
	LogPrintln("[T]", "===============================================================================================")
	LogPrintln("[T]", cli.vendor, "公司", cli.model, "产品e-Link自组网接口一致性测试报告")
	LogPrintln("[T]", "-----------------------------------------------------------------------------------------------")
	LogPrintln("[T]", "测试依据：", "《中国电信家庭终端与智能家庭网关自动连接的接口技术要求》(Q/CT2621-2017)")
	LogPrintln("[T]", "委托单位：", "北京微桥信息技术有限公司")
	LogPrintln("[T]", "测试地点：", "量子银座")
	LogPrintln("[T]", "测试时间：", time.Now())
	LogPrintln("[T]", "版 本 号：", "1.0")
	LogPrintln("[T]", "测试人员：", username)
	LogPrintln("[T]", "连接次数：", cli.connTimes)
	LogPrintln("[T]", "===============================================================================================")
	LogPrintln("[T]", "序号", "|", FW("测试接口名称", 40), "|", FW("测试用例名称", 34), "|", "测试结果")
	LogPrintln("[T]", "-----------------------------------------------------------------------------------------------")
	for _, v := range queue {
		if v.Interface != "" {
			count++
			index := fmt.Sprintf("%4v", count)
			title := FW(v.Interface, 40)
			name := FW(v.Name, 34)
			pass := "不通过"
			if v.Pass {
				pass = "通过"
			}
			LogPrintln("[T]", index, "|", title, "|", name, "|", pass)
		}
	}
	LogPrintln("[T]", "===============================================================================================")

	// 关闭dhcp服务器
	if err := srv.Wait(); err != nil {
		log.Print(err)
	}
	time.Sleep(time.Second)
}

func handleListen(cli *Client) {
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", *flagHost+":"+*flagPort)
	if err != nil {
		LogPrintln("[E]", "Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	LogPrintln("[I]", "Listening on "+*flagHost+":"+*flagPort)

	for {
		conn, err := l.Accept()
		if err != nil {
			LogPrintln("[E]", "Error accepting: ", err)
			os.Exit(1)
		}

		// logs an incoming message
		LogPrintln("[E]", "Connection", conn.RemoteAddr(), "->", conn.LocalAddr())
		cli.Run(conn)
	}
}
