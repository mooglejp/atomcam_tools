# ONVIF Relay Server — アーキテクチャと実装ノート

## 概要

atomcam_toolsカスタムFW導入済みの複数Atomcamを集約し、1台のONVIF対応IPカメラとしてNVR（Blue Iris, Frigate, Synology Surveillance Station等）に認識させる中継サーバー。

RTSPストリーミングは**mediamtx**に完全委譲し、本サーバーはONVIF SOAP/PTZ/Imaging/WS-Discoveryのみを担当する。

## 最終アーキテクチャ

```
NVRクライアント (Blue Iris, Frigate, VLC等)
    │
    ├── ONVIF SOAP (TCP :8080)  ──→  onvif-relay (Go)
    │     Device / Media / PTZ / Imaging / Events
    │
    ├── WS-Discovery (UDP :3702) ──→  onvif-relay (Go)
    │
    └── RTSP (TCP :8554)  ──→  mediamtx
          │
          ├─ H.264 + 音声トランスコード → ffmpeg (runOnDemand, RTSP publish)
          ├─ H.265 + 音声トランスコード → ffmpeg (runOnDemand, SRT/MPEG-TS publish)
          └─ トランスコード不要          → mediamtx直接ソースプロキシ
```

### コンテナ構成

| コンテナ | イメージ | 役割 |
|---|---|---|
| onvif-relay | 自前ビルド (Go + Alpine) | ONVIF SOAP, PTZ, Imaging, WS-Discovery |
| mediamtx | `bluenviron/mediamtx:latest-ffmpeg` | RTSP/HLS/WebRTC配信, ffmpegによる音声トランスコード |

**重要**: mediamtxのイメージは必ず `latest-ffmpeg` を使うこと。`latest` はscratch baseでffmpegが含まれない。

## ストリーミング経路の3パターン

### パターン1: H.264 + 音声トランスコード（RTSP publish）

```
Atomcam (RTSP/TCP) → ffmpeg → RTSP publish → mediamtx → クライアント
```

- mediamtxの `runOnDemand` でffmpegをオンデマンド起動
- 映像: copy（無変換）、音声: pcm_mulaw等にトランスコード
- ffmpegの出力先: `rtsp://localhost:$RTSP_PORT/$MTX_PATH`

```
ffmpeg -fflags +genpts -rtsp_transport tcp -i rtsp://<camera>/stream
  -map 0:v:0 -map 0:a:0?
  -c:v copy
  -c:a pcm_mulaw -ar 8k -ac 1 -async 1 -af volume=8
  -rtsp_transport tcp
  -f rtsp rtsp://localhost:$RTSP_PORT/$MTX_PATH
```

### パターン2: H.265/HEVC + 音声トランスコード（SRT publish）

```
Atomcam (RTSP/TCP) → ffmpeg → SRT/MPEG-TS publish → mediamtx → クライアント
```

- H.264と同じくffmpegオンデマンド起動だが、**出力先がSRT**
- RTSP出力ではなくSRTを使う理由は後述（HEVC SDP非互換問題）
- MPEG-TSコンテナは `pcm_mulaw`/`pcm_alaw` 非対応のため、音声コーデックを `aac` に自動変換
- サンプルレート: 48000Hz（AACの標準）

```
ffmpeg -fflags +genpts -rtsp_transport tcp -i rtsp://<camera>/stream
  -map 0:v:0 -map 0:a:0?
  -c:v copy
  -c:a aac -ar 48000 -ac 1 -async 1 -af volume=8
  -f mpegts "srt://localhost:8890?streamid=publish:<path>&pkt_size=1316"
```

### パターン3: トランスコード不要（直接ソースプロキシ）

```
Atomcam (RTSP/TCP) → mediamtx (source proxy) → クライアント
```

- mediamtxの `source` + `sourceOnDemand` で直接中継
- `rtspTransport: tcp` を設定し、上流カメラへの接続をTCPに強制

