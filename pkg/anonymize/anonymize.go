package anonymize

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/mitchellh/copystructure"
	"github.com/traefik/traefik/v2/pkg/config/dynamic"
	"github.com/traefik/traefik/v2/pkg/tls"
	"mvdan.cc/xurls/v2"
)

const (
	maskShort = "xxxx"
	maskLarge = maskShort + maskShort + maskShort + maskShort + maskShort + maskShort + maskShort + maskShort
)

// Do sends configuration.
func Do(baseConfig interface{}, indent bool) (string, error) {
	anomConfig, err := copystructure.Copy(baseConfig)
	if err != nil {
		return "", err
	}

	val := reflect.ValueOf(anomConfig)

	err = doOnStruct(val)
	if err != nil {
		return "", err
	}

	configJSON, err := marshal(anomConfig, indent)
	if err != nil {
		return "", err
	}

	return doOnJSON(string(configJSON)), nil
}

func doOnJSON(input string) string {
	mailExp := regexp.MustCompile(`\w[-.\w]*\w@\w[-.\w]*\w\.\w{2,3}"`)
	return xurls.Relaxed().ReplaceAllString(mailExp.ReplaceAllString(input, maskLarge+"\""), maskLarge)
}

func doOnStruct(field reflect.Value) error {
	if field.Type().AssignableTo(reflect.TypeOf(dynamic.PluginConf{})) {
		resetPlugin(field)
		return nil
	}

	switch field.Kind() {
	case reflect.Ptr:
		if !field.IsNil() {
			if err := doOnStruct(field.Elem()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < field.NumField(); i++ {
			fld := field.Field(i)
			stField := field.Type().Field(i)
			if !isExported(stField) {
				continue
			}

			if stField.Tag.Get("export") == "true" {
				// A struct field cannot be set it must be filled as pointer.
				if fld.Kind() == reflect.Struct {
					fldPtr := reflect.New(fld.Type())
					fldPtr.Elem().Set(fld)

					if err := doOnStruct(fldPtr); err != nil {
						return err
					}

					fld.Set(fldPtr.Elem())

					continue
				}

				if err := doOnStruct(fld); err != nil {
					return err
				}
			} else if err := reset(fld, stField.Name); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range field.MapKeys() {
			val := field.MapIndex(key)

			// A struct value cannot be set it must be filled as pointer.
			if val.Kind() == reflect.Struct {
				valPtr := reflect.New(val.Type())
				valPtr.Elem().Set(val)

				if err := doOnStruct(valPtr); err != nil {
					return err
				}

				field.SetMapIndex(key, valPtr.Elem())

				continue
			}

			if err := doOnStruct(val); err != nil {
				return err
			}
		}
	case reflect.Slice:
		for j := 0; j < field.Len(); j++ {
			if err := doOnStruct(field.Index(j)); err != nil {
				return err
			}
		}
	}

	return nil
}

func reset(field reflect.Value, name string) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot reset field %s", name)
	}

	switch field.Kind() {
	case reflect.Ptr:
		if !field.IsNil() {
			field.Set(reflect.Zero(field.Type()))
		}
	case reflect.Struct:
		if field.IsValid() {
			field.Set(reflect.Zero(field.Type()))
		}
	case reflect.String:
		if field.String() != "" {
			if field.Type().AssignableTo(reflect.TypeOf(tls.FileOrContent(""))) {
				field.Set(reflect.ValueOf(tls.FileOrContent(maskShort)))
			} else {
				field.Set(reflect.ValueOf(maskShort))
			}
		}
	case reflect.Map:
		if field.Len() > 0 {
			field.Set(reflect.MakeMap(field.Type()))
		}
	case reflect.Slice:
		if field.Len() > 0 {
			switch field.Type().Elem().Kind() {
			case reflect.String:
				slice := reflect.MakeSlice(field.Type(), field.Len(), field.Len())
				for j := 0; j < field.Len(); j++ {
					slice.Index(j).SetString(maskShort)
				}
				field.Set(slice)
			default:
				field.Set(reflect.MakeSlice(field.Type(), 0, 0))
			}
		}
	case reflect.Interface:
		if !field.IsNil() {
			return reset(field.Elem(), "")
		}
	default:
		// Primitive type
		field.Set(reflect.Zero(field.Type()))
	}
	return nil
}

// resetPlugin resets the plugin configuration so it keep the plugin name but not its configuration.
func resetPlugin(field reflect.Value) {
	for _, key := range field.MapKeys() {
		field.SetMapIndex(key, reflect.ValueOf(struct{}{}))
	}
}

// isExported return true is a struct field is exported, else false.
func isExported(f reflect.StructField) bool {
	if f.PkgPath != "" && !f.Anonymous {
		return false
	}
	return true
}

func marshal(anomConfig interface{}, indent bool) ([]byte, error) {
	if indent {
		return json.MarshalIndent(anomConfig, "", "  ")
	}
	return json.Marshal(anomConfig)
}
