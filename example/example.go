package main

import (
	"flag"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/cleoag/hls"
	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/format/rtmp"
	"golang.org/x/sync/errgroup"
)

func main() {

	modePtr := flag.Int("mode", 0, "HLS Mode (0,1,2)")
	fragLenPtr := flag.Int("fraglen", 500, "HLS Fragment Length (ms)")
	bufferLenPtr := flag.Int("bufferlen", 2, "HLS Buffer Length (sec)")
	initialDurationPtr := flag.Int("initialduration", 2, "HLS Initial duration (sec)")
	keepSegmentsPtr := flag.Int("keepsegments", 10, "HLS how many segments to keep")

	flag.Parse()

	pub := &hls.Publisher{Mode: hls.Mode(*modePtr), FragmentLength: time.Duration(*fragLenPtr) * time.Millisecond, BufferLength: time.Duration(*bufferLenPtr) * time.Second, InitialDuration: time.Duration(*initialDurationPtr) * time.Second, KeepSegments: *keepSegmentsPtr}
	rts := &rtmp.Server{Addr: ":1935",
		HandlePublish: func(c *rtmp.Conn) {
			defer c.Close()
			log.Println("publish started from", c.NetConn().RemoteAddr())
			if err := avutil.CopyFile(pub, c); err != nil {
				log.Printf("error: publishing from %s: %+v", c.NetConn().RemoteAddr(), err)
			}
		},
	}
	var eg errgroup.Group
	eg.Go(rts.ListenAndServe)

	http.Handle("/exit/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		pub.Close()
	}))

	http.Handle("/hls/", pub)
	http.Handle("/player.html", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r := strings.NewReader(home)
		http.ServeContent(rw, req, "player.html", time.Time{}, r)
	}))

	http.Handle("/links.html", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r := strings.NewReader(links)
		http.ServeContent(rw, req, "links.html", time.Time{}, r)
	}))

	log.Println("run ffmpeg")
	cmd := exec.Command("ffmpeg",
		"-threads", "2",
		"-re",
		"-fflags", "+genpts",
		"-stream_loop", "-1",
		"-i", "/home/den/encoded.mp4",
		"-c", "copy",
		//"-g", "100",
		//"-c:a", "aac", "-b:a", "128k",
		//"-b:v", "5M", "-c:v", "libx264", "-preset", "fast", "-tune", "zerolatency", "-profile:v", "high", "-level", "4.2",
		//"-filter:a", "'volume=0.1'",
		"-f", "flv",
		"rtmp://127.0.0.1:1935/live",
	)
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	go cmd.Run()
	//if err != nil {
	//	log.Fatalf("cmd.Run() failed with %s\n", err)
	//}

	eg.Go(func() error {
		//return http.ListenAndServeTLS(":8080", "server.crt", "server.key", nil)
		return http.ListenAndServe(":8080", nil)
	})
	log.Println("listening on rtmp://localhost:1935/live and http://localhost:8080/player.html")
	if err := eg.Wait(); err != nil {
		log.Println("error:", err)
	}
	//ffmpeg -threads 2 -re -fflags +genpts -stream_loop -1 -i ./input.mov -c copy -f flv rtmp://127.0.0.1:1935/live
}

const links = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>HLS demo</title>
</head>
<body>
<a href='/hls/index.m3u8'> m3u8</a>
<a href='https://stream.mux.com/v69RSHhFelSm4701snP22dYz2jICy4E4FUyk02rW4gxRM.m3u8'> bunny low latency </a>
</body>
</html>
`

const home = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>HLS demo</title>
<script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
</head>
<body>
<video id="video" muted autoplay controls width="1280" height="720"></video>
<script>
let config = {
 lowLatencyMode: true,
};
let videoSrc = '/hls/index.m3u8';
let video = document.getElementById('video');
 if (Hls.isSupported()) {
    var hls = new Hls(config);
    hls.loadSource(videoSrc);
    hls.attachMedia(video);
  } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
    video.src = videoSrc;
 }


// hls.on(Hls.Events.MANIFEST_PARSED, () => video.play());
</script>
<a href='/exit/'> close stream</a>
</br>
<a href='/hls/index.m3u8'> m3u8</a>
<a href='https://stream.mux.com/v69RSHhFelSm4701snP22dYz2jICy4E4FUyk02rW4gxRM.m3u8'> bunny low latency </a>
</body>
</html>
`
