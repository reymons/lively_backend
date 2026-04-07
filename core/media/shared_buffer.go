package media

import (
	"io"
	"sync"
	"sync/atomic"
)

type SharedBuffer struct {
	pool *sync.Pool
	buf  []byte
	refs atomic.Int32
	len  int
}

func NewSharedBuffer(pool *sync.Pool, buf []byte, initLen int) *SharedBuffer {
	if initLen > len(buf) {
		panic("initial length is greater than the buffer's length")
	}
	return &SharedBuffer{
		pool: pool,
		buf:  buf,
		len:  initLen,
	}
}

func (b *SharedBuffer) reset() {
	b.len = 0
}

func (b *SharedBuffer) CopySlice() *SharedBuffer {
	bytes := b.Bytes()
	data := make([]byte, len(bytes))
	copy(data, bytes)
	shared := NewSharedBuffer(b.pool, data, len(data))
	return shared
}

func (b *SharedBuffer) Len() int {
	return b.len
}

func (b *SharedBuffer) Write(data []byte) (int, error) {
	end := len(b.buf)
	if len(data) <= end-b.len {
		end = b.len + len(data)
	}
	copy(b.buf[b.len:end], data)

	size := end - b.len
	b.len += size

	if size != len(data) {
		return size, io.ErrShortWrite
	}

	return size, nil
}

func (b *SharedBuffer) ReadN(r io.Reader, toRead int) (int, error) {
	end := b.len + toRead
	if end > len(b.buf) {
		end = len(b.buf)
		toRead = end - b.len
	}

	var nRead int

	for nRead < toRead {
		n, err := r.Read(b.buf[b.len:end])

		if n > 0 {
			b.len += n
			nRead += n

			if nRead == toRead {
				return nRead, nil
			}
		}

		if err == io.EOF && nRead < toRead {
			return nRead, io.ErrUnexpectedEOF
		}

		if err != nil && err != io.EOF {
			return nRead, err
		}
	}

	return nRead, nil
}

func (b *SharedBuffer) Bytes() []byte {
	return b.buf[:b.len]
}

func (b *SharedBuffer) Acquire(n int) {
	b.refs.Add(int32(n))
}

func (b *SharedBuffer) Release() {
	if new := b.refs.Add(-1); new < 0 {
		panic("released more times than acquired")
	} else if new == 0 {
		b.reset()
		b.pool.Put(b)
	}
}
