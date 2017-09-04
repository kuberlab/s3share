package git

import (
	"fmt"
	"github.com/dreyk/s3share/pkg/util"
	"github.com/pborman/uuid"
	"log/syslog"
	"os"
	"path/filepath"
	"syscall"
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
		return err
	} else if isMounted {
		return nil
	}
	url := m.conf["url"].(string)
	rootMnt := filepath.Join(os.TempDir(), uuid.New())
	m.slog.Info(fmt.Sprintf("Clone %s to %s", url, rootMnt))
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return fmt.Errorf("Failed create tmp dir for clonning repo: %v", err)
	}
	out, err := util.ExecCommand(m.exec, "git", []string{"clone", url, rootMnt}, rootMnt)
	if err != nil {
		return fmt.Errorf("Failed clone repo out='%v' error='%v'", string(out), err)
	}
	m.slog.Info(fmt.Sprintf("Mount %s to %s", rootMnt, path))
	out, err = util.ExecCommand(m.exec, "mount", []string{"--bind", rootMnt, path}, "")
	if err != nil {
		return fmt.Errorf("Failed bind dirs out='%v' error='%v'", string(out), err)
	}
	if isMounted, err := util.IsMounted(path); err != nil {
		m.slog.Warning("Can't et mount status: " + err.Error())
	} else {
		m.slog.Info(fmt.Sprintf("Mount result is %v", isMounted))
	}
	return nil
}

func (m *GitFSMount) UnMount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return err
	} else if !isMounted {
		return nil
	}
	out, err := util.ExecCommand(m.exec, "findmnt", []string{"-n", path}, "")
	if err != nil {
		return fmt.Errorf("Failed find mount point", string(out), err)
	}
	err = syscall.Unmount(path, 0)
	if err != nil {
		return err
	}
	_, p2, err := util.ParseFindMntOut(string(out))
	if err != nil {
		m.slog.Warning(fmt.Sprintf("Can't dinf mount source in %s", out))
	} else {
		m.slog.Info(fmt.Sprintf("Clear mout source %s", p2))
		err := os.RemoveAll(p2)
		if err != nil {
			m.slog.Warning(fmt.Sprintf("Failed clear mount source %s: %v", p2, err))
		}
	}
	return nil
}
