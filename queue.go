package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// {
//	"type": "cfg",
//	"sequence": 123,
//	"mac": "mac",
//	"set": {
//		"roaming_set": {
//			"enable": "yes",
//			"threshold_rssi": -50,
//			"report_interval": 30,
//			"start_time": 60,
//			"start_rssi": -55
//		}
//	 }
// }
// ^ResponseKeyWord^roaming_report
// ^RecTimeOut^120
// ^Interface^漫游配置/终端RSSI上报(表18、19)
// ^MessageBox^请点击OK后在120秒后将下挂设备远离AP。
type TestItem struct {
	Request         interface{}
	RecTimeOut      int
	ResponseKeyWord []string
	Interface       string
	MessageBox      string
	Name            string
	Pass            bool
}

type TestQueue []*TestItem

const (
	STAMAC = "A03BE385997D"
)

func CreateTestItemFromFile(name string, mac string) *TestItem {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		LogPrintln("[E]", "Error:", err)
		return nil
	}

	f, err := os.Open(name)
	if err != nil {
		LogPrintln("[E]", "Error:", err)
		return nil
	}
	defer f.Close()

	var request string
	var timeout int
	var keywords []string
	var title string
	var message string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "^RecTimeOut^") {
			timeout, _ = strconv.Atoi(strings.TrimPrefix(line, "^RecTimeOut^"))
		} else if strings.HasPrefix(line, "^ResponseKeyWord^") {
			keywords = strings.Split(strings.TrimPrefix(line, "^ResponseKeyWord^"), "^")
			for i, v := range keywords {
				if v == STAMAC {
					keywords[i] = mac
				}
			}
		} else if strings.HasPrefix(line, "^Interface^") {
			title = strings.TrimPrefix(line, "^Interface^")
		} else if strings.HasPrefix(line, "^MessageBox^") {
			message = strings.TrimPrefix(line, "^MessageBox^")
		} else if strings.HasPrefix(line, "^") {
			LogPrintln("[W]", "Unknown line:", line)
		} else {
			request += line
		}
	}

	// 用指定的测试手机MAC地址替换请求头中的MAC地址
	request = strings.Replace(request, STAMAC, mac, -1)

	var item = new(TestItem)
	if err = json.Unmarshal([]byte(request), &item.Request); err != nil {
		LogPrintln("[E]", "Convert JSON string error:", err)
		item.Request = nil
	}
	item.Name = name
	item.RecTimeOut = timeout
	if item.RecTimeOut <= 0 {
		item.RecTimeOut = 5
	}
	item.ResponseKeyWord = keywords
	item.Interface = title
	item.MessageBox = message
	item.Pass = false
	return item
}

func CreateTestQueueFromFile(file string, mac string) (queue TestQueue, err error) {
	if _, err = os.Stat(file); os.IsNotExist(err) {
		return
	}

	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rec, _ := reader.ReadAll()
	for _, r := range rec {
		for _, c := range r {
			item := CreateTestItemFromFile(c, mac)
			if item != nil {
				queue = append(queue, item)
			}
		}
	}

	return
}
