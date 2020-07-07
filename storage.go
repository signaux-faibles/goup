package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"path/filepath"
	tusd "github.com/tus/tusd/pkg/handler"
	"github.com/spf13/viper"
)

func checkStorage(path string) error {
	err := checkDir(path)
	if err != nil {
		return err
	}
	err = checkDir("public")
	if err != nil {
		return err
	}
	return nil
}

func checkDir(path string) error {
	basePath := viper.GetString("basePath")
	fullPath := filepath.Join(basePath, path)
	file, err := os.Stat(fullPath);
	if os.IsNotExist(err) {
		err := os.Mkdir(fullPath, 0750)
		if err != nil {
			return fmt.Errorf("new directory %s can not be created", path)
		}
		group, err := user.LookupGroup(path)
		if err != nil {
			return err
		}
		gid, err := strconv.Atoi(group.Gid)
		if err != nil {
			return err
		}
		err = os.Chown(fullPath, -1, gid)
		if err != nil {
			return err
		}
	}
	if (file != nil && !file.IsDir()) {
		return fmt.Errorf("file %s should be a directory", path)
	}
	return nil
}

func linkFile(event tusd.HookEvent) error {
	file := event.Upload
	basePath := viper.GetString("basePath")
	if file.MetaData["goup-path"] == "" {
		return fmt.Errorf("this user should not be there, aborting")
	}
	var path string
	if file.MetaData["private"] == "true" {
		path = file.MetaData["goup-path"]
	} else {
		path = "public"
	}
	group, err := user.LookupGroup(path)
	if err != nil {
		return fmt.Errorf("group for %s does not exist, leaving file in tusd", path)
	}
	tusdFilePath := filepath.Join(basePath, "tusd", file.ID)
	finalFilePath := filepath.Join(basePath, path, file.ID)
	err = scanFile(tusdFilePath)
	if err != nil {
		return err
	}
	err = makeHardLinks(tusdFilePath, finalFilePath)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		return err
	}
	err = changeOwner(tusdFilePath, gid)
	if err != nil {
		return err
	}
	err = changePermissions(tusdFilePath)
	if err != nil {
		return err
	}
	return nil
}

func scanFile(path string) error {
	clamav := exec.Command(viper.GetString("clamavPath"), path)
	var b bytes.Buffer
	clamav.Stdout = &b
	clamav.Stderr = &b
	err := clamav.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			errorCode := exitError.ExitCode()
			if errorCode == 1 {
				return fmt.Errorf("virus found on file %s", path)
			}
			if errorCode == 2 {
				return fmt.Errorf("can't scan file %s, see details:\n%s", path, b.String())
			}
		}
		return err
	}
	return nil
}

func makeHardLinks(sourcePath string, targetPath string) error {
	err := os.Link(sourcePath + ".info", targetPath + ".info")
	if err != nil {
		return err
	}
	err = os.Link(sourcePath, targetPath)
	if err != nil {
		return err
	}
	return nil
}

func changeOwner(path string, gid int) error {
	err := os.Chown(path + ".info", -1, gid)
	if err != nil {
		return err
	}
	err = os.Chown(path, -1, gid)
	if err != nil {
		return err
	}
	return nil
}

func changePermissions(path string) error {
	mode := int(0640)
	err := os.Chmod(path + ".info", os.FileMode(mode))
	if err != nil {
		return err
	}
	err = os.Chmod(path, os.FileMode(mode))
	if err != nil {
		return err
	}
	return nil
}