**注意**: このモードはAtomcamのLIVE555サーバーとの相性問題あり（後述「ハマりポイント」参照）。音声トランスコードが必要な場合はパターン1/2を使うこと。

## mediamtx REST API v3

### エンドポイント

| 操作 | メソッド | パス |
|---|---|---|
| パス追加 | POST | `/v3/config/paths/add/{name}` |
| パス更新 | PATCH | `/v3/config/paths/patch/{name}` |
| パス削除 | DELETE | `/v3/config/paths/delete/{name}` |
| パス一覧 | GET | `/v3/config/paths/list` |
| パス取得 | GET | `/v3/config/paths/get/{name}` |
| グローバル設定 | GET | `/v3/config/global/get` |

### 注意事項

- **エンドポイント名に注意**: `edit` ではなく `patch`、`remove` ではなく `delete`。間違えると404が返る。
- パス名にスラッシュを含む場合（例: `camera/video0`）もそのままURLに入れてよい。
- パスが既に存在する場合のPOSTは `400 Bad Request` + `{"error":"path already exists"}` を返す。
- パスの設定を大幅に変更する場合（runOnDemand ↔ source切替等）、PATCHではなく**削除→再追加**が確実。

### パス設定の再登録戦略

onvif-relayは起動時に全パスを設定する。既存パスがある場合:

1. POST `/v3/config/paths/add/` を試行
2. 400（既存）の場合 → DELETE `/v3/config/paths/delete/` → 再度POST

この方式により、前回と異なるモード（ffmpeg→source等）への切替も確実に行える。

## mediamtx.yml 設定のポイント

```yaml
# REST API有効化（必須）
api: true
apiAddress: :9997

# クライアント接続はTCPのみ（Docker bridgeのUDP NAT問題回避）
rtspTransports: [tcp]

# Atomcamストリームはバースト的 - デフォルト10sでは短すぎる
readTimeout: 15m

# HEVCキーフレームは大きい - デフォルト512では不足
writeQueueSize: 4096

# Docker内部ネットワークからのAPI/publish/readを許可
# デフォルトは127.0.0.1のみ許可でDocker bridgeからアクセスできない
authInternalUsers:
  - user: any
    pass:
    ips: []        # ← 空配列 = 全IPアドレス許可
    permissions:
      - action: api
      - action: publish
      - action: read
      - action: playback
```

## Docker Compose 構成

```yaml
services:
  onvif-relay:
    build: .
    restart: unless-stopped
    ports:
      - "8080:8080"       # ONVIF SOAP
      - "3702:3702/udp"   # WS-Discovery
    volumes:
      - ./config:/config
    depends_on:
      - mediamtx

  mediamtx:
    image: bluenviron/mediamtx:latest-ffmpeg   # ← latest ではなく latest-ffmpeg
    restart: unless-stopped
    ports:
      - "8554:8554"       # RTSP
      - "8000:8000/udp"   # RTP
      - "8001:8001/udp"   # RTCP
      - "8888:8888"       # HLS
      - "8889:8889"       # WebRTC HTTP
      - "8189:8189/udp"   # WebRTC ICE
      - "9997:9997"       # REST API
    volumes:
      - ./config/mediamtx.yml:/mediamtx.yml   # ← ファイルマウント（ディレクトリではない）
```

## 依存関係

```
go 1.23.0
require gopkg.in/yaml.v3 v3.0.1
```

CGO不要。gortsplib等のRTSPライブラリは不要（mediamtxに委譲したため）。

---

## ハマりポイント一覧

### 1. mediamtx APIがDocker内部から401 Unauthorized

**症状**: onvif-relayからmediamtx API呼び出しが401で拒否される。

**原因**: mediamtxの `authInternalUsers` がデフォルトで `ips: [127.0.0.1, ::1]` に制限されている。Docker bridgeネットワークではコンテナ間通信が172.18.0.x等のIPになるため、アクセスが拒否される。

**対策**: mediamtx.ymlで `ips: []`（空配列=全許可）を設定。

### 2. mediamtx.ymlのボリュームマウントがディレクトリになる

