package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/containous/traefik/log"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const regular = `^(13[0-9]|14[579]|15[0-3,5-9]|16[6]|17[0135678]|18[0-9]|19[89])\d{8}$`

type WatchOnit struct {
	Proc      string
	Args      []string
	Contacts  []string
	HeartBeat int64
	APIADDR   string
}


func validate(mobileNum string) bool {
	reg := regexp.MustCompile(regular)
	return reg.MatchString(mobileNum)
}

func FromCmd(CmdArgs WatchOnit) WatchOnit {
	var contact, arg, num string

	//不确定是不是应该写一个-h或者用别的方式来设计.... 明天问
	flag.StringVar(&CmdArgs.Proc, "cmd", "", "需要监控的程序")
	flag.StringVar(&arg, "args", "", "程序启动的参数")
	flag.StringVar(&contact, "tel", "", "告警联系人电话,多个时用逗号分开")
	flag.StringVar(&CmdArgs.APIADDR, "api", "", "短信api地址")
	flag.Int64Var(&CmdArgs.HeartBeat, "hb", 60, "心跳频率,单位:秒")

	CmdArgs.Args[0] = arg
	numbers := strings.Split(contact, ",")

	for num = range numbers {
		if validate(num) {
			CmdArgs.Contacts = append(CmdArgs.Contacts, num)
		} else {
			fmt.Println("请输入合法的电话号码,多个时以逗号分开")
		}
	}

	return CmdArgs
}

func MsgOrNot(CmdArgs WatchOnit) string {
	if len(CmdArgs.Contacts) != 0 {
		MSG := make(map[string]interface{})
		MSG["mobiles"] = CmdArgs.Contacts
		MSG["content"] = fmt.Sprintf("%s又挂啦,修不了啦", CmdArgs.Proc)

		bytesData, _ := json.Marshal(MSG)
		/*
			if err != nil {
				fmt.Println(err.Error())
				log.Error(err)
				return string("")
			}
		*/
		reader := bytes.NewReader(bytesData)
		url := fmt.Sprintf("%s%s", CmdArgs.APIADDR, CmdArgs.Proc)
		req, _ := http.NewRequest("POST", url, reader)
		/*
			if err != nil {
				fmt.Println(err.Error())
				log.Error(err)
				return
			}
		*/
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		client := http.Client{}
		response, _ := client.Do(req)
		body, _ := ioutil.ReadAll(response.Body)
		var respmsg map[string]interface{}
		err := json.Unmarshal(body, &respmsg)
		if err != nil {
			fmt.Println(err.Error())
		}
		return respmsg["msg"].(string)
	}
	return string("没写联系人，我也不知道联系谁")
}

func uccu(cmd WatchOnit) {

	FromCmd(cmd)
	flag.Parse()

	Attr := &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	}
	//一旦监控的程序或者参数提交错误 是不是会引起这个程序无限重启导致死循环.... 不太明白这里为啥不用signal控制重启...
	p, err := os.StartProcess(cmd.Proc, cmd.Args, Attr)

	log.Info(p)

	if err != nil {
		log.Error(err)
		return
	}
	r, err := p.Wait()
	if err != nil {
		log.Error(err)
		return
	}
	log.Info(r)
	//  重启后发告警短信

	time.Sleep(time.Duration(cmd.HeartBeat) * time.Second)
}
func main() {
	var u WatchOnit
	for {
		defer MsgOrNot(u)
		uccu(u)
	}
}
