package userdatatemplate

import (
	"github.com/coreos/kube-aws/filereader/texttemplate"
	"github.com/coreos/kube-aws/gzipcompressor"
)

func GetString(filename string, data interface{}, compress bool) (string, error) {
	buf, err := texttemplate.GetBytesBuffer(filename, data)

	if err != nil {
		return "", err
	}

	if compress {
		return gzipcompressor.CompressData(buf.Bytes())
	}

	return buf.String(), nil
}
