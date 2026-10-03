package commands

import (
	"bufio"
	"fmt"
	"io"
	"lever/engine"
	"net"
	"os"
	"strconv"
)

func NewNcCmd() *engine.Command {
	var (
		listenMode bool
		udpMode    bool
		portStr    string
	)
	cmd := &engine.Command{
		Use:   "nc [hostname] [port]",
		Short: "网络连接",
		Long:  `建立任何 TCP/UDP 连接、监听端口、双向传递网络流或通过管道无损传输文件。`,
		Run: func(c *engine.Command, args []string) {
			flags := c.Flags()
			networkType := "tcp"
			if udpMode {
				networkType = "udp"
			}

			if listenMode {
				if portStr == "" && flags.NArg() > 0 {
					portStr = flags.Arg(0)
				}
				if portStr == "" {
					fmt.Fprintln(os.Stderr, "nc 错误: 监听模式下必须指定端口号 (例如: nc -l 8080)")
					os.Exit(1)
				}
				runNcServer(networkType, portStr)
				return
			}

			if flags.NArg() < 2 {
				fmt.Fprintln(os.Stderr, "nc 错误: 必须指定目标主机和端口号 (例如: nc 127.0.0.1 8080)")
				os.Exit(1)
			}
			host := flags.Arg(0)
			port := flags.Arg(1)
			runNcClient(networkType, host, port)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listenMode, "listen", "l", false, "监听模式，开启本地端口等待连接")
	flags.BoolVarP(&udpMode, "udp", "u", false, "切换为 UDP 模式 (默认使用 TCP)")
	flags.StringVarP(&portStr, "port", "p", "", "指定本地或远端的目标端口号")

	return cmd
}

func runNcServer(network, port string) {
	address := ":" + port
	fmt.Fprintf(os.Stderr, "Listening on [%s %s]...\n", network, address)

	if network == "tcp" {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			fmt.Fprintf(os.Stderr, "监听失败: %v\n", err)
			os.Exit(1)
		}
		defer listener.Close()

		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "接受连接失败: %v\n", err)
			return
		}
		defer conn.Close()
		fmt.Fprintf(os.Stderr, "Connection received from %s\n", conn.RemoteAddr().String())

		handleTCPConnection(conn)
	} else {
		portInt, _ := strconv.Atoi(port)
		conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: portInt})
		if err != nil {
			fmt.Fprintf(os.Stderr, "UDP监听失败: %v\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		buffer := make([]byte, 2048)
		for {
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				break
			}
			_, _ = os.Stdout.Write(buffer[:n])
		}
	}
}

func runNcClient(network, host, port string) {
	address := net.JoinHostPort(host, port)
	conn, err := net.Dial(network, address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接到 %s 失败: %v\n", address, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Fprintf(os.Stderr, "Connected to %s successfully.\n", address)
	if network == "tcp" {
		handleTCPConnection(conn)
	} else {
		go func() { _, _ = io.Copy(conn, os.Stdin) }()
		_, _ = io.Copy(os.Stdout, conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	tcpConn, ok := conn.(*net.TCPConn)

	if ok && tcpConn != nil {
		_ = tcpConn.SetReadBuffer(32768)
		_ = tcpConn.SetWriteBuffer(32768)
	}

	done := make(chan struct{})

	go func() {
		bufSrc := make([]byte, 32768)
		_, _ = io.CopyBuffer(conn, os.Stdin, bufSrc)
		if ok && tcpConn != nil {
			_ = tcpConn.CloseWrite()
		}
		close(done)
	}()

	bufferedStdout := bufio.NewWriterSize(os.Stdout, 32768)
	bufDest := make([]byte, 32768)

	_, _ = io.CopyBuffer(bufferedStdout, conn, bufDest)
	_ = bufferedStdout.Flush()
	<-done
}
