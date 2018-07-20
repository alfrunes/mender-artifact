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

// Change default minimum buffer allocation size to 512 bytes.
#define XD3_ALLOCSIZE 512

#include <stdlib.h>
#include <string.h>
#include <config.h>
#include <xdelta3.h>
#include <xdelta3.c>

// Wrap C macros (constants) into struct for access in go
struct xd3_constants {
    int allocSize;
    // config flag adler32 checksum
    int adler32;
    // stream flag flush
    int flush;
    // xd3_decode_input return values of interest //
    // XD3_INPUT: decode process require more input.
    int input;
    // XD3_OUTPUT: decode process has more input.
    int output;
    // XD3_GETSRCBLK: notification returned to initiate non-blocking source read.
    int getsrcblk;
    // XD3_GOTHEADER: notification returned following the VCDIFF header
    //                and first window header.
    int gotheader;
    // XD3_WINSTART: general notification returned for each window.
    int winstart;
    // XD3_WINFINISH: general notification following the compleate input and output
    //                of a window.
    int winfinish;
};
// make the macros available in go
struct xd3_constants xd3_const = {XD3_ALLOCSIZE, XD3_ADLER32,
                                  XD3_FLUSH, XD3_INPUT,
                                  XD3_OUTPUT, XD3_GETSRCBLK,
                                  XD3_GOTHEADER, XD3_WINSTART,
                                  XD3_WINFINISH};

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
#cgo CFLAGS: -I../vendor/xdelta3-3.0.11/
*/
import "C"
import (
	"errors"
	"fmt"
	"io"
	"unsafe"
)

/**
+-----------------------------------+
|artifact.mender:                   | (io.ReadCloser - device.go (image))
|...  file1 = core-image.patch      |==inFile=====++
+-----------------------------------+             ||
+-----------------------------------+             || srcFile.doPatch(inFile) -> outFile
|/dev/active-part  (ro-rootfs)      |==srcFile====++=========++
+-----------------------------------+ (io.Writer - delta.go) ||
+-----------------------------------+                        VV       +------------------+
|/dev/inactive-part                 |==outFile==========++===========>|updated partition |
+-----------------------------------+ (os.File - blockdevice)         +------------------+
*/

type DeltaDecoding struct {
	// srcFile : old file
	srcFile io.ReadSeeker
	// inFile  : patch file
	inFile io.Reader
	// outFile : patched file
	outFile io.Writer
	// buffer used by xdelta algorithm
	inBuf []byte
}

func NewDeltaDecoding (srcFile io.ReadSeeker, inFile io.Reader, outFile io.Writer, buf []byte) *DeltaDecoding {
    return &DeltaDecoding{
        srcFile: srcFile,
        inFile: inFile,
        outFile: outFile,
        inBuf: buf,
    }
}


// TODO: Decide whether size info should be part of patch
//       header or part of mender artifact meta data.

func (d *DeltaDecoding) Decode() error {
    return d.encodeDecode(C.codeFunc(C.xd3_decode_input))
}

func (d *DeltaDecoding) Encode() error {
    return d.encodeDecode(C.codeFunc(C.xd3_encode_input))
}

// TODO: Make a workaround to actually copy buffers instead
//       of constantly using C.CBytes which uses malloc and
//       hence need to be manually freed.
func (d *DeltaDecoding) encodeDecode(XD3_CODE C.codeFunc) error {

	stream := C.getStream()
	source := C.getSource()
	config := C.getConfig()

	if d.srcFile == nil || d.inFile == nil || d.outFile == nil {
		return errors.New("[DeltaDecoding]: Calling decode without configuring streams {source|patch|out}.")
	}

	if len(d.inBuf) == 0 {
		d.inBuf = make([]byte, int(C.xd3_const.allocSize))
	} else if len(d.inBuf) < int(C.xd3_const.allocSize) {
		return errors.New("XDelta3 buffer too small.")
	}

	// temporary buffer to emulate C-type fread
	buf := make([]byte, len(d.inBuf))

	// Configure source and stream
	C.memset(unsafe.Pointer(stream), 0, C.sizeof_xd3_stream)
	C.memset(unsafe.Pointer(source), 0, C.sizeof_xd3_stream)

	C.xd3_init_config(config, C.xd3_const.adler32)
	config.winsize = C.uint(len(d.inBuf))
	C.xd3_config_stream(stream, config)

	source.blksize = C.uint(len(d.inBuf))

	// Load first block from stream
	onblk, err := d.srcFile.Read(buf)
	if err != nil {
		return err
	}
	source.onblk = C.uint(onblk)
	source.curblkno = 0
	source.curblk = (*C.uchar)(C.CBytes(buf))

	C.xd3_set_source(stream, source)

	var ret C.int
	var Cbuf unsafe.Pointer
	for {
		// get a new chunk of patch file
		bytesRead, err := d.inFile.Read(d.inBuf)
		if bytesRead < len(d.inBuf) {
			C.xd3_set_flags(stream, C.xd3_const.flush|stream.flags)
		}
		Cbuf = C.CBytes(d.inBuf)
		C.xd3_avail_input(stream, (*C.uchar)(Cbuf), C.uint(bytesRead))
		for ret = C._xd3_code(XD3_CODE, stream); ret != C.xd3_const.input;
            ret = C._xd3_code(XD3_CODE, stream) {
			switch ret {
			case C.xd3_const.output:
				bytesWritten, err := d.outFile.Write(C.GoBytes(unsafe.Pointer(stream.next_out),
					C.int(stream.avail_out)))
				if err != nil {
					C.free(unsafe.Pointer(source.curblk))
					C.free(Cbuf)
					return err
				} else if bytesWritten != int(stream.avail_out) {
					C.free(unsafe.Pointer(source.curblk))
					C.free(Cbuf)
					return errors.New("Wrote an unexpected amount through xdelta stream, ABORTING.")
				}
				C.xd3_consume_output(stream)

			case C.xd3_const.getsrcblk:
				_, err = d.srcFile.Seek(int64(source.blksize)*int64(source.getblkno), io.SeekStart)
				C.free(unsafe.Pointer(source.curblk))
				if bytesRead, err = d.srcFile.Read(buf); err != nil {
					C.free(unsafe.Pointer(source.curblk))
					C.free(Cbuf)
					return err
				}
				source.curblk = (*C.uchar)(C.CBytes(buf))
				source.onblk = C.uint(bytesRead)
				source.curblkno = C.ulong(source.getblkno)

			case C.xd3_const.gotheader:
			case C.xd3_const.winstart:
			case C.xd3_const.winfinish:

			default:
				C.free(unsafe.Pointer(source.curblk))
				C.free(Cbuf)
				stream_msg := C.GoString(stream.msg)
				e := errors.New(fmt.Sprintf("Xdelta error: %s [exit code: %d].", stream_msg, int(ret)))
				return e
			}
		}
		C.free(Cbuf)
		if bytesRead != len(d.inBuf) {
			break
		}
	}
	C.free(unsafe.Pointer(source.curblk))
	C.xd3_close_stream(stream)
	C.xd3_free_stream(stream)

	return nil
}

