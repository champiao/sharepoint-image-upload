package msgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	chunkSize = 4 * 1024 * 1024
	maxSimple = 4 * 1024 * 1024
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type uploadSession struct {
	UploadURL string `json:"uploadUrl"`
}

// GetAccessToken obtém um token de acesso app-only via OAuth2 client_credentials.
func GetAccessToken(tenantID, clientID, clientSecret string) (string, error) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenantID)

	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("client_secret", clientSecret)
	params.Set("scope", "https://graph.microsoft.com/.default")
	params.Set("grant_type", "client_credentials")

	resp, err := http.PostForm(tokenURL, params)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("microsoft retornou %d: %s", resp.StatusCode, string(body))
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	return tr.AccessToken, nil
}

// UploadToSharePoint envia um arquivo para uma pasta do SharePoint via Microsoft Graph.
// Usa upload simples para arquivos < 4MB e upload em chunks para os demais.
func UploadToSharePoint(token, siteID, folder, localFile, filename string) error {
	data, err := os.ReadFile(localFile)
	if err != nil {
		return fmt.Errorf("ler arquivo: %w", err)
	}

	size := len(data)
	fmt.Printf("📁 Enviando: %s (%.2f MB)\n", filename, float64(size)/1024/1024)

	if size < maxSimple {
		fmt.Println("   Modo: upload simples")
		return uploadSimple(token, siteID, folder, filename, data)
	}

	fmt.Println("   Modo: upload em chunks (arquivo > 4MB)")
	return uploadChunked(token, siteID, folder, filename, data)
}

func uploadSimple(token, siteID, folder, filename string, data []byte) error {
	uploadURL := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/sites/%s/drive/root:%s/%s:/content",
		siteID, folder, filename,
	)

	req, _ := http.NewRequest("PUT", uploadURL, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("upload simples retornou %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func uploadChunked(token, siteID, folder, filename string, data []byte) error {
	sessionURL := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/sites/%s/drive/root:%s/%s:/createUploadSession",
		siteID, folder, filename,
	)

	sessionBody := map[string]interface{}{
		"item": map[string]string{
			"@microsoft.graph.conflictBehavior": "replace",
			"name":                              filename,
		},
	}
	sessionJSON, _ := json.Marshal(sessionBody)

	req, _ := http.NewRequest("POST", sessionURL, bytes.NewReader(sessionJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return fmt.Errorf("criar sessão: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("criar sessão retornou %d: %s", resp.StatusCode, string(body))
	}

	var session uploadSession
	json.Unmarshal(body, &session)
	fmt.Printf("   📤 Sessão criada, enviando em chunks de 4MB...\n")

	totalSize := len(data)
	offset := 0

	for offset < totalSize {
		end := offset + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := data[offset:end]
		contentRange := fmt.Sprintf("bytes %d-%d/%d", offset, end-1, totalSize)

		chunkReq, _ := http.NewRequest("PUT", session.UploadURL, bytes.NewReader(chunk))
		chunkReq.Header.Set("Content-Range", contentRange)
		chunkReq.Header.Set("Content-Type", "application/octet-stream")

		chunkResp, err := (&http.Client{}).Do(chunkReq)
		if err != nil {
			return fmt.Errorf("enviar chunk %d: %w", offset, err)
		}
		chunkBody, _ := io.ReadAll(chunkResp.Body)
		chunkResp.Body.Close()

		if chunkResp.StatusCode != 202 && chunkResp.StatusCode != 201 && chunkResp.StatusCode != 200 {
			return fmt.Errorf("chunk %d retornou %d: %s", offset, chunkResp.StatusCode, string(chunkBody))
		}

		offset = end
		fmt.Printf("   ⬆️  %.1f / %.1f MB\n",
			float64(offset)/1024/1024,
			float64(totalSize)/1024/1024,
		)
	}

	return nil
}
