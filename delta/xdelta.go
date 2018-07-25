// Copyright 2018 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package delta

/*
// CGO comment
// This comment is treated as c-code.

// Set window size to 4kB total memory usage approx. 8~12kB.
#define XD3_ALLOCSIZE 4096
#define XD3_BUILD_FAST 1

#include <stdlib.h>
#include <string.h>
#include <config.h>
#include <xdelta3.h>
#include <xdelta3.c>


#ifdef HAVE_LZMA_H_
#include <xdelta3-lzma.h>
#endif
// These structures has to reside in the C code in order
// for the cgo checks to pass.
static xd3_stream stream;
static xd3_source source;
static xd3_config config;
static inline xd3_stream *getStream () {return &stream;}
static inline xd3_source *getSource () {return &source;}
static inline xd3_config *getConfig () {return &config;}

// Wrap C function pointers to make them accessible in Go.
// from xdelta.h :
// int xd3_decode_input (xd3_stream *);
// int xd3_encode_input (xd3_stream *);
typedef int (*codeFunc) (xd3_stream *);
// encode / decode callback
int _xd3_code(codeFunc f, xd3_stream * stream) { return f(stream); }

// Add lzma compression library if tag is defined.
#cgo lzma LDFLAGS: -llzma
#cgo CFLAGS: -I../vendor/xdelta3-3.0.11
*/
import "C"
import (
	"io"
	"unsafe"

	"github.com/pkg/errors"
)

const ALLOCSIZE = C.XD3_ALLOCSIZE

/**
+-----------------------------------+
|artifact.mender:                   | (io.ReadCloser - device.go (image))
|...  file1 = core-image.patch      |==inFile=====++
+-----------------------------------+             ||
+-----------------------------------+             || srcFile.doPatch(inFile) -> outFile
|/dev/active-part  (ro-rootfs)      |==srcFile====++===========++
+-----------------------------------+ (blockdevice - delta.go) ||
+-----------------------------------+                          VV     +------------------+
|/dev/inactive-part                 |==outFile=================++====>|updated partition |
+-----------------------------------+ (blockdevice)                   +------------------+
*/

type DeltaDecoding struct {
	// srcFile : old file
	srcFile io.ReadSeeker
	// inFile  : patch file
	inFile io.Reader
	// outFile : patched file
	outFile io.Writer
	// buffer for the xdelta input stream
	// NOTE: Xdelta actually allocates another buffer of
	//       same dimension for the stream as well, thus
	//       the actual memory requirements are the double
	//       of this provided buffer.
	inBuf []byte
}

func NewDeltaDecoding(srcFile io.ReadSeeker, inFile io.Reader, outFile io.Writer, buf []byte) *DeltaDecoding {
	return &DeltaDecoding{
		srcFile: srcFile,
		inFile:  inFile,
		outFile: outFile,
		inBuf:   buf,
	}
}

// Decodes source and patch (in) to an updated revision (out).
func (d *DeltaDecoding) Decode() error {
	return d.encodeDecode(C.codeFunc(C.xd3_decode_input))
}

// Encodes source and updated revision (in) to a VCDIFF patch (out).
func (d *DeltaDecoding) Encode() error {
	return d.encodeDecode(C.codeFunc(C.xd3_encode_input))
}

