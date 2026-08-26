package main

import(
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type FileHeader struct{
	FileName 	string 	`json:"file_name"`
	FileSize 	int64 	`json:"file_size"`
	Hash 		string 	`json:"hash"`
}

type ProgressWriter struct{
	Total		int64
	Transferred int64
}

func (pw *ProgressWriter) Write(p []byte) (int, error){
	n := len(p)
	pw.Transferred += int64(n)
	percentage := float64(pw.Transferred) / float64(pw.Total) + 100
	fmt.Printf("\r Progress: %.2f%% (%d / %d bytes)", percentage, pw.Transferred, pw.Total)
	return n, nil
}

func main(){
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===========================")
	fmt.Println("	Go P2P File Transfer	")
	fmt.Println("===========================")
	fmt.Println("1. Send a file")
	fmt.Println("2. Receive a file")
	fmt.Println("Choose an option (1 or 2): ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice{
	case "1":
		runSender()
	case "2":
		runReceiver()
	default:
		fmt.Println("Invalid selection. Exiting...")
	}
}

// Windows File Picker
func getWindowsFilePath() string{
	psScript := `
	Add-Type -AssemblyName System/Windows.Forms
	$FileBrowser = New-Object System.Windows.Forms.OpenFileDialog
	$FileBrowser.Title = "Select a File to Send"
	$FileBrowser.ShowDialog() | Out-Null
	Write-Host $FileBrowser.FileName
	`
	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil{
		return ""
	}
	return strings.TrimSpace(string(output))
}

// UDP Discovery for finding receiver
func discoveryReceiverIP() (string, error){
	fmt.Println("Searching for receiver on local Wifi network...")
	pc, err := net.ListenPacket("udp4", ":9999")
	if err != nil{
		return "", err
	}
	defer pc.Close()

	_ = pc.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 1024)

	for{
		n, addr, err := pc.ReadFrom(buf)
		if err != nil{
			return "", fmt.Errorf("Receiver not found on network")
		}
		if string(buf[:n]) == "P2P_RECEIVER_HERE"{
			ip, _, _ := net.SplitHostPort(addr.String())
			return ip, nil
		}
	}
}

// Sender Function
func runSender(){
	receiverIP, err := discoveryReceiverIP()
	if err != nil{
		fmt.Println("Auto-discovery failed: ", err)
		return
	}
	fmt.Printf("Discovered Receiver at IP: %s\n", receiverIP)
	
	var filePath string

	// If running on Termux
	if runtime.GOOS == "android" || os.Getenv("TERMUX_VERSION") != ""{
		tempFile := "picked_mobile_file.dat"
		_ = os.Remove(tempFile)

		fmt.Println("Opening Android File Picker...")
		cmd := exec.Command("termux-storage-get", tempFile)
		_ = cmd.Start()

		fileFound := false
		for i := 0; i < 30; i++{
			time.Sleep(1 * time.Second)
			info, err := os.Stat(tempFile)
			if err == nil && info.Size() > 0{
				fileFound = true
				break
			}
		}

		if !fileFound{
			fmt.Println("Timed out or no file selected.")
			return
		}

		filePath = tempFile
		defer os.Remove(tempFile)
	}else if runtime.GOOS =="windows"{
		fmt.Println("Opening File Picker...")
		filePath = getWindowsFilePath()
		if filePath == ""{
			fmt.Println("No file selected.")
			return
		}
	} else{
		fmt.Print("Enter path of the local file to send: ")
		reader := bufio.NewReader(os.Stdin)
		inputPath, _ := reader.ReadString('\n')
		filePath = strings.TrimSpace(strings.Trim(inputPath, "\""))
	}

	file, err := os.Open(filePath)
	if err != nil{
		fmt.Println("Error opening file: ", err)
		return
	}
	defer file.Close()
	
	fileInfo, _ := file.Stat()

	// Calculate SHA-256
	fmt.Println("Calculating file checksun (SHA-256)...")
	hasher := sha256.New()
	_, _ = io.Copy(hasher, file)
	fileHash := hex.EncodeToString(hasher.Sum(nil))
	_, _ = file.Seek(0, 0)

	//Detect the type of extension
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	_, _ = file.Seek(0, 0)

	ext := ".dat"
	contentType := http.DetectContentType(buffer[:n])
	switch contentType{
	case "image/jpeg":
		ext = ".jpeg"
	case "image/jpg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	case "video/mp4":
		ext = ".mp4"
	case "application/pdf":
		ext = ".pdf"
	}

	sendFileName := filepath.Base(filePath)
	if sendFileName == "picked_mobile_file.dat"{
		sendFileName = fmt.Sprintf("received_%d%s", time.Now().Unix(), ext)
	}

	address := net.JoinHostPort(receiverIP, "9000")
	conn, err := net.Dial("tcp", address)
	if err != nil{
		fmt.Println("Error connecting to receiver: ", err)
		return
	}
	defer conn.Close()

	header := FileHeader{
		FileName: sendFileName,
		FileSize: fileInfo.Size(),
		Hash: fileHash,
	}
	headerData, _ := json.Marshal(header)

	headerLen := int32(len(headerData))
	conn.Write([]byte{
		byte(headerLen >> 24),
		byte(headerLen >> 16),
		byte(headerLen >> 8),
		byte(headerLen),
	})
	conn.Write(headerData)

	pw := &ProgressWriter{Total: fileInfo.Size()}
	tee := io.TeeReader(file, pw)

	_, err = io.Copy(conn, tee)
	if err != nil{
		fmt.Println("\n Error sending file: ", err)
		return
	}
	fmt.Println("\n Success! File transferred completely.")
}

