package main

import (
	"archive/tar"
	"bytes"
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

type encoder func(src []byte) string

func rawBytesHexEncode(src []byte) string {
	return fmt.Sprintf("%#v", src)
}

var coders map[string]encoder = map[string]encoder{
	"rawBytesHex": rawBytesHexEncode,
}

func main() {
	packageName := ""
	variableName := ""
	variableEncoding := ""
	flag.StringVar(&packageName, "p", "main", "`pagkage` name in generated file")
	flag.StringVar(&variableName, "v", "tarBytes", "variable name in generated file")
	flag.StringVar(&variableEncoding, "e", "rawBytesHex", "variable encoding")
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
	log.Printf("packing %s into %s as %s package to %s variable", dir, file, packageName, variableName)

	tpl, err := template.New("goTar").Parse(goTarTemplate)
	if err != nil {
		log.Fatalf("template parse error: %v", err)
	}

	tarBuffer := &bytes.Buffer{}
	tarFile, err := os.Create(fileBase + ".tar")
	if err != nil {
		log.Fatalf("tar file create error: %v", err)
	}
	defer tarFile.Close()

	//wg := &sync.WaitGroup{}
	//wg.Add(1)
	//go func() {
	//	Tar(dirAbs, tarBuffer, tarFile)
	//	wg.Done()
	//}()
	Tar(dirAbs, tarBuffer, tarFile)

	encoded := encodeFunc(tarBuffer.Bytes())

	//wg.Wait()

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, map[string]interface{}{
		"packageName":  packageName,
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

	if err := ioutil.WriteFile(file, fileContent, 0644); err != nil {
		log.Fatalf("file write error: %v", err)
	}
	log.Printf("file succesfully generated")

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

//this file was generated
//by dir2goTar on {{.time}}
//in {{.workingDir}} using arguments: {{.arguments}}

var {{.variableName}} = {{.dataString}}`
