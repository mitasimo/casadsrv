package main

import (
	"io/ioutil"
	"net/http"
	"testing"
)

func BenchmarkService(b *testing.B) {
	//b.SetParallelism(500)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := http.Get("http://localhost:1133")
			if err == nil {
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				//fmt.Fprintf(os.Stdout, "%s\n", body)
			}
		}
	})
}
