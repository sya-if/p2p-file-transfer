package main

import(
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

type FileHeader struct{
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

func main(){
	
	outputDir := "Received File"
	// Ensure the output directory exists
	err := os.MkdirAll(outputDir, 0755)
	if err != nil{
		fmt.Println("Error creating output directory: ", err)
		return
	}

	// Start listening on TCP port 9000
	listener, err:= net.Listen("tcp", ":9000")
	if err != nil{
		fmt.Println("Error starting listener: ", err)
		return
	}
	defer listener.Close()

	fmt.Println("Receiver waiting for connection on port 9000...")

	// Accept incoming TCP connection from Sender
	for{
		conn, err := listener.Accept()
		if err != nil{
			fmt.Println("Connection failed: ", err)
			continue
		}
		go handleConnection(conn, outputDir)
	}
}

func handleConnection(conn net.Conn, outputDir string){
	defer conn.Close()

	// Read JSON Header Length (4 bytes)
	var headerLen int32
	err := binary.Read(conn, binary.BigEndian, &headerLen)
	if err != nil{
		fmt.Println("Error reading header length: ", err)
		return
	}

	// Read JSON Header Payload
	headerBuf := make([]byte, headerLen)
	_, err = io.ReadFull(conn, headerBuf)
	if err != nil{
		fmt.Println("Error reading header payload: ", err)
		return
	}

	var header FileHeader
	err = json.Unmarshal(headerBuf, &header)
	if err != nil{
		fmt.Println("Error parsing header JSON: ", err)
		return
	}

	// Create file inside 'Received File' folder with original name
	savePath := filepath.Join(outputDir, header.FileName)
	outFile, err := os.Create(savePath)
	if err != nil{
		fmt.Println("Error creating destination file: ", err)
		return
	}
	defer outFile.Close()

	fmt.Printf("Receiving '%s' (%d bytes)...", header.FileName, header.FileSize)

	// Stream rest of TCP connection into file
	bytesReceived, err := io.Copy(outFile, conn)
	if err != nil{
		fmt.Println("Error writing to file: ", err)
		return
	}

	fmt.Printf("Saved to '%s' (%d bytes received)\n\n", savePath, bytesReceived)
}