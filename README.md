# sharepoint-image-uploader

API REST em Go para receber imagens via HTTP e enviá-las diretamente para uma pasta do **SharePoint** usando a Microsoft Graph API.

## Como funciona

1. Recebe uma imagem via `POST /image` (multipart/form-data, campo `image`)
2. Detecta o tipo MIME pelo conteúdo binário do arquivo
3. Salva localmente em `Images/` com nome gerado por timestamp e extensão correta
4. Obtém token de acesso via OAuth2 `client_credentials` (app-only, sem interação do usuário)
5. Envia para a pasta configurada no SharePoint (upload simples < 4MB, upload em chunks ≥ 4MB)
6. Retorna JSON com nome do arquivo, tamanho e tipo MIME

## Pré-requisitos

- **Go 1.21+**
- **App registrado no Azure** com a permissão `Sites.ReadWrite.All` (Application) e admin consent concedido

## Configuração

Copie o arquivo de exemplo e preencha com suas credenciais:

```bash
cp .env.example .env
```

| Variável | Descrição | Exemplo |
|---|---|---|
| `MS_CLIENT_ID` | Application (client) ID do App Registration | `71374387-ef45-...` |
| `MS_CLIENT_SECRET` | Client Secret gerado no App Registration | `7HX8Q~...` |
| `MS_TENANT_ID` | Directory (tenant) ID do Azure AD | `ac8a7146-...` |
| `SHAREPOINT_SITE_ID` | ID do site SharePoint (ver abaixo) | `org.sharepoint.com,guid,guid` |
| `SHAREPOINT_FOLDER` | Pasta de destino no SharePoint | `/Images` |
| `PORT` | Porta do servidor HTTP (padrão: `8080`) | `8080` |

### Como obter o `SHAREPOINT_SITE_ID`

Acesse o [Graph Explorer](https://developer.microsoft.com/graph/graph-explorer) autenticado e faça:

```
GET https://graph.microsoft.com/v1.0/sites?search=NomeDoCite
```

O campo `id` no resultado tem o formato `your-org.sharepoint.com,{siteGUID},{webGUID}`.

## Instalação e uso

```bash
# Clone e entre na pasta
git clone https://github.com/champiao/sharepoint-image-uploader
cd sharepoint-image-uploader

# Instale as dependências
go mod tidy

# Configure o ambiente
cp .env.example .env
# edite o .env com suas credenciais

# Execute
go run main.go
```

### Compilar binário

```bash
go build -o sharepoint-image-uploader .
./sharepoint-image-uploader
```

## Endpoints

### `POST /image`

Envia uma imagem para o SharePoint.

**Request:**

```bash
curl -X POST http://localhost:8080/image \
  -F "image=@/caminho/para/foto.jpg"
```

| Campo | Tipo | Descrição |
|---|---|---|
| `image` | file (multipart) | Arquivo de imagem a ser enviado |

**Resposta 201 Created:**

```json
{
  "mensagem": "imagem enviada com sucesso",
  "arquivo": "20260407_153022_foto_perfil.jpg",
  "tamanho": 204800,
  "mime_type": "image/jpeg"
}
```

**Erros:**

| Status | Motivo |
|---|---|
| `400 Bad Request` | Campo `image` ausente no formulário |
| `500 Internal Server Error` | Falha ao salvar localmente, autenticar ou enviar ao SharePoint |

## Configurando o App no Azure

1. Acesse [portal.azure.com](https://portal.azure.com) → **Azure Active Directory** → **App registrations**
2. Clique em **New registration** → dê um nome → **Register**
3. Anote o **Application (client) ID** e o **Directory (tenant) ID**
4. Vá em **Certificates & secrets** → **New client secret** → anote o valor gerado
5. Vá em **API permissions** → **Add a permission** → **Microsoft Graph** → **Application permissions**
6. Busque e adicione `Sites.ReadWrite.All`
7. Clique em **Grant admin consent** para conceder o consentimento administrativo

## Tipos de imagem suportados

| MIME Type | Extensão |
|---|---|
| `image/jpeg` | `.jpg` |
| `image/png` | `.png` |
| `image/gif` | `.gif` |
| `image/webp` | `.webp` |
| `image/bmp` | `.bmp` |
| `image/tiff` | `.tiff` |
| `image/svg+xml` | `.svg` |
| `image/avif` | `.avif` |
| `image/heic` | `.heic` |

## Estrutura do projeto

```
sharepoint-image-uploader/
├── main.go                 # Entry point: inicializa servidor Gin
├── handlers/
│   └── image.go            # Handler POST /image: MIME detection, save, upload
├── msgraph/
│   └── sharepoint.go       # Auth OAuth2 + upload para SharePoint via Graph API
├── Images/                 # Armazenamento local (gitignored, criado automaticamente)
├── .env.example            # Template de variáveis de ambiente
├── .gitignore
├── go.mod
└── README.md
```

## Licença

MIT License — veja [LICENSE](LICENSE) para detalhes.
