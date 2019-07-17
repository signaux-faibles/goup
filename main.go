package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Nerzal/gocloak"
	"github.com/tus/tusd"
	"github.com/tus/tusd/filestore"

	jwt "github.com/dgrijalva/jwt-go"
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

var keycloak gocloak.GoCloak

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
	claims := c.Keys["claims"].(*jwt.MapClaims)

	path, ok := (*claims)["goup_path"].(string)
	if !ok {
		fmt.Println("gnagnagna")
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

	keycloak = gocloak.NewClient(viper.GetString("keycloakHostname"))

	store := filestore.FileStore{
		Path: viper.GetString("basePath") + "/tusd/",
	}
	composer := tusd.NewStoreComposer()
	composer.UsesGetReader = false

	store.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:                "/files/",
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		RespectForwardedHeaders: true,
	})

	handler.CompleteUploads = processUpload()

	if err != nil {
		panic(fmt.Errorf("Unable to create handler: %s", err))
	}

	hostname := viper.GetString("hostname")

	// gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{hostname}
	config.AddAllowHeaders("Authorization", "Tus-Resumable", "Upload-Length", "Upload-Metadata", "Upload-Offset", "Location")
	config.AddAllowMethods("POST", "HEAD", "PATCH")
	config.AddExposeHeaders("Content-Length")
	router.Use(cors.New(config))
	router.POST("/files/*any", keycloakMiddleware, addMetadata, gin.WrapH(http.StripPrefix("/files/", handler)))
	router.HEAD("/files/*any", keycloakMiddleware, gin.WrapH(http.StripPrefix("/files/", handler)))
	router.PATCH("/files/*any", keycloakMiddleware, gin.WrapH(http.StripPrefix("/files/", handler)))
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

func keycloakMiddleware(c *gin.Context) {
	header := c.Request.Header["Authorization"][0]

	rawToken := strings.Split(header, " ")[1]

	token, claims, err := keycloak.DecodeAccessToken(rawToken, viper.GetString("keycloakRealm"))
	if errValid := claims.Valid(); err != nil && errValid != nil {
		c.AbortWithStatus(401)
	}

	c.Set("token", token)
	c.Set("claims", claims)

	c.Next()
}

func scopeFromClaims(claims *jwt.MapClaims) []string {
	resourceAccess := (*claims)["resource_access"].(map[string]interface{})
	client := (resourceAccess)["signauxfaibles"].(map[string]interface{})
	scope := (client)["roles"].([]interface{})

	var tags []string
	for _, tag := range scope {
		tags = append(tags, tag.(string))
	}
	return tags
}
