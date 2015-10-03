package main

import "net/http"
import "fmt"
import "encoding/json"
import "path/filepath"
import "io/ioutil"
import "os"
import "io"
import "./assetpack"
import "os/exec"

var assetStatJSON string
var serverAddress = "http://151.225.0.62:8989"
var downloadDir string

func CreateFile(path string) *os.File {
	var dir = filepath.Dir(path)
	err := os.MkdirAll(dir, 0777)
	if err != nil {
		panic(err)
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	return f
}

type ProgressReader func(written int64)

func CopyProgress(dst io.Writer, src io.Reader, pf ProgressReader) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)

				if pf != nil {
					pf(written)
				}
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err
}

func downloadFileData(path string, httpPath string) {
	outPath, err := filepath.Abs(downloadDir + path)
	if err != nil {
		panic(err)
	}

	out := CreateFile(outPath)

	res, err := http.Get(httpPath)
	if err != nil {
		fmt.Println(err)
	}

	r := func(written int64) {
		var percent = int64((float64(written) / float64(res.ContentLength)) * 100.0)

		fmt.Printf("\rDownloading '%s'...%v%%", path, percent)

		if percent >= 100 {
			fmt.Printf("\n")
		}
	}

	_, err = CopyProgress(out, res.Body, r)
	if err != nil {
		fmt.Println(err)
	}

	out.Close()
	res.Body.Close()
}

func downloadFilePatcher(path string) {
	var httpPath string = serverAddress + "/patcher/"
	httpPath = filepath.ToSlash(httpPath)

	downloadFileData(path, httpPath)
}

func downloadFile(path string) {
	var filePath string = serverAddress + "/files/" + path
	filePath = filepath.ToSlash(filePath)

	downloadFileData(path, filePath)
}

func copyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d := CreateFile(dst)
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func main() {
	// Clean up upgrade
	for {
		binAbs := os.Args[0]
		binDir := filepath.Dir(binAbs)
		bin := filepath.Base(binAbs)
		oldBin := binDir + "/~" + bin

		if _, err := os.Stat(oldBin); os.IsNotExist(err) {
			break
		}

		fmt.Println("Removing old patcher.")

		err := os.Remove(oldBin)
		if err == nil {
			break
		} else {
			fmt.Println(err)
		}
	}

	res, err := http.Get(serverAddress + "/stat/")
	if err != nil {
		fmt.Println(err)
	}

	stat, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println(err)
	}

	var assetPack assetpack.AssetPack
	err = json.Unmarshal(stat, &assetPack)
	if err != nil {
		fmt.Println(err)
	}

	// Create Temporary Download Dir
	createTemp()
	defer cleanTemp()

	// Check for self upgrade
	fmt.Printf("Checking for patcher update.\n")
	binPath := os.Args[0]
	bin := filepath.Base(binPath)
	hash, _ := assetpack.HashFile(binPath)

	if hash != assetPack.PatcherHash {
		fmt.Println("Found update for patcher.")
		downloadFilePatcher(bin)
		fmt.Print("Installing update...")
		_ = os.Rename(bin, "~"+bin)
		copyFile(bin, downloadDir+bin)

		fmt.Println("done.")
		fmt.Println("Restarting patcher.")
		cmd := exec.Command(binPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Start()
		if err != nil {
			fmt.Println(err)
		}

		cleanTemp()
		os.Exit(0)
	} else {
		fmt.Println("No update.")
	}

	for k, v := range assetPack.Assets {
		_ = k
		hash, _ := assetpack.HashFile(v.Path)

		if hash != v.Hash {
			downloadFile(v.Path)
			copyFile(v.Path, downloadDir+v.Path)
		}
	}
}

func createTemp() {
	wd, _ := os.Getwd()
	downloadDir, _ = ioutil.TempDir(wd, "~temp")
	downloadDir = downloadDir + "/"
	err := os.MkdirAll(downloadDir, 0777)
	if err != nil {
		fmt.Println(err)
	}
}

func cleanTemp() {
	err := os.RemoveAll(downloadDir)
	if err != nil {
		fmt.Println(err)
	}
}
