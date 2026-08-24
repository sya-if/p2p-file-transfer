package main

import(
	"fmt"
	"io"
	"net"
	"os"
)

func main(){
	if len(os.Args) < 3{
		fmt.Println("Usage: go run sender.go <RECEIVER_IP> <FILE_PATH>")
		fmt.Println("Example: go run sender.go 192.168.1.50 test.txt")
		return
	}

	receiverIP := os.Args[1]
	filePath := os.Args[2]

	//1. Open local file for reading
	file, err := os.Open(filePath)
	if err != nil{
		fmt.Println("Error opening file: ", err)
		return
	}
	defer file.Close()

	//2. Dial TCP connection to receiver IP on port 9000
	address := net.JoinHostPort(receiverIP, "9000")
	conn, err := net.Dial("tcp", address)
	if err != nil{
		fmt.Println("Error connecting to receiver: ", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Connected! Sending %s to %s...\n", filePath, address)

	//3. Stream file contents through the TCP secket
	bytesSent, err := io.Copy(conn, file)
	if err != nil{
		fmt.Println("Error sending file: ", err)
		return
	}

	fmt.Printf("Done! Sent %d bytes.\n", bytesSent)
}