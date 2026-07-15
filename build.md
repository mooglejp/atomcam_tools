# SD-Cardイメージの作成方法

## 必要な環境

- Windows / macOS / Linux
- GitHubにアクセスできる環境
- Dockerが実行可能な環境

※ 以下の手順はmacOS Sonomaで動作確認済みです。

## ビルド手順   

適当なディレクトリで以下のコマンドを実行します。
macOSの場合はDockerの起動にLimaを使用します。
事前にLima、Docker CLI、docker-composeをインストールしておいてください。

```sh
git clone https://github.com/mnakada/atomcam_tools
cd atomcam_tools
make lima  # macOSのみ
make
```

環境によりますが、約1時間で`atomcam_tools/atomcam_tools.zip`が生成されます。

**以下のファイルが含まれます:**
- `authorized_keys`
- `hostname`
- `factory_t31_ZMC6tiIDQN`
- `rootfs_hack.squashfs`


リモートログインを行う場合は、SSHの公開鍵を`authorized_keys`に追加してください:

```sh
cat ~/.ssh/id_rsa.pub >> ./target/authorized_keys
```

デバイス名を変更したい場合、`hostname`を編集してください（デフォルト: atomcam）:

```sh
echo "新しいホスト名" > ./target/hostname
```

**上記4つのファイルをSDカードにコピー**し、AtomCamに挿入して起動します。

> **注意**: 初回起動時は、スワップファイルの作成とSSHホストキーの生成のため、約40秒ほど時間がかかります。

ビルド環境はイメージ作成後、Docker上にコンテナが起動した状態になります。

コンテナに入るには以下のコマンドを使用します：

```sh
make login
```

Dockerコンテナを手動で起動:

```sh
make lima               # macOSの場合
docker-compose up -d    # Linuxの場合
```


## AtomCam内部の環境

このイメージでAtomCamを起動すると、glibcで生成されたMIPSEL版のLinux環境が起動します。

この環境内で`/atom`ディレクトリ以下に本来のAtomCamシステムを起動し、chroot監獄に閉じ込めています。

システム構成は以下の通りです:

- **SoC:** Ingenic T31 SoC
- **CPU:** MIPS32R5 I\$32K/D\$32K/L2\$128K
- **Kernel:** Linux 3.10.14 MIPSEL

# 起動シーケンス

AtomCamの起動シーケンスは以下の通りです:

#### 1. U-Boot

- カーネルに内蔵されたinitramfsの`/init`ディレクトリに配置されます

#### 2. Initramfs

- 内容は`initramfs_skeleton/`ディレクトリに格納されています
- カーネル起動時のコマンドラインで`/init`を実行するよう設定されています

#### 3. ツール更新

必要に応じて、ツールの更新処理が実行されます

#### 4. ルートファイルシステムの切り替え

- SD-Card上の`rootfs_hack.squashfs`をルートにswitch_rootします
- remountの処理を行います
- `/sbin/init（busybox）`を起動します

これにより、AtomCamのカスタムファームウェアが正常に起動します。

## `rootfs_hack.squashfs`

`rootfs_hack.squashfs`は、`configs/atomcam_defconfig`の設定でビルドされたイメージに`overlay_rootfs`を重ねたものです。

### 起動プロセス

1. `/sbin/init`が`inittab`に従って`/etc/init.d/rcS`を起動
2. `rcS`が`/etc/init.d/S*`を順番に実行
3. `/etc/init.d`の実行
- **シリアル接続時:** gettyは常駐メモリ削減のためデフォルト無効
- **AtomCam:** 背面LEDが青点滅から青点灯に変わるとSSHログイン可能


## `/etc/init.d/S16fwupdate`

AtomCamのファームウェアアップデートのシーケンスが実行中の場合、その処理を代行します。

## `/etc/init.d/S20mountfs`

overlayfsが使用できないため、bind mountでシステムのファイル/フォルダーの配置を入れ替えています。

## `/etc/init.d/S61atomcam`

/atom/以下に本来のATOMCamのシステムと幾つかのmount-pointを共通でアクセスできるようにmountします。

その後、chrootで/atom の`/tmp/system/bin/atom_init.sh`を呼び出します。

> **ここまではglibcの世界で動作しています。**

## `/atom/tmp/system/bin/atom_init.sh`

