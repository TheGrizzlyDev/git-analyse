package settings

import (
	"flag"
	"os"
	"path"
)

var (
	BasePath            = flag.String("base-path", path.Join(os.TempDir(), "git-analyse"), "")
	WorkspacePath       = path.Join(*BasePath, "workspace")
	CasPath             = path.Join(WorkspacePath, "cas")
	BisectWorkspacePath = path.Join(WorkspacePath, "bisect")
)

func init() {
	flag.Parse()

	os.MkdirAll(*BasePath, os.ModePerm)
	os.MkdirAll(WorkspacePath, os.ModePerm)
	os.MkdirAll(CasPath, os.ModePerm)
	os.MkdirAll(BisectWorkspacePath, os.ModePerm)
}
