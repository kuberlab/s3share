package util

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	Failure      = "Failure"
	Success      = "Success"
	NotSupported = "Not supported"
)

func IsMounted(mountpoint string) (bool, error) {
	mntpoint, err := os.Stat(mountpoint)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	parent, err := os.Stat(filepath.Join(mountpoint, ".."))
	if err != nil {
		return false, err
	}
	mntpointSt := mntpoint.Sys().(*syscall.Stat_t)
	parentSt := parent.Sys().(*syscall.Stat_t)
	return mntpointSt.Dev != parentSt.Dev, nil
}

func ExecCommand(exec Interface, command string, args []string, dir string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	if len(dir) > 0 {
		cmd.SetDir(dir)
	}
	return cmd.CombinedOutput()
}

func TryStopMountDaemon(path string) error {
	exec := NewExec()
	cid, err := MountDaemon(path, exec)
	if err != nil {
		return err
	}
	return StopDaemon(cid, exec)
}

func MountDaemon(path string, exec Interface) (string, error) {
	out, err := ExecCommand(exec, "docker", []string{"ps", "-a",
		"--filter",
		"label=flex.mount.path=" + path,
		"--format",
		`{{ .ID }}`,
	}, "")
	if err != nil {
		return "", fmt.Errorf("Failed list docker containers: %v, %v", string(out), err)
	}
	if len(out) > 0 {
		return strings.Trim(string(out), "\n"), nil
	} else {
		return "", nil
	}
}

func CheckDaemon(id string, exec Interface) error {
	// docker inspect c4e1f5a1af33 --format '{{ .State.Status }} {{ .State.ExitCode}}'
	args := []string{"inspect", id, "--format", "{{ .State.Status }}"}
	out, err := ExecCommand(exec, "docker", args, "")
	if err != nil {
		return err
	}
	outS := strings.Trim(string(out), "\n")

	if outS == "exited" {
		return errors.New("Daemon is exited")
	} else {
		return nil
	}
}

func DaemonLogs(id string, exec Interface) (string, error) {
	// docker inspect c4e1f5a1af33 --format '{{ .State.Status }} {{ .State.ExitCode}}'
	args := []string{"logs", id}
	out, err := ExecCommand(exec, "docker", args, "")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func StopDaemon(id string, exec Interface) error {
	out, err := ExecCommand(exec, "docker", []string{"rm", "--force", id}, "")
	if err != nil {
		return fmt.Errorf("Failed remove docker container: %v, %v %v", id, out, err)
	}
	return nil
}

func ParseFindMntOut(out string) (string, string, error) {
	//root/test1 /dev/mapper/ASRock--vg-home[/tmp/test2] ext4   rw,relatime,errors=remount-ro,data=ordered
	p1 := strings.Split(out, "/n")
	if len(p1) < 1 {
		return "", "", fmt.Errorf("Mount not found in '%v'", out)
	} else {
		p1 := strings.Split(p1[0], " ")
		if len(p1) < 2 {
			return "", "", fmt.Errorf("Mount not found in '%v'", out)
		}
		i1 := strings.Index(p1[1], "[")
		if i1 < 0 {
			return "", "", fmt.Errorf("Mount not found in '%v'", out)
		}
		i2 := strings.Index(p1[1], "]")
		if i2 < 0 {
			return "", "", fmt.Errorf("Mount not found in '%v'", out)
		}
		return p1[0], (p1[0])[i1:i2], nil
	}
}

func GetSecretString(conf map[string]interface{}, name string) (string, error) {
	if v, ok := conf["kubernetes.io/secret/"+name]; !ok {
		return "", fmt.Errorf("Secret '%s' not found", name)
	} else {
		if s, ok := v.(string); ok {
			sb, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return "", fmt.Errorf("Failed decode secret '%s'", name)
			}
			return strings.Trim(strings.Trim(string(sb), "\n"), "\r"), nil
		} else {
			return "", fmt.Errorf("Bad secret '%s' value", name)
		}
	}

}

func LocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}