**症状**: `./config/mediamtx.yml:/mediamtx.yml` のマウントで、mediamtx.ymlファイルが存在しない状態でコンテナを起動すると、Dockerが**ディレクトリ**として作成してしまう。

**対策**: `docker compose up` の前に必ず `config/mediamtx.yml` ファイルを作成しておく。

### 3. `depends_on` はサービスの準備完了を待たない

**症状**: onvif-relayが起動した時点でmediamtx APIがまだ準備できておらず、connection refusedになる。

**対策**: `WaitReady()` メソッドで2秒間隔のポーリングリトライ（最大30秒）を実装。

### 4. `bluenviron/mediamtx:latest` にffmpegが含まれない

**症状**: `runOnDemand` でffmpegを起動しようとすると `executable file not found in $PATH`。

**原因**: `latest` タグはscratchベースのミニマルイメージ。ffmpegを含むのは `latest-ffmpeg` タグ。

**対策**: `bluenviron/mediamtx:latest-ffmpeg` を使用。

### 5. Docker Desktop/WSL2で `network_mode: host` が使えない

**症状**: `network_mode: host` を設定してもホストからコンテナにアクセスできない。

**原因**: Docker DesktopはWSL2のVM内で動作しており、`host` ネットワークはVM内部のネットワーク。ホストOS（Windows）からはアクセスできない。

**対策**: bridgeネットワーク + 明示的なポートマッピングを使用。

### 6. Atomcam LIVE555がSPS/PPSをSDPに含まない

**症状**: ffmpegが `dimensions not set` / `Could not write header` でストリーム情報を取得できない。

**原因**: AtomcamのLIVE555 RTSPサーバーはSDP応答にH.264のSPS/PPSパラメータを含まない。実際のRTPパケットからのみ取得可能。

**対策**:
- `-rtsp_transport tcp` でTCP接続（UDPだとDocker NATでパケットロスしやすい）
- `-fflags +genpts` でタイムスタンプ生成
- `-probesize` や `-analyzeduration` を小さくしすぎない（デフォルト値を使用）

### 7. ffmpegのHEVC RTSP出力がmediamtxで拒否される（SDP非互換）

**症状**: ffmpegからRTSP publishすると `invalid SDP: media 1 is invalid: invalid sprop-pps` エラー。

**原因**: ffmpegのRTSP muxerがHEVCストリームのSDP `sprop-pps` 属性にカンマ区切りで複数PPS値を記述する。gortsplib（mediamtxが使用するRTSPライブラリ）はこの形式をパースできない。

**対策**: HEVCストリームはRTSP出力の代わりに**SRT (MPEG-TS) 出力**を使用。
```
-f mpegts "srt://localhost:8890?streamid=publish:$MTX_PATH&pkt_size=1316"
```

### 8. MPEG-TSコンテナが pcm_mulaw/pcm_alaw をサポートしない

**症状**: SRT publish時に mediamtx が `skipping track 2 (unsupported codec)` と警告し、音声トラックが無視される。

**原因**: MPEG-TSコンテナはPCM mu-law/A-lawを標準でサポートしていない。

**対策**: SRT出力時は音声コーデックを自動的に `aac`（48kHz）に変換。H.264のRTSP出力時は `pcm_mulaw`（8kHz）のまま。

### 9. mediamtxソースプロキシがAtomcamから約65秒で切断される

**症状**: mediamtxの `source` + `sourceOnDemand` モードで直接プロキシすると、約65秒後にストリームが停止する。HLS/WebRTCでも同様。

**原因**: AtomcamのLIVE555サーバーのセッションタイムアウト（約65秒）。mediamtxのRTSPクライアントからのkeepaliveが不十分と推測される。

**対策**: 音声トランスコードが必要なストリームは全てffmpeg経由（runOnDemand）にする。ffmpegは適切にRTSP keepaliveを送信するため切断されない。トランスコード不要で直接プロキシを使う場合は `readTimeout: 15m` を設定し、切断時の自動再接続に期待する。

