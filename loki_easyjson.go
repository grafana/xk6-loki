// Code generated by easyjson for marshaling/unmarshaling. DO NOT EDIT.

package loki

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

func easyjson3fd435f7DecodeGithubComGrafanaXk6Loki(in *jlexer.Lexer, out *JSONStream) {
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
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "stream":
			if in.IsNull() {
				in.Skip()
			} else {
				in.Delim('{')
				out.Stream = make(map[string]string)
				for !in.IsDelim('}') {
					key := string(in.String())
					in.WantColon()
					var v1 string
					v1 = string(in.String())
					(out.Stream)[key] = v1
					in.WantComma()
				}
				in.Delim('}')
			}
		case "values":
			if in.IsNull() {
				in.Skip()
				out.Values = nil
			} else {
				in.Delim('[')
				if out.Values == nil {
					if !in.IsDelim(']') {
						out.Values = make([][]string, 0, 2)
					} else {
						out.Values = [][]string{}
					}
				} else {
					out.Values = (out.Values)[:0]
				}
				for !in.IsDelim(']') {
					var v2 []string
					if in.IsNull() {
						in.Skip()
						v2 = nil
					} else {
						in.Delim('[')
						if v2 == nil {
							if !in.IsDelim(']') {
								v2 = make([]string, 0, 4)
							} else {
								v2 = []string{}
							}
						} else {
							v2 = (v2)[:0]
						}
						for !in.IsDelim(']') {
							var v3 string
							v3 = string(in.String())
							v2 = append(v2, v3)
							in.WantComma()
						}
						in.Delim(']')
					}
					out.Values = append(out.Values, v2)
					in.WantComma()
				}
				in.Delim(']')
			}
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
func easyjson3fd435f7EncodeGithubComGrafanaXk6Loki(out *jwriter.Writer, in JSONStream) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"stream\":"
		out.RawString(prefix[1:])
		if in.Stream == nil && (out.Flags&jwriter.NilMapAsEmpty) == 0 {
			out.RawString(`null`)
		} else {
			out.RawByte('{')
			v4First := true
			for v4Name, v4Value := range in.Stream {
				if v4First {
					v4First = false
				} else {
					out.RawByte(',')
				}
				out.String(string(v4Name))
				out.RawByte(':')
				out.String(string(v4Value))
			}
			out.RawByte('}')
		}
	}
	{
		const prefix string = ",\"values\":"
		out.RawString(prefix)
		if in.Values == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v5, v6 := range in.Values {
				if v5 > 0 {
					out.RawByte(',')
				}
				if v6 == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
					out.RawString("null")
				} else {
					out.RawByte('[')
					for v7, v8 := range v6 {
						if v7 > 0 {
							out.RawByte(',')
						}
						out.String(string(v8))
					}
					out.RawByte(']')
				}
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v JSONStream) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson3fd435f7EncodeGithubComGrafanaXk6Loki(w, v)
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *JSONStream) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson3fd435f7DecodeGithubComGrafanaXk6Loki(l, v)
}
func easyjson3fd435f7DecodeGithubComGrafanaXk6Loki1(in *jlexer.Lexer, out *JSONPushRequest) {
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
		key := in.UnsafeFieldName(false)
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "streams":
			if in.IsNull() {
				in.Skip()
				out.Streams = nil
			} else {
				in.Delim('[')
				if out.Streams == nil {
					if !in.IsDelim(']') {
						out.Streams = make([]JSONStream, 0, 2)
					} else {
						out.Streams = []JSONStream{}
					}
				} else {
					out.Streams = (out.Streams)[:0]
				}
				for !in.IsDelim(']') {
					var v9 JSONStream
					(v9).UnmarshalEasyJSON(in)
					out.Streams = append(out.Streams, v9)
					in.WantComma()
				}
				in.Delim(']')
			}
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
func easyjson3fd435f7EncodeGithubComGrafanaXk6Loki1(out *jwriter.Writer, in JSONPushRequest) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"streams\":"
		out.RawString(prefix[1:])
		if in.Streams == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v10, v11 := range in.Streams {
				if v10 > 0 {
					out.RawByte(',')
				}
				(v11).MarshalEasyJSON(out)
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v JSONPushRequest) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson3fd435f7EncodeGithubComGrafanaXk6Loki1(w, v)
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *JSONPushRequest) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson3fd435f7DecodeGithubComGrafanaXk6Loki1(l, v)
}