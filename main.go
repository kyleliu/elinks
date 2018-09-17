package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"time"
)

var host = flag.String("host", "", "侦听地址")
var port = flag.String("port", "32768", "侦听端口")
var file = flag.String("file", "TestQueue.txt", "测试队列文件名")
var tmac = flag.String("tmac", "", "测试手机的MAC地址")

func main() {
	flag.Parse()

	testMAC := strings.ToUpper(strings.Replace(*tmac, ":", "", -1))
	if len(testMAC) != 12 {
		flag.Usage()
		LogPrintln("E", "无效的测试手机MAC地址[", testMAC, "]")
		os.Exit(1)
	}

	// Init test queue
	queue, err := CreateTestQueueFromFile(*file, testMAC)
	if queue == nil {
		flag.Usage()
		LogPrintln("E", "解析TestQueue错误：", err)
		os.Exit(1)
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
}

func handleListen(cli *Client) {
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", *host+":"+*port)
	if err != nil {
		LogPrintln("[E]", "Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	LogPrintln("[I]", "Listening on "+*host+":"+*port)

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
