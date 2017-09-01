package host

import (
	"github.com/dreyk/s3share/pkg/util"
	"os/exec"
	"syscall"
)

type HostFSMount struct {
	Conf map[string]interface{}
}

func NewHostFSMount(conf map[string]interface{}) *HostFSMount {
	return &HostFSMount{Conf: conf}
}

func (m *HostFSMount) Mount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return err
	} else if isMounted {
		return nil
	}
	//mount --bind /root/kuberlab-up /root/test-mount
	mpath := m.Conf["path"].(string)
	cmd := exec.Command("mount", "--bind",mpath,path)
	return cmd.Run()
}

func (m *HostFSMount) UnMount(path string) error {
	if isMounted, err := util.IsMounted(path); err != nil {
		return err
	} else if !isMounted {
		return nil
	}
	return syscall.Unmount(path, 0)
}
