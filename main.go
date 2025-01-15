package main

import (
	"embed"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"github.com/pkg/errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed go.mod
var resources embed.FS

var (
	sourcePath = flag.String("s", "", "source directory")
	destPath   = flag.String("d", "", "destination directory")
	countGen   = flag.Int("c", 0, "number of backup generations")
	dry        = flag.Bool("n", false, "dry run")
)

func init() {
	common.Init("", "", "", "", "Create incremental backups with RSYNC", "", "", "", &resources, nil, nil, run, 0)
}

func checkRsync() (string, error) {
	rsyncPath, err := exec.LookPath("rsync")
	if common.Error(err) {
		return "", errors.Wrap(err, "Please install 'rsync'")
	}

	return rsyncPath, nil
}

func checkPath(path string) error {
	if !common.FileExists(path) || !common.IsDirectory(path) {
		return &common.ErrFileNotFound{path}
	}

	return nil
}

func backupName(gen int) string {
	return fmt.Sprintf("%s-Backup-%03d", filepath.Base(*sourcePath), gen)
}

func run() error {
	rsyncPath, err := checkRsync()
	if common.Error(err) {
		return err
	}

	err = checkPath(*sourcePath)
	if common.Error(err) {
		return err
	}

	err = checkPath(*destPath)
	if common.Error(err) {
		return err
	}

	if strings.HasSuffix(*sourcePath, string(os.PathSeparator)) || strings.HasSuffix(*sourcePath, "/") {
		*sourcePath = (*sourcePath)[:len(*sourcePath)-1]
	}

	if strings.HasSuffix(*destPath, string(os.PathSeparator)) || strings.HasSuffix(*destPath, "/") {
		*destPath = (*destPath)[:len(*destPath)-1]
	}

	prefix := ""
	if *dry {
		prefix = "Would "
	}

	start := time.Now()
	b := false

	for i := *countGen; ; i++ {
		path := filepath.Join(*destPath, backupName(i))
		if !common.FileExists(path) {
			break
		}

		common.Info("%sRemove obsolete path: %s", prefix, path)

		if !*dry {
			err := os.RemoveAll(path)
			if common.Error(err) {
				return err
			}
		}

		b = true
	}

	if b {
		common.Info("")
	}

	b = false

	for i := *countGen - 1; i >= 1; i-- {
		oldPath := filepath.Join(*destPath, backupName(i))
		newPath := filepath.Join(*destPath, backupName(i+1))

		if !common.FileExists(oldPath) {
			continue
		}

		common.Info("%sMove %s to %s", prefix, oldPath, newPath)

		if !*dry {
			err := os.Rename(oldPath, newPath)
			if common.Error(err) {
				return err
			}
		}

		b = true
	}

	if b {
		common.Info("")
	}

	link := fmt.Sprintf("%s%s", filepath.Join(*destPath, backupName(2)), string(os.PathSeparator))
	source := fmt.Sprintf("%s%s", *sourcePath, string(os.PathSeparator))
	dest := filepath.Join(*destPath, backupName(1))

	args := []string{}

	args = append(args, "--archive", "--verbose", "--safe-links", "--delete")

	if common.FileExists(link) {
		args = append(args, fmt.Sprintf("--link-dest=%s", link))
	}

	args = append(args, source, dest)

	cmd := exec.Command(rsyncPath, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	common.Info("%sExecute rsync: %s", prefix, common.CmdToString(cmd))

	if !*dry {
		err = cmd.Run()

		common.Info("")
		common.Info("Time needed: %v", time.Since(start))

		if common.Error(err) {
			return err
		}
	}

	return nil
}

func main() {
	common.Run([]string{"s", "d", "c"})
}
