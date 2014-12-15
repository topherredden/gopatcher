package main

import "net/http"
import "fmt"
//import "flag"
import "encoding/json"
import "path/filepath"
import "io/ioutil"
import "os"
import "io"
import "./assetpack"
import "bitbucket.org/kardianos/osext"
import "os/exec"

var assetStatJSON string
var serverAddress = "http://151.226.92.200:8080"

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

func downloadFile(path string) {
    var filePath string = serverAddress + "/files/" + path
    filePath = filepath.ToSlash(filePath)

    outPath, err := filepath.Abs("download/" + path)
    if err != nil {
        panic(err)
    }

    out := CreateFile(outPath)

    res, err := http.Get(filePath)
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
        binAbs, _ := osext.Executable()
        binDir := filepath.Dir(binAbs)
        bin := filepath.Base(binAbs)
        oldBin := binDir + "/~" + bin

        if _, err := os.Stat(oldBin); os.IsNotExist(err) {
            break
        }

        err := os.Remove(oldBin)
        if err == nil {
            break
        } else {
            fmt.Println(err)
        }
    }
    
    //var dir = flag.String("dir", "files", "Directory to serve files from")
    //var port = flag.Int("port", 8080, "Port for HTTP server to listen on")
    //flag.Parse()

    // Convert path to absolute
    //absDir, _ := filepath.Abs(*dir)

    //assetPack := assetpack.Load(absDir)

    /*b, err := json.MarshalIndent(assetPack, "", "")
    assetStatJSON = string(b)
    if err != nil {
        panic(err)
    }*/

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

    // Check for self upgrade
    binPath, _ := osext.Executable()
    bin := filepath.Base(binPath)
    //binDir := filepath.Dir(binPath)
    binAsset, ok := assetPack.Assets[bin]
    if ok {
        hash, _ := assetpack.HashFile(bin)

        if hash != binAsset.Hash {
            fmt.Println("Found update for patcher.")
            downloadFile(binAsset.Path)
            fmt.Println("Installing update...")
            _ = os.Rename(bin, "~" + bin)
            copyFile(bin, "download/" + bin)
            //fmt.Println("Please restart patcher.")
            //fmt.Println("Restarting patcher.")

            /*var procAttr os.ProcAttr 
            procAttr.Files = []*os.File{nil, nil, nil}
            _, err := os.StartProcess(binPath, nil, &procAttr)
            if err != nil {
                fmt.Println(err)
            }*/
            cmd := exec.Command(binPath)
            cmd.Stdin = os.Stdin
            cmd.Stdout = os.Stdout
            cmd.Stderr = os.Stderr
            err := cmd.Start()
            if err != nil {
                fmt.Println(err)
            }

            os.Exit(0)
        }
    }

    for k, v := range assetPack.Assets {
        _ = k
        hash, _ := assetpack.HashFile(v.Path)

        if hash != v.Hash {
            downloadFile(v.Path)
            copyFile(v.Path, "download/" + v.Path)
        }
    }

    err = os.RemoveAll("download")
    if err != nil {
        fmt.Println(err)
    }
}
