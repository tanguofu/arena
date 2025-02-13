// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type MyFormatter struct{}

func (m *MyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	b.WriteString(entry.Time.Format("2006-01-02 15:04:05"))
	b.WriteString(" ")
	b.WriteString(entry.Level.String())
	b.WriteString(" ")
	if entry.HasCaller() {
		fileName := filepath.Base(entry.Caller.File)
		b.WriteString(fileName)
		b.WriteString(":")

		// funcNames := strings.Split(entry.Caller.Function, ".")
		// b.WriteString(funcNames[len(funcNames)-1])
		// b.WriteString(":")

		b.WriteString(fmt.Sprintf("%d", entry.Caller.Line))
		b.WriteString(" ")
	}
	b.WriteString(entry.Message)
	b.WriteString("\n")

	return b.Bytes(), nil
}

// SetLogLevel sets the logrus logging level
func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Fatalf("Unknown level: %s", level)
	}

	log.SetReportCaller(true)
	logrus.SetFormatter(&MyFormatter{})
}
