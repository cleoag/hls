package mp4

import (
	"github.com/nareix/joy4/format/mp4/mp4io"
	"github.com/nareix/joy4/utils/bits/pio"
	"io"
)

func ReadMOOVAtom(r io.ReadSeeker) (atoms []mp4io.Atom, err error) {
	for {
		offset, _ := r.Seek(0, 1)
		taghdr := make([]byte, 8)
		if _, err = io.ReadFull(r, taghdr); err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		size := pio.U32BE(taghdr[0:])
		tag := mp4io.Tag(pio.U32BE(taghdr[4:]))

		var atom mp4io.Atom

		if tag == mp4io.MOOV {
			atom = &mp4io.Movie{}
			b := make([]byte, int(size))
			if _, err = io.ReadFull(r, b[8:]); err != nil {
				return
			}
			copy(b, taghdr)
			if _, err = atom.Unmarshal(b, int(offset)); err != nil {
				return
			}
			atoms = append(atoms, atom)
			return
		} else {
			if _, err = r.Seek(int64(size)-8, 1); err != nil {
				return
			}
		}
	}
	return
}
func ReadMOOFAtom(r io.ReadSeeker) (atoms []mp4io.Atom, err error) {
	for {
		offset, _ := r.Seek(0, 1)
		taghdr := make([]byte, 8)
		if _, err = io.ReadFull(r, taghdr); err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
		size := pio.U32BE(taghdr[0:])
		tag := mp4io.Tag(pio.U32BE(taghdr[4:]))

		var atom mp4io.Atom

		if tag == mp4io.MOOF {
			atom = &mp4io.MovieFrag{}
			b := make([]byte, int(size))
			if _, err = io.ReadFull(r, b[8:]); err != nil {
				return
			}
			copy(b, taghdr)
			if _, err = atom.Unmarshal(b, int(offset)); err != nil {
				return
			}
			atoms = append(atoms, atom)
			return
		} else {
			if _, err = r.Seek(int64(size)-8, 1); err != nil {
				return
			}
		}
	}
	return
}
