package mapcmd

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

	trainutils "github.com/bytearena/bytearena/ba/utils"

	bettererrors "github.com/xtuc/better-errors"
)

const (
	MANIFEST_URL = "https://dltrainer.bytearena.com/manifest.json"
)

type manifest struct {
	Md5 string `json:"md5"`
	Url string `json:"url"`
}

func MapUpdateAction(debug func(str string)) {
	mapChecksum, err := GetLocalMapChecksum()
	if err != nil {
		// Local map has never been downloaded
		fmt.Println("Map does not exist locally; will have to be fetched.")
	}

	fmt.Println("Downloading map manifest from " + MANIFEST_URL)

	mapManifest, errManifest := DownloadAndGetManifest()
	if errManifest != nil {
		trainutils.FailWith(errManifest)
	}

	if mapChecksum != mapManifest.Md5 {
		debug("The map is outdated, downloading the new version...")

		err := DownloadMap(mapManifest)

		if err != nil {
			trainutils.FailWith(err)
		}
	} else {
		debug("The map is already up to date!")
	}
}

func GetMapLocation(mapName string) string {
	user, err := user.Current()

	if err != nil {
		trainutils.FailWith(err)
	}

	baConfigDir := path.Join(user.HomeDir, ".bytearena")

	err = os.MkdirAll(baConfigDir, os.ModePerm)

	if err != nil {
		trainutils.FailWith(err)
	}

	return path.Join(baConfigDir, mapName+".zip")
}

func GetLocalMapChecksum() (string, error) {
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

func DownloadMap(manifest manifest) error {
	head, errHead := http.Head(manifest.Url)
	head.Body.Close()

	if errHead != nil {
		return bettererrors.
			NewFromString("Could not get map").
			With(bettererrors.NewFromErr(errHead)).
			SetContext("url", manifest.Url)
	}

	if head.StatusCode != 200 {
		msg := fmt.Sprintf("Could not get map (%s): server returned code %s", manifest.Url, head.Status)
		return bettererrors.NewFromString(msg)
	}

	fileSize := int(head.ContentLength)

	res, errGet := http.Get(manifest.Url)

	if errGet != nil {
		return bettererrors.
			NewFromString("Could not get map").
			With(errHead).
			SetContext("url", manifest.Url)
	}

	file, errOpen := os.OpenFile(GetMapLocation("map"), os.O_WRONLY|os.O_CREATE, 0755)

	if errOpen != nil {
		return bettererrors.
			NewFromString("Could not open destination file").
			With(errOpen).
			SetContext("location", GetMapLocation("map"))
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

func DownloadAndGetManifest() (manifest, error) {
	var manifest manifest

	res, err := http.Get(MANIFEST_URL)

	if err != nil {
		return manifest, bettererrors.
			NewFromString("Could not download manifest").
			With(bettererrors.NewFromErr(err)).
			SetContext("manifest url", MANIFEST_URL)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		msg := fmt.Sprintf("Could not download manifest (%s): server returned code %s", MANIFEST_URL, res.Status)
		return manifest, bettererrors.NewFromString(msg)
	}

	data, _ := ioutil.ReadAll(res.Body)

	err = json.Unmarshal(data, &manifest)

	if err != nil {
		return manifest, bettererrors.
			NewFromString("Could not parse manifest").
			With(bettererrors.NewFromErr(err)).
			SetContext("manifest url", MANIFEST_URL)
	}

	return manifest, nil
}

func IsMapLocally() bool {
	_, err := os.Stat(GetMapLocation("map"))

	return !os.IsNotExist(err)
}

func getMapLocally() (*os.File, error) {
	f, err := os.OpenFile(GetMapLocation("map"), os.O_RDONLY, 0755)

	if err != nil {
		return nil, bettererrors.
			NewFromString("Could not open map file").
			With(bettererrors.NewFromErr(err)).
			SetContext("map file", GetMapLocation("map"))
	}

	return f, nil
}

func updateMap(mapManifest manifest, debug func(str string)) error {
	if IsMapLocally() {
		mapChecksum, err := GetLocalMapChecksum()
		if err != nil {
			return err
		}

		if mapChecksum != mapManifest.Md5 {
			debug("The map is outdated, downloading the new version...")

			err := DownloadMap(mapManifest)

			if err != nil {
				return err
			}
		}
	} else {
		debug("Map doesn't exists locally, downloading...")

		err := DownloadMap(mapManifest)

		if err != nil {
			return err
		}
	}

	return nil
}
