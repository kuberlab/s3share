package share

import (
	"fmt"
	"github.com/dreyk/s3share/pkg/share/git"
	"log/syslog"
)

type Share interface {
	Mount(path string) error
	UnMount(path string) error
}

func NewShare(slog *syslog.Writer, c map[string]interface{}) (Share, error) {
	if t, ok := c["kuberlabFS"]; ok {
		if s, ok := t.(string); ok {
			if s == "" {
				return nil, fmt.Errorf("FS type to share is not defined")
			} else {
				switch s {
				case "git":
					return git.NewGitFSMount(slog, c), nil
				default:
					return nil, fmt.Errorf("FS type '%s' is not supported", s)
				}
			}
		} else {
			return nil, fmt.Errorf("Not supported FS type format")
		}
	} else {
		return nil, fmt.Errorf("FS type to share is not defined")
	}
}