本来のAtomCamの初期化シーケンスを実行します。ここからuClibcの環境に入ります。
iCamera_app実行時にlibcallback.soを介在させ、各種パッチを適用して処理を追加しています。WebHookやatom-log等でiCamera_appのログを読む必要がある場合だけ、名前付きFIFOファイル（`/var/run/atomapp`）に出力します。

これを実行するとウォッチドッグが起動するため、assistとiCamera_appは停止できなくなります。

認識機能などの機能はiCamera_app内にあるわけではなく、クラウドから読み込まれて実行されているようです。

# 各種スクリプト

## `/atom/bin/mv`, `/atom/bin/rm`

AtomCamのiCloud_appが動体検知をクラウドに送信後に削除する際のrmコマンド、1分ごとのSD-Cardへの記録ファイルを/tmpから移動するmvコマンドを置き換えて、WebHookのイベントを送信するためのスクリプトです。軽量ビルドではNAS/CIFS記録は無効です。

## `/scripts/cmd`

iCamera_app内部パラメータや動作を変更するためのコマンドを実行する`atomcmd`バイナリです。

## `/scripts/cruise.sh`

AtomSwingでのクルーズ動作を実行するためのスクリプトです。

## `/scripts/hack_ini_reconfig.sh`

バージョンアップでhack_iniの互換性がない場合に引き継ぎ処理をするためのスクリプトです。

## `/scripts/health_check.sh`

定期的にネットワークの健全性のチェックを行うスクリプトです。

## `/scripts/lighttpd.sh`

WebUIのlighttpdの起動処理と認証の切り替え等の処理を行うスクリプトです。

## `/scripts/memory_check.sh`

定期的にメモリーの状態をログに記録するスクリプトです。

## `/scripts/motor_init`

AtomSwingでモーターの初期位置動作をするスクリプトです。

## `/scripts/network_init.sh`

ネットワークの接続をするための初期化スクリプトです。

## `/scripts/reboot.sh`

WebUIの定期リブート設定をcrontabで指定時間に実行するためのスクリプトです。

同期してリブートを実行します。

## `/scripts/remove_old.sh`

指定時間経過した録画データを削除するためのスクリプトです。

## `/scripts/rtspserver.sh`

init.d/S58rtspserverとWebUIのRTSPのオン/オフから呼ばれます。

v4l2rtspserverをオン/オフします。

`RTSP_DSCP`を`LIVE555_DSCP`としてv4l2rtspserverに渡し、live555側で送信ソケットにDSCPを設定します。

圧縮映像フレームの経路診断は`/scripts/cmd video <ch> diag on`で有効化し、
`/scripts/cmd video <ch> diag off`で無効化します。有効時はiCamera_app側のwrite直前に、
フレーム連番、時刻、サイズ、FNV-1a 64bitハッシュ、write結果を`tools.log`へ出力します。
`v4l2rtspserver`を`-vv`で起動すると、V4L2読取直後とLIVE555への配送直前にも同じハッシュ、
NALの分割番号、受入上限、切り詰めバイト数を出力します。

## `/scripts/set_crontab.sh`

`reboot.sh`や`timelapse.sh`を起動する時刻をcrontabに設定するためのスクリプトです。

## `/scripts/set_icamera_config.sh`

`iCamera_app`起動直後に設定しておくべき設定値を処理するためのスクリプトです。

## `/scripts/timelapse.sh`

タイムラプスの開始処理、終了時のファイル処理のスクリプトです。

## `/usr/bin/atomwebcmd`

`/var/www/cgi-bin/exec.cgi`から名前付きFIFO経由でコマンドを実行します。

CGIの実行は`www-data`アカウントでの実行なのでシステム制御系のコマンドは直接実行できないため、コマンドを受けて実行して問題ないものだけ実行する構造にしています。

## `/usr/bin/atomrecpostd`

`WEBHOOK_RECORD_EVENT`または`WEBHOOK_RECORD_UPLOAD`が有効な場合、`/media/mmc/record`をinotifyで監視します。

