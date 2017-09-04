package util

import (
	"encoding/base64"
	"fmt"
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
