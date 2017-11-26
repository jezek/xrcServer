package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

type coder interface {
	encode(src io.Reader, dst io.Writer) (nSrc, nDst int, err error)
	decodeFuncString(variableName string) string
}

type nestedError struct {
	error
	parent error
}

func (e nestedError) Error() string {
	return fmt.Sprintf("%s, %s", e.parent.Error(), e.error.Error())
}

type bytesDecCoder struct {
	bufferLen int
}

func (c bytesDecCoder) decodeFuncString(variableName string) string {
	return "//bytesDec coder\nreturn " + variableName
}

func (c bytesDecCoder) encode(src io.Reader, dst io.Writer) (nSrc, nDst int, err error) {
	log.Printf("hello encode")
	defer func() {
		log.Printf("bye encode")
	}()

	header, footer := "[]byte {", "}"
	nr, nw := 0, 0

	nw, err = dst.Write([]byte(header))
	nDst += nw
	if err != nil {
		return
	}
	defer func() {
		var e error
		nw, e = dst.Write([]byte(footer))
		nDst += nw
		if e != nil {
			if err == nil {
				err = e
			} else {
				err = nestedError{e, err}
			}
		}
	}()

	if c.bufferLen == 0 {
		c.bufferLen = 1024
	}
	b := make([]byte, c.bufferLen)
	for {
		b = b[:cap(b)]
		nr, err = src.Read(b)
		nSrc += nr
		b := b[:nr]
		if err != nil {
			if err == io.EOF {
				err = nil
				if len(b) > 0 {
					s := ""
					sep := ","
					for i, bv := range b {
						if i == len(b)-1 {
							sep = ""
						}
						s += fmt.Sprintf("%d%s", bv, sep)
					}
					nw, err = dst.Write([]byte(s))
					nDst += nw
				}
			} else {
				if len(b) > 0 {
					var e error
					s := ""
					sep := ","
					for i, bv := range b {
						if i == len(b)-1 {
							sep = ""
						}
						s += fmt.Sprintf("%d%s", bv, sep)
					}
					nw, e = dst.Write([]byte(s))
					nDst += nw
					if e != nil {
						err = nestedError{e, err}
					}
				}
			}
			return
		} else {
			if len(b) > 0 {
				s := ""
				for _, bv := range b {
					s += fmt.Sprintf("%d,", bv)
				}
				nw, err = dst.Write([]byte(s))
				nDst += nw
				if err != nil {
					return
				}
			}
		}
	}
}

var coders map[string]coder = map[string]coder{
	"bytesDec": bytesDecCoder{},
}

func main() {
	packageName := ""
	variableName := ""
	variableEncoding := ""
	funcName := ""
	flag.StringVar(&packageName, "p", "main", "`pagkage` name in generated file")
	flag.StringVar(&variableName, "v", "tarBytes", "variable name in generated file")
	flag.StringVar(&variableEncoding, "e", "bytesDec", "variable encoding")
	flag.StringVar(&funcName, "f", "TarBytes", "function name in generated file")
	flag.Parse()

	coder := coders[variableEncoding]
	if coder == nil {
		log.Fatalf("unknown encoding: %s", variableEncoding)
	}

	dir := ""
	if flag.NArg() == 1 {
		dir = flag.Arg(0)
		stat, err := os.Stat(dir)
		if err != nil {
			log.Fatalf("can not get dir stat: %v", err)
		}

		if !stat.IsDir() {
			log.Fatalf("argument is not a directory: %s", dir)
		}
	} else if flag.NArg() > 1 {
		log.Fatalf("want exactly 0 or 1 argument, got %d: %v", flag.NArg(), flag.Args())
	}

	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("can not get absolute path: %v", err)
	}

	fileBase := filepath.Base(dirAbs)
	if fileBase == string(filepath.Separator) {
		fileBase = "root"
	}

	file := fileBase + ".tar.go"
	log.Printf("packing %s into %s as %s package to %s variable", dir, file, packageName, variableName)

	tpl, err := template.New("goTar").Parse(goTarTemplate)
	if err != nil {
		log.Fatalf("template parse error: %v", err)
	}

	data := []byte{0, 1, 2, 63, 64, 65, 126, 127, 128, 191, 192, 193, 253, 254, 255}
	dataEncodedBuffer := &bytes.Buffer{}

	if r, w, err := coder.encode(bytes.NewReader(data), dataEncodedBuffer); err != nil {
		log.Fatalf("data encoding error: %v (read: %d, write: %d)", err.Error(), r, w)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, map[string]interface{}{
		"packageName":  packageName,
		"variableName": variableName,
		"funcName":     funcName,
		"funcBody":     coder.decodeFuncString(variableName),
		"arguments":    os.Args[1:],
		"time":         time.Now(),
		"workingDir":   dirAbs,
		"dataString":   string(dataEncodedBuffer.Bytes()),
	}); err != nil {
		log.Fatalf("template execute error: %v", err)
	}

	fileContent, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("resulting template formating error: %v", err)
	}
	log.Printf("generated:\n%s\n", string(fileContent))

}

const goTarTemplate = `
package {{.packageName}}

//this file was generated
//by dir2goTar on {{.time}}
//in {{.workingDir}} using arguments: {{.arguments}}

var {{.variableName}} = {{.dataString}}

func {{.funcName}}() []byte {
	{{.funcBody}}
}
`
