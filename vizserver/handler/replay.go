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

func Replay(recorder recording.Recorder, basepath string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["recordId"]

		_, err := os.Stat(recorder.GetDirectory() + "/record-" + id + ".bin")

		if os.IsNotExist(err) {
			w.Write([]byte("Record not found"))
			return
		}

		vizhtml, err := ioutil.ReadFile(basepath + "index.html")
		if err != nil {
			w.Write([]byte("ERROR: could not render arena"))
			return
		}

		var vizhtmlTemplate = template.Must(template.New("").Parse(string(vizhtml)))
		vizhtmlTemplate.Execute(w, struct {
			WsURL string
			Rand  int64
			Tps   int
		}{
			WsURL: "ws://" + r.Host + "/record/" + id + "/ws",
			Rand:  time.Now().Unix(),
			Tps:   10, // FIXME(sven): get metadata from record
		})

	}
}
