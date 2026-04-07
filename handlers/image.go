package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/champiao/sharepoint-image-uploader/msgraph"
	"github.com/gin-gonic/gin"
)

// Config contém as credenciais e configurações do SharePoint.
type Config struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	SiteID       string
	Folder       string
}

// ImageHandler lida com o upload de imagens.
type ImageHandler struct {
	cfg Config
}

// NewImageHandler cria um novo ImageHandler com a configuração fornecida.
func NewImageHandler(cfg Config) *ImageHandler {
	return &ImageHandler{cfg: cfg}
}

var nonSafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

var mimeExtensions = map[string]string{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/gif":     ".gif",
	"image/webp":    ".webp",
	"image/bmp":     ".bmp",
	"image/tiff":    ".tiff",
	"image/svg+xml": ".svg",
	"image/avif":    ".avif",
	"image/heic":    ".heic",
}

func buildFilename(original, ext string) string {
	ts := time.Now().Format("20060102_150405")
	base := strings.TrimSuffix(original, filepath.Ext(original))
	safe := strings.Trim(nonSafeChars.ReplaceAllString(base, "_"), "_")
	if safe == "" {
		safe = "image"
	}
	return fmt.Sprintf("%s_%s%s", ts, safe, ext)
}

// Upload recebe uma imagem via multipart form, salva localmente e envia para o SharePoint.
func (h *ImageHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'image' obrigatório"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao abrir arquivo"})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao ler arquivo"})
		return
	}

	// Detecta MIME pelo conteúdo binário (primeiros 512 bytes)
	mimeType := http.DetectContentType(data)

	// Resolve extensão pelo MIME
	ext, ok := mimeExtensions[mimeType]
	if !ok {
		// Fallback: tenta o Content-Type enviado pelo cliente
		ctHeader := fileHeader.Header.Get("Content-Type")
		ext = mimeExtensions[ctHeader]
		// Caso especial para SVG (detectado como text/xml ou text/plain)
		if ext == "" && (strings.HasPrefix(mimeType, "text/") || mimeType == "application/xml") {
			if ctHeader == "image/svg+xml" {
				ext = ".svg"
			}
		}
		if ext == "" {
			ext = ".bin"
		}
	}

	filename := buildFilename(fileHeader.Filename, ext)
	localPath := filepath.Join("Images", filename)

	if err := os.WriteFile(localPath, data, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao salvar arquivo localmente"})
		return
	}

	token, err := msgraph.GetAccessToken(h.cfg.TenantID, h.cfg.ClientID, h.cfg.ClientSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao autenticar com Microsoft"})
		return
	}

	if err := msgraph.UploadToSharePoint(token, h.cfg.SiteID, h.cfg.Folder, localPath, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "falha no upload para SharePoint",
			"detalhe": err.Error(),
		})
		return
	}

	os.Remove(localPath)

	c.JSON(http.StatusCreated, gin.H{
		"mensagem":  "imagem enviada com sucesso",
		"arquivo":   filename,
		"tamanho":   len(data),
		"mime_type": mimeType,
	})
}
