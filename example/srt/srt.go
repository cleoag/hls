package srt

import (
	"fmt"
	"log"

	"github.com/haivision/srtgo"
	"github.com/nareix/joy4/format/ts"
)

type Server struct {
	Host string
	Port int

	HandlePublish func(*Conn)
	HandleConn    func(*Conn)
}

type Conn struct {
	srtSocket  *srtgo.SrtSocket
	Dmx        *ts.Demuxer
	publishing bool
}

func NewConn(srtSocket *srtgo.SrtSocket) *Conn {
	conn := &Conn{}
	conn.srtSocket = srtSocket
	conn.Dmx = ts.NewDemuxer(srtSocket)
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

	srtServer := srtgo.NewSrtSocket(self.Host, uint16(self.Port), map[string]string{})

	// Start listening for incoming connections
	err := srtServer.Listen(1)
	if err != nil {
		log.Fatalf("Failed to listen on SRT socket: %v", err)
	}
	defer srtServer.Close()

	// Accept an incoming SRT connection

	//srt.dmx = ts.NewDemuxer(srtConn)

	for {
		srtConn, _, err := srtServer.Accept()
		if err != nil {
			log.Fatalf("Failed to accept SRT connection: %v", err)
		}
		defer srtConn.Close()

		demuxer := ts.NewDemuxer(srtConn)
		for {
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
		/*
			conn := NewConn(srtConn)
			go func() {
				for {
					pkt, err := conn.Dmx.ReadPacket()
					if err != nil {
						fmt.Println("srt: server: err:", err)
						break
					}
					// print packet type
					fmt.Println("srt: server: pkt:", pkt.IsKeyFrame, pkt.Idx, pkt.Time)
					// print packet data
					fmt.Println("srt: server: pkt:", pkt.Data)

				}
				//err := self.handleConn(conn)
				fmt.Println("srt: server: client closed err:", err)
			}()
		*/
	}

}

func (self *Conn) Close() {
	self.srtSocket.Close()
}
