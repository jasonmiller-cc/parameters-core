package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"
)

// ApplyEnv walks the struct pointed to by dst, looks for `env:"VAR"` tags,
// and overwrites the field with the corresponding environment variable value
// when one is set. Only basic types (string, int, bool) are handled.
func ApplyEnv(prefix string, dst any) {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	applyEnvValue(prefix, v.Elem())
}

func applyEnvValue(prefix string, v reflect.Value) {
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		fv := v.Field(i)

		if !fv.CanSet() {
			continue
		}

		if f.Type.Kind() == reflect.Struct {
			applyEnvValue(prefix, fv)
			continue
		}

		tag := f.Tag.Get("env")
		if tag == "" {
			continue
		}

		key := strings.ToUpper(prefix) + "_" + strings.ToUpper(tag)
		val, ok := os.LookupEnv(key)
		if !ok {
			// Also try without prefix.
			val, ok = os.LookupEnv(strings.ToUpper(tag))
		}
		if !ok {
			continue
		}

		switch f.Type.Kind() {
		case reflect.String:
			fv.SetString(val)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				fv.SetInt(n)
			}
		case reflect.Bool:
			if b, err := strconv.ParseBool(val); err == nil {
				fv.SetBool(b)
			}
		}
	}
}
