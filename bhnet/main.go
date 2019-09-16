package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
)

type cli struct {
	listen            bool
	port              int
	execute           string
	command           bool
	uploadDestination string
	target            string
	help              bool
}

var inputCli cli

func initCommandLine() {
	var usageStr string
	var defaultVal interface{}

	usageStr = "print help message"
	defaultVal = false
	flag.BoolVar(&inputCli.help, "h", defaultVal.(bool), usageStr)
	flag.BoolVar(&inputCli.help, "help", defaultVal.(bool), usageStr)

	usageStr = "listen on [host]:[port] for incoming connections"
	defaultVal = false
	flag.BoolVar(&inputCli.listen, "l", defaultVal.(bool), usageStr)
	flag.BoolVar(&inputCli.listen, "listen", defaultVal.(bool), usageStr)

	usageStr = "execute the given file upon receiving a connection"
	defaultVal = ""
	flag.StringVar(&inputCli.execute, "e", defaultVal.(string), usageStr)
	flag.StringVar(&inputCli.execute, "execute", defaultVal.(string), usageStr)

	usageStr = "initialize a command shell"
	defaultVal = false
	flag.BoolVar(&inputCli.command, "c", defaultVal.(bool), usageStr)
	flag.BoolVar(&inputCli.command, "command", defaultVal.(bool), usageStr)

	usageStr = "upon receiving connection upload a file and write to [destination]"
	defaultVal = ""
	flag.StringVar(&inputCli.uploadDestination, "u", defaultVal.(string), usageStr)
	flag.StringVar(&inputCli.uploadDestination, "upload", defaultVal.(string), usageStr)

	usageStr = "target host"
	defaultVal = ""
	flag.StringVar(&inputCli.uploadDestination, "t", defaultVal.(string), usageStr)
	flag.StringVar(&inputCli.uploadDestination, "target", defaultVal.(string), usageStr)

	usageStr = "listen port"
	defaultVal = 0
	flag.IntVar(&inputCli.port, "p", defaultVal.(int), usageStr)
	flag.IntVar(&inputCli.port, "port", defaultVal.(int), usageStr)

	flag.Parse()
	if inputCli.help {
		flag.PrintDefaults()
		os.Exit(0)
	}
}

// to be a client
func clientSender() {
	tcpClient, err := net.Dial("tcp", fmt.Sprintf("%s:%d", inputCli.target, inputCli.port))
	if err != nil {
		log.Println(err)
		return
	}
	defer tcpClient.Close()

	for {
		// read data from stdio
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan() // use `for scanner.Scan()` to keep reading
		buffer := scanner.Text()
		// send data to target:port
		fmt.Fprint(tcpClient, buffer)

		// receive from target
		var receivedBuf bytes.Buffer
		reader := bufio.NewReader(tcpClient)
		for {
			readBuffer := make([]byte, 4096)
			n, err := reader.Read(readBuffer)
			if err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				break
			}
			receivedBuf.Write(readBuffer[:n])
			if n < 4096 {
				break
			}
		}
		fmt.Println(receivedBuf.String())
	}
}

// this is for incoming connections
func serverLoop() {
	// listen all ip when target is empty string
	tcpServer, err := net.Listen("tcp", fmt.Sprintf("%s:%d", inputCli.target, inputCli.port))
	if err != nil {
		log.Println(err)
		return
	}
	defer tcpServer.Close()
	for {
		conn, err := tcpServer.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleClient(conn)
	}
}

// handle connections
func handleClient(c net.Conn) {
	// check for upload
	if inputCli.uploadDestination != "" {
		// read from conn
		var buffer bytes.Buffer
		reader := bufio.NewReader(c)
		for {
			readBuffer := make([]byte, 4096)
			n, err := reader.Read(readBuffer)
			if err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				break
			}
			buffer.Write(readBuffer[:n])
			if n < 4096 {
				break
			}
		}
		f, err := os.OpenFile(inputCli.uploadDestination, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
		if err != nil {
			log.Println(err)
			return
		}
		defer f.Close()
		n, err := buffer.WriteTo(f)
		if err != nil {
			log.Println("Failed save to file, ", err)
		} else {
			log.Println("Successfully save to file ", n)
		}
	}

	// check for execute command
	if inputCli.execute != "" {
		output := runCommand(inputCli.execute)
		fmt.Fprint(c, output)
	}

	// initial a command shell
	if inputCli.command {
		for {
			// show a simple prompt
			fmt.Fprint(c, "<BHP:#> ")

			// read command until EOF or '\n'
			// read from conn
			var buffer bytes.Buffer
			reader := bufio.NewReader(c)
			for {
				readBuffer := make([]byte, 4096)
				n, err := reader.Read(readBuffer)
				if err != nil {
					if err != io.EOF {
						// stop the program
						log.Println(err)
						return
					}
					break
				}
				buffer.Write(readBuffer[:n])
				if n < 4096 {
					break
				}
			}

			// run the command and send to output back
			output := runCommand(buffer.String())
			fmt.Fprint(c, output)
		}
	}
}

// run the command as another process
func runCommand(execute string) string {
	log.Println("run command:", execute)
	cmd := exec.Command("cmd.exe", "/C", execute)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {

		return fmt.Sprint(err, ": ", stderr.String())
	}
	return out.String()
}

func main() {
	initCommandLine()
	if !inputCli.listen && inputCli.port > 0 {
		// read stdin and send buffer to target:port
		clientSender()
	}

	if inputCli.listen {
		serverLoop()
	}
}
