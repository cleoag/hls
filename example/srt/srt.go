package srt

import (
	"fmt"
	"io"
	"log"

	"github.com/haivision/srtgo"
)

type ByteSliceReadSeeker struct {
	data []byte
	pos  int64
}

func (b *ByteSliceReadSeeker) Read(p []byte) (n int, err error) {
	if b.pos >= int64(len(b.data)) {
		return 0, io.EOF
	}

	n = copy(p, b.data[b.pos:])
	b.pos += int64(n)
	return
}

func (b *ByteSliceReadSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return 0, fmt.Errorf("negative position")
		}
		b.pos = offset
	case io.SeekCurrent:
		newPos := b.pos + offset
		if newPos < 0 {
			return 0, fmt.Errorf("negative position")
		}
		b.pos = newPos
	case io.SeekEnd:
		newPos := int64(len(b.data)) + offset
		if newPos < 0 {
			return 0, fmt.Errorf("negative position")
		}
		b.pos = newPos
	default:
		return 0, fmt.Errorf("invalid whence value")
	}

	return b.pos, nil
}

type Server struct {
	Host string
	Port int
}

type Conn struct {
	srtSocket *srtgo.SrtSocket
	r         *ByteSliceReadSeeker
}

func NewConn(srtSocket *srtgo.SrtSocket) *Conn {
	conn := &Conn{}
	conn.srtSocket = srtSocket
	conn.r = &ByteSliceReadSeeker{}
	return conn
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

	conn := NewConn(srtConn)
	conn.r.data = make([]byte, 300000) // adjust buffer size as needed
	tmp := make([]byte, 20500)
	isEOF := false
	//go func() {
	// loop until EOF
	pos := 0
	for i := 0; i < 10; i++ {
		n, err := srtConn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Println("srt connection read error:", err)
			} else {
				fmt.Println("srt connection EOF:", err)
			}
			isEOF = true
			break
		}
		copy(conn.r.data[pos:], tmp[:n])
		pos += n
		log.Println("srt: server: read:", n, "bytes", "total:", len(conn.r.data), "pos:", pos)
	}
	//}()

	demuxer := NewDemuxer(conn.r)

	//read stream info
	for isEOF == false {

		streams, err := demuxer.Streams()
		if err != nil {
			fmt.Println("read streams:", err)
			break
		}
		if len(streams) > 0 {
			fmt.Println("srt: server: streams:", streams)
			break
		}
	}

	/*
		for {
			if isEOF {
				break
			}
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

	*/

	return nil
}

func (self *Conn) Close() {
	self.srtSocket.Close()
}
