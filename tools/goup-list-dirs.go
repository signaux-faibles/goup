package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/user"
	"syscall"

	"github.com/tus/tusd"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// FileInfo contient les informations souhaitées
type FileInfo struct {
	FileName string         `json:"filename"`
	Type     string         `json:"type"`
	Owner    string         `json:"owner"`
	Group    string         `json:"group"`
	Size     int64          `json:"size"`
	Mode     string         `json:"mode"`
	TusdInfo *tusd.FileInfo `json:"tusdInfo,omitempty"`
}

func main() {
	viper.SetConfigName("listdir")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	hostname := viper.GetString("hostname")
	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{hostname}
	router.Use(cors.New(config))
	router.GET("/list", list)

	bind := viper.GetString("bind")
	router.Run(bind)
}

func list(c *gin.Context) {
	basePath := viper.GetString("basePath")
	list, err := listDir(basePath)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, list)
}

func listDir(path string) ([]FileInfo, error) {
	var ret []FileInfo
	dirs, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		userName, err := user.LookupId(fmt.Sprint(d.Sys().(*syscall.Stat_t).Uid))
		if err != nil {
			return nil, err
		}

		groupName, err := user.LookupGroupId(fmt.Sprint(d.Sys().(*syscall.Stat_t).Gid))
		if err != nil {
			return nil, err
		}
		f := FileInfo{
			FileName: path + "/" + d.Name(),
			Owner:    userName.Username,
			Group:    groupName.Name,
			Size:     d.Size(),
			Mode:     d.Mode().String(),
		}
		if d.IsDir() {
			f.Type = "directory"
			subDirs, err := listDir(path + "/" + d.Name())
			if err != nil {
				return nil, err
			}
			ret = append(ret, f)
			ret = append(ret, subDirs...)
		} else {
			f.Type = "file"
			l := len(d.Name())
			if d.Name()[l-4:l] == "info" {
				nfo, err := ioutil.ReadFile(path + "/" + d.Name())
				if err != nil {
					return nil, err
				}
				var info tusd.FileInfo
				json.Unmarshal(nfo, &info)
				f.TusdInfo = &info
			}
			ret = append(ret, f)
		}
	}
	return ret, nil
}
