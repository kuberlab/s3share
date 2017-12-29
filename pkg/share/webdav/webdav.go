package webdav

import (
	"fmt"
	"log/syslog"
	"strings"
	"syscall"

	"github.com/kuberlab/s3share/pkg/util"
)

type Mount struct {
	slog *syslog.Writer
	conf map[string]interface{}
	exec util.Interface
}

func NewWebDavMount(slog *syslog.Writer, conf map[string]interface{}) *Mount {
	return &Mount{
		slog: slog,
		conf: conf,
		exec: util.NewExec(),
	}
}

func (m *Mount) Mount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return fmt.Errorf("Failed test mount %v", err)
	} else if isMounted {
		return nil
	}
	urlRaw, ok := m.conf["serverURL"]
	var url string
	if !ok {
		ip, err := util.LocalIP()
		if err != nil {
			return err
		}
		url = fmt.Sprintf("http://%v:30082/webdav", ip)
	} else {
		url = urlRaw.(string)
	}
	url = strings.TrimSuffix(url, "/")
	url = fmt.Sprintf("%v/%v/%v/%v", url, m.conf["workspace"], m.conf["dataset"], m.conf["version"])

	// echo "pass" | mount -t davfs url path -o ro -o username='u'
	out, err := util.ExecCommand(
		m.exec,
		"bash",
		[]string{
			"-c",
			fmt.Sprintf(`echo "pass" | mount -t davfs "%v" "%v" -o ro -o username="u"`, url, path)},
		"",
	)
	if err != nil {
		return fmt.Errorf("Failed mount davfs out='%v' error='%v'", string(out), err)
	}

	if isMounted, err := util.IsMounted(path); err != nil {
		m.slog.Warning("Can't get mount status: " + err.Error())
	} else {
		m.slog.Info(fmt.Sprintf("Mount result is %v", isMounted))
	}
	return nil
}

func (m *Mount) UnMount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return fmt.Errorf("Failed test mount %v", err)
	} else if !isMounted {
		return nil
	}
	return syscall.Unmount(path, 0)
}
