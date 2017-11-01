package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

	"github.com/cheggaaa/pb"
	"github.com/pkg/errors"
)

const (
	MANIFEST_URL = "https://dltrainer.bytearena.com/manifest.json"
)

type manifest struct {
	Md5 string `json:"md5"`
	Url string `json:"url"`
}

func getMapLocation() string {
	user, err := user.Current()

	if err != nil {
		failWith(err)
	}

	baConfigDir := path.Join(user.HomeDir, ".bytearena")

	err = os.MkdirAll(baConfigDir, os.ModePerm)

	if err != nil {
		failWith(err)
	}

	return path.Join(baConfigDir, "map.zip")
}

func getLocalMapChecksum() (string, error) {
	file, err := getMapLocally()
	defer file.Close()

	if err != nil {
		return "", err
	}

	h := md5.New()
	if _, err := io.Copy(h, file); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadMap(manifest manifest) error {
	head, errHead := http.Head(manifest.Url)
	head.Body.Close()

	if errHead != nil {
		return errors.Wrapf(errHead, "Could not get map (%s)", manifest.Url)
	}

	if head.StatusCode != 200 {
		return fmt.Errorf("Could not get map (%s): server returned code %s", manifest.Url, head.Status)
	}

	fileSize := int(head.ContentLength)

	res, errGet := http.Get(manifest.Url)

	if errGet != nil {
		return errors.Wrapf(errHead, "Could not get map (%s)", manifest.Url)
	}

	file, errOpen := os.OpenFile(getMapLocation(), os.O_WRONLY|os.O_CREATE, 0755)

	if errOpen != nil {
		return errors.Wrapf(errHead, "Could not open destination file (%s)", getMapLocation())
	}

	bar := pb.New(fileSize)
	bar.SetWidth(80)
	bar.Start()

	rd := bar.NewProxyReader(res.Body)
	io.Copy(file, rd)

	file.Close()
	bar.Finish()

	return nil
}

func downloadAndGetManifest() (manifest, error) {
	var manifest manifest

	res, err := http.Get(MANIFEST_URL)

	if err != nil {
		return manifest, errors.Wrapf(err, "Could not download manifest (%s)", MANIFEST_URL)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return manifest, fmt.Errorf("Could not download manifest (%s): server returned code %s", MANIFEST_URL, res.Status)
	}

	data, _ := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(data, &manifest)

	if err != nil {
		return manifest, errors.Wrapf(err, "Could not parse manifest (%s)", MANIFEST_URL)
	}

	return manifest, nil
}

func isMapLocally() bool {
	_, err := os.Stat(getMapLocation())

	return !os.IsNotExist(err)
}

func getMapLocally() (*os.File, error) {
	f, err := os.OpenFile(getMapLocation(), os.O_RDONLY, 0755)

	if err != nil {
		return nil, errors.Wrapf(err, "Could not open map file (%s)", getMapLocation())
	}

	return f, nil
}
