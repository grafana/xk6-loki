package flog

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

type Flog struct {
	rand     *rand.Rand
	gofakeit *gofakeit.Faker
}

func New(rand *rand.Rand, faker *gofakeit.Faker) *Flog {
	return &Flog{rand: rand, gofakeit: faker}
}

// TODO: something faster, saner and safer...
func formatExtra(format string, data [][2]string) string {
	result := make([]string, len(data))
	for i, kv := range data {
		if format == "json" {
			result[i] = fmt.Sprintf(", %q: %q", kv[0], kv[1])
		} else {
			result[i] = fmt.Sprintf(" %q", kv[0]+"="+kv[1])
		}
	}

	return strings.Join(result, "")
}

func (f *Flog) LogLine(format string, t time.Time, extraData [][2]string) string {
	extra := formatExtra(format, extraData)
	switch format {
	case "apache_common":
		return f.NewApacheCommonLog(t, extra)
	case "apache_combined":
		return f.NewApacheCombinedLog(t, extra)
	case "apache_error":
		return f.NewApacheErrorLog(t, extra)
	case "rfc3164":
		return f.NewRFC3164Log(t, extra)
	case "rfc5424":
		return f.NewRFC5424Log(t, extra)
	case "common_log":
		return f.NewCommonLogFormat(t, extra)
	case "json":
		return f.NewJSONLogFormat(t, extra)
	case "logfmt":
		return f.NewLogFmtLogFormat(t, extra)
	default:
		return ""
	}
}
