package handler

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/bytearena/bytearena/common/recording"
	"github.com/gorilla/mux"
)

func Replay(recordStore recording.RecordStoreInterface, basepath string, CDNBaseURL string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["recordId"]

		if !recordStore.RecordExists(id) {
			w.Write([]byte("Record not found"))
			return
		}

		vizhtml, err := ioutil.ReadFile(basepath + "index.html")
		if err != nil {
			w.Write([]byte("ERROR: could not render arena"))
			return
		}

		protocol := "ws"

		if os.Getenv("ENV") == "prod" {
			protocol = "wss"
		}

		var vizhtmlTemplate = template.Must(template.New("").Parse(string(vizhtml)))
		vizhtmlTemplate.Execute(w, struct {
			WsURL      string
			CDNBaseURL string
			Rand       int64
			Tps        int
		}{
			WsURL:      protocol + "://" + r.Host + "/record/" + id + "/ws",
			CDNBaseURL: CDNBaseURL,
			Rand:       time.Now().Unix(),
			Tps:        10, // FIXME(sven): get metadata from record
		})

	}
}
