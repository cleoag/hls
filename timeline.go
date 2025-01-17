package hls

import (
	"fmt"
	"time"

	"github.com/cleoag/hls/internal/segment"
)

const (
	defaultInitialDuration = 2 * time.Second
	defaultBufferLength    = 10 * time.Second
)

// start a new segment
func (p *Publisher) newSegment(start time.Duration, programTime time.Time) error {
	if len(p.primary.segments) != 0 {
		// flush and finalize previous segment
		if err := p.flush(); err != nil {
			return err
		}
		for _, track := range p.tracks {
			track.current().Finalize(start)
		}
	}
	initialDur := p.targetDuration()
	nextMSN := p.baseMSN + segment.MSN(len(p.primary.segments))
	for trackID, track := range p.tracks {
		track.frag.NewSegment()
		name := fmt.Sprintf("%d%s%d%s", trackID, p.pid, nextMSN, track.hdr.SegmentExtension)
		//name := fmt.Sprintf("%d%s%d%s", trackID, "dc", nextMSN, track.hdr.SegmentExtension)
		seg, err := segment.New(name, p.WorkDir, track.hdr.SegmentContentType, start, p.nextDCN, programTime)
		if err != nil {
			return err
		}
		//log.Println("---> new segment", name)
		// add the new segment and remove the old
		track.segments = append(track.segments, seg)
	}
	p.trimSegments(initialDur)
	p.snapshot(initialDur)
	p.nextDCN = false
	return nil
}

// calculate the longest segment duration
func (p *Publisher) targetDuration() time.Duration {
	maxTime := p.primary.frag.Duration() // pending segment duration
	for _, seg := range p.primary.segments {
		if dur := seg.Duration(); dur > maxTime {
			maxTime = dur
		}
	}
	maxTime = maxTime.Round(time.Second)
	if maxTime == 0 {
		maxTime = p.InitialDuration
	}
	if maxTime == 0 {
		maxTime = defaultInitialDuration
	}
	return maxTime
}

// remove the oldest segment until the total length is less than configured
func (p *Publisher) trimSegments(segmentLen time.Duration) {
	goalLen := p.BufferLength
	if goalLen == 0 {
		goalLen = defaultBufferLength
	}
	keepSegmentsLen := p.KeepSegments
	keepSegments := int((goalLen+segmentLen-1)/segmentLen + 1)
	if keepSegments <= keepSegmentsLen {
		keepSegments = keepSegmentsLen
	}
	n := len(p.primary.segments) - keepSegments
	if n <= 0 {
		return
	}
	p.baseMSN += segment.MSN(n)
	for _, track := range p.tracks {
		for _, seg := range track.segments[:n] {
			if track == p.primary && seg.Discontinuous() {
				p.baseDCN++
			}
			seg.Release()
		}
		track.segments = track.segments[n:]
	}
}

// make a fragment for every track
func (p *Publisher) flush() error {
	for _, track := range p.tracks {
		f, err := track.frag.Fragment()
		if err != nil {
			return err
		} else if f.Bytes != nil {
			if err := track.current().Append(f); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *track) current() *segment.Segment {
	if len(t.segments) == 0 {
		return nil
	}
	return t.segments[len(t.segments)-1]
}
