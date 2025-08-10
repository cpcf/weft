package render

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"
)

func DefaultFuncMap() template.FuncMap {
	funcs := template.FuncMap{
		"snake":      toSnakeCase,
		"camel":      toCamelCase,
		"pascal":     toPascalCase,
		"kebab":      toKebabCase,
		"lower":      strings.ToLower,
		"upper":      strings.ToUpper,
		"title":      strings.Title,
		"trim":       strings.TrimSpace,
		"trimLeft":   strings.TrimLeft,
		"trimRight":  strings.TrimRight,
		"trimPrefix": strings.TrimPrefix,
		"trimSuffix": strings.TrimSuffix,
		"replace":    strings.ReplaceAll,
		"split":      strings.Split,
		"join":       strings.Join,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"repeat":     strings.Repeat,

		"formatSlice": formatSlice,
		"filter":      filterSlice,
		"map":         mapSlice,
		"first":       getFirst,
		"last":        getLast,
		"rest":        getRest,
		"reverse":     reverseSlice,
		"sort":        sortSlice,
		"unique":      uniqueSlice,
		"len":         getLength,
		"isEmpty":     isEmpty,
		"isNotEmpty":  isNotEmpty,

		"plural":    pluralize,
		"singular":  singularize,
		"humanize":  humanize,
		"indent":    indentLines,
		"quote":     quote,
		"squote":    singleQuote,
		"comment":   comment,
		"goComment": goComment,

		"add":      add,
		"subtract": subtract,
		"multiply": multiply,
		"divide":   divide,
		"mod":      modulo,
		"max":      maximum,
		"min":      minimum,
		"abs":      absolute,

		"now":         time.Now,
		"formatTime":  formatTime,
		"parseTime":   parseTime,
		"toUnix":      toUnixTime,
		"fromUnix":    fromUnixTime,
		"addDuration": addDuration,
		"subDuration": subtractDuration,

		"default":  defaultValue,
		"coalesce": coalesce,
		"ternary":  ternary,
		"isNil":    isNil,
		"isNotNil": isNotNil,
		"toString": toString,
		"toInt":    toInt,
		"toBool":   toBool,
		"typeOf":   typeOf,
		"kindOf":   kindOf,
	}

	return funcs
}

func ExtendedFuncMap() template.FuncMap {
	funcs := DefaultFuncMap()

	extended := template.FuncMap{
		"uuid":       generateUUID,
		"md5":        calculateMD5,
		"sha1":       calculateSHA1,
		"sha256":     calculateSHA256,
		"base64":     encodeBase64,
		"base64dec":  decodeBase64,
		"urlQuery":   urlQueryEscape,
		"urlPath":    urlPathEscape,
		"jsonEscape": jsonEscape,
		"yamlEscape": yamlEscape,
		"htmlEscape": htmlEscape,
		"cssEscape":  cssEscape,
		"jsEscape":   jsEscape,

		"env":       getEnvVar,
		"expandEnv": expandEnvVars,
		"hasEnv":    hasEnvVar,

		"regexMatch":   regexMatch,
		"regexReplace": regexReplace,
		"regexSplit":   regexSplit,
		"regexFind":    regexFind,

		"pathJoin":  pathJoin,
		"pathBase":  pathBase,
		"pathDir":   pathDir,
		"pathExt":   pathExt,
		"pathClean": pathClean,
		"pathIsAbs": pathIsAbs,

		"semver":        parseSemver,
		"semverMajor":   semverMajor,
		"semverMinor":   semverMinor,
		"semverPatch":   semverPatch,
		"semverCompare": semverCompare,

		"genPassword": generatePassword,
		"randInt":     randomInt,
		"randString":  randomString,
		"shuffle":     shuffleSlice,

		"wrap":     wrapText,
		"truncate": truncateString,
		"center":   centerString,
		"pad":      padString,
		"padLeft":  padStringLeft,
		"padRight": padStringRight,
	}

	maps.Copy(funcs, extended)

	return funcs
}

