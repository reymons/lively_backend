package flv

func decode3BytesBE(b []byte) uint32 {
	if len(b) < 3 {
		panic("size should be at least 3 bytes")
	}

	return uint32(b[0])<<16 |
		uint32(b[1])<<8 |
		uint32(b[2])
}
