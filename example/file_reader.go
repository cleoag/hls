package main

import (
	"fmt"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/format"
	"log"
)

func main() {

	format.RegisterAll()

	//file, err := avutil.Open("/home/den/Videos/segmented.mp4")
	file, err := avutil.Open("/home/den/Videos/video_in.mp4")
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}

	streams, err := file.Streams()
	if err != nil {
		log.Fatalf("Error getting streams: %v", err)
	}

	// Create a buffer for AV packets
	avPackets := make([]av.Packet, 0)
	// Create a buffer for stream info
	streamInfo := make([]av.CodecData, 0)

	// Read the file continuously until EOF
	i := 0
	for {
		pkt, err := file.ReadPacket()
		if err != nil {
			break
		}

		// Demux the packet to the buffer
		avPackets = append(avPackets, pkt)
		streamInfo = append(streamInfo, streams[pkt.Idx])
		i++

		// Print the packet info
		fmt.Println("Packet Info:", i, pkt.Idx, pkt.IsKeyFrame, pkt.Time, pkt.CompositionTime, len(pkt.Data))

		// Remove the printed packet from the buffer
		avPackets = avPackets[1:]
	}
	fmt.Println("Total packets:", i)
}
