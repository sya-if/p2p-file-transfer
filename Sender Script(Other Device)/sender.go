package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Metadata header sent before raw file bytes
type FileHeader struct{
	FileName string `json:"file_name"`
	FileSize int64 `json:"file-size"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run sender.go <RECEIVER_IP> <FILE_PATH>")
		return
	}

	receiverIP := os.Args[1]
	var filePath string

	// If file path is provided in arguments, use it
	// Otherwise, invoke Termux Android File Picker!
	if len(os.Args) >= 3{
		filePath = os.Args[2]
	}else{
		tempFile := "picked_mobile_file.dat"
		_ = os.Remove(tempFile)

		fmt.Println("Opening Android File Picker...")
		// Run termux-storage-get via OS command
		cmd := exec.Command("termux-storage-get", tempFile)
		_ = cmd.Start()
		
		// Wait up to 30 seconds for the user to pick a file
		fileFound := false
		for i := 0; i < 30; i++{
			time.Sleep(1 * time.Second)
			if info, err := os.Stat(tempFile); err == nil && info.Size() > 0{
				fileFound = true
				break
			}
		}

		if !fileFound{
			fmt.Println("Timed out or no file was selected.")
			return
		}

		// Feedback immediately after picker closes
		fmt.Println("File selected! Preparing transfer...")
		//exec.Command("termux-toast", "File selected! Sending to laptop...").Run()

		filePath = tempFile
		// Clean up the temporary file when main finishes
		defer os.Remove(tempFile)
	}

	// 1. Open local file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 2. Get file details
	fileInfo, err := file.Stat()
	if err != nil{
		fmt.Println("Error reading file info: ", err)
		return
	}

	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	_, _ = file.Seek(0, 0)

	contentType := http.DetectContentType(buffer[:n])
	fmt.Printf("Detected file type: %s\n", contentType)

	ext := ".dat"
	switch contentType{
		case "image/jpg":
			ext = ".jpg"
		case "image/jpeg":
			ext = ".jpeg"
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

	// Assign clean name with real extension
	sendFileName := filepath.Base(filePath)
	if sendFileName == "picked_mobile_file.dat"{
		sendFileName = fmt.Sprintf("received_%d%s", time.Now().Unix(), ext)
	}

	// Build TCP Header
	header := FileHeader{
		FileName:sendFileName,
		FileSize: fileInfo.Size(),
	}

	// Connect to Receiver
	address := net.JoinHostPort(receiverIP, "9000")
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting to receiver:", err)
		return
	}
	defer conn.Close()

	headerData, err := json.Marshal(header)
	if err != nil{
		fmt.Println("Error encoding metadata: ", err)
		return
	}

	// 4. Send length of header (4bytes) followed by header payload
	headerLen := int32(len(headerData))
	conn.Write([]byte{
		byte(headerLen >> 24),
		byte(headerLen >> 16),
		byte(headerLen >> 8),
		byte(headerLen),
	})
	conn.Write(headerData)

	// 5. Stream raw file payload
	fmt.Printf("Sending '%s' (%d bytes) to %s...\n", header.FileName, header.FileSize, address)
	bytesSent, err := io.Copy(conn, file)
	if err != nil {
		fmt.Println("Error sending file payload: ", err)
		return
	}

	fmt.Printf("Done! Sent %d bytes successfully.\n", bytesSent)
}

