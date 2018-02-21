package share

import (
	"fmt"
	"log/syslog"

	"github.com/kuberlab/s3share/pkg/share/download"
	"github.com/kuberlab/s3share/pkg/share/git"
	"github.com/kuberlab/s3share/pkg/share/plukefs"
	"github.com/kuberlab/s3share/pkg/share/s3share"
	"github.com/kuberlab/s3share/pkg/share/webdav"
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
				case "download":
					return download.NewDownloadMount(slog, c), nil
				case "git":
					return git.NewGitFSMount(slog, c), nil
				case "plukefs":
					return plukefs.NewPlukeFSMount(slog, c), nil
				case "s3":
					return s3share.NewS3FSMount(slog, c), nil
				case "webdav":
					return webdav.NewWebDavMount(slog, c), nil
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
