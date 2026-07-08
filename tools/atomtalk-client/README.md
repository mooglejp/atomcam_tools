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

The default Windows capture path uses ffmpeg's WASAPI input:

```powershell
ffmpeg -f wasapi -i default ...
```

If your ffmpeg build does not support WASAPI, use DirectShow and pass the device name:

```powershell
ffmpeg -list_devices true -f dshow -i dummy
.\atomtalk-client.exe -host 192.168.105.196 -format dshow -input 'audio=Microphone (USB Audio Device)'
```

Use `Ctrl+C` to stop. The client sends an `ATOMTALK STOP` control packet before exiting.
