# atomcam_tools

これはATOMCam/ATOMCam2/AtomSwing/WyzeCamV3の機能を拡張するツールキットです。  
標準的な機能に満足できないユーザーが、各自の責任において機能を拡張するためのツール・スクリプトをまとめたものです。  
利用にあたって、当然のことながらカメラメーカへの問い合わせなどは厳に慎んでください。  
それ以外、自由に利用できますが、仮に悪用し、他人へ迷惑をかけた場合の責任は、その設定を行った者が負うべきものであることを理解してください。  
質問、動作不具合、機能のリクエストは　Issues を立ててください。

<br>

AtomSwingの画面

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/camera.jpg" /></kbd>

WyzeCamV3の画面

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/wyzecam.jpg" /></kbd>

<br>

## 実現される機能
- WebUI (Port: 80)
  - ATOMCamのアプリから設定できない追加機能について設定します。
  - アカウントとパスワードを設定可能です。

- Web経由でSD-Cardの記録映像を見ることができます。

- NASへの保存
  - 軽量ビルドでは無効です。

- Time Lapse録画

  - 定期的に周期と回数を指定してTime Lapse録画を実行します。
  - SD-Cardのtime_lapse/フォルダに中間ファイルを生成し、最後にmp4に変換して指定メディアに記録します。

- RTSPServer(Port:8554,8080)
  - RTSP streaming を送出します。
  - AtomCamの場合、Main(video0)に1080p AVC/HEVC、Sub(video1)に360p HEVCを出しています。
  - WyzeCamV3の場合、Main(video0)に1080p AVC、Sub(video1)に320p AVCを出しています。

- hostname設定
  - microSDカードのhostnameファイルを編集することでデバイス名を変更できます（WebUIからも変更可能）
  - hostnameの命名規則は英数と-(hyphen)のみ（RFC952,1123で規定)です。\_(underscore)は使用できません。defaultはatomcamになっています。
  - 軽量化のため、このビルドではavahi/mDNSによる`[hostname].local`解決は無効です。

- sshd (Port:22)
  - microSDカードのroot directoryにsshの公開鍵をauthorized\_keysの名前のファイルで置いてください。rootアカウントなのでパスワードではloginできない設定になっています。
  - `ssh root@[ATOMCamのIPアドレス]`でloginできます。

- webHook機能
  - 各種イベント発生時に指定したURLにpostで通知します。

