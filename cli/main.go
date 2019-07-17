package main

import (
	"fmt"
	"os"

	"github.com/spf13/viper"

	"github.com/eventials/go-tus"
)

func usage() {
	fmt.Println("Usage: sftus.exe [file1] [file2]")
	fmt.Println("Il faut un fichier de configuration ?")
}

func getToken() {
	type Login struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	endpoint := viper.GetString("endpoint")
	// username := viper.GetString("username")
	// password := viper.GetString("password")

	f, err := os.Open(os.Args[1])

	if err != nil {
		panic(err)
	}

	defer f.Close()

	// create the tus client.
	client, _ := tus.NewClient(endpoint, nil)

	// create an upload from a file.
	upload, _ := tus.NewUploadFromFile(f)

	// create the uploader.
	uploader, _ := client.CreateUpload(upload)

	// start the uploading process.
	uploader.Upload()
}
