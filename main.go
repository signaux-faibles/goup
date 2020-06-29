package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"strings"

	"github.com/Nerzal/gocloak/v5"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"

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

func processUpload() chan tusd.HookEvent {
	channel := make(chan tusd.HookEvent)
	go func() {
		for event := range channel {
			err := linkFile(event)
			email := event.Upload.MetaData["email"]
			filename := event.Upload.MetaData["filename"]
			if err != nil {
				fmt.Println(err)
				if email != "" && filename != "" {
					err := sendFailureEmail(email, filename)
					if err != nil {
						fmt.Println(err)
					}
				}
			} else {
				if email != "" && filename != "" {
					err := sendSuccessEmail(email, filename)
					if err != nil {
						fmt.Println(err)
					}
				}
			}
		}
	}()
	return channel
}

func sendSuccessEmail(to string, filename string) error {
	msg := []byte("From: Signaux Faibles <" + viper.GetString("fromEmailAddress") + ">\r\n" +
		"To: " + to + "\r\n" +
		"Subject: Fichier transmis avec succès\n" +
		"\r\n" +
		"Le fichier \"" + filename + "\" a bien été transmis à Signaux Faibles.\r\n")
	// Sending "Bcc" messages is accomplished by including an email address in the to parameter but not including it in the msg headers
	err := smtp.SendMail(viper.GetString("smtpHost"), nil, viper.GetString("fromEmailAddress"), []string{to, viper.GetString("fromEmailAddress")}, msg)
	if err != nil {
		return err
	}
	return nil
}

func sendFailureEmail(to string, filename string) error {
	msg := []byte("From: Signaux Faibles <" + viper.GetString("fromEmailAddress") + ">\r\n" +
		"To: " + to + "\r\n" +
		"Subject: Fichier non transmis\n" +
		"\r\n" +
		"Un problème est survenu lors de la transmission du fichier \"" + filename + "\" à Signaux Faibles.\r\n")
	// Sending "Bcc" messages is accomplished by including an email address in the to parameter but not including it in the msg headers
	err := smtp.SendMail(viper.GetString("smtpHost"), nil, viper.GetString("fromEmailAddress"), []string{to, viper.GetString("fromEmailAddress")}, msg)
	if err != nil {
		return err
	}
	return nil
}

func addMetadata(c *gin.Context) {
	metadata := c.Request.Header.Get("upload-metadata")
	claims := c.Keys["claims"].(*jwt.MapClaims)

	path, ok := (*claims)["goup_path"].(string)
	if !ok {
		c.JSON(403, "Forbidden")
		c.Abort()
		return
	}

	err := checkStorage(path)
	if err != nil {
		c.JSON(500, err.Error())
		c.Abort()
		return
	}

	metadata = metadata + ", goup-path " + base64.StdEncoding.EncodeToString([]byte(path))
	email, ok := (*claims)["email"].(string)
	if ok {
		metadata = metadata + ", email " + base64.StdEncoding.EncodeToString([]byte(email))
	}
	
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
	config.AddAllowHeaders("Authorization", "Tus-Resumable", "Upload-Length", "Upload-Metadata", "Upload-Offset", "Location", "X-HTTP-Method-Override")
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
