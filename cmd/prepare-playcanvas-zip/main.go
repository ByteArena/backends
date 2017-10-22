package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bytearena/bytearena/common/utils"
)

func main() {
	// 1. Décompresser le zip
	// 2. Lire index.html
	// 3. Trouver le SCENE_PATH et le zip ID
	// 4. Déplacer tous les fichiers files/assets/**/* dans /assets/

	zipOutPath, err := ioutil.TempDir("", "pc-tmp-zip")
	if err != nil {
		log.Fatal(err)
	}

	newOutPath, err := ioutil.TempDir("", "pc-tmp-new")
	if err != nil {
		log.Fatal(err)
	}

	zipfilepath := flag.String("in", "", "Input zip file; required")
	flag.Parse()

	if *zipfilepath == "" {
		fmt.Println("--in is required; ex: --in ~/file.zip")
		os.Exit(1)
	}

	unzippedfiles, err := unzip(*zipfilepath, zipOutPath)
	if err != nil {
		panic(err)
	}

	defer os.RemoveAll(zipOutPath) // clean up

	// On applique le patch
	curdir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	cmd := exec.Command(
		"patch",
		"-p", "1",
		"--directory", zipOutPath,
		"-i", curdir+"/patches/island.patch",
	)

	cmd.Env = nil
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		fmt.Println("Error while patching source files.")
		os.Exit(1)
	}

	// On détermine l'ID de la scene PC

	indexContents, err := ioutil.ReadFile(zipOutPath + "/index.html")
	if err != nil {
		panic(err)
	}

	re := regexp.MustCompile(`SCENE_PATH = "(\d+).json";`)
	res := re.FindAllStringSubmatch(string(indexContents), -1)
	if len(res) != 1 {
		panic("Could not determine SCENE_PATH")
	}

	zipid := res[0][1]

	// On détermine le chemin du fichier de modèle
	configJson, err := ioutil.ReadFile(zipOutPath + "/config.json")
	if err != nil {
		panic(err)
	}

	type configAsset struct {
		Type string `json:"type"`
		Name string `json:"name"`
		File struct {
			Filename string `json:"filename"`
			Url      string `json:"url"`
		} `json:"file"`
	}

	configWhole := make(map[string]json.RawMessage)
	json.Unmarshal(configJson, &configWhole)
	configAssets := make(map[string]configAsset)
	json.Unmarshal(configWhole["assets"], &configAssets)

	modelUrl := ""
	modelFilename := ""
	for _, asset := range configAssets {
		if asset.Type == "model" {
			modelUrl = asset.File.Url
			modelFilename = asset.File.Filename
			break
		}
	}

	if modelUrl == "" || modelFilename == "" {
		panic("Could not determine the name of the model asset")
	}

	// On crée le réceptacle
	err = os.MkdirAll(newOutPath, 0700)
	if err != nil {
		panic("Could not create output dir.")
	}

	// On crée les répertoires
	dirs := []string{
		"assets",
		"assets/js",
		"assets/css",
		"assets/json",
		"assets/img",
	}

	for _, dir := range dirs {
		err = os.MkdirAll(newOutPath+"/"+dir, 0700)
		if err != nil {
			panic(err)
		}
	}

	// On copie les fichiers
	filecopy := make(map[string]string)
	//filecopy["index.html"] = "index.html"
	filecopy["manifest.json"] = "assets/json/manifest.json"
	filecopy["playcanvas-stable.min.js"] = "assets/js/playcanvas-stable.min.js"
	filecopy["styles.css"] = "assets/css/styles.css"
	filecopy["__loading__.js"] = "assets/js/loading.js"
	filecopy["__start__.js"] = "assets/js/start.js"
	//filecopy["config.json"] = "assets/json/config.json"
	filecopy[zipid+".json"] = "assets/json/scene.json"
	filecopy[modelUrl] = "assets/json/model.json"

	assetsRename := make(map[string]string)
	assetsRename[modelUrl] = "assets/json/model.json"
	assetsRename[modelFilename] = "model.json"

	for _, filepath := range unzippedfiles {

		filePrefix := zipOutPath + "/files/assets/"

		if !strings.HasPrefix(filepath, filePrefix) {
			continue
		}

		filename := path.Base(filepath)
		relFilePath := "files/assets/" + strings.TrimPrefix(filepath, filePrefix)

		if _, ok := filecopy[relFilePath]; ok {
			// la copie de ce fichier est définie manuellement
			continue
		}

		if strings.HasSuffix(filename, ".png") || strings.HasSuffix(filename, ".jpg") {
			filecopy[relFilePath] = "assets/img/" + filename
		} else if strings.HasSuffix(filename, ".json") {
			filecopy[relFilePath] = "assets/json/" + filename
		} else if strings.HasSuffix(filename, ".js") {
			filecopy[relFilePath] = "assets/js/" + filename
		} else {
			panic("Unknown asset type " + filename)
		}

		assetsRename[relFilePath] = filecopy[relFilePath]

	}

	for from, to := range filecopy {
		utils.CopyFile(zipOutPath+"/"+from, newOutPath+"/"+to)
	}

	// On remplace les chemins dans index.html

	indexReplacements := make(map[string]string)
	indexReplacements["styles.css"] = "assets/css/styles.css"
	indexReplacements["manifest.json"] = "assets/json/manifest.json"
	indexReplacements["playcanvas-stable.min.js"] = "assets/js/playcanvas-stable.min.js"
	indexReplacements[zipid+".json"] = "assets/json/scene.json"
	indexReplacements["config.json"] = "assets/json/config.json"
	indexReplacements["SCRIPTS = ["] = "//SCRIPTS = ["
	indexReplacements["__start__.js"] = "assets/js/start.js"
	indexReplacements["__loading__.js"] = "assets/js/loading.js"

	indexContentsStr := string(indexContents)
	for from, to := range indexReplacements {
		indexContentsStr = strings.Replace(indexContentsStr, from, to, -1)
	}

	err = ioutil.WriteFile(newOutPath+"/index.html", []byte(indexContentsStr), 0700)
	if err != nil {
		panic(err)
	}

	// On remplace les chemins dans config.json

	configContents, err := ioutil.ReadFile(zipOutPath + "/config.json")
	if err != nil {
		panic(err)
	}

	configContentsStr := string(configContents)
	for from, to := range assetsRename {
		configContentsStr = strings.Replace(configContentsStr, from, to, -1)
	}

	err = ioutil.WriteFile(newOutPath+"/assets/json/config.json", []byte(configContentsStr), 0700)
	if err != nil {
		panic(err)
	}

	// On gzippe le fichier de modèle
	cmdGzip := exec.Command(
		"gzip",
		"--best",
		newOutPath+"/"+filecopy[modelUrl],
	)

	cmdGzip.Env = nil
	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	err = cmdGzip.Run()
	if err != nil {
		fmt.Println("Error while gzipping model file.")
		os.Exit(1)
	}

	fmt.Println("Output: " + newOutPath)
}

func unzip(src, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		defer rc.Close()

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {

			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)

		} else {

			filenames = append(filenames, fpath)

			// Make File
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
			}

			err = os.MkdirAll(fdir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
				return filenames, err
			}
			f, err := os.OpenFile(
				fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return filenames, err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return filenames, err
			}

		}
	}
	return filenames, nil
}
