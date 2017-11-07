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

	playcanvaszippath := flag.String("playcanvaszip", "", "Playcanvas zip file; required")
	vizdirpath := flag.String("vizdir", "", "Viz checkout dir; required")
	moveTo := flag.String("moveto", "", "If set, built zip will be moved to given file; ex: --moveto /path/to/map.zip")
	flag.Parse()

	paramError := false

	if *playcanvaszippath == "" {
		fmt.Println("--playcanvaszip is required")
		paramError = true
	}

	if *vizdirpath == "" {
		fmt.Println("--vizdir is required")
		paramError = true
	}

	if paramError {
		os.Exit(1)
	}

	unzippedfiles, err := unzip(*playcanvaszippath, zipOutPath)
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
	configJSON, err := ioutil.ReadFile(zipOutPath + "/config.json")
	if err != nil {
		panic(err)
	}

	type configAsset struct {
		Type string `json:"type"`
		Name string `json:"name"`
		File struct {
			Filename string `json:"filename"`
			URL      string `json:"url"`
		} `json:"file"`
	}

	configWhole := make(map[string]json.RawMessage)
	json.Unmarshal(configJSON, &configWhole)
	configAssets := make(map[string]configAsset)
	json.Unmarshal(configWhole["assets"], &configAssets)

	modelURL := ""
	modelFilename := ""
	for _, asset := range configAssets {
		if asset.Type == "model" {
			modelURL = asset.File.URL
			modelFilename = asset.File.Filename
			break
		}
	}

	if modelURL == "" || modelFilename == "" {
		panic("Could not determine the name of the model asset")
	}

	// On crée le réceptacle
	err = os.MkdirAll(newOutPath, 0700)
	if err != nil {
		panic("Could not create output dir.")
	}

	// On crée les répertoires
	dirs := []string{
		"js",
		"css",
		"json",
		"img",
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
	filecopy["manifest.json"] = "json/manifest.json"
	filecopy["playcanvas-stable.min.js"] = "js/playcanvas-stable.min.js"
	filecopy["styles.css"] = "css/styles.css"
	filecopy["__loading__.js"] = "js/loading.js"
	filecopy["__start__.js"] = "js/start.js"
	filecopy["__game-scripts.js"] = "js/game-scripts.js"
	filecopy[zipid+".json"] = "json/scene.json"
	filecopy[modelURL] = "json/model.json"

	assetsRename := make(map[string]string)
	assetsRename[modelURL] = "json/model.json"
	//assetsRename[modelFilename] = "model.json"
	assetsRename["__game-scripts.js"] = "js/game-scripts.js"

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
			filecopy[relFilePath] = "img/" + filename
		} else if strings.HasSuffix(filename, ".json") {
			filecopy[relFilePath] = "json/" + filename
		} else if strings.HasSuffix(filename, ".js") {
			filecopy[relFilePath] = "js/" + filename
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
	indexReplacements["styles.css"] = "css/styles.css"
	indexReplacements["manifest.json"] = "json/manifest.json"
	indexReplacements["playcanvas-stable.min.js"] = "js/playcanvas-stable.min.js"
	indexReplacements[zipid+".json"] = "json/scene.json"
	indexReplacements["config.json"] = "json/config.json"
	indexReplacements["SCRIPTS = ["] = "//SCRIPTS = ["
	indexReplacements["__start__.js"] = "js/start.js"
	indexReplacements["__loading__.js"] = "js/loading.js"

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

	err = ioutil.WriteFile(newOutPath+"/json/config.json", []byte(configContentsStr), 0700)
	if err != nil {
		panic(err)
	}

	// On gzippe le fichier de modèle
	cmdGzip := exec.Command(
		"gzip",
		"--best",
		newOutPath+"/"+filecopy[modelURL],
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

	///////////////////////////////////////////////////////////////////////////
	// Bundling mappack (map assets + viz lib)
	///////////////////////////////////////////////////////////////////////////

	// /map/* <= pc assets
	// /lib/* <= viz lib
	// /index.html <= the used by the viz service (trainer or viz-server)

	bundleOutPath, err := ioutil.TempDir("", "bundle-mappack")
	if err != nil {
		log.Fatal(err)
	}

	mapDistDirPath := bundleOutPath + "/map"
	libDistDirPath := bundleOutPath + "/lib"

	os.MkdirAll(libDistDirPath, os.ModePerm)

	// Building viz lib
	cmdBuild := exec.Command(
		"npm",
		"run",
		"install-and-build",
	)
	cmdBuild.Dir = *vizdirpath

	cmdBuild.Env = nil
	cmdBuild.Stderr = os.Stderr
	cmdBuild.Stdout = os.Stdout
	err = cmdBuild.Run()
	if err != nil {
		fmt.Println("Error while building viz js.")
		os.Exit(1)
	}

	// Bundling assets and js
	utils.Check(utils.CopyDir(newOutPath, mapDistDirPath), "Could not copy map dir")
	utils.Check(utils.CopyFile(*vizdirpath+"/lib/bytearenaviz.min.js", libDistDirPath+"/bytearenaviz.min.js"), "Could not copy lib js")
	utils.Check(utils.CopyFile(*vizdirpath+"/index.html", bundleOutPath+"/index.html"), "Could not index.html")

	// Zipping payload
	zipPath := bundleOutPath + ".zip"
	cmdZip := exec.Command(
		"zip",
		"-r",
		zipPath,
		".",
	)
	cmdZip.Dir = bundleOutPath

	cmdZip.Env = nil
	cmdZip.Stderr = os.Stderr
	cmdZip.Stdout = os.Stdout
	err = cmdZip.Run()
	if err != nil {
		fmt.Println("Error while zipping bundle.")
		os.Exit(1)
	}

	if *moveTo != "" {
		utils.Check(utils.CopyFile(zipPath, *moveTo), "Could not move bundle to specified path")
		fmt.Println("Bundle:", *moveTo)
	} else {
		fmt.Println("Bundle:", zipPath)
	}
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
