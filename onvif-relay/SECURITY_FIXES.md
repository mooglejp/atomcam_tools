# セキュリティ修正ガイド

## 監査日: 2026-02-16
## 検出された脆弱性: 18件（Critical: 3, High: 5, Medium: 6, Low: 4）

---

## 🔴 Critical修正（最優先）

### C-1. ONVIF認証の実装

**現状**: すべてのONVIFエンドポイントが認証なしでアクセス可能

**修正手順**:

1. **各ハンドラに認証チェックを追加**

`internal/onvif/server.go`の各ハンドラ（handleDeviceService、handleMediaService、handlePTZService、handleImagingService）の先頭に以下を追加：

```go
// GetSystemDateAndTime以外は認証必須
if action != "GetSystemDateAndTime" {
    // SOAPエンベロープからSecurityヘッダを抽出
    var envelope soap.Envelope
    if err := xml.Unmarshal(body, &envelope); err != nil {
        s.sendFault(w, soap.NewNotAuthorizedFault())
        return
    }

    if envelope.Header == nil || len(envelope.Header.Content) == 0 {
        s.sendFault(w, soap.NewNotAuthorizedFault())
        return
    }

    // WS-UsernameToken検証
    err := soap.ValidateUsernameToken(
        envelope.Header.Content,
        s.config.Server.Auth.Username,
        s.config.Server.Auth.Password,
    )
    if err != nil {
        log.Printf("Authentication failed for %s: %v", action, err)
        s.sendFault(w, soap.NewNotAuthorizedFault())
        return
    }
}
```

2. **スナップショットエンドポイントにHTTP Basic認証を追加**

`internal/snapshot/proxy.go`のHandlerメソッドに：

```go
// HTTP Basic認証チェック
username, password, ok := r.BasicAuth()
if !ok || username != p.username || password != p.password {
    w.Header().Set("WWW-Authenticate", `Basic realm="ONVIF Relay"`)
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

`snapshot.Proxy`構造体にusername/passwordフィールドを追加し、`NewProxy`で渡す。

---

### C-2. カメラコマンドのホワイトリスト検証

**修正**: `internal/camera/client.go`の`SendCommand`メソッドに検証を追加

```go
func (c *Client) SendCommand(command string) error {
    // コマンドプレフィックスのホワイトリスト
    allowedPrefixes := []string{"move ", "video ", "property "}

    allowed := false
    for _, prefix := range allowedPrefixes {
        if strings.HasPrefix(command, prefix) {
            allowed = true
            break
        }
    }

    if !allowed {
        return fmt.Errorf("command not allowed: %s", command)
    }

    // 既存のコード...
}
```

---

### C-3. RTSP URLからの認証情報除去

**修正**: `internal/mediamtx/client.go`の`BuildFFmpegCommand`を変更

```go
// 認証情報なしのRTSP URL
sourceURL := fmt.Sprintf("rtsp://%s:%d/%s", camera.Host, camera.RTSPPort, stream.Path)

// ffmpegの-user/-passwordオプションで認証情報を渡す
cmd := fmt.Sprintf("ffmpeg -fflags +genpts -rtsp_transport tcp")
if camera.Username != "" && camera.Password != "" {
    cmd += fmt.Sprintf(" -user %s -password %s",
        shellEscape(camera.Username),
        shellEscape(camera.Password))
}
cmd += fmt.Sprintf(" -i %s -map 0:v:0 -map 0:a:0? -c:v copy", sourceURL)
```

ただし、この方法でも`-password`がmediamtx API経由で見える可能性があるため、完全な解決には環境変数経由の認証情報渡しが必要。

---

## 🟠 High修正

### H-1. WS-Discovery XMLインジェクション対策

**修正**: `internal/discovery/wsdiscovery.go`の`buildProbeMatch`をXMLマーシャリングに変更

```go
import "encoding/xml"

