package smftool

import "errors"

func variableLengthUint32(b []byte) (uint32, int, error) {
	var prod uint32
	for i := 0; i < 4; i++ {
		if i >= len(b) {
			break
		}

		if (b[i] & 0x80) != 0 {
			prod = (prod | uint32(b[i]&0x7F)) << 7
		} else {
			return prod | uint32(b[i]), i + 1, nil
		}
	}

	return 0, 0, errors.New("smftool: invalid variable length byte count")
}
