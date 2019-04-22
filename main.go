package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/tus/tusd"
	"github.com/tus/tusd/filestore"

	jwt "github.com/appleboy/gin-jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Payload est l'object véhiculé par le token datapi
type Payload struct {
	Email string                 `json:"email"`
	Scope []string               `json:"scope"`
	Value map[string]interface{} `json:"value"`
}

// File décrit les metadonnées d'un fichier
type File struct {
	Domain string   `json:"domain"`
	Name   string   `json:"name"`
	Tags   []string `json:"tags"`
}

func payloadHandler(data interface{}) jwt.MapClaims {
	if v, ok := data.(*Payload); ok {
		return jwt.MapClaims{
			"email": v.Email,
			"scope": v.Scope,
			"value": v.Value,
		}
	}
	return jwt.MapClaims{}
}

func identityHandler(c *gin.Context) interface{} {
	claims := jwt.ExtractClaims(c)
	email, ok := claims["email"].(string)
	if !ok {
		return nil
	}

	user := email

	return &user
}

func authorizatorHandler(data interface{}, c *gin.Context) bool {
	return true
}

func unauthorizedHandler(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"code":    code,
		"message": message,
	})
}

func authenticator(c *gin.Context) (interface{}, error) {
	var loginVals struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBind(&loginVals); err != nil {
		return "", jwt.ErrMissingLoginValues
	}
	userID := loginVals.Email
	password := loginVals.Password

	if userID == "user1" && password == "upload" {
		return &Payload{
			Email: loginVals.Email,
			Scope: []string{},
			Value: map[string]interface{}{
				"goup-path": "user1",
			},
		}, nil
	}

	if userID == "user2" && password == "upload" {
		return &Payload{
			Email: loginVals.Email,
			Scope: []string{},
			Value: map[string]interface{}{
				"goup-path": "user2",
			},
		}, nil
	}

	if userID == "user3" && password == "noupload" {
		return &Payload{
			Email: loginVals.Email,
			Scope: []string{},
		}, nil
	}

	return nil, jwt.ErrFailedAuthentication
}

func processUpload() chan tusd.FileInfo {
	channel := make(chan tusd.FileInfo)
	go func() {
		for file := range channel {
			err := linkFile(file)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	return channel
}

func addMetadata(c *gin.Context) {
	metadata := c.Request.Header.Get("upload-metadata")
	claims := jwt.ExtractClaims(c)

	value, ok := claims["value"].(map[string]interface{})
	if !ok {
		c.JSON(403, "Forbidden")
		c.Abort()
		return
	}

	pathInterface, ok := value["goup-path"]
	if !ok {
		c.JSON(403, "Forbidden")
		c.Abort()
		return
	}

	path, ok := pathInterface.(string)
	if !ok {
		c.JSON(403, "Forbidden")
		c.Abort()
		return
	}

	err := checkStorage(path)
	if err != nil {
		fmt.Println(path)
		c.JSON(500, err.Error())
		c.Abort()
		return
	}

	metadata = metadata + ", goup-path " + base64.StdEncoding.EncodeToString([]byte(path))
	c.Request.Header.Set("upload-metadata", metadata)
	c.Next()
}

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	if !isDirectory(viper.GetString("basePath") + "/tusd/") {
		panic("Absence du répertoire de base")
	}

	store := filestore.FileStore{
		Path: viper.GetString("basePath") + "/tusd/",
	}
	composer := tusd.NewStoreComposer()
	composer.UsesGetReader = false

	store.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:              "/files/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
	})

	handler.CompleteUploads = processUpload()

	if err != nil {
		panic(fmt.Errorf("Unable to create handler: %s", err))
	}

	authMiddleware, err := jwt.New(&jwt.GinJWTMiddleware{
		Realm:           "Signaux-Faibles",
		Key:             []byte(viper.GetString("jwtSecret")),
		SendCookie:      false,
		Timeout:         time.Hour,
		MaxRefresh:      time.Hour,
		IdentityKey:     "id",
		PayloadFunc:     payloadHandler,
		IdentityHandler: identityHandler,
		Authenticator:   authenticator,
		Authorizator:    authorizatorHandler,
		Unauthorized:    unauthorizedHandler,
		TokenLookup:     "header: Authorization, query: token",
		TokenHeadName:   "Bearer",
		TimeFunc:        time.Now,
	})

	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://goup.signaux-faibles.beta.gouv.fr/"}
	config.AddAllowHeaders("Authorization", "tus-resumable", "upload-length", "upload-metadata", "upload-offset", "Location")
	router.Use(cors.New(config))
	router.POST("/login", authMiddleware.LoginHandler)
	router.GET("/list", authMiddleware.MiddlewareFunc(), list)

	router.POST("/files/*any", authMiddleware.MiddlewareFunc(), addMetadata, gin.WrapH(http.StripPrefix("/files/", handler)))
	router.HEAD("/files/*any", authMiddleware.MiddlewareFunc(), gin.WrapH(http.StripPrefix("/files/", handler)))
	router.PATCH("/files/*any", authMiddleware.MiddlewareFunc(), gin.WrapH(http.StripPrefix("/files/", handler)))
	bind := viper.GetString("bind")
	router.Run(bind)
}

func isDirectory(path string) bool {
	fileInfo, _ := os.Stat(path)

	if fileInfo != nil && fileInfo.IsDir() {
		return true
	}
	return false
}

func list(c *gin.Context) {
	a := jwt.ExtractClaims(c)

	basePath := viper.GetString("basePath")
	uploadPath, ok := a["value"].(map[string]interface{})["upload_path"].(string)

	if !ok {
		c.JSON(403, "L'utilisateur n'a pas les permissions nécessaires")
		return
	}

	if !isDirectory(basePath+"/"+uploadPath) || !isDirectory(basePath+"/"+uploadPath+"/private") {
		c.JSON(500, "Stockage mal configuré")
		return
	}

	public, err := ioutil.ReadDir(basePath + "/" + uploadPath)
	if err != nil {
		c.JSON(500, "Erreur sur le répertoire public: "+err.Error())
	}

	private, err := ioutil.ReadDir(basePath + "/" + uploadPath + "/private")
	if err != nil {
		c.JSON(500, "Erreur sur le répertoire privé: "+err.Error())
	}

	var files struct {
		Public  []File `json:"public,omitempty"`
		Private []File `json:"private,omitempty"`
	}

	for _, p := range public {
		if p.Name() != "private" && !p.IsDir() {
			files.Public = append(files.Public, File{
				Name:   p.Name(),
				Domain: "public",
			})
		}
	}

	for _, p := range private {
		files.Private = append(files.Private, File{
			Name:   p.Name(),
			Domain: "private",
		})
	}

	c.JSON(200, files)
}

func upload(h *tusd.Handler) func(*gin.Context) {
	return gin.WrapH(h)
}
