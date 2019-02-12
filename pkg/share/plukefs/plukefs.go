package plukefs

import (
	"fmt"
	"log/syslog"
	"syscall"
	"time"

	"errors"
	"github.com/kuberlab/s3share/pkg/util"
	"strings"
)

type PlukeFSMount struct {
	slog *syslog.Writer
	exec util.Interface
	conf map[string]interface{}
}

func NewPlukeFSMount(slog *syslog.Writer, conf map[string]interface{}) *PlukeFSMount {
	return &PlukeFSMount{slog: slog, conf: conf, exec: util.NewExec()}
}

func (m *PlukeFSMount) Mount(path string) error {
	start := time.Now()
	defer func() {
		m.slog.Info(fmt.Sprintf("Time to mount: .%3f", time.Since(start).Seconds()))
	}()
	cid, err := util.MountDaemon(path, m.exec)
	if err != nil {
		return nil
	}
	if cid != "" {
		if isMounted, err := util.IsMounted(path); err != nil {
			return err
		} else if isMounted {
			return nil
		} else {
			m.slog.Warning(fmt.Sprintf("Mount point '%s' doesn't exist but container '%s' is running", path, cid))
			if err := util.StopDaemon(cid, m.exec); err != nil {
				return err
			}

		}
	} else {
		if isMounted, err := util.IsMounted(path); err != nil {
			return err
		} else if isMounted {
			m.slog.Warning(fmt.Sprintf("Mount point '%s' exists but container is not running", path))
			err := syscall.Unmount(path, 0)
			if err != nil {
				m.slog.Warning(fmt.Sprintf("Failed unmount stalled mount '%s': %v", path, err))
				return err
			}
		}
	}

	urlRaw, ok := m.conf["server"]
	var server string
	if !ok {
		ip, err := util.LocalIP()
		if err != nil {
			return err
		}
		server = fmt.Sprintf("http://%v:30802", ip)
	} else {
		server = urlRaw.(string)
	}

	var secret = ""
	if _, ok := m.conf["kubernetes.io/secret/token"]; ok {
		token, err := util.GetSecretString(m.conf, "token")
		if err != nil {
			return err
		}
		secret = token
	}
	/*
		docker run -it --rm --mount \
		type=bind,source=$(pwd)/mount,target=/mnt/mountpoint,bind-propagation=shared \
		--privileged kuberlab/plukefs:latest \
		plukefs --debug -o workspace=kuberlab-demo -o name=styles \
		-o version=1.0.0 -o server=http://192.168.0.9:8082 -o mountPoint=/mnt/mountpoint
	*/

	args1 := []string{
		"run",
		"-d",
		"--privileged",
		"-l",
		"flex.mount.path=" + path,
		"--mount",
		"type=bind,source=" + path + ",target=/mnt/mountpoint,bind-propagation=shared",
		"--cap-add",
		"SYS_ADMIN",
	}

	dsType, ok := m.conf["type"]
	if !ok || dsType == "" {
		dsType = "dataset"
	}

	// Check for required params
	_, okSW := m.conf["secret_workspace"]
	_, okOW := m.conf["object_workspace"]
	_, okN := m.conf["name"]
	_, okV := m.conf["version"]

	ok = okSW && okOW && okN && okV
	if !ok {
		return fmt.Errorf(
			"secret_workspace, object_workspace, name, version are required.",
		)
	}

	args2 := []string{
		"kuberlab/plukefs:latest",
		"plukefs",
		//"--debug",
		"-o",
		fmt.Sprintf("secret_workspace=%v", m.conf["secret_workspace"]),
		"-o",
		fmt.Sprintf("object_workspace=%v", m.conf["object_workspace"]),
		"-o",
		fmt.Sprintf("name=%v", m.conf["name"]),
		"-o",
		fmt.Sprintf("version=%v", m.conf["version"]),
		"-o",
		fmt.Sprintf("type=%v", dsType),
		"-o",
		fmt.Sprintf("server=%v", server),
		"-o",
		fmt.Sprintf("secret=%v", secret),
		"-o",
		"mountPoint=/mnt/mountpoint",
	}

	out, err := util.ExecCommand(m.exec, "docker", append(args1, args2...), "")
	if err != nil {
		return fmt.Errorf("Failed mount s3fs out='%v' error='%v'", string(out), err)
	} else {
		m.slog.Info(fmt.Sprintf("Start conntainer result %s", string(out)))
	}

	cid = strings.Trim(string(out), "\n")

	// Wait mount success.
	timeout := time.NewTimer(time.Minute * 2)
	ticker := time.NewTicker(time.Second * 2)
	mounted := false
	for {
		select {
		case <-ticker.C:
			isMounted, _ := util.IsMounted(path)
			if isMounted {
				mounted = true
				break
			}
			if err = util.CheckDaemon(cid, m.exec); err != nil {
				logs, err := util.DaemonLogs(cid, m.exec)
				util.StopDaemon(cid, m.exec)
				util.ExecCommand(m.exec, "umount", []string{"-f", path}, "")
				if err == nil {
					m.slog.Err(logs)
					return errors.New(logs)
				}
				return errors.New("Failed mount: mount daemon has been failed")
			}
		case <-timeout.C:
			m.slog.Err("Failed mount FS: timeout.")
			util.StopDaemon(cid, m.exec)
			util.ExecCommand(m.exec, "umount", []string{"-f", path}, "")
			return fmt.Errorf("Failed mount: timed out")
		}
		if mounted {
			break
		}
	}

	return nil
}

func (m *PlukeFSMount) UnMount(path string) error {
	// Unused ??
	return nil
}
