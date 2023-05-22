package srt

import (
	"bytes"
	"fmt"
	"github.com/nareix/joy4/format/mp4"
	"io"
	"log"

	"github.com/haivision/srtgo"
)

type SrtReader struct {
	buffer bytes.Buffer
	sock   *srtgo.SrtSocket
}

func (s *SrtReader) Read(p []byte) (n int, err error) {
	return s.buffer.Read(p)
}

func (s *SrtReader) Seek(offset int64, whence int) (int64, error) {
	// SRT is a livestreaming protocol and doesn't support seeking
	return 0, nil
}

type Server struct {
	Host string
	Port int

	HandlePublish func(*Conn)
	HandleConn    func(*Conn)
}

type Conn struct {
	srtSocket  *srtgo.SrtSocket
	publishing bool
}

func NewConn(srtSocket *srtgo.SrtSocket) *Conn {
	conn := &Conn{}
	conn.srtSocket = srtSocket
	conn.publishing = true
	return conn
}

func (self *Server) handleConn(conn *Conn) (err error) {
	if self.HandleConn != nil {
		self.HandleConn(conn)
	} else {
		if conn.publishing {
			if self.HandlePublish != nil {
				self.HandlePublish(conn)
			}
		}
	}

	return
}

func (self *Server) ListenAndServe() error {

	options := make(map[string]string)
	options["transtype"] = "live"
	options["payloadsize"] = "1316"

	srtServer := srtgo.NewSrtSocket(self.Host, uint16(self.Port), options)

	// Start listening for incoming connections
	err := srtServer.Listen(1)
	if err != nil {
		log.Fatalf("Failed to listen on SRT socket: %v", err)
	}
	defer srtServer.Close()

	srtConn, _, err := srtServer.Accept()
	if err != nil {
		log.Fatalf("Failed to accept SRT connection: %v", err)
	}
	defer srtConn.Close()

	srtReader := &SrtReader{sock: srtConn}

	tmp := make([]byte, 2048) // adjust buffer size as needed
	//	go func() {
	// loop until EOF
	for i := 0; i < 10; i++ {
		n, err := srtConn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		log.Println("read:", n)
		srtReader.buffer.Write(tmp[:n])
	}
	//	}()

	demuxer := mp4.NewDemuxer(srtReader)

	for {
		// Read a packet
		pkt, err := demuxer.ReadPacket()

		if err != nil {
			fmt.Println("srt: server: err:", err)
			break
		}
		// print packet type
		fmt.Println("srt: server: pkt:", pkt.IsKeyFrame, pkt.Idx, pkt.Time)
		// print packet data
		fmt.Println("srt: server: pkt:", pkt.Data)

	}

	return nil
}

func (self *Conn) Close() {
	self.srtSocket.Close()
}
