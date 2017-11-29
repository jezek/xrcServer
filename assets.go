package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

//go:generate go run embeed/dir2goTar.go -v assetsTarBytes ./assets

//sets application assets directory to assetsDir and parses templates.
//if assetsDir is empty, then a tmporary directory is created and used as application assets directory.
//the tmporary directory is filled with assetes embeded in binary via "go generate" command before building from sorce.
//functions returns a function for cleaning up template dir and error
//if not nil error is returned, then running clean up is not neccessary
func (app *application) parseAssets(assetsDir string) (func(), error) {
	noCleanUpFunction := func() {
		log.Printf("nothing to clean up")
	}

	var localCleanUp func()
	defer func() {
		//log.Printf("local clean up")
		if localCleanUp != nil {
			localCleanUp()
		} else {
			//log.Printf("nothing to clean up localy")
		}
	}()

	if app.assets != "" {
		log.Printf("assets dir is allready setted: %s", app.assets)
		return noCleanUpFunction, nil
	}

	if assetsDir == "" {
		if len(assetsTarBytes) == 0 {
			return noCleanUpFunction, fmt.Errorf("no assets tar bytes. use \"go generate\" to embed assets into binary, or run xrcServer with -asets flag")
		}

		//create
		assets, err := ioutil.TempDir(os.TempDir(), "xrcServer_assets_")
		if err != nil {
			return noCleanUpFunction, err
		}
		localCleanUp = func() {
			log.Printf("cleaning up temporary assets directory")
			os.RemoveAll(assets)
		}
		log.Printf("temporary assets directory created: %s", assets)
		app.assets = assets

		tarBuffer := bytes.NewReader(assetsTarBytes)
		tarReader := tar.NewReader(tarBuffer)

		//extract
		for {
			header, err := tarReader.Next()
			if err != nil {
				if err == io.EOF {
					//end of tar
					break
				}
				return noCleanUpFunction, err
			}

			path := filepath.Join(assets, header.Name)
			info := header.FileInfo()
			if info.IsDir() {
				if err = os.MkdirAll(path, info.Mode()); err != nil {
					return noCleanUpFunction, err
				}
				continue
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return noCleanUpFunction, err
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return noCleanUpFunction, err
			}
		}
	} else {
		app.assets = assetsDir
	}

	//parse & set templates
	{
		template, err := template.ParseFiles(filepath.Join(app.assets, "index.tmpl"))
		if err != nil {
			return noCleanUpFunction, err
		}
		app.homeTemplate = template
	}

	{
		template, err := template.ParseFiles(filepath.Join(app.assets, "pair.tmpl"))
		if err != nil {
			return noCleanUpFunction, err
		}
		app.pairTemplate = template
	}

	if localCleanUp == nil {
		return noCleanUpFunction, nil
	}

	cleanUpFunction := localCleanUp
	localCleanUp = nil
	return cleanUpFunction, nil
}

func (app *application) cleanAssets() {
	if app.assets == "" {
		return
	}
	log.Printf("assets dir removed: %s", app.assets)
}
