package main

import (
	"html/template"
	"os"
	"reflect"

	"github.com/bytearena/bytearena/game/deathmatch/mailboxmessages"
)

const (
	docTemplate = `
### {{.Title}}
| name | type |
|---|---|
{{ range $value := .Fields }}| {{ $value.Name }} | {{ $value.Type }} |
{{ end }}
`
)

type DocEntry struct {
	Title  string
	Fields []reflect.StructField
}

func main() {
	generateDocumentationFor(mailboxmessages.Score{})
	generateDocumentationFor(mailboxmessages.Stats{})
	generateDocumentationFor(mailboxmessages.YouAreRespawning{})
	generateDocumentationFor(mailboxmessages.YouHaveBeenFragged{})
	generateDocumentationFor(mailboxmessages.YouHaveBeenHit{})
	generateDocumentationFor(mailboxmessages.YouHaveFragged{})
	generateDocumentationFor(mailboxmessages.YouHaveHit{})
	generateDocumentationFor(mailboxmessages.YouHaveRespawned{})
}

func generateDocumentationFor(messagestruct mailboxmessages.MailboxMessageInterface) {
	structType := reflect.TypeOf(messagestruct)

	entry := DocEntry{
		Title:  structType.Name(),
		Fields: make([]reflect.StructField, 0),
	}

	for i := 0; i < structType.NumField(); i++ {
		entry.Fields = append(entry.Fields, structType.Field(i))
	}

	tmpl, err := template.New("").Parse(docTemplate)

	err = tmpl.Execute(os.Stdout, entry)

	if err != nil {
		panic(err)
	}
}