録画mp4が完成したら、JSONの`recordEvent`通知、または`ffmpeg`によるQuickTime形式への変換とHTTP POSTを行います。`WEBHOOK_RECORD_UPLOAD_DELAY_SEC`でPOST開始を遅延し、`WEBHOOK_RECORD_UPLOAD_TARGET_SEC`で1分定常録画の送信時間をおおよそ指定できます。`inotifywait`やshell pipelineを常駐させずに定常録画のPOST連携を行うための軽量daemonです。

## `/usr/bin/atomtalkd`

`ATOMTALK_ENABLE=on` の場合だけ起動します。

PC側クライアントからUDPで受け取った8kHz/mono/S16LE PCMをFIFOに書き込み、`iCamera_app` 内の `talk` コマンド経由でカメラのスピーカーへ出力します。音声データはWebUIを通しません。

## `/usr/bin/atomhookd`

iCamera_appのログを受けてWebHookのイベントやタイムラプス完了を拾います。録画ファイルPOSTだけを使う構成では起動しません。

iCamera_appの実行環境では制限があるため、名前付きFIFO経由でログを受けて必要に応じてcurlでポストしています。

## `/var/www/cgi-bin/cmd.cgi`

WebUIからのコマンドを名前付きパイプ経由でwebcmd.shに渡しています。

## `/var/www/cgi-bin/get_jpeg.cgi`

WebUIで表示するJPEG画像を取得しています。

## `/var/www/cgi-bin/hack_ini.cgi`

WebUIで使用している設定値の取得、設定をします。

## `/var/www/cgi-bin/diagnostics.cgi`

WebUIの診断タブ用に、負荷、メモリ、ストレージ、主要プロセス、主要設定、RTSP/録画POSTログ末尾を軽量なテキスト形式で返します。

## `/var/www/cgi-bin/hello.cgi`

モバイルアプリからのアクセス時の要求に応答するためのCGIです。

## `/var/www/cgi-bin/video_isp.cgi`

カメラ設定の詳細設定項目を操作するCGIです。

## `/var/www/cgi-bin/watermark.cgi`

システム設定のロゴを設定するためのCGIです。

# WebUI

`web/`ディレクトリ以下にWebUIのソースコードがあります。
WebUIはVue.jsとElementUIで記述しています。
ターゲット環境はMIPSELなのでNode.jsの最新のバージョンは未対応です。

そのため、フロントエンド側のみビルドして、バックエンド側はlighttpdとCGIで対応し、フロントエンドからaxios経由でアクセスする構造にしています。

WebUIの画面は`web/source/vue/Setting.vue`に記述しています。

# Docker環境

Docker環境では `/src` が `atomcam_tools/` にマップされています。

以下、基本的にDocker内のコマンドは下記のディレクトリから実行します:

```
root@ac0375635c01:/atomtools# cd /atomtools/build/buildroot-2016.02
```

rootfsはglibc環境でDocker内のgccを使用します。
ビルド時にgccも生成されます。

**gccのプレフィックスは以下の通りです:**
`/atomtools/build/buildroot-2016.02/output/host/usr/bin/mipsel-ingenic-linux-gnu-`

ATOMCam本来のシステムのカメラアプリiCamera_appはuClibc環境でビルドされています。
そのため、iCamera_appのハック用のlibcallback.soのビルドにはuClibc環境が必要です。
別途**cross tools-ng-1.26.0**を導入しています。

**uClibc用gccのプレフィックスは以下の通りです:**
`/atomtools/build/cross/mips-uclibc/bin/mipsel-ingenic-linux-uclibc-`

# 各種変更時のビルド方法

### initramfs, kernelのconfigを変更した場合

```sh
make linux-rebuild
make
```

これでビルドされ、`atomcam_tools/target`にコピーされます。

### rootfs内のファイルやbusyboxのmenuconfigを修正した場合

```sh
make
```

これでビルドされ、`atomcam_tools/target`にコピーされます。

### rootfsに含まれるパッケージを変更した場合

```sh
make menuconfig
make
```

これでビルドされ、`atomcam_tools/target`にコピーされます。

### 個別のパッケージをリビルドする場合

```sh
make <package>-rebuild
make
```

### busyboxのコマンド等の設定を変更する場合

```sh
make busybox-menuconfig
make
```

これでrootfsがビルドされます。

### kernelの設定を変更する場合

```sh
make linux-menuconfig
make linux-rebuild
make
```

これでビルドされ、`atomcam_tools/target`にコピーされます。
