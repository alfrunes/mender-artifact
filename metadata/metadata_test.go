// Copyright 2016 Mender Software AS
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

package metadata

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateInfo(t *testing.T) {
	var validateTests = []struct {
		in  Info
		err error
	}{
		{Info{Format: "", Version: 0}, ErrValidatingData},
		{Info{Format: "", Version: 1}, ErrValidatingData},
		{Info{Format: "format"}, ErrValidatingData},
		{Info{}, ErrValidatingData},
		{Info{Format: "format", Version: 1}, nil},
	}

	for _, tt := range validateTests {
		e := tt.in.Validate()
		assert.Equal(t, e, tt.err)
	}
}

func TestValidateHeaderInfo(t *testing.T) {
	var validateTests = []struct {
		in  HeaderInfo
		err error
	}{
		{HeaderInfo{}, ErrValidatingData},
		{HeaderInfo{Updates: []UpdateType{}}, ErrValidatingData},
		{HeaderInfo{Updates: []UpdateType{{Type: ""}}}, ErrValidatingData},
		{HeaderInfo{Updates: []UpdateType{{Type: "update"}, {}}}, ErrValidatingData},
		{HeaderInfo{Updates: []UpdateType{{}, {Type: "update"}}}, ErrValidatingData},
		{HeaderInfo{Updates: []UpdateType{{Type: "update"}, {Type: ""}}}, ErrValidatingData},
		{HeaderInfo{Updates: []UpdateType{{Type: "update"}}}, nil},
		{HeaderInfo{Updates: []UpdateType{{Type: "update"}, {Type: "update"}}}, nil},
	}
	for idx, tt := range validateTests {
		e := tt.in.Validate()
		assert.Equal(t, e, tt.err, "failing test: %v (%v)", idx, tt)
	}
}

func TestValidateTypeInfo(t *testing.T) {
	var validateTests = []struct {
		in  TypeInfo
		err error
	}{
		{TypeInfo{}, ErrValidatingData},
		{TypeInfo{Rootfs: ""}, ErrValidatingData},
		{TypeInfo{Rootfs: "image-type"}, nil},
	}

	for _, tt := range validateTests {
		e := tt.in.Validate()
		assert.Equal(t, e, tt.err)
	}
}

func TestValidateMetadata(t *testing.T) {
	var validateTests = []struct {
		in  Metadata
		err error
	}{
		{Metadata{}, ErrValidatingData},
		{Metadata{make(map[string]interface{})}, ErrValidatingData},
		{Metadata{map[string]interface{}{}}, ErrValidatingData},
		{Metadata{map[string]interface{}{"": nil}}, ErrValidatingData},
		{Metadata{map[string]interface{}{"key": nil}}, ErrValidatingData},
		{Metadata{map[string]interface{}{"key": "val"}}, ErrValidatingData},
		{Metadata{map[string]interface{}{"DeviceType": "type"}}, ErrValidatingData},
		{Metadata{map[string]interface{}{"DeviceType": nil, "ImageID": "image"}}, ErrValidatingData},
		{Metadata{map[string]interface{}{"DeviceType": "device", "ImageID": "image"}}, nil},
		{Metadata{map[string]interface{}{"DeviceType": "device", "ImageID": "image", "Data": "data"}}, nil},
	}

	for _, tt := range validateTests {
		e := tt.in.Validate()
		assert.Equal(t, e, tt.err, "failing test: %v", tt)
	}
}

func TestValidateFiles(t *testing.T) {
	var validateTests = []struct {
		in  Files
		err error
	}{
		{Files{}, ErrValidatingData},
		{Files{Files: []File{}}, ErrValidatingData},
		{Files{Files: []File{{File: ""}}}, ErrValidatingData},
		{Files{Files: []File{{File: "file"}}}, nil},
		{Files{Files: []File{{File: "file"}, {}}}, ErrValidatingData},
		{Files{Files: []File{{File: "file"}, {File: "file_next"}}}, nil},
	}
	for idx, tt := range validateTests {
		e := tt.in.Validate()
		assert.Equal(t, e, tt.err, "failing test: %v (%v)", idx, tt)
	}
}

func MakeFakeUpdateDir(updateDir string, elements []DirEntry) error {
	for _, elem := range elements {
		if elem.IsDir {
			if err := os.MkdirAll(path.Join(updateDir, elem.Path), os.ModeDir|os.ModePerm); err != nil {
				return err
			}
		} else {
			if _, err := os.Create(path.Join(updateDir, elem.Path)); err != nil {
				return err
			}
		}
	}
	return nil
}

var dirStructOK = []DirEntry{
	{Path: "files", IsDir: false},
	{Path: "type-info", IsDir: false},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true},
	{Path: "scripts/pre", IsDir: true},
	{Path: "scripts/post", IsDir: true},
	{Path: "scripts/check", IsDir: true},
}

var dirStructMultipleUpdates = []DirEntry{
	{Path: "files", IsDir: false},
	{Path: "type-info", IsDir: false},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "checksums/image_next.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "signatures/image_next.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
	{Path: "scripts/post", IsDir: true, Required: false},
	{Path: "scripts/check", IsDir: true, Required: false},
}

