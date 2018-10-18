package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

var directory = "/tmp/default"
var CORS = false

const ChunkSize = 65536


func check(e error) {
	if e != nil {
		panic(e)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fileName := path.Join(directory, path.Base(r.URL.Path))
	if CORS {
		(w).Header().Set("Access-Control-Allow-Origin", "*")
		(w).Header().Set("Access-Control-Allow-Methods", "GET, PUT")
		(w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	}

	switch r.Method {

	case http.MethodPut:
		if _, err := os.Stat(fileName); !os.IsNotExist(err) {
			os.Remove(fileName)
			fmt.Printf("%s already existed, was deleted\n", fileName)
		}

		f, err := os.Create(fileName)
		check(err)
		isOpen, err := os.Create(fileName + "_is_open")
		isOpen.Close()
		defer f.Close()
		defer os.Remove(fileName + "_is_open")
		_, err = io.Copy(f, r.Body)
		if err != nil {
			return
		}

	case http.MethodGet:

		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			fmt.Printf("%s not found, cannot GET it\n", fileName)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		isOpenPath := fileName + "_is_open"

		if _, err := os.Stat(isOpenPath); os.IsNotExist(err) {
			http.ServeFile(w, r, fileName)
		} else {
			f, err := os.Open(fileName)
			check(err)
			b1 := make([]byte, ChunkSize)
			done := false
			for done != true {
				hasClosed := false
				if _, err := os.Stat(isOpenPath); os.IsNotExist(err) {
					hasClosed = true
				}
				readLength, err := f.Read(b1)
				if err != io.EOF {
					check(err)
				}
				if readLength != ChunkSize && hasClosed {
					done = true
				}
				if readLength > 0 {
					_, err = w.Write(b1[0:readLength])
					check(err)
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					} else {
						log.Println("Damn, no flush")
					}
				}
			}
		}
		return

	default:
		fmt.Printf("%s\n", "A method other than Get or Put was attempted, not supported")
	}
	return
}

func main() {

	directory = *flag.String("d", "/tmp", "The local path to and from which files will be served")
	port := flag.Int("p", 8080, "The port number of the server")
	CORSptr := flag.Bool("CORS", false, "add CORS headers to allow full local file access")
	flag.Parse()
	CORS = *CORSptr

	m := http.NewServeMux()
	s := http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           m,
		ReadHeaderTimeout: 3 * time.Second,
		//ReadTimeout:       60 * time.Second,
		//WriteTimeout:      60 * time.Second,
		ErrorLog:          nil,
	}

	fmt.Printf("Http server started, listening on port %d\n", *port)
	fmt.Printf("Serving files out of %s\n", directory)

	m.HandleFunc("/", handler)
	log.Fatal(s.ListenAndServe())
}
