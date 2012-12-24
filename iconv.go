//
// iconv.go
//
package iconv

// #cgo LDFLAGS: -liconv
// #include <iconv.h>
// #include <errno.h>
// #include <stdlib.h>
import "C"

import (
	"bytes"
	"io"
	"syscall"
	"unsafe"
)

const EILSEQ = syscall.Errno(C.EILSEQ)
const E2BIG = syscall.Errno(C.E2BIG)

const DefaultBufSize = 512

type Iconv struct {
	Handle C.iconv_t
}

func Open(tocode string, fromcode string) (cd Iconv, err error) {
	ctocode := C.CString(tocode)
	cfromcode := C.CString(fromcode)
	defer func() {
		C.free(unsafe.Pointer(ctocode))
		C.free(unsafe.Pointer(cfromcode))
	}()
	ret, err := C.iconv_open(ctocode, cfromcode)
	if err != nil {
		return
	}
	cd = Iconv{ret}
	return
}

func (cd Iconv) Close() error {
	_, err := C.iconv_close(cd.Handle)
	return err
}

func (cd Iconv) Conv(b []byte, outbuf []byte) (out []byte, inleft int, err error) {

	outn, inleft, err := cd.Do(b, len(b), outbuf)
	out = outbuf[:outn]
	if err == nil && err != E2BIG {
		return
	}

	w := bytes.NewBuffer(nil)
	w.Write(out)

	inleft, err = cd.DoWrite(w, b[len(b)-inleft:], inleft, outbuf)
	out = w.Bytes()
	return
}

func (cd Iconv) ConvString(s string) string {
	var outbuf [DefaultBufSize]byte
	s1, _, _ := cd.Conv([]byte(s), outbuf[:])
	return string(s1)
}

func (cd Iconv) Do(inbuf []byte, in int, outbuf []byte) (out, inleft int, err error) {

	if in == 0 {
		return
	}

	inbytes := C.size_t(in)
	inptr := &inbuf[0]

	outbytes := C.size_t(len(outbuf))
	outptr := &outbuf[0]
	_, err = C.iconv(cd.Handle,
		(**C.char)(unsafe.Pointer(&inptr)), &inbytes,
		(**C.char)(unsafe.Pointer(&outptr)), &outbytes)

	out = len(outbuf) - int(outbytes)
	inleft = int(inbytes)
	return
}

func (cd Iconv) DoWrite(w io.Writer, inbuf []byte, in int, outbuf []byte) (inleft int, err error) {

	if in == 0 {
		return
	}

	inbytes := C.size_t(in)
	inptr := &inbuf[0]

	for inbytes > 0 {
		outbytes := C.size_t(len(outbuf))
		outptr := &outbuf[0]
		_, err = C.iconv(cd.Handle,
			(**C.char)(unsafe.Pointer(&inptr)), &inbytes,
			(**C.char)(unsafe.Pointer(&outptr)), &outbytes)
		w.Write(outbuf[:len(outbuf)-int(outbytes)])
		if err != nil && err != E2BIG {
			return int(inbytes), err
		}
	}

	return 0, nil
}
