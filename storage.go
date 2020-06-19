package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

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
	file := event.Upload
	linkFile := viper.GetString("linkFile")
	basePath := viper.GetString("basePath")

	if checkGroup(file.MetaData["goup-path"]) {
		group := "public"
		if file.MetaData["private"] == "true" {
			group = file.MetaData["goup-path"]
		}

		var b bytes.Buffer
		cmd := exec.Command("sudo", linkFile, basePath, file.ID, group)
		cmd.Stderr = &b
		cmd.Stdout = &b
		cmd.Run()
		fmt.Println(b.String())
	}

	return nil
}
