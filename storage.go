package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	tusd "github.com/tus/tusd/pkg/handler"

	"github.com/spf13/viper"
)

func checkStorage(users ...string) error {
	store := viper.GetString("basePath")
	dirs, err := ioutil.ReadDir(store)
	if err != nil {
		return err
	}

	usersMap := arrayToMap(users)
	dirsMap := filesToMap(dirs)
	for k := range usersMap {
		if _, ok := dirsMap[k]; !ok {
			public, err := os.Stat(store + "/" + k + "-public")
			if err != nil || !public.IsDir() {
				return errors.New("Stockage public manquant pour " + k)
			}

			private, err := os.Stat(store + "/" + k + "/private")
			if err != nil || !private.IsDir() {
				return errors.New("Stockage privé manquant pour " + k)
			}
		}
	}
	return nil
}

func arrayToMap(array []string) map[string]struct{} {
	m := make(map[string]struct{})

	for _, v := range array {
		m[v] = struct{}{}
	}

	return m
}

func filesToMap(array []os.FileInfo) map[string]struct{} {
	m := make(map[string]struct{})

	for _, v := range array {
		m[v.Name()] = struct{}{}
	}

	return m
}

func checkGroup(goupPath string) bool {

	return true
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
