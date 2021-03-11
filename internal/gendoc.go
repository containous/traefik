package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/traefik/paerser/env"
	"github.com/traefik/paerser/flag"
	"github.com/traefik/paerser/generator"
	"github.com/traefik/paerser/parser"
	"github.com/traefik/traefik/v2/cmd"
	"github.com/traefik/traefik/v2/pkg/log"
)

func main() {
	genStaticConfDoc("./docs/content/reference/static-configuration/env-ref.md", "", func(i interface{}) ([]parser.Flat, error) {
		return env.Encode(env.DefaultNamePrefix, i)
	})
	genStaticConfDoc("./docs/content/reference/static-configuration/cli-ref.md", "--", flag.Encode)
	genKVDynConfDoc("./docs/content/reference/dynamic-configuration/kv-ref.md")
}

func genStaticConfDoc(outputFile, prefix string, encodeFn func(interface{}) ([]parser.Flat, error)) {
	logger := log.WithoutContext().WithField("file", outputFile)

	element := &cmd.NewTraefikConfiguration().Configuration

	generator.Generate(element)

	flats, err := encodeFn(element)
	if err != nil {
		logger.Fatal(err)
	}

	err = os.RemoveAll(outputFile)
	if err != nil {
		logger.Fatal(err)
	}

	file, err := os.OpenFile(outputFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		logger.Fatal(err)
	}

	defer file.Close()

	w := errWriter{w: file}

	w.writeln(`<!--
CODE GENERATED AUTOMATICALLY
THIS FILE MUST NOT BE EDITED BY HAND
-->`)
	w.writeln()

	for i, flat := range flats {
		// TODO must be move into the flats creation.
		if flat.Name == "experimental.plugins.<name>" || flat.Name == "TRAEFIK_EXPERIMENTAL_PLUGINS_<NAME>" {
			continue
		}

		if prefix == "" {
			w.writeln("`" + prefix + strings.ReplaceAll(flat.Name, "[0]", "_n") + "`:  ")
		} else {
			w.writeln("`" + prefix + strings.ReplaceAll(flat.Name, "[0]", "[n]") + "`:  ")
		}

		if flat.Default == "" {
			w.writeln(flat.Description)
		} else {
			w.writeln(flat.Description + " (Default: ```" + flat.Default + "```)")
		}

		if i < len(flats)-1 {
			w.writeln()
		}
	}

	if w.err != nil {
		logger.Fatal(err)
	}
}

type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) writeln(a ...interface{}) {
	if ew.err != nil {
		return
	}

	_, ew.err = fmt.Fprintln(ew.w, a...)
}

func genKVDynConfDoc(outputFile string) {
	dynConfPath := "./docs/content/reference/dynamic-configuration/file.toml"
	conf := map[string]interface{}{}
	_, err := toml.DecodeFile(dynConfPath, &conf)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}

	store := storeWriter{data: map[string]string{}}
	c := client{store: store}
	err = c.load("traefik", conf)
	if err != nil {
		log.Fatal(err)
	}

	var keys []string
	for k := range store.data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		_, _ = fmt.Fprintf(file, "| `%s` | `%s` |\n", k, store.data[k])
	}
}

type storeWriter struct {
	data map[string]string
}

func (f storeWriter) Put(key string, value []byte, options []string) error {
	f.data[key] = string(value)
	return nil
}

type client struct {
	store storeWriter
}

func (c client) load(parentKey string, conf map[string]interface{}) error {
	for k, v := range conf {
		switch entry := v.(type) {
		case map[string]interface{}:
			key := path.Join(parentKey, k)

			if len(entry) == 0 {
				err := c.store.Put(key, nil, nil)
				if err != nil {
					return err
				}
			} else {
				err := c.load(key, entry)
				if err != nil {
					return err
				}
			}
		case []map[string]interface{}:
			for i, o := range entry {
				key := path.Join(parentKey, k, strconv.Itoa(i))

				if err := c.load(key, o); err != nil {
					return err
				}
			}
		case []interface{}:
			for i, o := range entry {
				key := path.Join(parentKey, k, strconv.Itoa(i))

				err := c.store.Put(key, []byte(fmt.Sprintf("%v", o)), nil)
				if err != nil {
					return err
				}
			}
		default:
			key := path.Join(parentKey, k)

			err := c.store.Put(key, []byte(fmt.Sprintf("%v", v)), nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
