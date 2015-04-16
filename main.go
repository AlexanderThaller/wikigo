package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/AlexanderThaller/logger"
	"github.com/juju/errgo"
	"github.com/julienschmidt/httprouter"
)

const (
	PagesFolder = "pages"
	Binding     = ":12522"
	Name        = "wikgo"
)

func init() {
	logger.SetLevel(".", logger.Trace)
}

func main() {
	l := logger.New(Name, "main")

	router := httprouter.New()
	router.GET("/", rootHandler)
	router.GET("/pages/*path", pagesHandler)

	l.Notice("Listening on ", Binding)
	err := http.ListenAndServe(Binding, router)
	if err != nil {
		l.Alert(errgo.Notef(err, "can not listen on binding"))
		os.Exit(1)
	}
}

func rootHandler(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	http.Redirect(wr, re, "/pages/", 301)
}

func pagesHandlerIndex(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	path := "index.adoc"
	fmt.Fprintf(wr, "Path: "+path)
}

func pagesHandler(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	l := logger.New(Name, "pagesHandler")

	path := "./" + re.URL.Path
	stat, err := os.Stat(path)
	if err != nil {
		printerr(l, wr, errgo.Notef(err, "can not stat path"))
		return
	}

	if stat.Mode().IsDir() {
		pagesHandlerDirectory(wr, re, ps)
		return
	}

	if stat.Mode().IsRegular() {
		pagesHandlerFile(wr, re, ps)
		return
	}

	printerr(l, wr, errgo.New("this is not a directory and not a regular file!"))
}

func pagesHandlerDirectory(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	l := logger.New(Name, "pagesHandlerDirectory")

	path := "./" + re.URL.Path
	files, err := ioutil.ReadDir(path)
	if err != nil {
		printerr(l, wr, errgo.Notef(err, "can not read from directory"))
		return
	}

	fmt.Fprintf(wr, `<!DOCTYPE html>
  <html lang="en">
  <head>
  <meta charset="utf-8">
  <title>title</title>
  </head>
  <body>`)
	for _, file := range files {
		filepath := re.URL.Path + "/" + file.Name()
		fmt.Fprintf(wr, "<a href="+filepath+">"+file.Name()+"</a>")
		fmt.Fprintf(wr, "<br>\n")
	}
	fmt.Fprintf(wr, `</body>
  </html>`)
}

func pagesHandlerFile(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	l := logger.New(Name, "pagesHandlerDirectory")

	path := "./" + re.URL.Path
	file, err := os.Open(path)
	if err != nil {
		printerr(l, wr, errgo.Notef(err, "can not open file for reading"))
		return
	}
	defer file.Close()

	l.Trace("Filepath Extention: ", filepath.Ext(path))
	switch filepath.Ext(path) {
	case ".asciidoc":
		err = AsciiDoctor(file, wr)
		if err != nil {
			printerr(l, wr, errgo.Notef(err, "can not format file with asciidoctor"))
			return
		}

	default:
		_, err = io.Copy(wr, file)
		if err != nil {
			printerr(l, wr, errgo.Notef(err, "can not copy file to response writer"))
			return
		}
	}
}

func printerr(l logger.Logger, wr http.ResponseWriter, err error) {
	l.Error(errgo.Details(err))
	fmt.Fprintf(wr, errgo.Details(err))

	return
}

func AsciiDoctor(reader io.Reader, writer io.Writer) error {
	stderr := new(bytes.Buffer)

	command := exec.Command("asciidoctor", "-")
	command.Stdin = reader
	command.Stdout = writer
	command.Stderr = stderr

	err := command.Run()
	if err != nil {
		return errgo.Notef(errgo.Notef(err, "can not run asciidoctor"),
			stderr.String())
	}

	return nil
}
