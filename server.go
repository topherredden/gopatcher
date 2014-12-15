package main

import "net/http"
import "fmt"
import "flag"
import "encoding/json"
import "./assetpack"
import "path/filepath"
import "github.com/howeyc/fsnotify"

var assetStatJSON string
var watcher *fsnotify.Watcher

func statHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "%s", assetStatJSON)
}

func watcherLoop() {
    for {
        select {
        case ev := <-watcher.Event:
            fmt.Println("Event")
        case err := <-watcher.Error:
            fmt.Println("Error")
        }
    }
}

func main() {
    var dir = flag.String("dir", "files", "Directory to serve files from")
    var port = flag.Int("port", 8080, "Port for HTTP server to listen on")
    flag.Parse()

    // Create Watcher
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        panic(err)
    }

    // Convert path to absolute
    absDir, _ := filepath.Abs(*dir)
    portString := fmt.Sprintf(":%v", *port)

    err = watcher.Watch(absDir)
    if err != nil {
        panic(err)
    }

    assetPack := assetpack.Load(absDir)

    b, err := json.MarshalIndent(assetPack, "", "")
    assetStatJSON = string(b)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Serving directory '%v' on port '%v'\n", absDir, *port)

    fileServer := http.StripPrefix("/files/", http.FileServer(http.Dir(absDir)))
    http.Handle("/files/", fileServer)
    http.HandleFunc("/stat/", statHandler)
    http.ListenAndServe(portString, nil)
}
