# atomtalk-client

PC microphone audio to ATOMCam speaker bridge.

The client captures audio with `ffmpeg`, converts it to 8000 Hz mono signed 16-bit little-endian PCM, and sends it to `atomtalkd` over UDP. The camera firmware must have `ATOMTALK_ENABLE=on`.

## Windows build

From WSL or Linux:

```sh
cd tools/atomtalk-client
GOOS=windows GOARCH=amd64 go build -o atomtalk-client.exe .
```

## Windows usage

Install `ffmpeg.exe` and keep it in `PATH`, then run:

```powershell
.\atomtalk-client.exe -host 192.168.105.196 -token "your-token"
```

To send through `onvif-relay` instead of directly to the camera:

```powershell
.\atomtalk-client.exe `
  -relay-url http://192.168.1.10:8080/talk/living-room `
  -relay-user onvif_user `
  -relay-pass onvif_password
```

To play an audio file instead of microphone input, use `-file`. The client lets ffmpeg auto-detect the input format and adds `-re` so playback is sent in real time. File playback appends 1000 ms of silence by default so the camera speaker can drain its output buffer; tune this with `-tail-ms`.

```powershell
.\atomtalk-client.exe `
  -host 192.168.105.196 `
  -token "your-token" `
  -file "D:\Git\Irodori-TTS\outputs\no-leave.wav" `
  -tail-ms 1000
```

Relay mode can use the same file input:

```powershell
.\atomtalk-client.exe `
  -relay-url http://192.168.1.10:8080/talk/living-room `
  -relay-user onvif_user `
  -relay-pass onvif_password `
  -file "D:\Git\Irodori-TTS\outputs\no-leave.wav"
```

The default Windows capture path uses ffmpeg's DirectShow input and auto-selects the first audio capture device:

```powershell
ffmpeg -list_devices true -f dshow -i dummy
ffmpeg -f dshow -i "audio=<first audio device>" ...
```

To pass the device name yourself:

```powershell
ffmpeg -list_devices true -f dshow -i dummy
.\atomtalk-client.exe -host 192.168.105.196 -format dshow -input 'audio=Microphone (USB Audio Device)'
```

If your ffmpeg build supports WASAPI and you prefer it:

```powershell
.\atomtalk-client.exe -host 192.168.105.196 -format wasapi -input default
```

Use `Ctrl+C` to stop. The client sends an `ATOMTALK STOP` control packet before exiting.