// encodeDecode is a general function to encode / decode a byte stream
// to / from a patch / updated revision respectively. The actual encoding
// and decoding happens in xd3_encode_input and xd3_decode_input functions
// respectively, and the preparations of the streams are completely the same
// only the meaning of "in" and "out" is swapped.
func (d *DeltaDecoding) encodeDecode(XD3_CODE C.codeFunc) error {

	stream := C.getStream()
	source := C.getSource()
	config := C.getConfig()

	if d.srcFile == nil || d.inFile == nil || d.outFile == nil {
		return errors.New("[DeltaDecoding]: Calling decode without configuring streams {source|patch|out}.")
	}

	// Setup buffers and initiate c interface
	if len(d.inBuf) == 0 {
		d.inBuf = make([]byte, int(C.XD3_ALLOCSIZE))
	} else if len(d.inBuf) < int(C.XD3_ALLOCSIZE) {
		return errors.New("XDelta3 buffer too small.")
	}

	// temporary buffer for source
	buf := make([]byte, len(d.inBuf))

	// Configure source and stream
	C.memset(unsafe.Pointer(stream), 0, C.sizeof_xd3_stream)
	C.memset(unsafe.Pointer(source), 0, C.sizeof_xd3_stream)

	C.xd3_init_config(config, C.XD3_ADLER32)
	config.winsize = C.uint(len(d.inBuf))
	C.xd3_config_stream(stream, config)
	source.curblk = (*C.uchar)(unsafe.Pointer(&buf[0]))
	source.blksize = C.uint(len(buf))

	// Load first block from stream
	onblk, err := d.srcFile.Read(buf)
	if err != nil {
		C.xd3_free_stream(stream)
		return errors.Wrapf(err, "Xdelta: error reading from source file")
	}
	source.onblk = C.uint(onblk)
	source.curblkno = 0

	C.xd3_set_source(stream, source)

	var ret C.int
	for {
		// get a new chunk of patch file
		bytesRead, err := d.inFile.Read(d.inBuf)
		if bytesRead < len(d.inBuf) {
			C.xd3_set_flags(stream, C.XD3_FLUSH|stream.flags)
		}
		C.xd3_avail_input(stream, (*C.uchar)(unsafe.Pointer(&d.inBuf[0])), C.uint(bytesRead))
		for ret = C._xd3_code(XD3_CODE, stream); ret != C.XD3_INPUT; ret = C._xd3_code(XD3_CODE, stream) {
			switch ret {
			case C.XD3_OUTPUT:
				bytesWritten, err := d.outFile.Write(C.GoBytes(unsafe.Pointer(stream.next_out),
					                                C.int(stream.avail_out)))
				if err != nil {
					C.xd3_close_stream(stream)
					C.xd3_free_stream(stream)
					return err
				} else if bytesWritten != int(stream.avail_out) {
					C.xd3_close_stream(stream)
					C.xd3_free_stream(stream)
					return errors.New("Wrote an unexpected amount through xdelta stream, ABORTING.")
				}
				C.xd3_consume_output(stream)

			case C.XD3_GETSRCBLK:
				_, err = d.srcFile.Seek(int64(source.blksize)*int64(source.getblkno), io.SeekStart)
				if err != nil {
					C.xd3_close_stream(stream)
					C.xd3_free_stream(stream)
					return errors.Wrapf(err, "Xdelta: error seeking in source file")
				}
				if bytesRead, err = d.srcFile.Read(buf); err != nil {
					C.xd3_close_stream(stream)
					C.xd3_free_stream(stream)
					return errors.Wrapf(err, "Xdelta: error reading from source file")
				}
				source.curblk = (*C.uchar)(unsafe.Pointer(&buf[0]))
				source.onblk = C.uint(bytesRead)
				source.curblkno = C.ulong(source.getblkno)

			case C.XD3_GOTHEADER:
			case C.XD3_WINSTART:
			case C.XD3_WINFINISH:

			case C.XD3_INVALID_INPUT:
				C.xd3_close_stream(stream)
				C.xd3_free_stream(stream)
				return errors.Errorf("Xdelta error: %s [exit code: %d]. Possibly a source-patch mismatch.",
					C.GoString(stream.msg), int(ret))
			default:
				C.xd3_close_stream(stream)
				C.xd3_free_stream(stream)
				return errors.Errorf("Xdelta error: %s [exit code: %d].",
					C.GoString(stream.msg), int(ret))
			}
		}
		// do {...} while (bytesRead != len(d.inBuf))
		if bytesRead != len(d.inBuf) {
			break
		}
	}

	C.xd3_close_stream(stream)
	C.xd3_free_stream(stream)

	return nil
}
