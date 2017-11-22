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

func easyjson9b336149DecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(in *jlexer.Lexer, out *YouHaveFragged) {
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
		case "who":
			out.Who = string(in.String())
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
func easyjson9b336149EncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(out *jwriter.Writer, in YouHaveFragged) {
	out.RawByte('{')
	first := true
	_ = first
	if !first {
		out.RawByte(',')
	}
	first = false
	out.RawString("\"who\":")
	out.String(string(in.Who))
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v YouHaveFragged) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson9b336149EncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v YouHaveFragged) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson9b336149EncodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *YouHaveFragged) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson9b336149DecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *YouHaveFragged) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson9b336149DecodeGithubComBytearenaBytearenaGameDeathmatchMailboxmessages(l, v)
}
