# ONVIF Relay Server

複数のAtomCamカメラを1台のONVIF対応IPカメラとして集約する中継サーバー。Blue Iris、Frigate、Synology Surveillance Station等のNVRから利用可能。

## 特徴

- ✅ **ONVIF完全対応**: Device, Media, PTZ, Imaging サービス実装
- ✅ **WS-Discovery**: 自動デバイス検出
- ✅ **マルチストリーム**: H.264/H.265対応、複数解像度
- ✅ **PTZ制御**: パン/チルト/ズーム操作
- ✅ **Imaging制御**: 明るさ、コントラスト、IR切替
- ✅ **マルチアーキテクチャ**: AMD64, ARM64, ARM v7対応
- ✅ **セキュリティ強化**: 認証、入力検証、DoS防止

## クイックスタート

### 1. 設定ファイルの準備

```bash
cd onvif-relay
cp config.example.yaml config/config.yaml
```

`config/config.yaml`を編集してカメラ情報を設定：

```yaml
server:
  onvif_port: 8080
  device_name: "AtomCam ONVIF Relay"
  discovery: true
  auth:
    username: "your_username"  # ONVIF認証用
    password: "your_password"
  mediamtx:
    api: "http://mediamtx:9997"
    rtsp_port: 8554

cameras:
  - name: "camera1"
    host: "192.168.1.100"
    username: "admin"          # カメラ認証用
    password: "camera_password"
    rtsp_port: 8554
    http_port: 80
    capabilities:
      ptz: true
      ir: true
    streams:
      - path: "video0_unicast"
        resolution: "1920x1080"
        codec: "h264"
        profile_name: "Main"
```

### 2. Docker Composeで起動

#### ローカルビルド版

```bash
docker compose up -d --build
```

#### プリビルドイメージ版

```yaml
# docker-compose.ymlを編集
services:
  onvif-relay:
    image: ghcr.io/mooglejp/atomcam_tools/onvif-relay:latest
    # build: .  # この行をコメントアウト
```

```bash
docker compose up -d
```

### 3. 動作確認

```bash
# ONVIF GetSystemDateAndTime テスト
curl -X POST \
  -H "Content-Type: application/soap+xml" \
  -d '<?xml version="1.0"?><s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"><s:Body><GetSystemDateAndTime xmlns="http://www.onvif.org/ver10/device/wsdl"/></s:Body></s:Envelope>' \
  http://localhost:8080/onvif/device_service

# ログ確認
docker compose logs -f onvif-relay
```

## アーキテクチャ

```
NVRクライアント
    │
    ├── ONVIF SOAP (8080)  → onvif-relay
    ├── WS-Discovery (3702/udp)
    └── RTSP (8554) → mediamtx → AtomCam
```

詳細は [ARCHITECTURE.md](ARCHITECTURE.md) を参照。

## 対応プラットフォーム

GitHub Actionsで自動ビルドされるマルチアーキテクチャイメージ：

| プラットフォーム | アーキテクチャ | 用途 |
|---|---|---|
| `linux/amd64` | x86_64 | PC、サーバー |
| `linux/arm64` | ARM 64-bit | Raspberry Pi 4/5, Apple Silicon |
| `linux/arm/v7` | ARM 32-bit | Raspberry Pi 3 |

## 利用可能なイメージタグ

| タグ | 説明 |
|---|---|
| `latest` | 最新の安定版（mainブランチ） |
| `vX.Y.Z` | 特定バージョン |
| `main-sha-XXXXXXX` | 特定コミット |

## 開発

### ローカルビルド

```bash
cd onvif-relay
go build -o bin/onvif-relay ./cmd/onvif-relay
./bin/onvif-relay -config config/config.yaml
```

### テスト

```bash
# セキュリティ監査
# （6回の反復監査で46件の問題を修正済み）

# 動作確認
docker compose up -d
curl http://localhost:8080/onvif/device_service
```

## セキュリティ

本プロジェクトは包括的なセキュリティ監査を実施済み：

- ✅ 定数時間認証比較（タイミング攻撃防止）
- ✅ 入力検証（シェルインジェクション、パストラバーサル対策）
- ✅ DoS防止（リクエストサイズ制限、タイムアウト）
- ✅ Nonce リプレイ攻撃防止
- ✅ ゴルーチンリーク防止
- ✅ グレースフルシャットダウン

詳細は [SECURITY_FIXES.md](SECURITY_FIXES.md) を参照。

## トラブルシューティング

### RTP packet size 警告

```
INF [path xxx] RTP packets are too big, remuxing them
```

→ `-max_delay 500000`オプションで対応済み（自動）

### Non-monotonic DTS 警告

```
Non-monotonic DTS in output stream
```

→ タイムスタンプ正規化オプションで対応済み（自動）

### カメラが検出されない

1. ネットワーク接続を確認
2. カメラのRTSPポート（通常8554）が開いているか確認
3. 認証情報が正しいか確認
4. ヘルスチェッカーログを確認：`docker compose logs onvif-relay | grep "health check"`

## ライセンス

（元リポジトリのライセンスに従う）

## 貢献

- セキュリティ問題: GitHubのSecurity Advisoriesで報告
- バグ報告: GitHubのIssuesで報告
- 機能要望: GitHubのIssuesで提案

## クレジット

- **元プロジェクト**: [mnakada/atomcam_tools](https://github.com/mnakada/atomcam_tools)
- **フォーク**: [mooglejp/atomcam_tools](https://github.com/mooglejp/atomcam_tools)
- **ONVIF Relay実装**: Claude Sonnet 4.5との共同開発
