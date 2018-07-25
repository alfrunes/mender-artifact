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

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	td, err := ioutil.TempDir("/tmp/", "mender-xdelta")
	assert.NoError(t, err)
	defer os.RemoveAll(td)
	buf := make([]byte, ALLOCSIZE)

	// Open and create  src, in and outFile
	// And prepare for decoding
	oldFile, err := os.OpenFile(path.Join(td, "src"), os.O_CREATE|os.O_RDWR, 0)
	assert.NoError(t, err)
	newFile, err := os.OpenFile(path.Join(td, "in"), os.O_CREATE|os.O_RDWR, 0)
	assert.NoError(t, err)
	patchFile, err := os.OpenFile(path.Join(td, "patch"), os.O_CREATE|os.O_RDWR, 0)
	assert.NoError(t, err)

	// fill in some gibberish
	_, err = oldFile.Write([]byte("foo; this is an old bar"))
	assert.NoError(t, err)
	_, err = newFile.Write([]byte("foo; this is a new, much better, bar"))
	assert.NoError(t, err)
	// reset seek head as if we just opened it
	_, err = oldFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = newFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)

	// generate patch
	dd := NewDeltaDecoding(oldFile, newFile, patchFile, buf)

	err = dd.Encode()
	assert.NoError(t, err)

	outFile, err := os.OpenFile(path.Join(td, "out"), os.O_CREATE|os.O_RDWR, 0)
	assert.NoError(t, err)
	// reset seek head
	_, err = oldFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = patchFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)

	dd.inFile = patchFile
	dd.outFile = outFile

	// Decode generated patch.
	err = dd.Decode()
	assert.NoError(t, err)
	_, err = outFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = newFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	out, err := ioutil.ReadAll(outFile)
	assert.NoError(t, err)
	new, err := ioutil.ReadAll(newFile)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(out, new))

	// Appending to the source file should not change the result
	oldFile.Write([]byte(`dangeling crap`))
	_, err = oldFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = patchFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = outFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)

	err = dd.Decode()
	assert.NoError(t, err)
	_, err = outFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = newFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	out, err = ioutil.ReadAll(outFile)
	assert.NoError(t, err)
	new, err = ioutil.ReadAll(newFile)
	assert.NoError(t, err)
	assert.True(t, bytes.Equal(out, new))

	// Prepending however gives error in the window checksum
	corruptFile, err := os.OpenFile(path.Join(td, "cor"), os.O_CREATE|os.O_RDWR, 0)
	assert.NoError(t, err)
	_, err = corruptFile.Write([]byte(`to foo or not to bar. `))
	assert.NoError(t, err)
	_, err = corruptFile.Write(out)
	assert.NoError(t, err)

	_, err = corruptFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)
	_, err = patchFile.Seek(0, os.SEEK_SET)
	assert.NoError(t, err)

	dd.srcFile = corruptFile

	err = dd.Decode()
	assert.Error(t, err)
}
