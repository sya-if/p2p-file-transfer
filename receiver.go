package main

import(
	"fmt"
	"io"
	"net"
	"os"
)

func main(){
	
	//1. Start listening on TCP port 9000
	listener, err:= net.Listen("tcp", ":9000")
	if err != nil{
		fmt.Println("Error starting listener: ", err)
	}
	defer listener.Close()

	fmt.Println("Receiver waiting for connection on port 9000...")

	//2. Accept incoming TCP connection from Sender
	conn, err := listener.Accept()
	if err != nil{
		fmt.Println("Connection failed: ", err)
		return
	}
	defer conn.Close()

	fmt.Println("Sender connected! Receiving file...")

	//3. Create destination file on disk
	dstFile, err := os.Create("received_output.bin")
	if err != nil{
		fmt.Println("Error creating file: ", err)
		return
	}
	defer dstFile.Close()

	//4. Stream bytes from TCP socket directly into the file buffer
	//io.Copy automatically handles chunking in 32KB buffers under the hood
	bytesCopied, err := io.Copy(dstFile, conn)
	if err != nil{
		fmt.Println("Transfer error: ", err)
		return
	}
	fmt.Printf("Success! Received %d bytes and saved to received_output.bin\n", bytesCopied)
}