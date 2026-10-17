package avatar

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func BenchmarkServeMark(b *testing.B) {
	mux := http.NewServeMux()
	Routes(mux, StylePoly)
	for _, target := range []string{"/avatar/v2/night_owl", "/avatar/v2/night_owl?style=pixel"} {
		b.Run(target, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				rec := httptest.NewRecorder()
				mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			}
		})
	}
}

// BenchmarkServeMarkParallel is many visitors loading the same faces at
// once: hits must not queue behind each other.
func BenchmarkServeMarkParallel(b *testing.B) {
	mux := http.NewServeMux()
	Routes(mux, StylePoly)
	names := make([]string, 200)
	for i := range names {
		names[i] = "/avatar/v2/person" + strconv.Itoa(i)
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, names[i%len(names)], nil))
			i++
		}
	})
}
