ffmpeg -re -i /home/den/Videos/video_in.mp4 -seg_duration 1 -movflags frag_keyframe+empty_moov+default_base_moof -f mp4 'srt://127.0.0.1:12345?pkt_size=1316'
