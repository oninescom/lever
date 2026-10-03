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
		Short: "Network connections",
		Long:  `Connect or listen over TCP or UDP, exchange data in both directions, and transfer files through pipes.`,
		RunE: func(c *engine.Command, args []string) error {
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
					return fmt.Errorf("nc error: listen mode requires a port (for example: nc -l 8080)")
				}
				return runNcServer(networkType, portStr)
			}

			if flags.NArg() < 2 {
				return fmt.Errorf("nc error: specify a host and port (for example: nc 127.0.0.1 8080)")
			}
			host := flags.Arg(0)
			port := flags.Arg(1)
			return runNcClient(networkType, host, port)
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&listenMode, "listen", "l", false, "Listen on a local port for connections")
	flags.BoolVarP(&udpMode, "udp", "u", false, "Use UDP instead of TCP")
	flags.StringVarP(&portStr, "port", "p", "", "Local or remote port number")

	return cmd
}

func runNcServer(network, port string) error {
	address := ":" + port
	fmt.Fprintf(os.Stderr, "Listening on [%s %s]...\n", network, address)

	if network == "tcp" {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return fmt.Errorf("listen failed: %w", err)
		}
		defer listener.Close()

		conn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept failed: %w", err)
		}
		defer conn.Close()
		fmt.Fprintf(os.Stderr, "Connection received from %s\n", conn.RemoteAddr().String())

		handleTCPConnection(conn)
		return nil
	} else {
		portInt, _ := strconv.Atoi(port)
		conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: portInt})
		if err != nil {
			return fmt.Errorf("UDP listen failed: %w", err)
		}
		defer conn.Close()

		buffer := make([]byte, 2048)
		for {
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return err
			}
			if _, err := os.Stdout.Write(buffer[:n]); err != nil {
				return err
			}
		}
	}
}

func runNcClient(network, host, port string) error {
	address := net.JoinHostPort(host, port)
	conn, err := net.Dial(network, address)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", address, err)
	}
	defer conn.Close()
	fmt.Fprintf(os.Stderr, "Connected to %s successfully.\n", address)
	if network == "tcp" {
		handleTCPConnection(conn)
		return nil
	} else {
		go func() { _, _ = io.Copy(conn, os.Stdin) }()
		_, err = io.Copy(os.Stdout, conn)
		return err
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
