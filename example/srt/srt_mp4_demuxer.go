package srt

import (
	"fmt"
	"github.com/cleoag/hls/example/srt/mp4"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format/mp4/mp4io"
	"io"
)

type Demuxer struct {
	r         io.ReadSeeker
	streams   []*Stream
	movieAtom *mp4io.Movie
}

func NewDemuxer(r io.ReadSeeker) *Demuxer {
	return &Demuxer{
		r: r,
	}
}
func (self *Demuxer) Streams() (streams []av.CodecData, err error) {
	if err = self.probe(); err != nil {
		return
	}
	for _, stream := range self.streams {
		streams = append(streams, stream.CodecData)
	}
	return
}
func (self *Demuxer) probe() (err error) {
	if self.movieAtom != nil {
		return
	}

	var moov *mp4io.Movie
	var atoms []mp4io.Atom

	if atoms, err = mp4.ReadMOOVAtom(self.r); err != nil {
		return
	}
	if _, err = self.r.Seek(0, 0); err != nil {
		return
	}

	for _, atom := range atoms {
		if atom.Tag() == mp4io.MOOV {
			moov = atom.(*mp4io.Movie)
		}
	}

	if moov == nil {
		err = fmt.Errorf("mp4: 'moov' atom not found")
		return
	}

	self.streams = []*Stream{}
	for i, atrack := range moov.Tracks {
		stream := &Stream{
			trackAtom: atrack,
			demuxer:   self,
			idx:       i,
		}
		if atrack.Media != nil && atrack.Media.Info != nil && atrack.Media.Info.Sample != nil {
			stream.sample = atrack.Media.Info.Sample
			stream.timeScale = int64(atrack.Media.Header.TimeScale)
		} else {
			err = fmt.Errorf("mp4: sample table not found")
			return
		}

		if avc1 := atrack.GetAVC1Conf(); avc1 != nil {
			if stream.CodecData, err = h264parser.NewCodecDataFromAVCDecoderConfRecord(avc1.Data); err != nil {
				return
			}
			self.streams = append(self.streams, stream)
		} else if esds := atrack.GetElemStreamDesc(); esds != nil {
			if stream.CodecData, err = aacparser.NewCodecDataFromMPEG4AudioConfigBytes(esds.DecConfig); err != nil {
				return
			}
			self.streams = append(self.streams, stream)
		}
	}

	self.movieAtom = moov
	return
}