func defaultValue(def any, given any) any {
	if given == nil {
		return def
	}

	if reflect.ValueOf(given).Kind() == reflect.String {
		if given.(string) == "" {
			return def
		}
	}

	return given
}

func coalesce(values ...any) any {
	for _, v := range values {
		if v != nil {
			if reflect.ValueOf(v).Kind() == reflect.String && v.(string) != "" {
				return v
			} else if reflect.ValueOf(v).Kind() != reflect.String {
				return v
			}
		}
	}
	return nil
}

func ternary(condition bool, trueVal, falseVal any) any {
	if condition {
		return trueVal
	}
	return falseVal
}

func isNil(value any) bool {
	return value == nil || reflect.ValueOf(value).IsNil()
}

func isNotNil(value any) bool {
	return !isNil(value)
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%v", value)
}

func toInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

func toBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	case int:
		return v != 0
	case float64:
		return v != 0
	default:
		return false
	}
}

func typeOf(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}

func kindOf(value any) string {
	if value == nil {
		return "invalid"
	}
	return reflect.ValueOf(value).Kind().String()
}

func getLength(value any) int {
	if value == nil {
		return 0
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return v.Len()
	default:
		return 0
	}
}

func isEmpty(value any) bool {
	return getLength(value) == 0
}

func isNotEmpty(value any) bool {
	return !isEmpty(value)
}

func add(a, b any) (any, error) {
	return performMath(a, b, func(x, y float64) float64 { return x + y })
}

func subtract(a, b any) (any, error) {
	return performMath(a, b, func(x, y float64) float64 { return x - y })
}

func multiply(a, b any) (any, error) {
	return performMath(a, b, func(x, y float64) float64 { return x * y })
}

func divide(a, b any) (any, error) {
	return performMath(a, b, func(x, y float64) float64 { return x / y })
}

func modulo(a, b any) (any, error) {
	aInt, err := toInt(a)
	if err != nil {
		return nil, err
	}
	bInt, err := toInt(b)
	if err != nil {
		return nil, err
	}
	return aInt % bInt, nil
}

func performMath(a, b any, op func(float64, float64) float64) (any, error) {
	aFloat, err := toFloat64(a)
	if err != nil {
		return nil, err
	}
	bFloat, err := toFloat64(b)
	if err != nil {
		return nil, err
	}
	result := op(aFloat, bFloat)

	if result == float64(int64(result)) {
		return int64(result), nil
	}
	return result, nil
}

func toFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func maximum(values ...any) (any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("max requires at least one argument")
	}

	max := values[0]
	maxFloat, err := toFloat64(max)
	if err != nil {
		return nil, err
	}

	for _, v := range values[1:] {
		vFloat, err := toFloat64(v)
		if err != nil {
			return nil, err
		}
		if vFloat > maxFloat {
			max = v
			maxFloat = vFloat
		}
	}

	return max, nil
}

func minimum(values ...any) (any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("min requires at least one argument")
	}

	min := values[0]
	minFloat, err := toFloat64(min)
	if err != nil {
		return nil, err
	}

	for _, v := range values[1:] {
		vFloat, err := toFloat64(v)
		if err != nil {
			return nil, err
		}
		if vFloat < minFloat {
			min = v
			minFloat = vFloat
		}
	}

	return min, nil
}

func absolute(value any) (any, error) {
	f, err := toFloat64(value)
	if err != nil {
		return nil, err
	}
	if f < 0 {
		f = -f
	}

	if f == float64(int64(f)) {
		return int64(f), nil
	}
	return f, nil
}

func formatTime(t time.Time, layout string) string {
	return t.Format(layout)
}

func parseTime(layout, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

func toUnixTime(t time.Time) int64 {
	return t.Unix()
}

func fromUnixTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

func addDuration(t time.Time, duration string) (time.Time, error) {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return time.Time{}, err
	}
	return t.Add(d), nil
}

func subtractDuration(t time.Time, duration string) (time.Time, error) {
	d, err := time.ParseDuration(duration)
	if err != nil {
		return time.Time{}, err
	}
	return t.Add(-d), nil
}
