package s3share

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dreyk/s3share/pkg/util"
	"log/syslog"
	"syscall"
)

type S3FSMount struct {
	slog *syslog.Writer
	exec util.Interface
	conf map[string]interface{}
}

func NewS3FSMount(slog *syslog.Writer, conf map[string]interface{}) *S3FSMount {
	return &S3FSMount{slog: slog, conf: conf, exec: util.NewExec()}
}

func (m *S3FSMount) Mount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return err
	} else if isMounted {
		return nil
	}
	bucket := m.conf["bucket"].(string)
	args1 := []string{
		"run",
		"-d",
		"--privileged",
		"-l",
		"flex.mount.path=" + path,
		"-v",
		path + ":/mnt/mountpoint:shared",
		"--cap-add",
		"SYS_ADMIN",
	}
	args2 := []string{
		"kuberlab/s3fs",
		bucket,
		"/mnt/mountpoint",
		"-o",
		"passwd_file=/etc/s3secret/passwd-s3fs",
	}
	if v, ok := m.conf["kubernetes.io/secret/aws_access_key_id"]; ok {
		id, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return errors.New("Failed decode aws key id")
		}
		secret, err := base64.StdEncoding.DecodeString(m.conf["kubernetes.io/secret/aws_access_key"].(string))
		if err != nil {
			return errors.New("Failed decode aws key")
		}
		args1 = append(args1,
			"-e",
			fmt.Sprintf("S3User=%s", string(id)),
			"-e",
			fmt.Sprintf("S3Secret=%s", string(secret)),
		)
	}

	out, err := util.ExecCommand(m.exec, "docker", append(args1, args2...), "")
	if err != nil {
		return fmt.Errorf("Failed mount s3fs out='%v' error='%v'", string(out), err)
	}

	return nil
}

func (m *S3FSMount) UnMount(path string) error {
	out, err := util.ExecCommand(m.exec, "docker", []string{"ps",
		"--filter",
		"label=flex.mount.path=" + path,
		"--format",
		`"{{.ID}}"`,
	}, "")
	if err != nil {
		m.slog.Warning(fmt.Sprintf("Failed list docker containers: %v, %v", string(out), err))
		return fmt.Errorf("Failed list docker containers: %v, %v", string(out), err)
	}
	if len(out) > 0 {
		m.slog.Info("Terminating container " + string(out))
		out, err := util.ExecCommand(m.exec, "docker", []string{"rm",
			string(out),
		}, "")
		m.slog.Warning(fmt.Sprintf("Failed remove docker container: %v, %v", string(out), err))
		return fmt.Errorf("Failed remove docker container: %v, %v", string(out), err)
	}
	if isMounted, err := util.IsMounted(path); err != nil {
		return err
	} else if !isMounted {
		return nil
	}
	return syscall.Unmount(path, 0)
}
