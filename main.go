package main

import (
	"fmt"
	"os"

	"github.com/champiao/sharepoint-image-uploader/handlers"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "❌  Variável de ambiente ausente: %s\n", key)
		os.Exit(1)
	}
	return v
}

func main() {
	godotenv.Load()

	cfg := handlers.Config{
		ClientID:     mustEnv("MS_CLIENT_ID"),
		ClientSecret: mustEnv("MS_CLIENT_SECRET"),
		TenantID:     mustEnv("MS_TENANT_ID"),
		SiteID:       mustEnv("SHAREPOINT_SITE_ID"),
		Folder:       mustEnv("SHAREPOINT_FOLDER"),
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := os.MkdirAll("Images", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌  Não foi possível criar pasta Images: %v\n", err)
		os.Exit(1)
	}

	r := gin.Default()
	r.POST("/image", handlers.NewImageHandler(cfg).Upload)

	fmt.Printf("🚀  Servidor iniciado na porta %s\n", port)
	r.Run(":" + port)
}
