package download

import (
	"fmt"
	"io/ioutil"
	"log/syslog"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/kuberlab/s3share/pkg/util"
)

type Mount struct {
	slog *syslog.Writer
	conf map[string]interface{}
	exec util.Interface
}

func NewDownloadMount(slog *syslog.Writer, conf map[string]interface{}) *Mount {
	return &Mount{
		slog: slog,
		conf: conf,
		exec: util.NewExec(),
	}
}

func (m *Mount) EnsureDownloaderContainer() error {
	var err error
	var lockFile *os.File
	for {
		lockFile, err = os.OpenFile("/tmp/pluk.lock", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if os.IsExist(err) {
			time.Sleep(time.Second * 2)
		}
		if os.IsNotExist(err) || err == nil {
			break
		}
	}
	defer os.Remove("/tmp/pluk.lock")
	defer lockFile.Close()

	cmd := m.exec.Command(
		"docker",
		"inspect",
		"pluk-downloader",
	)
	_, err = cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	//os.Mkdir("/pluk-downloader", os.ModePerm)

	// Start container and wait some secs
	// docker run -d -e PLUK_URL=https://dev.kuberlab.io/pluk/v1
	// -v /pluk-downloader:/pluk-downloader --name pluk-downloader
	// --network=host kuberlab/pluk-downloader:latest
	cmd = m.exec.Command(
		"docker",
		"run",
		"-d",
		"-e",
		"PLUK_URL=http://127.0.0.1:30802/pluk/v1",
		//"PLUK_URL=https://dev.kuberlab.io/pluk/v1",
		"-e",
		"DEBUG=true",
		"-e",
		"DOWNLOAD_DIR=/pluk-tmp",
		"-v",
		"/pluk-tmp:/pluk-tmp",
		"--network=host",
		"--name",
		"pluk-downloader",
		"--restart",
		"always",
		"kuberlab/pluk-downloader:latest",
	)
	_, err = cmd.CombinedOutput()
	if err == nil {
		time.Sleep(time.Second * 2)
	}

	return err
}

func (m *Mount) IsMounted(mountpoint string) (bool, error) {
	_, err := os.Stat(mountpoint)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	out, err := util.ExecCommand(m.exec, "bash", []string{"-c", fmt.Sprintf("mount | grep %v | xargs echo", mountpoint)}, "")
	if err != nil {
		return false, err
	}
	if strings.Contains(string(out), mountpoint) {
		return true, nil
	} else {
		return false, nil
	}
}

func (m *Mount) Mount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return fmt.Errorf("Failed test mount %v", err)
	} else if isMounted {
		return nil
	}

	if err := m.EnsureDownloaderContainer(); err != nil {
		return err
	}

	url := fmt.Sprintf("http://127.0.0.1:8084/v1/download/%v/%v/%v", m.conf["workspace"], m.conf["dataset"], m.conf["version"])

	var password = ""
	if _, ok := m.conf["kubernetes.io/secret/token"]; ok {
		token, err := util.GetSecretString(m.conf, "token")
		if err != nil {
			return err
		}
		password = token
	}
	// http://127.0.0.1:8084/v1/download/{workspace}/{dataset}/{version}
	// -H "X-Workspace-Name: kuberlab-demo" -H "X-Workspace-Secret: $secret"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Workspace-Name", fmt.Sprintf("%v", m.conf["workspace"]))
	req.Header.Set("X-Workspace-Secret", password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		data, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%v: %v", resp.StatusCode, string(data))
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	datasetPath := string(data)
	// mount --rbind <dataset-path> <mount-path> -o ro
	out, err := util.ExecCommand(
		m.exec,
		"mount",
		[]string{
			"--rbind",
			datasetPath,
			path,
			"-o",
			"ro",
		},
		"",
	)
	if err != nil {
		return fmt.Errorf("Failed mount tmpfs out='%v' error='%v'", string(out), err)
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
