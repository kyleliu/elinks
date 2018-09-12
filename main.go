package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"
	"time"
)

var host = flag.String("host", "", "Specific listen address")
var port = flag.String("port", "32768", "Specific listen port")
var test = flag.String("test", "TestQueue.txt", "Specific test queue file")

func main() {
	flag.Parse()

	// Init test queue
	queue := CreateTestQueueFromFile(*test)
	if queue == nil {
		fmt.Println("Parse TestQueue error!")
		os.Exit(1)
	}

	// Start to listen
	cli := NewClient()
	go handleListen(&cli)

	// Wait client ready
	cli.WaitReady()

	// Do tests
	for _, q := range queue.Children {
		if q.Interface != "" {
			fmt.Printf("[T] \"%s\"(%s) 正在测试 ... \n", q.Interface, q.Name)
		}

		// Prompt user to press key
		if q.MessageBox != "" {
			fmt.Printf("[T] --- %s ---\n", q.MessageBox)
			fmt.Printf("[T] --- 按回车键继续 >>>>>> ")
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')
		}

		// Send request
		cli.SendRequest(q.Request)

		// Wait response and check keywords
		q.Pass = cli.WaitAndCheckResponse(
			q.RecTimeOut, q.ResponseKeyWord)

		// Show result
		if q.Interface != "" {
			fmt.Printf("[T] \"%s\"(%s) 测试结果 [%s]\n",
				q.Interface, q.Name, strconv.FormatBool(q.Pass))
		}
	}

	// Auto get tester's name
	username := "nobody"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	// Dump test result
	fmt.Printf("[T] =========================================================================================\n")
	fmt.Printf("[T] %s 公司 %s 产品e-Link自组网接口一致性测试报告\n", cli.vendor, cli.model)
	fmt.Printf("[T] -----------------------------------------------------------------------------------------\n")
	fmt.Printf("[T] 测试依据：《中国电信家庭终端与智能家庭网关自动连接的接口技术要求》(Q/CT2621-2017)\n")
	fmt.Printf("[T] 委托单位：北京微桥信息技术有限公司\n")
	fmt.Printf("[T] 测试地点：量子银座\n")
	fmt.Printf("[T] 测试时间：%v\n", time.Now())
	fmt.Printf("[T] 版 本 号：1.0\n")
	fmt.Printf("[T] 测试人员：%s\n", username)
	fmt.Printf("[T] =========================================================================================\n")
	fmt.Printf("[T] 序号|                测试接口名称            |     测试用例名称   |测试结果\n")
	for i, v := range queue.Children {
		if v.Interface != "" {
			fmt.Printf("[T] %2d|%-40s|%-20s|[%s]\n",
				i+1, v.Interface, v.Name, strconv.FormatBool(v.Pass))
		}
	}
}

func handleListen(cli *Client) {
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", *host+":"+*port)
	if err != nil {
		fmt.Println("Error listening:", err)
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Listening on " + *host + ":" + *port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err)
			os.Exit(1)
		}

		// logs an incoming message
		fmt.Printf("Connection %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		cli.Run(conn)
	}
}
