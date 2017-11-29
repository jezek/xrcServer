package main

import (
	"archive/tar"
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type encoder func(src []byte) ([]string, string)

func rawBytesHexEncode(src []byte) ([]string, string) {
	return nil, fmt.Sprintf("%#v", src)
}

func rawBytesDecEncode(src []byte) ([]string, string) {
	return nil, "[]byte{" + strings.Join(strings.Split(strings.Trim(fmt.Sprintf("%v", src), "[]"), " "), ",") + "}"
}

func base16Encode(src []byte) ([]string, string) {
	return []string{"encoding/hex"}, `func() []byte {
		b, _ := hex.DecodeString("` + hex.EncodeToString(src) + `")
		return b
	}()`
}

var coders map[string]encoder = map[string]encoder{
	"rawBytesHex": rawBytesHexEncode,
	"rawBytesDec": rawBytesDecEncode,
	"base16":      base16Encode,
}

func main() {
	packageName := ""
	variableName := ""
	variableEncoding := ""
	debug := false
	flag.StringVar(&packageName, "p", "main", "`pagkage` name in generated file")
	flag.StringVar(&variableName, "v", "tarBytes", "variable name in generated file")
	flag.StringVar(&variableEncoding, "e", "rawBytesHex", "variable encoding")
	flag.BoolVar(&debug, "debug", false, "debug mode. Result will be writen to stdout instead of file")
	flag.Parse()

	encodeFunc := coders[variableEncoding]
	if encodeFunc == nil {
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
	log.Printf("packing %s into %s as %s package to %s variable using %s encoding", dir, file, packageName, variableName, variableEncoding)

	tpl, err := template.New("goTar").Parse(goTarTemplate)
	if err != nil {
		log.Fatalf("template parse error: %v", err)
	}

	tarBuffer := &bytes.Buffer{}
	outBuffers := []io.Writer{tarBuffer}

	if !debug {
		tarFile, err := os.Create(fileBase + ".tar")
		if err != nil {
			log.Printf("tar file create error: %v", err)
		} else {
			defer tarFile.Close()
			outBuffers = append(outBuffers, tarFile)
		}
	}

	Tar(dirAbs, outBuffers...)

	imports, encoded := encodeFunc(tarBuffer.Bytes())

	//if debug {
	//	log.Printf("encoded: %s", encoded)
	//}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, map[string]interface{}{
		"packageName":  packageName,
		"imports":      imports,
		"variableName": variableName,
		"arguments":    os.Args[1:],
		"time":         time.Now(),
		"workingDir":   dirAbs,
		"dataString":   encoded,
	}); err != nil {
		log.Fatalf("template execute error: %v", err)
	}

	fileContent, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("resulting template formating error: %v", err)
	}

	if debug == false {
		if err := ioutil.WriteFile(file, fileContent, 0644); err != nil {
			log.Fatalf("file write error: %v", err)
		}
		log.Printf("file succesfully generated")
	} else {
		fmt.Print(string(fileContent))
	}

}

//from: https://medium.com/@skdomino/taring-untaring-files-in-go-6b07cf56bc07
//Tar takes a source and variable writers and walks 'source' writing each file
//found to the tar writer; the purpose for accepting multiple writers is to allow
//for multiple outputs (for example a file, or md5 hash)
func Tar(src string, writers ...io.Writer) error {

	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return err
	}

	mw := io.MultiWriter(writers...)

	tw := tar.NewWriter(mw)
	defer tw.Close()

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		//log.Printf("hello walk: %s", file)
		//defer log.Printf("bye walk: %s", file)

		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		if header.Name == "" {
			return nil
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		//log.Printf("writen header: %s", header.Name)

		if !fi.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		defer f.Close()
		if err != nil {
			return err
		}

		n, err := io.Copy(tw, f)
		if err != nil {
			return err
		}
		log.Printf("tar added: %s (%d bytes)", header.Name, n)

		return nil
	})
}

const goTarTemplate = `
package {{.packageName}}

{{if .imports}}
import (
	{{range .imports}}
	"{{.}}"
	{{end}}
)
{{end}}

//this file was generated
//by dir2goTar on {{.time}}
//in {{.workingDir}} using arguments: {{.arguments}}

var {{.variableName}} = {{.dataString}}`
