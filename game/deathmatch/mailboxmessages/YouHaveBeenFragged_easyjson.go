// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package mailboxmessages

import (
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

func easyjsonD875cbDecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(in *jlexer.Lexer, out *YouHaveBeenFragged) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "by":
			out.By = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjsonD875cbEncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(out *jwriter.Writer, in YouHaveBeenFragged) {
	out.RawByte('{')
	first := true
	_ = first
	if !first {
		out.RawByte(',')
	}
	first = false
	out.RawString("\"by\":")
	out.String(string(in.By))
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v YouHaveBeenFragged) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjsonD875cbEncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v YouHaveBeenFragged) MarshalEasyJSON(w *jwriter.Writer) {
	easyjsonD875cbEncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *YouHaveBeenFragged) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjsonD875cbDecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *YouHaveBeenFragged) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjsonD875cbDecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(l, v)
}
