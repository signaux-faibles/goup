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
	fmt.Println("checkStorage")
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
	if (!file.IsDir()) {
		return fmt.Errorf("file %s should be a directory", path)
	}
	return nil
}

func linkFile(event tusd.HookEvent) error {
	var err error
	var group *user.Group
	var store string

	file := event.Upload
	basePath := viper.GetString("basePath")
	tusdStore := basePath + "/tusd/"

	if file.MetaData["goup-path"] == "" {
		return fmt.Errorf("this user should not be there, aborting")
	}

	if file.MetaData["private"] == "true" {
		store = basePath + "/" + file.MetaData["goup-path"] + "/"
		group, err = user.LookupGroup(file.MetaData["goup-path"])
	} else {
		store = basePath + "/" + "public/"
		group, err = user.LookupGroup("public")
	}

	if err != nil {
		return fmt.Errorf("group for %s does not exist, leaving file in tusd", store)
	}

	clamav := exec.Command("/usr/bin/clamscan", tusdStore+file.ID)
	var b bytes.Buffer
	clamav.Stdout = &b
	clamav.Stderr = &b
	errClamav := clamav.Run()
	if errClamav != nil {
		errorCode := errClamav.(*exec.ExitError).ExitCode()
		if errorCode == 1 {
			return fmt.Errorf("virus found on file %s", tusdStore+file.ID)
		}
		if errorCode == 2 {
			return fmt.Errorf("couldn't scan file %s \n, detail:\n%s", tusdStore+file.ID, b.String())
		}
	}

	err = os.Link(tusdStore+file.ID+".info", store+file.ID+".info")
	if err != nil {
		return err
	}

	err = os.Link(tusdStore+file.ID, store+file.ID)
	if err != nil {
		return err
	}

	gid, _ := strconv.Atoi(group.Gid)
	err = os.Chown(tusdStore+file.ID+".info", -1, gid)
	if err != nil {
		return err
	}

	err = os.Chown(tusdStore+file.ID, -1, gid)
	if err != nil {
		return err
	}

	return nil
}
