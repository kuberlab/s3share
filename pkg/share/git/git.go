package git

import (
	"fmt"
	"log/syslog"
	"syscall"

	"github.com/kuberlab/s3share/pkg/util"
)

type GitFSMount struct {
	slog *syslog.Writer
	exec util.Interface
	conf map[string]interface{}
}

func NewGitFSMount(slog *syslog.Writer, conf map[string]interface{}) *GitFSMount {
	return &GitFSMount{slog: slog, conf: conf, exec: util.NewExec()}
}

func (m *GitFSMount) Mount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return fmt.Errorf("Failed test mount %v", err)
	} else if isMounted {
		return nil
	}
	out, err := util.ExecCommand(m.exec, "mount", []string{"-t", "tmpfs", "tmpfs", path}, "")
	if err != nil {
		return fmt.Errorf("Failed mount tmpfs out='%v' error='%v'", string(out), err)
	}
	url := m.conf["url"].(string)
	out, err = util.ExecCommand(m.exec, "git", []string{"clone", url, path}, path)
	if err != nil {
		return fmt.Errorf("Failed clone repo out='%v' error='%v'", string(out), err)
	}
	if isMounted, err := util.IsMounted(path); err != nil {
		m.slog.Warning("Can't get mount status: " + err.Error())
	} else {
		m.slog.Info(fmt.Sprintf("Mount result is %v", isMounted))
	}
	return nil
}

func (m *GitFSMount) UnMount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return fmt.Errorf("Failed test mount %v", err)
	} else if !isMounted {
		return nil
	}
	return syscall.Unmount(path, 0)
}