var dirStructOKHaveScripts = []DirEntry{
	{Path: "files", IsDir: false},
	{Path: "type-info", IsDir: false},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
	{Path: "scripts/pre/0000_install.sh", IsDir: false, Required: false},
	{Path: "scripts/pre/0001_install.sh", IsDir: false, Required: false},
	{Path: "scripts/post", IsDir: true, Required: false},
	{Path: "scripts/check", IsDir: true, Required: false},
}

var dirStructTypeError = []DirEntry{
	{Path: "files", IsDir: false},
	// type-info should be a file
	{Path: "type-info", IsDir: true},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
	{Path: "scripts/post", IsDir: true, Required: false},
	{Path: "scripts/check", IsDir: true, Required: false},
}

var dirStructInvalidContent = []DirEntry{
	// can not contain unsupported elements
	{Path: "not-supported", IsDir: true, Required: true},
	{Path: "files", IsDir: false},
	{Path: "type-info", IsDir: false},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
	{Path: "scripts/post", IsDir: true, Required: false},
	{Path: "scripts/check", IsDir: true, Required: false},
}

var dirStructInvalidNestedDirs = []DirEntry{
	{Path: "files", IsDir: false},
	{Path: "type-info", IsDir: false},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
	{Path: "scripts/post", IsDir: true, Required: false},
	{Path: "scripts/check", IsDir: true, Required: false},
	{Path: "scripts/unsupported_dir", IsDir: true, Required: true},
}

var dirStructMissingRequired = []DirEntry{
	{Path: "files", IsDir: false},
	// does not contain meta-data and type-info
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
	{Path: "scripts/post", IsDir: true, Required: false},
	{Path: "scripts/check", IsDir: true, Required: false},
}

var dirStructMissingOptional = []DirEntry{
	{Path: "files", IsDir: false},
	{Path: "type-info", IsDir: false},
	{Path: "meta-data", IsDir: false},
	{Path: "checksums", IsDir: true},
	{Path: "checksums/image.sha", IsDir: false},
	{Path: "signatures", IsDir: true},
	{Path: "signatures/image.sig", IsDir: false},
	{Path: "scripts", IsDir: true, Required: false},
	{Path: "scripts/pre", IsDir: true, Required: false},
}

var testMetadataHeaderFormat = map[string]DirEntry{
	// while calling filepath.Walk() `.` (root) directory is included
	// when iterating throug entries in the tree
	".":               {Path: ".", IsDir: true, Required: false},
	"files":           {Path: "files", IsDir: false, Required: false},
	"meta-data":       {Path: "meta-data", IsDir: false, Required: true},
	"type-info":       {Path: "type-info", IsDir: false, Required: true},
	"checksums":       {Path: "checksums", IsDir: true, Required: false},
	"checksums/*":     {Path: "checksums", IsDir: false, Required: false},
	"signatures":      {Path: "signatures", IsDir: true, Required: true},
	"signatures/*":    {Path: "signatures", IsDir: false, Required: true},
	"scripts":         {Path: "scripts", IsDir: true, Required: false},
	"scripts/pre":     {Path: "scripts/pre", IsDir: true, Required: false},
	"scripts/pre/*":   {Path: "scripts/pre", IsDir: false, Required: false},
	"scripts/post":    {Path: "scripts/post", IsDir: true, Required: false},
	"scripts/post/*":  {Path: "scripts/post", IsDir: false, Required: false},
	"scripts/check":   {Path: "scripts/check", IsDir: true, Required: false},
	"scripts/check/*": {Path: "scripts/check/*", IsDir: false, Required: false},
}

func TestDirectoryStructure(t *testing.T) {
	var validateTests = []struct {
		dirContent []DirEntry
		err        error
	}{
		{dirStructOK, nil},
		{dirStructMultipleUpdates, nil},
		{dirStructOKHaveScripts, nil},
		{dirStructTypeError, ErrInvalidMetadataElemType},
		{dirStructInvalidContent, ErrUnsupportedElement},
		{dirStructInvalidNestedDirs, ErrUnsupportedElement},
		{dirStructMissingRequired, ErrMissingMetadataElem},
		{dirStructMissingOptional, nil},
	}

	for _, tt := range validateTests {
		updateTestDir, _ := ioutil.TempDir("", "update")
		defer os.RemoveAll(updateTestDir)
		err := MakeFakeUpdateDir(updateTestDir, tt.dirContent)
		assert.NoError(t, err)

		header := ArtifactHeader{Artifacts: testMetadataHeaderFormat}

		err = header.CheckHeaderStructure(updateTestDir)
		assert.Equal(t, tt.err, err)
	}
}

func TestDirectoryStructureNotExist(t *testing.T) {
	header := ArtifactHeader{Artifacts: testMetadataHeaderFormat}
	err := header.CheckHeaderStructure("non-existing-directory")
	assert.Equal(t, os.ErrNotExist, err)
}