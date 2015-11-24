package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/AlexanderThaller/logger"
	"github.com/juju/errgo"
	"github.com/julienschmidt/httprouter"
	"github.com/spf13/viper"
)

const (
	// Name is the name of the application. Used in logging.
	Name = "wikgo"
)

var (
	BuildHash string
	BuildTime string
)

func init() {
	err := configure()
	if err != nil {
		panic(err)
	}

	loglevel, err := logger.ParsePriority(viper.GetString("LogLevel"))
	if err != nil {
		panic(err)
	}

	err = logger.SetLevel(".", loglevel)
	if err != nil {
		panic(err)
	}
}

func main() {
	l := logger.New(Name, "main")
	l.Info("Version: ", fmt.Sprintf("%v-b%v", BuildHash, BuildTime))

	pagesFolder := viper.GetString("PagesFolder")
	binding := viper.GetString("Binding")

	router := httprouter.New()
	router.GET("/", rootHandler)
	router.GET("/"+pagesFolder+"/*path", pagesHandler)

	l.Notice("Listening on ", binding)
	err := http.ListenAndServe(binding, router)
	if err != nil {
		l.Alert(errgo.Notef(err, "can not listen on binding"))
		os.Exit(1)
	}
}

func rootHandler(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	pagesFolder := viper.GetString("PagesFolder")
	http.Redirect(wr, re, "/"+pagesFolder+"/", 301)
}

func pagesHandler(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	l := logger.New(Name, "pagesHandler")
	timestart := time.Now()
	ip, _, _ := net.SplitHostPort(re.RemoteAddr)

	path := path.Clean("./" + re.URL.Path)
	l.Notice("Sending ", path, " to ", ip)

	stat, err := os.Stat(path)
	if err != nil {
		printerr(l, wr, errgo.Notef(err, "can not stat path"))
		return
	}

	if stat.Mode().IsDir() {
		l.Trace("Filetype: Directory")
		pagesHandlerDirectory(wr, re, ps)
	}

	if stat.Mode().IsRegular() {
		l.Trace("Filetype: File")
		pagesHandlerFile(wr, re, ps)
	}

	if !stat.Mode().IsDir() && !stat.Mode().IsRegular() {
		l.Error("Filetype is not a directory and not a regular file. Something is strange.")
	}

	l.Debug("Sent ", path, " (", time.Since(timestart), ")")
}

func pagesHandlerDirectory(wr http.ResponseWriter, re *http.Request, ps httprouter.Params) {
	l := logger.New(Name, "pagesHandlerDirectory")

	urlpath := "./" + re.URL.Path
	files, err := ioutil.ReadDir(urlpath)
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
		url, err := url.Parse(path.Clean(re.URL.Path + "/" + file.Name()))
		if err != nil {
			l.Error(errgo.Notef(err, "can not escape url"))
		}

		fmt.Fprintf(wr, "<a href=%s>%s</a>", url.String(), file.Name())
		fmt.Fprint(wr, "<br>\n")
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
		err = asciiDoctor(file, wr)
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

func asciiDoctor(reader io.Reader, writer io.Writer) error {
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