//Broadcast presence for auto-discovery
func startUDPBroadcast(stopChan chan bool){
	broadcastAddr, _ := net.ResolveUDPAddr("udp4", "255.255.255.255:9999")
	conn, err := net.DialUDP("udp4", nil, broadcastAddr)
	if err != nil{
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for{
		select{
		case <-stopChan:
			return
		case <-ticker.C:
			_, _ = conn.Write([]byte("P2P_RECEIVER_HERE"))
		}
	}
}

// Receiver Function
func runReceiver(){
	stopBroadcast := make(chan bool)
	go startUDPBroadcast(stopBroadcast)
	
	listener, err := net.Listen("tcp", "0.0.0.0:9000")
	if err != nil{
		fmt.Println("Error starting listener: ", err)
		return
	}
	defer listener.Close()

	fmt.Println("Receiver running on prot 9000.Waiting for incoming connection...")
	conn, err := listener.Accept()
	stopBroadcast <- true
	
	if err != nil{
		fmt.Println("Connection error: ", err)
		return
	}
	defer conn.Close()

	var headerLen int32
	lenBuf := make([]byte, 4)
	_, _ = io.ReadFull(conn, lenBuf)
	headerLen = int32(lenBuf[0])<<24 | int32(lenBuf[1])<<16 | int32(lenBuf[2])<<8 | int32(lenBuf[3])

	headerBuf := make([]byte, headerLen)
	_, _ = io.ReadFull(conn, headerBuf)

	var header FileHeader
	_ = json.Unmarshal(headerBuf, &header)

	fmt.Printf("Receiving '%s' (%d bytes)...\n", header.FileName, header.FileSize)

	outFile, _ := os.Create(header.FileName)
	
	buf := make([]byte, 32*1024)
	var totalReceived int64

	for{
		n, err := conn.Read(buf)
		if n > 0{
			_, _ = outFile.Write(buf[:n])
			totalReceived += int64(n)
			pct := float64(totalReceived) / float64(header.FileSize) * 100
			fmt.Printf("Receiving: %.2f%% (%d / %d bytes)", pct, totalReceived, header.FileSize)
		}
		if err == io.EOF{
			break
		}
	}
	outFile.Close()

	fmt.Println("\n Verifiying checksum integrity...")
	savedFile, _ := os.Open(header.FileName)
	hasher := sha256.New()
	_, _ = io.Copy(hasher, savedFile)
	savedFile.Close()

	receivedHash := hex.EncodeToString(hasher.Sum(nil))

	if receivedHash == header.Hash{
		fmt.Println("Checksum verified! File is 100% intact.")
	}else{
		fmt.Println("Checksum mismatch! Transferred file may be corrupted.")
	}
}

