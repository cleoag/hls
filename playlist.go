package hls

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"eaglesong.dev/hls/internal/segment"
)

const maxFutureMSN = 3

// lock-free snapshot of HLS state for readers
type hlsState struct {
	segments []segment.Cursor
	playlist []byte
	etag     string
	first    segment.MSN
	complete segment.PartMSN
	parser   segment.Parser
}

// Get a segment by MSN. Returns the zero value if it isn't available.
func (s hlsState) Get(msn segment.MSN) (c segment.Cursor, ok bool) {
	if msn > s.complete.MSN+maxFutureMSN {
		// too far in the future
		return segment.Cursor{}, false
	}
	idx := int(msn - s.first)
	if idx < 0 {
		// expired
		return segment.Cursor{}, false
	} else if idx >= len(s.segments) {
		// ready soon
		return segment.Cursor{}, true
	}
	// ready now
	return s.segments[idx], true
}

// publish a lock-free snapshot of segments and playlist
func (p *Publisher) snapshot(initialDur time.Duration) {
	if initialDur == 0 {
		initialDur = p.targetDuration()
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "#EXTM3U\n#EXT-X-VERSION:9\n#EXT-X-TARGETDURATION:%d\n", int(initialDur.Seconds()))
	fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", p.baseMSN)
	if p.baseDCN != 0 {
		fmt.Fprintf(&b, "#EXT-X-DISCONTINUITY-SEQUENCE:%d\n", p.baseDCN)
	}
	fragLen := p.FragmentLength
	if fragLen == 0 {
		fragLen = defaultFragmentLength
	}
	if fragLen < 0 {
		fmt.Fprintf(&b, "#EXT-X-SERVER-CONTROL:CAN-BLOCK-RELOAD=YES\n")
	} else {
		fmt.Fprintf(&b, "#EXT-X-SERVER-CONTROL:HOLD-BACK=%f,PART-HOLD-BACK=1,CAN-BLOCK-RELOAD=YES\n", 1.5*initialDur.Seconds())
		fmt.Fprintf(&b, "#EXT-X-PART-INF:PART-TARGET=%f\n", fragLen.Seconds())
	}
	b.WriteString("#EXT-X-MAP:URI=\"init.mp4\"\n")
	cursors := make([]segment.Cursor, len(p.segments))
	completeIndex := -1
	completeParts := -1
	for i, seg := range p.segments {
		cursors[i] = seg.Cursor()
		if seg.Final() {
			completeIndex = i
		} else if i == completeIndex+1 {
			completeParts = seg.Parts()
		}
		includeParts := fragLen > 0 && i >= len(p.segments)-3
		seg.Format(&b, includeParts)
	}
	digest := fnv.New128a()
	digest.Write(b.Bytes())
	p.state.Store(hlsState{
		segments: cursors,
		playlist: b.Bytes(),
		etag:     fmt.Sprintf("\"%x\"", digest.Sum(nil)),
		parser:   p.names.Parser(),
		first:    p.baseMSN,
		complete: segment.PartMSN{
			MSN:  p.baseMSN + segment.MSN(completeIndex),
			Part: completeParts,
		},
	})
	p.notifySegment()
}

func (p *Publisher) servePlaylist(rw http.ResponseWriter, req *http.Request, state hlsState) {
	want, err := parseBlock(req.URL.Query())
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	} else if want.MSN > state.complete.MSN+maxFutureMSN {
		http.Error(rw, "_HLS_msn is in the distant future", 400)
		return
	}
	if !state.complete.Satisfies(want) {
		state = p.waitForSegment(req.Context(), want)
		if len(state.segments) == 0 {
			// timeout or stream disappeared
			http.NotFound(rw, req)
			return
		}
	}
	rw.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	rw.Header().Set("Etag", state.etag)
	http.ServeContent(rw, req, "", time.Time{}, bytes.NewReader(state.playlist))
}

type (
	subscriber chan struct{}
	subMap     map[subscriber]struct{}
)

// block until segment with the given number is ready or ctx is cancelled
func (p *Publisher) waitForSegment(ctx context.Context, want segment.PartMSN) hlsState {
	ctx, cancel := context.WithTimeout(ctx, 35*time.Second)
	defer cancel()
	// subscribe to segment updates
	ch := make(subscriber, 1)
	p.subsMu.Lock()
	if p.subs == nil {
		p.subs = make(subMap)
	}
	p.subs[ch] = struct{}{}
	p.subsMu.Unlock()
	defer func() {
		// unsubscribe
		p.subsMu.Lock()
		delete(p.subs, ch)
		p.subsMu.Unlock()
	}()
	for {
		state, ok := p.state.Load().(hlsState)
		if !ok {
			return hlsState{}
		}
		if state.complete.Satisfies(want) {
			return state
		}
		// wait for notify or for timeout/disconnect
		select {
		case <-ch:
		case <-ctx.Done():
			return hlsState{}
		}
	}
}

// notify subscribers that segment is ready
func (p *Publisher) notifySegment() {
	p.subsMu.Lock()
	defer p.subsMu.Unlock()
	for sub := range p.subs {
		// non-blocking send
		select {
		case sub <- struct{}{}:
		default:
		}
	}
}

func parseBlock(q url.Values) (want segment.PartMSN, err error) {
	want = segment.PartMSN{MSN: -1, Part: -1}
	v := q.Get("_HLS_msn")
	if v == "" {
		// not blocking
		return
	}
	vv, err := strconv.ParseInt(v, 10, 64)
	if err != nil || vv < 0 {
		return want, errors.New("invalid _HLS_msn")
	}
	want.MSN = segment.MSN(vv)
	v = q.Get("_HLS_part")
	if v == "" {
		// block for whole segment
		return
	}
	vv, err = strconv.ParseInt(v, 10, 64)
	if err != nil || vv < 0 {
		return want, errors.New("invalid _HLS_part")
	}
	// block for part
	want.Part = int(vv)
	return
}