- 動体検知アラームの不感知期間を短縮

  - Atomcamの動体検知は一度検知すると５分間検知しない仕様ですが、この検知しない期間を30秒に短縮します。
  - メーカーへの迷惑防止のためCloudへの通知、video/jpegのuploadは５分以内の再送をブロックしています。このため、アプリへの通知も５分以内の再検知時は通知されません。
  - SD-Cardには記録されます。(検知時の12秒間のファイル、検知時を含む１分間のファイル共）
  - webHook機能もイベントごとに発生します。必要な場合はwebHook経由で通知を組んでください。RasberryPi上でNode-REDを動かすのがお手軽です。

- atomcam_toolsのupdate機能
  - GitHubのLatestイメージをダウンロードして更新する機能です。
  - 回線状況にもよりますが、３分程度かかります。
  - AtomCamのFWのupdateはWebUIからはできません。アプリからupdateをしてください。

- AtomCamのFW update対応
  - アプリからAtomCamのFW updateをするときにSD-Cardを抜かなくてもできるようになりました。
  - 商品とのubootと同じシーケンスでflashメモリ内の領域の消去と書き込みが行われます。
  - update途中で電源が落ちることがないよう気をつけてください。

- AtomSwingのpan/tilt制御

  - WebUIからのpan/tilt操作

    但し自動追跡をonにしていると映像が動くことで物体認識して取り合いになります。

  - pan/tilt座標系の初期化のためのリセット動作をするボタンをメンテナンスに追加

- AtomSwingのクルーズシーケンス制御

  - WebUIからのpan/tiltと待ち時間の登録
  - 待ち時間中の動体検知、動体追尾の選択

- iCamera_app起動後に実行するshell-scriptの追加

  - SD-Cardにpost_icamera.shファイルが存在すると実行
  - RootFSがReadOnlyになったため、hook pointとして追加

- FAT32＋exFAT構成に対応(FAT32上限の2TB以上のSD-Cardを使いたい場合)

  - **但し、256MB以上のメディアではAtomCamのFWのbugでSD-Cardのrecordフォルダのファイルが消されていきます** 
    - [SD-Cardのrecord以下のファイルが保存日数前に削除される #91](https://github.com/mnakada/atomcam_tools/issues/91#top)

  - u-bootがFATFSしか見えないので、以下の構成でSD-Cardにpartitionを切った構成に対応
  - 1st partition : FATFS boot  将来の余裕をみて16MB以上
    - factory_t31_ZMC6tiIDQNのみ入れる
  - 2nd partition : ExFAT atomtools (残りのサイズ）
    - 残りのrootfs_hack.squashfs, hostname, authorized_keys, etcを入れてください

- Videoのbitrateの設定

  - MainのHDのチャンネルは300~2000bpsの範囲
    - 高い値にすると負荷が増えます
  - Subの360p/320pのチャンネルは100~500bpsの範囲
    - 高い値にすると負荷が増えます

- Videoのframerateの設定

  - 1~30fpsの範囲
    - 25くらいまでが妥当、それ以上を設定するとたまに間に合わなくなって動作がおかしくなります

- モバイルアプリの設定（全てではない）に対応しました。

  - GUIが複雑になるものは対応していません。

- watermarkの置き換え対応

  - 左下のwatermark(ロゴ)の置き換えに対応しました。


<br>


## セキュリティに関わる重要事項
上記項目に書いてある各ポートが利用可能となります。  
現時点ではこのポートはセキュリティ上の懸念材料となりますので、  
ネットワークのセキュリティーを各自十分に保つように心がけてください。
WebUI、video、jpegに関してはLAN内からは自由に見える設定になっています。
sshは物理的にSD-Cardへアクセスして公開鍵を書かないとloginできないようにしています。
ただし、ATOMCamはSSID,passwordを平文でカメラ内のフラッシュメモリ（SD-Cardではない）に保存しているのでカメラを盗難されて中を見られるとWiFiにアクセスされる可能性がありますのでご注意ください。

<br>

## ファームウェアの更新の注意

 atomcam_toolsは以下のATOMCamのVersionで動作確認しています。更新した場合機能が使えなくなることもあるのでご注意ください。

ATOMCam Ver.4.33.3.68, 4.33.3.73

ATOMCam2 Ver.4.58.0.139, 4.58.0.154, 4.58.0.160

ATOMSwing Ver.4.37.1.152, 4.37.1.162, 4.37.1.166

WyzeCamV3 Ver.4.36.9.139 （Experimental release)  4.36.10.xx以降のVer.では動作しません。

**WyzeCamV3は未検証／未対応の機能があります。また、H.265は対応していません。**

<br>


## 関連記事

Qiitaに少し解説を書いています。

[Qiita.com ATOMCam2を少し改造して導入してみた](https://qiita.com/mnakada/items/7d0fbcb6e629e1ddbd0c)

[Qiita.com AtomSwingを少し改造して遊んでみた](https://qiita.com/mnakada/items/5da19a302b0f7521f225)

[atomcam_toolsのtimelapseの負荷低減](https://qiita.com/mnakada/items/eddbf8b6f0095e279095)

[ATOMSwingのtimelapseで星座追尾](https://qiita.com/mnakada/items/e2337f7b73e3a9e1539c)

[atomcam_toolsのMobileApp代替機能の実装](https://qiita.com/mnakada/items/310954397db6f16fe0b5)

[Devinがatomcam_toolsの構造を説明してくれるページ](https://deepwiki.com/mnakada/atomcam_tools)

[Devinがatomcam_toolsの構造を説明してくれるページ-日本語翻訳-](https://translate.google.com/translate?sl=auto&tl=ja&u=https://deepwiki.com/mnakada/atomcam_tools)

（linkが途切れるので上の英語のページを開いてブラウザの翻訳機能で見た方がいいかも）

<br>

## 使用法

https://github.com/mnakada/atomcam_tools/releases/latest

からatomcam_tools.zipをダウンロードし、適当なツールで解凍します。
<img src="https://github.com/mnakada/atomcam_tools/blob/images/extract.png">

解凍されて出てきたすべてのファイルを、ATOMCamで使用可能なmicroSDカードのルートフォルダに保存します 。

**複数台運用する場合はhostnameを修正して重複しない様に設定してください。**

保存したmicroSDカードをATOMCamに入れて電源を入れます。

**SD-Cardはできるだけ高速なものをFAT-FSで使用してください。**

AtomCamのu-bootがexFATに対応していないため、exFATだとhackが認識されず起動されません。

また、メモリ不足をSD-Card上のswapで補っているため、遅いメディアだと負荷が高くなりすぎてうまく動かない事があります。（Class10以上を推奨）

<br>


## Web設定画面

ブラウザで`http://[ATOMCamのIPアドレス]`を開くと設定画面にアクセスできます。

軽量化のためmDNSは無効です。ATOMCam純正アプリやIPアドレス確認ツールなどでATOMCamのIPアドレスを確認してください。

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/camera.jpg" /></kbd>

上のタイトル帯には左にタイトルとtoolsのVer.、真ん中にCameraの種類とVer.、その右にカメラ内の時間、右端に言語設定とこの説明ページへのlinkボタンがあります。

左側には各ページの切り替えのタブがあります。ここを選択する事でページが切り替わります。

以下、各タブのページの説明になります。

<br>

### カメラ画像

最初に上記のカメラ画像のページが開きます。

AtomSwingの場合はPan/Tiltのスライダーがカメラ画面の左と下に表示され、つまんで動かす事ができます。

AtomCam / WyzeCamV3の場合はスライダーは表示されません。

右下にはナイトビジョンのon/auto/offの切り替えボタンがあります。

<br>

### カメラ設定（ATOMCam/ATOMSwingのみ）

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/CameraSettings.jpg" /></kbd>

ATOMアプリのカメラの設定項目の機能をWebからも設定できるようにしました。

このページの設定はATOMアプリ同様に切り替えたタイミングでカメラに反映されるため、右下の設定ボタンはありません。

#### 機能設定

##### - ナイトビジョン

ナイトビジョンのon/auto/offの切り替えをします。

onにすると暗視の白黒画像、offにするとカラー画像、autoにすると環境に応じて自動切り替えになります。

##### - 切り替えタイミング（auto選択時のみ)

ナイトビジョンがautoの時の切り替えタイミングの設定です。

切り替えタイミングを暗い／非常に暗いから選択します。

##### - 赤外線ライト（on/auto選択時のみ）

ナイトビジョンの時に赤外線ライトを点灯するかどうかの選択です。

#### 検出設定

##### - モーション検知

モーション検知でモーション検知録画をするかどうかの設定です。

ここをonにしていないと録画設定のモーション検知録画が動作しません。

##### - 感度調整

モーション検知の感度を高／中／低から選択します。

##### - 検知エリア

モーション検知を全領域と選択範囲で検知するかを切り替えます。

選択範囲にした場合は画面のオレンジの枠で選択範囲を設定します。

##### - サウンド検知

サウンド検知でモーション検知録画をするかどうかの設定です。

##### - 感度調整

サウンド検知の感度を高／中／低から選択します。

##### - 火災／CO警報音検知

火災／CO警報音検知でモーション録画をするかどうかの設定です。

##### - モーションタグ

モーション検知時に緑色の枠を表示します。

##### - 録画モード

連続録画の記録をモーション検知してなくても録画するか、検知時のみ行うか、録画しないかの設定です。

#### その他

##### - ステータスランプ

正面のLEDの点灯を設定します。

##### - 画像180°反転

画像を180°回転させます。

##### - タイムスタンプ

画面右側のタイムスタンプの表示を設定します。

##### - ロゴ

画面左側のwatermark(ロゴ)の表示を設定します。

defaultは　atomcam_tools　の文字がロゴの代わりに表示されます。

ロゴの変更はシステム設定からできます。

元々のATOMのロゴに戻すにはSD-Cardのwatermark.bgraを削除して空のwatermark.brga ファイルを置いてください。

#### 詳細設定

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/AdvancedSettings.jpg" /></kbd>

##### - 諸々設定

カメラのISPの設定値を変更します。変更は即時反映されて再起動後も引き継がれます。

defaultの設定に戻す場合は右側のrefreshボタンを押してください。

色々画質変更できるのですが、contrast~sharpness以外は詳細わからないので説明は割愛します。

それぞれのタイトルはSDKのAPIのコメントからとっています。

（カメラの設定に詳しい方いたらIssuesにでもコメント上げてもらえれば追記します）

<br>

### SD Card

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/sdcard.jpg" /></kbd>

カメラ内のSD-Cardの映像記録のフォルダーにアクセスできます。

alarm_recordはモーション検知録画、recordは連続録画、time_lapseはタイムラプスの保存されているフォルダーです。

ファイルをクリックすることでmp4, jpegファイルを開くことができます。

低負荷でSD-Cardの記録ファイルを確認できます。

##### <br>

### 録画設定

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/recording.jpg" /></kbd>

#### 連続録画

SD-Cardのrecordフォルダーに１分ごとのファイルが記録されます。

モバイルアプリからアクセスされるためSD-Cardのファイル名は変更できません。そのため保存するパスの設定はできません。

#### モーション検知録画

ATOMCamアプリで設定した検出時にクラウドサーバーに保存される12秒の映像と静止画をSD-Cardにも記録します。記録されるフォルダーは alarm_record です。

！！！　「録画ファイルの自動削除」で保存日数を指定しないと一杯になっても削除されません　！！！

以下、各録画項目内の設定です。

##### - SD-Card録画

SD-Cardへの記録をします。

##### - NAS録画

軽量ビルドでは無効です。

##### - 保存するPATH

指定フォルダ以下のファイルのPATHを指定します。最後に拡張子が自動的に付加されます。

書式はstrftimeの指定形式です。最後の/以降はファイル名として後ろに.mp4が付加されます。

例えば %Y%m%d/%H%M%S を指定すると/alarm_record/20211128/223000.mp4 というようなファイルが作られます。 

フォルダがない場合は自動で作成します。

##### - ファイルの自動削除

保存日数で指定した日を超えたファイルを自動削除する機能です。

自動で削除したくない場合はoffにしてください。

容量にご注意ください。

##### - 保存日数

録画ファイルを保存しておく日数です。この日数経過すると自動的に削除されます。

##### - 録画スケジュール

スケジュールを選ぶと、曜日と時間帯を指定する項目が追加されます。

映像を記録する時間帯を設定します。

右端のー/＋で指定項目を削除/追加できます。複数の項目はor条件で効きます。

##### - JPEG記録の停止

 /media/mmc/recordディレクトリへのJPEGファイルの記録を停止します。

<br>

### タイムラプス設定

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/timelapse.jpg" /></kbd>

SD-Cardに指定した一定時間毎に映像を記録します。早送りのような映像が作成できます。

指定した全記録が終わるまではmp4ファイルが完成しません。

設定項目は下記以外は録画設定と同じです。

##### - サンプリング設定

開始する曜日と時間を指定します。

録画の完了する時間が計算されますが、この時刻の後にmp4への変換処理をするので次の録画スタートまで５分以上間を空けてください。

開始時刻から周期と回数で指定された枚数の記録をとり指定されたフレームレートの録画ファイルを生成します。

右側の＋／ーで設定項目を増減できます。

**サンプリングの時間設定が重ならない様に設定してください。**

開始時間に他のサンプリングが実行中の場合、そのサンプリングはskipされます。

##### - 出力fps

出力ファイルの再生フレームレートを指定します。数値は1秒間の表示枚数です。

##### - 動作状態

動作中は進捗を表示します。

##### - 中止

動作中に停止したい場合は、Lockを外して中止ボタンを押してください。

mp4ファイルを生成して中止します。

<br>

### メディア設定

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/media.jpg" /></kbd>

#### SD-Card設定

##### - 録画を直接記録

offにすると一旦RAM-Diskにファイルを生成してからSD-Cardにコピーします。

onにするとSD-Cardに直接ファイルを生成します。

SD-Cardのスピードや各種設定によって最適な設定が分からないため設定を変えて試してください。

SD-Cardに記録される常時録画かモーション検知録画のファイルに効果があります。

##### - SD-Card消去

SD-Cardのrecord, alarm_record, time_lapseフォルダの中身を消去します。

SD-Cardにtoolが入っているため、アプリからのSD-Cardのフォーマットをdisableしています。

その代替手段として用意しています。

Lockスイッチを解除してからEraseボタンを押してください。

<br>

#### NAS設定

軽量ビルドでは無効です。

<br>

### 配信設定

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/streaming.jpg" /></kbd>

#### RTSP

**※ 負荷が重いためSD-Cardのネットワークアクセスと同時使用は推奨しません。**

**画面は説明のために全てonにしていますが、配信自体も処理が重たい為使わないところはできるだけoffにしてください。**

####  Main

Main(video0)側のRTSPストリーミングを行います。 

HD AVCの出力になります。

##### - 音声

Main側の音声を切り替えます。

音声を配信したくない場合はoffにします。RTSPだけを使う場合はクライアントに合わせて選択してください。

##### - URL

Main側のVLC media playerの「ネットワークストリーミングを開く」で入力するURLが表示されます。

####  Main HEVC (ATOMCamのみ)

Main(video2)側のRTSPストリーミングを行います。 

HD HEVCの出力になります。

##### - 音声

Main HEVC側の音声を切り替えます。

RTSPのみの対応なので音声はどれでも構いません。

##### - URL

Main HEVC側のVLC media playerの「ネットワークストリーミングを開く」で入力するURLが表示されます。

#### Sub

Sub(video1)側のRTSPストリーミングを行います。 

360p/320pの出力になります。

##### - 音声

Sub側の音声を切り替えます。

RTSPのみの対応なので音声はどれでも構いません。

##### - URL

Sub側のVLC media playerの「ネットワークストリーミングを開く」で入力するURLが表示されます。

#### RTSP over HTTP

RTSPをUDPでなくHTTP経由で送出します。

AtomCamからPCまでの経路でのパケット伝送が保障されますが、再送により遅延が発生する可能性があります。

変更するとRTSP Main/SubのURLが変わります。

#### パスワード認証

RTSPにパスワード認証を要求します。

##### - アカウント

RTSPのアカウントを設定します。

##### - パスワード

RTSPのパスワードを設定します。

### スピーカー送話

PC側の送話アプリからUDPで送った8kHz/mono/S16LE PCMをカメラのスピーカーへ出力します。

WebUIはdaemonの有効化、UDPポート、音量、トークンの設定だけを行います。音声データはWebUIを通らないため、ブラウザのマイク/HTTPS制約を避けられます。

デフォルトではoffです。使用する場合は配信設定タブの「スピーカー送話」をonにして保存します。トークンを空欄にすると同一LANからのUDP音声を認証なしで受け付けるため、通常は空白を含まない任意のトークンを設定してください。

PC側クライアントは `tools/atomtalk-client` にあります。Windows向けには次のようにビルドできます。

```sh
cd tools/atomtalk-client
GOOS=windows GOARCH=amd64 go build -o atomtalk-client.exe .
```

実行例:

```powershell
.\atomtalk-client.exe -host 192.168.105.196 -token "your-token"
```

ONVIF relay経由で送話する場合は、同梱の `onvif-relay/` を起動し、relayの設定で対象カメラの `talk.enabled` と `talk.token` を設定します。PC側クライアントは `-relay-url http://<relay>:8080/talk/<camera>` を指定して送信できます。

<br>

### イベント通知

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/event.jpg" /></kbd>

#### WebHook

動体検知や録画ファイルの書き込み等のイベントのタイミングで指定のURLに通知します。

##### - 通知URL

WebHookを受け取るURLを指定します。今のところ実験的な実装なのでLAN内のnon-secureなpostを想定しています。

{ type: 'event名', data: あれば何か }の形式でpostします。

##### - 未認証接続許可

URLが自己証明書等の認証されていない証明書を提示しても接続を許可します。

##### - 動体検知

動体検知が働いた時に通知URLに type: alarmEvent をpostします。

##### - 動体認識情報

動体検知の認識情報取得時に通知URLに type: recognitionNotify, data: recognition data をpostします。

##### - 動体検知録画終了

動体検知での録画が終了した時に通知URLに type: uploadVideoFinish をpostします。

##### - 動体検知録画転送

動体検知での録画が終了した時に通知URLに mime:video/mp4で録画ファイル をpostします。

##### - 動体検知静止画保存

動体検知での静止画保存完了時に通知URLに type: uploadPictureFinishをpostします。

##### - 動体検知静止画転送

動体検知での静止画保存完了時に通知URLに mime:image/jpegで静止画ファイルをpostします。

##### - 定常録画保存

１分間の定常録画が終了するたびに通知URLに type: recordEventをpostします。

##### - 定常録画POST

１分間の定常録画mp4が保存されるたびに、録画ファイルをQuickTime形式に変換して指定URLへPOSTします。

`post_icamera.sh`で`inotifywait`を常駐させる代わりに、FW内の`atomrecpostd`が`/media/mmc/record`を監視します。POSTは`ffmpeg`から直接行うため、変換時に追加の`curl`プロセスやshell pipelineは使いません。

送信時の主な形式は次の通りです。

- `Content-Type: video/quicktime`
- `x-video-name: /media/mmc/record/...`
- 映像: copy
- 音声: `pcm_mulaw`, 8000Hz, mono
- `creation_time`: 録画ファイルパスから生成したJST時刻

POST先URLが空の場合は通知URLを使用します。

##### - タイムラプス開始

タイムラプス開始時に通知URLに type: timelapseStart, data: timelapse pathをpostします。

##### - タイムラプス記録

タイムラプス記録のたびに通知URLに type: timelapseEvent, data: timelapse pathをpostします。

##### - タイムラプス録画終了

タイムラプス録画終了時に通知URLに type: timelapseFinish, data: timelapse pathをpostします。

<br>

### クルーズ設定（AtomSwingのみ）

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/cruise.jpg" /></kbd>

#### Swing座標初期化

Swingのpan/tilt座標系を初期化します。

両側の端点に当てることでモーターの動作範囲をリセットさせるための動作をします。

#### クルーズ動作

クルーズ動作のOn/Offの設定とシーケンス登録

以下の項目を登録順に動作し、最後までいくと最初に戻って繰り返します。

シーケンスの１項目を選択中は薄緑の色がつきます。色がついてる項目を編集できます。

右端のー、＋マークで項目の削除、追加ができます。

編集するときは、一旦クルーズをOffにして設定してからにしないとカメラの方向が動いてやりにくいです。

##### - pan, tilt,速度

カメラを向ける方向と移動時の速度を指定します。

pan,tiltは数値を直接入力するか、Jpeg表示の左、下にあるスライダーで方向を制御しても入力できます。

速度は１が低速、９が高速です。

##### - 動作後待機時間

カメラの移動が終わってから、次の移動までの待機時間を秒数で指定します。

##### - 検知

待機時間中に動体検知すると、待機時間を延長します。

動体検知が終わってからの待機時間は下の項目の検知後待機時間になります。

##### - 追尾

待機時間中に動体検知すると、動体を追尾し、待機時間を延長します。

検知後待機時間は上記の検知と同様です。

##### - 検知後待機時間

検知後の待機時間中も動体検知は働いて、待機時間の間検知しないと次のシーケンスに進みます。

##### - 追尾速度

追尾時の移動速度です。

速度は１が低速、９が高速です。

<br>

### システム設定

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/system.jpg" /></kbd>

#### デバイス設定

##### - デバイス名

カメラのデバイス名を設定します。
ここで設定した名前はホスト名に使用されます。

##### - ログイン認証

アカウントとパスワードを設定することでWebUIの簡易ログイン認証を行います。

md5によるdigest認証なので安全ではありません。LAN内での簡易認証として使用してください。

##### - アカウント

１文字以上の英数記号を使用できます。但し':'と'\\'は使用できません。

特にエラーチェックはしてないので入れるとおかしくなるかもしれません。

##### - パスワード

１文字以上の英数記号を使用できます。但し':'と'\\'は使用できません。

特にエラーチェックはしてないので入れるとおかしくなるかもしれません。

####  動体検知

##### - 動体検知周期の短縮

動体検知アラームの不感知期間を５分から30秒に短縮します。

設定変更時には反映するために再起動されます。

##### - 動体検知録画upload停止

動体検知時にAtomTechのawsサーバーに12秒間の録画データがuploadされますが、このuploadを停止します。これを停止するとAtomCamのアプリから録画データを見る事ができなくなります。

設定変更時には反映するために再起動されます。

#### ビデオ設定

##### - フレームレート

１秒あたりのフレーム数(1-28fps)を設定します。

##### - Mainの帯域

Main HDのビットレート(300-2000bps)を設定します。

##### - MainHEVCの帯域 (ATOMCamのみ)

Main H.265/HEVC HDのビットレート(300-2000bps)を設定します。

##### - Subの帯域

Sub360p/320pのビットレート(100-500bps)を設定します。

#### ロゴ

##### - PNGイメージ

枠内にPNGイメージをdropすることで画面左下のロゴを変更することができます。

PNGファイルはRGBA 各色8bitのフォーマットで500px X 200px以内で作成してください。

設定ボタンを押すと反映されます。

<br>

### メンテナンス

<kbd><img src="https://github.com/mnakada/atomcam_tools/blob/images/maintenance.jpg" /></kbd>

#### モニタリング

##### - Network確認

LANとの接続を確認し、再接続を試みます。

##### - 異常時再起動

LANとの接続ができない状態が継続した場合、システムの再起動をします。

##### - ping

定期的な疎通確認を行います。

##### - 通知URL

疎通確認をするURLを指定します。

ここで指定したURLにhttp getを１分毎に行います。

#### Update

##### - Update

GitHubのLatest VersionにUpdate します。GitHubからLatest Versionをダウンロードして展開して書き込み再起動します。180秒くらいかかります。

 現在のtoolのバージョン（タイトル部に表示されています）がLatest Versionより古い場合のみUpdateすることができます。

台数が多い場合や回線が細い場合、PCで[GitHubのLatest Version](https://github.com/mooglejp/atomcam_tools/releases/latest)のatomcam_tools.zipをダウンロードし、展開せずにSD-Cardのupdateフォルダに入れて、リブートすることでもUpdateできます。この場合はVerのチェックは行われません。

このforkのtoolバージョンは、fork元のベースバージョンにカスタムリビジョンを付けた `2.5.19-moogle.3` のような形式で表記します。`2.5.19` がベース、`moogle.3` がこのfork側のリビジョンです。

##### - カスタム更新ZIPファイル

Updateの更新ZIPファイルを指定のURLから取得します。

独自のpackage buildをする場合の用途になります。

GitHub ActionsでビルドしたZIPを直接指定する場合は、通常のActions artifactではなくRelease assetのURLを指定してください。このリポジトリのワークフローは`main`ブランチまたは手動実行で`ci-latest`リリースに`atomcam_tools.zip`をアップロードします。

```text
https://github.com/<owner>/<repo>/releases/download/ci-latest/atomcam_tools.zip
```

`v*`タグで実行した場合は通常のリリースassetとしてアップロードされるため、Latest Releaseを使う運用では次のURLも利用できます。

```text
https://github.com/<owner>/<repo>/releases/latest/download/atomcam_tools.zip
```

#### 再起動

##### - 定期リスタート

カメラのシステムを指定したスケジュールで再起動します。

ネットワークの不調など、何らかの理由でATOMCamが連続稼働することができない場合の対応ですが、必ずしもこれによって問題が解決するとは限りません。  

##### - リブート

AtomCamを再起動します。

Lockスイッチを解除してRebootボタンを押してください。

再起動に60~80秒くらいかかります。

<br>

### Copyright

LICENSEファイルを参照してください
