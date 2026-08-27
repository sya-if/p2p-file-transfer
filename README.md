## Installation & Usage

### Option 1: Download Pre-built Binary (Recommended)
Download the latest `p2p-transfer-windows.exe` from the [Releases page](https://github.com/sya-if/p2p-file-transfer/releases).

### Option 2: Build from Source
If you have Go installed, clone this repo and build the binary:

```bash
# Clone the repository
git clone [https://github.com/YOUR_USERNAME/YOUR_REPO.git](https://github.com/YOUR_USERNAME/YOUR_REPO.git)
cd YOUR_REPO

# Build for Windows (PowerShell)
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -o p2p-transfer-windows.exe main.go

# Build for Android (Termux)
$env:GOOS="android"; $env:GOARCH="arm64"; go build -o p2p-transfer-android main.go
cp /sdcard/Download/p2p-transfer-android ~/
cd ~

# Run the app
.\p2p-transfer-windows.exe
