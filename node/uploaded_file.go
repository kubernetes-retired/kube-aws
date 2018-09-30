package node

import (
	"encoding/base64"
	"fmt"

	"github.com/kubernetes-incubator/kube-aws/gzipcompressor"
)

type UploadedFile struct {
	Path    string
	Content UploadedFileContent
}

type UploadedFileContent struct {
	bytes []byte
}

func NewUploadedFileContent(bytes []byte) UploadedFileContent {
	return UploadedFileContent{
		bytes: bytes,
	}
}

func (c UploadedFileContent) ToBase64() UploadedFileContent {
	bytes := []byte(base64.StdEncoding.EncodeToString(c.bytes))
	return UploadedFileContent{
		bytes: bytes,
	}
}

func (c UploadedFileContent) ToGzip() UploadedFileContent {
	bytes, err := gzipcompressor.BytesToBytes(c.bytes)
	if err != nil {
		panic(fmt.Errorf("Unexpected error in ToGzip: %v", err))
	}
	return UploadedFileContent{
		bytes: bytes,
	}
}

func (c UploadedFileContent) String() string {
	return string(c.bytes)
}
