package s3share

import (
	"fmt"
	//"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/credentials"
	//"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dreyk/s3share/pkg/util"
	"log/syslog"
	"strings"
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
	cid, err := m.mountDaemon(path)
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
			if err := m.stopDaemon(path, cid); err != nil {
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
				m.slog.Warning(fmt.Sprintf("Failed unmount stailed mount '%s': %v", path, err))
				return err
			}
		}
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
		"passwd_file=/etc/passwd-s3fs",
		"-o",
		"multireq_max=5",
		"-f",
	}
	//var awsSession *session.Session
	if _, ok := m.conf["kubernetes.io/secret/aws_access_key_id"]; ok {
		id, err := util.GetSecretString(m.conf, "aws_access_key_id")
		if err != nil {
			return err
		}
		secret, err := util.GetSecretString(m.conf, "aws_access_key")
		if err != nil {
			return err
		}
		/*awsSession = session.New(&aws.Config{
			Credentials: credentials.NewStaticCredentials(id, secret, ""),
		})*/
		args1 = append(args1,
			"-e",
			fmt.Sprintf("S3User=%s", id),
			"-e",
			fmt.Sprintf("S3Secret=%s", secret),
		)
	} else {
		/*awsSession = session.New()*/
		args1 = append(args1,
			"-e",
			"S3User=''",
			"-e",
			"S3Secret=''",
		)
	}
	/*s3s := s3.New(awsSession, &aws.Config{})
	_, err = s3s.GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: &bucket,
	})
	if err != nil {
		m.slog.Warning(fmt.Sprintf("Get bucket location error %v", err))
		return fmt.Errorf("Bucket request failed: %v", err)
	}*/
	out, err := util.ExecCommand(m.exec, "docker", append(args1, args2...), "")
	if err != nil {
		return fmt.Errorf("Failed mount s3fs out='%v' error='%v'", string(out), err)
	} else {
		m.slog.Info(fmt.Sprintf("Start conntainer result %s", string(out)))
	}

	return nil
}

func (m *S3FSMount) UnMount(path string) error {
	cid, err := m.mountDaemon(path)
	if err != nil {
		return err
	}
	if cid != "" {
		if err := m.stopDaemon(path, cid); err != nil {
			return err
		}
	}
	if isMounted, err := util.IsMounted(path); err != nil {
		return err
	} else if !isMounted {
		return nil
	}
	return syscall.Unmount(path, 0)
}

func (m *S3FSMount) stopDaemon(path string, id string) error {
	m.slog.Info("Stoping container " + id)
	out, err := util.ExecCommand(m.exec, "docker", []string{"stop",
		id,
	}, "")
	if err != nil {
		m.slog.Warning(fmt.Sprintf("Failed stop docker container: %v, %v %v", id, out, err))
		return fmt.Errorf("Failed stop docker container: %v,%v %v", id, out, err)
	}
	m.slog.Info("Terminating container " + id)
	out, err = util.ExecCommand(m.exec, "docker", []string{"rm",
		id,
	}, "")
	if err != nil {
		m.slog.Warning(fmt.Sprintf("Failed remove docker container: %v, %v %v", id, out, err))
		return fmt.Errorf("Failed remove docker container: %v, %v %v", id, out, err)
	}
	return nil
}
func (m *S3FSMount) mountDaemon(path string) (string, error) {
	out, err := util.ExecCommand(m.exec, "docker", []string{"ps",
		"--filter",
		"label=flex.mount.path=" + path,
		"--format",
		`"{{.ID}}"`,
	}, "")
	if err != nil {
		m.slog.Warning(fmt.Sprintf("Failed list docker containers: %v, %v", string(out), err))
		return "", fmt.Errorf("Failed list docker containers: %v, %v", string(out), err)
	}
	if len(out) > 0 {
		return strings.Trim(string(out), "\n"), nil
	} else {
		return "", nil
	}
}
