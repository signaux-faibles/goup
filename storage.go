package main

import (
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"

	"github.com/tus/tusd"

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

func linkFile(file tusd.FileInfo) error {
	basePath := viper.GetString("basePath")
	tusdStore := basePath + "/tusd/"

	usersgroup, err := user.LookupGroup("users")
	if err != nil {
		return err
	}
	goupgroup, err := user.LookupGroup("goup")
	if err != nil {
		return err
	}
	user, err := user.Lookup(file.MetaData["goup-path"])
	if err != nil {
		return err
	}

	owner, _ := strconv.Atoi(user.Uid)
	group, _ := strconv.Atoi(usersgroup.Gid)

	var userStore string
	if file.MetaData["private"] == "true" {
		userStore = basePath + "/" + file.MetaData["goup-path"] + "/"
		group, _ = strconv.Atoi(goupgroup.Gid)
	} else {
		userStore = basePath + "/" + "public/"
	}

	err = os.Link(tusdStore+file.ID+".info", userStore+file.ID+".info")
	if err != nil {
		return err
	}

	err = os.Link(tusdStore+file.ID+".bin", userStore+file.ID+".bin")
	if err != nil {
		return err
	}

	err = os.Chmod(tusdStore+file.ID+".info", 0660)
	if err != nil {
		return err
	}

	err = os.Chown(tusdStore+file.ID+".info", owner, group)
	if err != nil {
		return err
	}

	err = os.Chmod(tusdStore+file.ID+".bin", 0660)
	if err != nil {
		return err
	}

	err = os.Chown(tusdStore+file.ID+".bin", owner, group)
	if err != nil {
		return err
	}

	return nil
}
