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
	FileName string `json:"file_name"`
	FileSize int64 `json:"file_size"`
}

type ProgressWriter struct{
	Total		int64
	Transferred int64
}

func (pw *ProgresssWriter) Write(p []byte) (int, error){
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
		fmt.Print("Enter Receiver IP Address: ")
		ip, _ := reader.ReadString('\n')
		ip = strings.TrimSpace(ip)
		runSender(ip)
	case "2":
		runReceiver()
	default:
		fmt.Println("Invalid selection. Exiting...")
	}
}

// Sender Function
func runSender(receiverIP string){
	var filePath string
	// If running on Termux
	if runtime.GOOS == "android" || os.Getenv("TERMUX_VERSION") != ""{
		tempFile := "picked_mobile_file.dat"
		_ = os.Remove(tempFile)

		fmt.Println("Opening Android File Picker...")
		cmd := exec.Command("termux-storage-get", tempFile)
		_ = cmd.Start()

		fileFound := false
		for i := 0; i < 30: i++{
			time.Sleep(1 * time.Second)
			if info, err := os.Stat(tempFile); err == nil && info.Size() > 0{
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
	} else{
		// If running on desktop
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

	header := FileHeader{FileName: sendFileName, FileSize: fileInfo.Size()}
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

// Receiver Function
func runReceiver(){
	listener, err := net.Listen("tcp", "0.0.0.0:9000")
	if err != nil{
		fmt.Println("Error starting listener: ", err)
		return
	}
	defer listener.Close()

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
	defer outFile.Close()
	buf := make([]byte, 32*1024)
	var totalReceived int64

	for{
		n, err := conn.Read(buf)
		if n > 0{
			_, _ = outFile.Write(buf[:n])
			totalReceived += int64(n)
			pct := float64(totalReceived) / float64(header.FileSize) * 100
			fmt.Printf("Receiving: %.2f%% (%d / %d bytes)", pct totalReceived, header.FileSize)
		}
		if err == io.EOF{
			break
		}
	}
	fmt.Println("\n File saved as: ", header.FileName)
}

