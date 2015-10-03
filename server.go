package main

import "net/http"
import "fmt"
import "flag"
import "time"
import "encoding/json"
import "./assetpack"
import "path/filepath"

var assetStatJSON string
var assetAbsDir string
var patcherAbsDir string

func statHandler(w http.ResponseWriter, r *http.Request) {
	refreshAssets(assetAbsDir, patcherAbsDir)
	fmt.Fprintf(w, "%s", assetStatJSON)
}

func patcherHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Serving Patcher '%s'\n", patcherAbsDir)
	http.ServeFile(w, r, patcherAbsDir)
}

func fileWatcher(watchPath string) {
	for {
		fmt.Print("Checking for file changes...")
		fmt.Println("complete.")
		time.Sleep(time.Second * 10)
	}
}

func refreshAssets(absDir string, patcherDir string) {
	fmt.Printf("Refreshing dir '%s'\n", absDir)
	assetPack := assetpack.Load(absDir, patcherDir)

	b, err := json.MarshalIndent(assetPack, "", "")
	assetStatJSON = string(b)
	if err != nil {
		panic(err)
	}
}

func main() {
	var dir = flag.String("dir", "files", "Directory to serve files from")
	var port = flag.Int("port", 8989, "Port for HTTP server to listen on")
	var patcher = flag.String("patcher", "patcher.exe", "Patcher binary")

	flag.Parse()

	// Convert path to absolute
	assetAbsDir, _ = filepath.Abs(*dir)
	patcherAbsDir, _ = filepath.Abs(*patcher)
	portString := fmt.Sprintf(":%v", *port)

	// Load Asset stats
	refreshAssets(assetAbsDir, patcherAbsDir)

	// Start Watcher
	//go fileWatcher(absDir)

	fmt.Printf("Serving directory '%v' on port '%v'\n", assetAbsDir, *port)

	fileServer := http.StripPrefix("/files/", http.FileServer(http.Dir(assetAbsDir)))
	http.Handle("/files/", fileServer)
	http.HandleFunc("/stat/", statHandler)
	http.HandleFunc("/patcher/", patcherHandler)
	http.ListenAndServe(portString, nil)
}