### 10. Docker bridge環境でRTSP UDPが機能しない

**症状**: VLC等がRTSP UDPでmediamtxに接続すると映像が映らない。UDPフォールバック後にTCPで映る。

**原因**: Docker bridge NATではRTPのUDP戻りパケットがホストに到達しない。ポートマッピング(8000/8001)はあるが、VLCが使うランダムクライアントポートはマッピングされていない。

**対策**: `rtspTransports: [tcp]` でTCPのみに制限し、クライアントのUDP試行をスキップ。

### 11. 音声タイムスタンプの不連続 (Non-monotonic DTS)

**症状**: ffmpegが `Non-monotonic DTS in output stream` 警告を大量に出力。

**原因**: Atomcamの音声ストリームのタイムスタンプが不規則。

**対策**: `-async 1` オプションでffmpegが音声サンプルの挿入/破棄によりタイムスタンプを自動補正。

### 12. mediamtx REST API v3のエンドポイント名

**症状**: `/v3/config/paths/edit/` や `/v3/config/paths/remove/` が404を返す。

**原因**: 正しいエンドポイント名は `patch`（editではない）と `delete`（removeではない）。ドキュメントやバージョンによって異なる記述があり混乱しやすい。

**対策**: 正しいエンドポイント:
- 更新: `PATCH /v3/config/paths/patch/{name}`
- 削除: `DELETE /v3/config/paths/delete/{name}`

---

## ディレクトリ構成

```
onvif-relay/
├── cmd/onvif-relay/
│   └── main.go                  # エントリーポイント
├── internal/
│   ├── config/
│   │   └── config.go            # YAML設定ロード・バリデーション
│   ├── mediamtx/
│   │   └── client.go            # mediamtx REST APIクライアント
│   ├── onvif/
│   │   ├── server.go            # HTTPサーバー・SOAPルーティング
│   │   ├── device/service.go    # Deviceサービス
│   │   ├── media/service.go     # Mediaサービス
│   │   ├── ptz/service.go       # PTZサービス
│   │   ├── imaging/service.go   # Imagingサービス
│   │   └── ...
│   ├── camera/
│   │   ├── registry.go          # カメラレジストリ
│   │   ├── client.go            # cmd.cgi HTTPクライアント
│   │   ├── ptz.go               # PTZ座標変換
│   │   ├── imaging.go           # IR/Imaging制御
│   │   └── health.go            # ヘルスチェック
│   ├── discovery/
│   │   └── wsdiscovery.go       # WS-Discovery UDPレスポンダー
│   └── snapshot/
│       └── proxy.go             # JPEGスナップショットプロキシ
├── config/
│   └── mediamtx.yml             # mediamtx設定（実行時にボリュームマウント）
├── config.example.yaml          # onvif-relay設定サンプル
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── ARCHITECTURE.md              # このファイル
```

## 起動シーケンス

1. `docker compose up -d`
2. mediamtxコンテナ起動 → REST API (:9997) 待受開始
3. onvif-relayコンテナ起動 → 設定ファイル読み込み
4. `WaitReady()` でmediamtx APIにポーリング（2秒間隔、最大30秒）
5. `ConfigurePaths()` で全カメラ/ストリームのパスをAPI経由で登録
6. WS-Discoveryレスポンダー起動 (UDP :3702)
7. ONVIF HTTPサーバー起動 (:8080)
8. ヘルスチェッカー起動（30秒間隔でカメラの死活監視）
9. クライアント接続待ち

## クライアントからのストリーム再生フロー

1. クライアントが `rtsp://relay:8554/<camera>/<stream>` にDESCRIBE
2. mediamtxがパス設定を参照
3. `runOnDemand` の場合: ffmpegプロセスをオンデマンド起動
4. `source` の場合: 上流カメラにRTSP接続
5. ストリームが利用可能になったらクライアントにSDP応答
6. RTPパケット配信開始
7. 全クライアント切断後、`runOnDemandCloseAfter` (10s) 経過でffmpeg終了
