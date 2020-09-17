/* Adapted from https://github.com/ipsusila/gogo/blob/master/http/formuploader.go */

/*
MIT License
Copyright (c) 2017 I Putu Susila
Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package util

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

//FormUploader represents HTTP multipart form submission.
type FormUploader interface {
	AddField(name, value string) FormUploader
	AddFields(fields map[string]string) FormUploader
	Fields(name string) []string
	AddFiles(fieldName string, basePath string, files ...string) error
	Files() []string
	SetChunkSize(size int64) FormUploader
	ChunkSize() int64
	Post(client *http.Client, targetURL string, headers map[string]string) (*http.Response, error)
	Put(client *http.Client, targetURL string, headers map[string]string) (*http.Response, error)
}

type formPart interface {
	newPart(buf *bytes.Buffer, mpw *multipart.Writer) (int64, error)
	writeTo(chunk []byte, w io.Writer) error
	close() error
}

type fieldPart struct {
	name   string //field name
	value  string //field value
	mpData []byte //multipart data
}

type filePart struct {
	file      *os.File //File handler
	fileSize  int64    //File size
	baseName  string   //Base name (may differs from orignal base-name)
	filePath  string   //Original filename
	fieldName string   //Name of field in multipart content
	mpBegin   []byte   //Beginning of the multipart
}

type endPart struct {
	mpData []byte //multipart data
}

type formUploader struct {
	chunkSize int64
	fields    []*fieldPart
	files     []*filePart
}

//NewFormUploader creates form uploder instance.
func NewFormUploader() FormUploader {
	return &formUploader{chunkSize: 1024 * 10}
}

//Write all the data to writer.
func writeExactly(w io.Writer, data []byte) error {
	if ndata := len(data); ndata > 0 {
		nw := 0
		for nw < ndata {
			n, err := w.Write(data[nw:])
			if err != nil {
				return err
			}
			nw += n
		}
	}
	return nil
}

func (p *fieldPart) newPart(buf *bytes.Buffer, mpw *multipart.Writer) (int64, error) {
	//Create multipart data
	if err := mpw.WriteField(p.name, p.value); err != nil {
		return 0, err
	}

	//Read writen data from buffer
	n := buf.Len()
	if cap(p.mpData) < n {
		p.mpData = make([]byte, n)
	}
	nr, err := buf.Read(p.mpData)
	if err != nil {
		return 0, err
	}
	//correctly assign data len
	p.mpData = p.mpData[:nr]

	return int64(nr), nil
}
func (p *fieldPart) writeTo(chunk []byte, w io.Writer) error {
	//chunk is not used.
	return writeExactly(w, p.mpData)
}
func (p *fieldPart) close() error {
	//do nothing
	return nil
}

func (p *filePart) newPart(buf *bytes.Buffer, mpw *multipart.Writer) (int64, error) {
	//make sure file is closed
	if err := p.close(); err != nil {
		return 0, err
	}

	//open file and get information
	file, err := os.Open(p.filePath)
	if err != nil {
		return 0, err
	}
	fi, err := file.Stat()
	if err != nil {
		return 0, err
	}
	p.file = file
	p.fileSize = fi.Size()

	//Create file part (the content is not writen)
	if _, err := mpw.CreateFormFile(p.fieldName, p.baseName); err != nil {
		return 0, err
	}

	n := buf.Len()
	if cap(p.mpBegin) < n {
		p.mpBegin = make([]byte, n)
	}
	nr, err := buf.Read(p.mpBegin)
	if err != nil {
		return int64(nr), err
	}
	//correctly assign data len
	p.mpBegin = p.mpBegin[:nr]

	return int64(n) + p.fileSize, nil
}
func (p *filePart) writeTo(chunk []byte, w io.Writer) error {
	//write multipart begin
	if err := writeExactly(w, p.mpBegin); err != nil {
		return err
	}

	if nw, err := io.CopyBuffer(w, p.file, chunk); err != nil {
		return err
	} else if nw != p.fileSize {
		//size doesn't match
		return fmt.Errorf("file size (%v) != copy size (%v)", p.fileSize, nw)
	}

	return nil
}
func (p *filePart) close() error {
	if f := p.file; f != nil {
		p.file = nil
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (p *endPart) newPart(buf *bytes.Buffer, mpw *multipart.Writer) (int64, error) {
	//end boundary
	if err := mpw.Close(); err != nil {
		return 0, err
	}

	//Read writen data from buffer
	n := buf.Len()
	if cap(p.mpData) < n {
		p.mpData = make([]byte, n)
	}
	nr, err := buf.Read(p.mpData)
	if err != nil {
		return 0, err
	}
	//correctly assign data len
	p.mpData = p.mpData[:nr]

	return int64(nr), nil

}
func (p *endPart) writeTo(chunk []byte, w io.Writer) error {
	//chunk is not used.
	return writeExactly(w, p.mpData)
}
func (p *endPart) close() error {
	return nil
}

func (fu *formUploader) AddField(name, value string) FormUploader {
	fp := &fieldPart{name: name, value: value}
	fu.fields = append(fu.fields, fp)

	return fu
}
func (fu *formUploader) AddFields(fields map[string]string) FormUploader {
	for key, val := range fields {
		fp := &fieldPart{name: key, value: val}
		fu.fields = append(fu.fields, fp)
	}
	return fu
}
func (fu *formUploader) Fields(name string) []string {
	fields := []string{}
	for _, field := range fu.fields {
		if field.name == name {
			fields = append(fields, field.value)
		}
	}
	return fields
}
func (fu *formUploader) AddFiles(fieldName string, basePath string, files ...string) (err error) {
	basePath, err = filepath.Abs(basePath)
	if err != nil {
		return
	}
	for _, file := range files {
		fp := &filePart{
			fieldName: fieldName,
			filePath:  filepath.Join(basePath, file),
			baseName:  "drop/" + file,
		}
		fu.files = append(fu.files, fp)
	}
	return
}
func (fu *formUploader) Files() []string {
	filePaths := []string{}
	for _, fp := range fu.files {
		filePaths = append(filePaths, fp.filePath)
	}
	return filePaths
}
func (fu *formUploader) SetChunkSize(size int64) FormUploader {
	fu.chunkSize = size
	return fu
}
func (fu *formUploader) ChunkSize() int64 {
	return fu.chunkSize
}
func (fu *formUploader) Post(client *http.Client, targetURL string, headers map[string]string) (*http.Response, error) {
	return fu.submit(client, targetURL, "POST", headers)
}
func (fu *formUploader) Put(client *http.Client, targetURL string, headers map[string]string) (*http.Response, error) {
	return fu.submit(client, targetURL, "PUT", headers)
}
func (fu *formUploader) submit(client *http.Client, targetURL, method string, headers map[string]string) (*http.Response, error) {
	buf := &bytes.Buffer{}
	mpw := multipart.NewWriter(buf)

	//List of form parts: fields, files, end
	parts := []formPart{}
	for _, p := range fu.fields {
		parts = append(parts, p)
	}
	for _, p := range fu.files {
		parts = append(parts, p)
	}
	parts = append(parts, &endPart{})

	//create parts and calculate size.
	totalContentLen := int64(0)
	for _, p := range parts {
		n, err := p.newPart(buf, mpw)
		if err != nil {
			return nil, err
		}
		totalContentLen += n
	}

	//close form parts when done.
	defer func() {
		for _, p := range parts {
			p.close()
		}
	}()

	//Pipe for connecting reader and writer.
	//Reader side will be connected to request,
	//while writer side will be used for writing
	//multipart form content.
	reader, writer := io.Pipe()
	defer reader.Close()

	//Write parts content
	var routineErr error
	go func() {
		defer writer.Close()

		//allocate buffer for reading file.
		chunk := make([]byte, fu.chunkSize)
		for _, p := range parts {
			if err := p.writeTo(chunk, writer); err != nil {
				routineErr = err
				break
			}
		}
	}()

	//construct HTTP client Request with rd
	req, err := http.NewRequest(method, targetURL, reader)
	if err != nil {
		return nil, err
	}

	//user headers
	if len(headers) > 0 {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	//required headers
	req.Header.Set("Content-Type", mpw.FormDataContentType())
	req.ContentLength = totalContentLen

	//process request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	} else if routineErr != nil {
		return nil, routineErr
	}

	return resp, nil
}
