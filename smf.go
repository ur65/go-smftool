package smftool

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	HeaderLength      = 14
	TrackHeaderLength = 8
)

// MsgType is channel message type
type MsgType byte

const (
	// META Event
	MTSequenceNumber    MsgType = 0x00
	MTText              MsgType = 0x01
	MTCopyrightNotice   MsgType = 0x02
	MTSequenceTrackName MsgType = 0x03
	MTInstrumentName    MsgType = 0x04
	MTLyric             MsgType = 0x05
	MTMarker            MsgType = 0x06
	MTCuePoint          MsgType = 0x07
	MTChannelPrefix     MsgType = 0x20
	MTEOT               MsgType = 0x2F
	MTSetTempo          MsgType = 0x51
	MTSMPTEOffset       MsgType = 0x54
	MTTimeSignature     MsgType = 0x58
	MTKeySignature      MsgType = 0x59
	MTSequencerSpecific MsgType = 0x7F

	// SysEx Event
	MTSysEx        MsgType = 0xF0
	MTSpecialSysEx MsgType = 0xF7

	// MIDI Event
	MTNoteOff         MsgType = 0x80
	MTNoteOn          MsgType = 0x90
	MTKeyPressure     MsgType = 0xA0
	MTController      MsgType = 0xB0
	MTProgramChange   MsgType = 0xC0
	MTChannelPressure MsgType = 0xD0
	MTPitchBend       MsgType = 0xE0
)

type Header struct {
	ID       [4]byte
	Size     uint32 // always 00 00 00 06
	Format   uint16
	NumTrack uint16
	Division uint16
}

func readHeader(r io.Reader) (Header, error) {
	h := Header{}

	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return Header{}, err
	}

	if !bytes.Equal(h.ID[:], []byte("MThd")) {
		return Header{}, errors.New("smftool: invalid header: not found 'MThd'")
	}

	if h.Division&0x8000 != 0 {
		return Header{}, fmt.Errorf("smftool: unsupported SMPTE timing (have: 0x%X)", h.Division)
	}

	return h, nil
}

type Event struct {
	Type    MsgType
	Channel byte
	Delta   uint32
	Length  int
	Value   []byte
}

func parseEvent(buf []byte, prevEvent *Event) (*Event, error) {
	b := buf

	delta, n, err := variableLengthUint32(b)
	if err != nil {
		return nil, err
	}
	b = b[n:]

	status := b[0]

	// running status rule
	if status&0x80 == 0 {
		if prevEvent == nil {
			return nil, errors.New("smftool: invalid running status")
		}
		// set previous status
		status = byte(prevEvent.Type) | prevEvent.Channel
	} else {
		b = b[1:]
	}

	// Meta Event
	switch MsgType(status) {
	case 0xFF: // META Event
		etype, b := MsgType(b[0]), b[1:]
		size, n, err := variableLengthUint32(b)
		if err != nil {
			return nil, err
		}
		b = b[n:]
		value, b := b[:size], b[size:]

		e := &Event{
			Type:   etype,
			Delta:  delta,
			Length: len(buf) - len(b),
			Value:  value,
		}
		return e, nil

	case MTSysEx, MTSpecialSysEx: // System Exclusive Event
		size, n, err := variableLengthUint32(b)
		if err != nil {
			return nil, err
		}
		b = b[n:]
		value, b := b[:size], b[size:]

		e := &Event{
			Type:   MsgType(status),
			Delta:  delta,
			Length: len(buf) - len(b),
			Value:  value,
		}
		return e, nil
	}

	// MIDI Event
	etype := MsgType(status & 0xF0)
	channel := status & 0x0F
	var value []byte
	switch etype {
	case MTNoteOff, MTNoteOn, MTKeyPressure, MTController, MTPitchBend:
		value, b = b[:2], b[2:]
	case MTProgramChange, MTChannelPressure:
		value, b = b[:1], b[1:]
	default: // ignore messages
		value, b = b[:1], b[1:]
	}

	e := &Event{
		Type:    etype,
		Channel: channel,
		Delta:   delta,
		Length:  len(buf) - len(b),
		Value:   value,
	}

	return e, nil
}

type TrackHeader struct {
	ID     [4]byte
	Length uint32
}

type Track struct {
	Header TrackHeader
	Events []*Event
}

func readTrack(r io.Reader) (*Track, error) {
	h := TrackHeader{}
	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return nil, err
	}

	if !bytes.Equal(h.ID[:], []byte("MTrk")) {
		return nil, errors.New("smftool: invalid track header: not found 'MTrk'")
	}

	buf := make([]byte, h.Length)
	n, err := io.ReadFull(r, buf)
	if n < len(buf) {
		return nil, errors.New("smftool: invalid track: unexpected end of track")
	}
	if err != nil {
		return nil, err
	}

	var (
		events     = make([]*Event, 0)
		prev       *Event
		ePos, eEnd int = 0, int(h.Length)
	)

	for ePos < eEnd {
		e, err := parseEvent(buf, prev)
		if err != nil {
			return nil, err
		}
		buf = buf[e.Length:]

		ePos += e.Length
		if ePos > eEnd || (e.Type == MTEOT && ePos != eEnd) {
			return nil, errors.New("smftool: invalid track: unexpected end of track")
		}

		events = append(events, e)
		prev = e
	}

	t := &Track{
		Header: h,
		Events: events,
	}

	return t, nil
}

func readTracks(r io.Reader, n int) ([]*Track, error) {
	tracks := make([]*Track, n)
	for i := range tracks {
		t, err := readTrack(r)
		if err != nil {
			return nil, err
		}
		tracks[i] = t
	}
	return tracks, nil
}

type SMF struct {
	Header Header
	Tracks []*Track
}

func Decode(r io.Reader) (*SMF, error) {
	h, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	tracks, err := readTracks(r, int(h.NumTrack))
	if err != nil {
		return nil, err
	}

	s := &SMF{
		Header: h,
		Tracks: tracks,
	}

	return s, nil
}
