package main

import (
	"fmt"
	"strings"
)

type D2Writer struct {
	data []string
}

func NewD2Writer() *D2Writer {
	return &D2Writer{}
}
func (w *D2Writer) Write(key string, values ...string) {
	if len(values) == 0 {
		w.data = append(w.data, key)
	} else {
		for _, value := range values {
			w.data = append(w.data, fmt.Sprintf("%s: %s", key, value))
		}
	}
}

func (w *D2Writer) String() string {
	return strings.Join(w.data, "\n")
}
