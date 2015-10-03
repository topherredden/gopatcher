package assetpack

import "fmt"
import "bytes"
import "crypto/sha1"
import "os"
import "bufio"
import "io"
import "path/filepath"
import "encoding/hex"

type AssetInfo struct {
	Hash string
	Path string
	Name string
	Dir  string
}

type AssetPack struct {
	GlobalHash  string
	Assets      map[string]AssetInfo
	PatcherHash string
}

func HashFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}

	h := sha1.New()
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	r := bufio.NewReader(f)
	io.Copy(h, r)
	s := hex.EncodeToString(h.Sum(nil))

	f.Close()

	return s, nil
}

func Load(path string, patcherPath string) AssetPack {
	var assetPack AssetPack
	var absPath string

	absPath, _ = filepath.Abs(path)
	assetPack.Assets = make(map[string]AssetInfo)

	// Visitor Function
	visit := func(path string, f os.FileInfo, err error) error {
		if _, err := os.Stat(path); err == nil && f.IsDir() == false {
			hash, _ := HashFile(path)

			var assetInfo AssetInfo
			assetInfo.Hash = hash
			assetInfo.Path, _ = filepath.Rel(absPath, path)
			assetInfo.Name = f.Name()
			assetInfo.Dir = filepath.Dir(assetInfo.Path)

			assetPack.Assets[assetInfo.Path] = assetInfo
		}
		return nil
	}

	// Hash files
	err := filepath.Walk(absPath, visit)
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
	}

	// Hash Patcher
	assetPack.PatcherHash, _ = HashFile(patcherPath)

	// Calculate global hash
	var hashBuffer bytes.Buffer
	for i := range assetPack.Assets {
		hashBuffer.WriteString(assetPack.Assets[i].Hash)
	}
	h := sha1.New()
	h.Write(hashBuffer.Bytes())
	s := hex.EncodeToString(h.Sum(nil))
	assetPack.GlobalHash = s

	return assetPack
}
