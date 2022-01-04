package smftool

import (
	"bytes"
	"errors"
	"io"
)

func SwapTrack(dst io.Writer, src io.Reader, a, b int) error {
	buf, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	smf, err := Decode(bytes.NewReader(buf))
	if err != nil {
		return err
	}

	h := smf.Header
	tracks := smf.Tracks

	if (a <= 0 && int(h.NumTrack) <= a) || (b <= 0 && int(h.NumTrack) <= b) {
		return errors.New("smftool: invalid a or b")
	}

	tstarts := make([]int, len(tracks))
	tends := make([]int, len(tracks))
	tstarts[0] = HeaderLength
	tends[0] = tstarts[0] + TrackHeaderLength + int(tracks[0].Header.Length)
	for i := 1; i < len(tracks); i++ {
		tstarts[i] = tends[i-1]
		tends[i] = tstarts[i] + TrackHeaderLength + int(tracks[i].Header.Length)
	}

	// swap a and b
	tstarts[a], tstarts[b] = tstarts[b], tstarts[a]
	tends[a], tends[b] = tends[b], tends[a]

	dst.Write(buf[:HeaderLength])
	for i := range tracks {
		dst.Write(buf[tstarts[i]:tends[i]])
	}

	return nil
}