type ProbeMatchEnvelope struct {
    XMLName xml.Name `xml:"http://www.w3.org/2003/05/soap-envelope Envelope"`
    Header  ProbeMatchHeader
    Body    ProbeMatchBody
}

// 構造体定義を追加し、xml.Marshalで生成
```

---

### H-2. Nonceリプレイ攻撃対策

**修正**: `internal/onvif/soap/auth.go`に使用済みNonceキャッシュを追加

```go
var (
    usedNonces sync.Map // map[string]time.Time
    nonceTTL = 5 * time.Minute
)

// ValidateUsernameToken内でNonceチェック
if nonce != "" {
    if _, exists := usedNonces.LoadOrStore(nonce, time.Now()); exists {
        return fmt.Errorf("nonce already used (replay attack)")
    }
}

// 定期的にクリーンアップするgoroutineを起動
go cleanupNonces()
```

---

### H-3. HTTP Digest認証の改善

**修正**: `pkg/digest/auth.go`

```go
// nc（nonce counter）をリクエストごとにインクリメント
type Transport struct {
    // ...
    nc uint32 // atomic counter
}

// cnonce生成を修正
func generateCnonce() string {
    b := make([]byte, 16)
    rand.Read(b)
    return fmt.Sprintf("%x", b)
}
```

---

### H-4. DoS対策

**修正**: `internal/onvif/server.go`のHTTPサーバー設定

```go
s.httpServer = &http.Server{
    Addr:           fmt.Sprintf(":%d", cfg.Server.OnvifPort),
    Handler:        mux,
    ReadTimeout:    30 * time.Second,
    WriteTimeout:   30 * time.Second,
    MaxHeaderBytes: 1 << 20, // 1MB
}
```

各ハンドラでのbody読み込みを制限：

```go
body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB制限
```

---

### H-5. TLS対応

**修正**: 設定ファイルにTLSオプションを追加

```yaml
server:
  onvif_port: 8080
  tls:
    enabled: false  # TLS有効化
    cert_file: /config/cert.pem
    key_file: /config/key.pem
```

`internal/onvif/server.go`:

```go
if cfg.Server.TLS.Enabled {
    return s.httpServer.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
}
return s.httpServer.ListenAndServe()
```

---

## 🟡 Medium修正

### M-1. WS-Discoveryバッファのレースコンディション対策

```go
// buffer[:n]のコピーを渡す
dataCopy := make([]byte, n)
copy(dataCopy, buffer[:n])
go r.handleProbe(dataCopy, remoteAddr)
```

### M-2. 環境変数経由のパスワード設定サポート

設定ファイルで`${ENV_VAR}`構文をサポート。

### M-3. エラーメッセージの汎用化

内部エラーの詳細をログのみに記録し、クライアントには汎用メッセージを返す。

### M-4. SHA-256サポート

HTTP Digest認証でSHA-256をサポート（AtomCamの対応状況に依存）。

### M-5. パス名のバリデーション

カメラ名とストリームパスを正規表現で検証：

```go
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

if !validNamePattern.MatchString(c.Name) {
    return fmt.Errorf("invalid camera name: %s", c.Name)
}
```

### M-6. challengesマップのTTL

TTL付きキャッシュに置き換え、またはホスト名のみをキーとして使用。

---

## 修正優先順位

1. **C-1 (認証実装)** - 即座に対応必須
2. **H-4 (DoS対策)** - 設定変更のみで対応可能
3. **C-2 (コマンド検証)** - コード変更少量
4. **H-2 (リプレイ攻撃)** - セキュリティ強化
5. その他

---

## テスト方法

### 認証テスト
```bash
# 認証なしでアクセス → 401エラーを期待
curl -X POST http://localhost:8080/onvif/device_service

# WS-UsernameToken付きでアクセス → 成功を期待
# （ONVIFクライアントツールを使用）
```

### DoS対策テスト
```bash
# 大きなリクエストボディ送信 → 拒否を期待
dd if=/dev/zero bs=2M count=1 | curl -X POST -d @- http://localhost:8080/onvif/device_service
```
