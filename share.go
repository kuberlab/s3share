package main

import (
	"encoding/json"
	"fmt"
	"github.com/dreyk/s3share/pkg/share"
	"github.com/dreyk/s3share/pkg/util"
	"log/syslog"
	"os"
	"strings"
)

var slog *syslog.Writer

func main() {
	args := os.Args
	if len(args) < 2 {
		log("unknown",ResultStatus{
			Status: util.Failure,
			Message: fmt.Sprintf("Wrong args number: %d",len(args)-1),
		})
		os.Exit(-1)
	}
	var err error
	slog, err = syslog.New(syslog.LOG_WARNING|syslog.LOG_DAEMON, "kuberlab-share")
	if err != nil {
		panic(err)
	}
	switch args[1] {
	case "mount":
		if len(args) < 4 {
			log("mount",ResultStatus{
				Status: util.Failure,
				Message: fmt.Sprintf("Wrong args number: %d",len(args)-1),
			})
			os.Exit(-1)
		}
		mount(args[2], args[3])
	case "init":
		log("init",ResultStatus{
			Status:       util.Success,
			Capabilities: map[string]interface{}{"attach": false},
		})
	case "unmount":
		if len(args) < 4 {
			log("unmount",ResultStatus{
				Status: util.Failure,
				Message: fmt.Sprintf("Wrong args number: %d",len(args)-1),
			})
			os.Exit(-1)
		}
		unmount(args[2], args[3])
	default:
		log(args[1],ResultStatus{
			Status: util.NotSupported,
		})
	}

}


type ResultStatus struct {
	Status       string                 `json:"status"`
	Message      string                 `json:"message"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

func mount(path string, conf string) {
	slog.Info(fmt.Sprintf("Mount request '%s' data '%s'", path, conf))
	s := getShare("mount",conf)
	err := s.Mount(path)
	if err != nil {
		log("mount",ResultStatus{
			Status:  util.Failure,
			Message: err.Error(),
		})
		os.Exit(1)
	}
	log("mount",ResultStatus{
		Status: util.Success,
	})
}
func unmount(path string, conf string) {
	slog.Info(fmt.Sprintf("Unmount request '%s' data '%s'", path, conf))
	s := getShare("unmount",conf)
	err := s.UnMount(path)
	if err != nil {
		log("unmount",ResultStatus{
			Status:  util.Failure,
			Message: err.Error(),
		})
		os.Exit(1)
	}
	log("unmount",ResultStatus{
		Status: util.Success,
	})
}

func getShare(command string,conf string) share.Share {
	c, err := getConf(conf)
	if err != nil {
		log(command,ResultStatus{
			Status:  util.Failure,
			Message: err.Error(),
		})
		os.Exit(1)
	}
	s, err := share.NewShare(c)
	if err != nil {
		log(command,ResultStatus{
			Status:  util.Failure,
			Message: err.Error(),
		})
		os.Exit(1)
	}
	return s

}

func log(command string,res ResultStatus) {
	slog.Info(fmt.Sprintf("Command '%s' result '%s' message: %s", command,res.Status,res.Message))
	log0(res)
}
func log0(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(v)
}

func getConf(s string) (map[string]interface{}, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	var c map[string]interface{}
	err := dec.Decode(&c)
	if err != nil {
		return nil, fmt.Errorf("Decode share param failed: %v", err)
	}
	return c, nil
}
