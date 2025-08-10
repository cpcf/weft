package render

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	var prevChar rune
	var prevWasUpper bool

	for i, char := range s {
		isUpper := unicode.IsUpper(char)
		isLetter := unicode.IsLetter(char)
		isDigit := unicode.IsDigit(char)

		if i > 0 && isUpper && !prevWasUpper && (unicode.IsLower(prevChar) || unicode.IsDigit(prevChar)) {
			result.WriteRune('_')
		}

		if i > 0 && isDigit && unicode.IsLetter(prevChar) {
			result.WriteRune('_')
		}

		if i > 0 && isLetter && unicode.IsDigit(prevChar) {
			result.WriteRune('_')
		}

		if isLetter || isDigit {
			result.WriteRune(unicode.ToLower(char))
		} else if char == ' ' || char == '-' {
			result.WriteRune('_')
		}

		prevChar = char
		prevWasUpper = isUpper
	}

	return strings.Trim(result.String(), "_")
}

func toCamelCase(s string) string {
	if s == "" {
		return ""
	}

	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])
	for _, word := range words[1:] {
		if len(word) > 0 {
			result += strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return result
}

func toPascalCase(s string) string {
	if s == "" {
		return ""
	}

	words := splitWords(s)
	var result strings.Builder

	for _, word := range words {
		if len(word) > 0 {
			result.WriteString(strings.ToUpper(string(word[0])) + strings.ToLower(word[1:]))
		}
	}

	return result.String()
}

func toKebabCase(s string) string {
	if s == "" {
		return ""
	}

	words := splitWords(s)
	var result []string

	for _, word := range words {
		if len(word) > 0 {
			result = append(result, strings.ToLower(word))
		}
	}

	return strings.Join(result, "-")
}

func splitWords(s string) []string {
	if s == "" {
		return nil
	}

	var words []string
	var current strings.Builder
	var prevChar rune
	var prevWasUpper bool

	for i, char := range s {
		isUpper := unicode.IsUpper(char)
		isLetter := unicode.IsLetter(char)
		isDigit := unicode.IsDigit(char)

		if char == ' ' || char == '_' || char == '-' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		} else if i > 0 && isUpper && !prevWasUpper && (unicode.IsLower(prevChar) || unicode.IsDigit(prevChar)) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(char)
		} else if i > 0 && isLetter && unicode.IsDigit(prevChar) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(char)
		} else if i > 0 && isDigit && unicode.IsLetter(prevChar) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			current.WriteRune(char)
		} else if isLetter || isDigit {
			current.WriteRune(char)
		}

		prevChar = char
		prevWasUpper = isUpper
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

func pluralize(word string) string {
	if word == "" {
		return ""
	}

	word = strings.ToLower(word)

	irregularPlurals := map[string]string{
		"person":     "people",
		"man":        "men",
		"woman":      "women",
		"child":      "children",
		"tooth":      "teeth",
		"foot":       "feet",
		"mouse":      "mice",
		"goose":      "geese",
		"ox":         "oxen",
		"datum":      "data",
		"medium":     "media",
		"criterion":  "criteria",
		"phenomenon": "phenomena",
	}

	if plural, exists := irregularPlurals[word]; exists {
		return plural
	}

	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	if strings.HasSuffix(word, "y") && len(word) > 1 {
		if !strings.ContainsRune("aeiou", rune(word[len(word)-2])) {
			return word[:len(word)-1] + "ies"
		}
	}

	if strings.HasSuffix(word, "f") {
		return word[:len(word)-1] + "ves"
	}

	if strings.HasSuffix(word, "fe") {
		return word[:len(word)-2] + "ves"
	}

	return word + "s"
}

func singularize(word string) string {
	if word == "" {
		return ""
	}

	word = strings.ToLower(word)

	irregularSingulars := map[string]string{
		"people":    "person",
		"men":       "man",
		"women":     "woman",
		"children":  "child",
		"teeth":     "tooth",
		"feet":      "foot",
		"mice":      "mouse",
		"geese":     "goose",
		"oxen":      "ox",
		"data":      "datum",
		"media":     "medium",
		"criteria":  "criterion",
		"phenomena": "phenomenon",
	}

	if singular, exists := irregularSingulars[word]; exists {
		return singular
	}

	if strings.HasSuffix(word, "ies") && len(word) > 3 {
		return word[:len(word)-3] + "y"
	}

	if strings.HasSuffix(word, "ves") && len(word) > 3 {
		return word[:len(word)-3] + "f"
	}

	if strings.HasSuffix(word, "es") && len(word) > 2 {
		base := word[:len(word)-2]
		if strings.HasSuffix(base, "s") || strings.HasSuffix(base, "x") ||
			strings.HasSuffix(base, "z") || strings.HasSuffix(base, "ch") ||
			strings.HasSuffix(base, "sh") {
			return base
		}
	}

	if strings.HasSuffix(word, "s") && len(word) > 1 {
		return word[:len(word)-1]
	}

	return word
}

func humanize(s string) string {
	if s == "" {
		return ""
	}

	words := splitWords(s)
	var result []string

	for _, word := range words {
		if len(word) > 0 {
			result = append(result, strings.ToUpper(string(word[0]))+strings.ToLower(word[1:]))
		}
	}

	return strings.Join(result, " ")
}

func indentLines(text string, indent int) string {
	if text == "" {
		return ""
	}

	indentStr := strings.Repeat(" ", indent)
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indentStr + line
		}
	}

	return strings.Join(lines, "\n")
}

func quote(s string) string {
	return fmt.Sprintf(`"%s"`, s)
}

func singleQuote(s string) string {
	return fmt.Sprintf("'%s'", s)
}

func comment(text, prefix string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = prefix + " " + line
		}
	}

	return strings.Join(lines, "\n")
}

func goComment(text string) string {
	return comment(text, "//")
}

func generateUUID() string {
	return uuid.New().String()
}

func calculateMD5(text string) string {
	hash := md5.Sum([]byte(text))
	return fmt.Sprintf("%x", hash)
}

func calculateSHA1(text string) string {
	hash := sha1.Sum([]byte(text))
	return fmt.Sprintf("%x", hash)
}

func calculateSHA256(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash)
}

func encodeBase64(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

func decodeBase64(text string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

func urlPathEscape(s string) string {
	return url.PathEscape(s)
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

func yamlEscape(s string) string {
	if needsQuoting(s) {
		return quote(s)
	}
	return s
}

func needsQuoting(s string) bool {
	if s == "" {
		return true
	}

	specialChars := ":{}[]|>*&!%#`@,"
	if strings.ContainsAny(s, specialChars) {
		return true
	}

	if strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return true
	}

	return false
}

func htmlEscape(s string) string {
	return html.EscapeString(s)
}

func cssEscape(s string) string {
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteString(fmt.Sprintf("\\%X", r))
		}
	}
	return result.String()
}

func jsEscape(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			result.WriteString("\\\\")
		case '"':
			result.WriteString("\\\"")
		case '\'':
			result.WriteString("\\'")
		case '\n':
			result.WriteString("\\n")
		case '\r':
			result.WriteString("\\r")
		case '\t':
			result.WriteString("\\t")
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

func getEnvVar(name string) string {
	return os.Getenv(name)
}

func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}

func hasEnvVar(name string) bool {
	_, exists := os.LookupEnv(name)
	return exists
}

func regexMatch(pattern, text string) (bool, error) {
	return regexp.MatchString(pattern, text)
}

func regexReplace(pattern, replacement, text string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllString(text, replacement), nil
}

func regexSplit(pattern, text string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.Split(text, -1), nil
}

func regexFind(pattern, text string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.FindAllString(text, -1), nil
}

func pathJoin(paths ...string) string {
	return filepath.Join(paths...)
}

func pathBase(path string) string {
	return filepath.Base(path)
}

func pathDir(path string) string {
	return filepath.Dir(path)
}

func pathExt(path string) string {
	return filepath.Ext(path)
}

func pathClean(path string) string {
	return filepath.Clean(path)
}

func pathIsAbs(path string) bool {
	return filepath.IsAbs(path)
}

func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

	rand.Seed(time.Now().UnixNano())
	password := make([]byte, length)

	for i := range password {
		password[i] = charset[rand.Intn(len(charset))]
	}

	return string(password)
}

func randomInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	rand.Seed(time.Now().UnixNano())
	result := make([]byte, length)

	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var result strings.Builder
	var currentLine strings.Builder
	lineLength := 0

	for _, word := range words {
		wordLength := utf8.RuneCountInString(word)

		if lineLength > 0 && lineLength+wordLength+1 > width {
			result.WriteString(currentLine.String())
			result.WriteRune('\n')
			currentLine.Reset()
			lineLength = 0
		}

		if lineLength > 0 {
			currentLine.WriteRune(' ')
			lineLength++
		}

		currentLine.WriteString(word)
		lineLength += wordLength
	}

	if currentLine.Len() > 0 {
		result.WriteString(currentLine.String())
	}

	return result.String()
}

func truncateString(s string, length int) string {
	if length <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= length {
		return s
	}

	if length <= 3 {
		return string(runes[:length])
	}

	return string(runes[:length-3]) + "..."
}

func centerString(s string, width int) string {
	if width <= 0 {
		return s
	}

	length := utf8.RuneCountInString(s)
	if length >= width {
		return s
	}

	padding := width - length
	leftPad := padding / 2
	rightPad := padding - leftPad

	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

func padString(s string, width int) string {
	return padStringLeft(s, width)
}

func padStringLeft(s string, width int) string {
	length := utf8.RuneCountInString(s)
	if length >= width {
		return s
	}
	return strings.Repeat(" ", width-length) + s
}

func padStringRight(s string, width int) string {
	length := utf8.RuneCountInString(s)
	if length >= width {
		return s
	}
	return s + strings.Repeat(" ", width-length)
}

func parseSemver(version string) map[string]any {
	re := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	matches := re.FindStringSubmatch(version)

	if len(matches) < 4 {
		return map[string]any{
			"major":      0,
			"minor":      0,
			"patch":      0,
			"prerelease": "",
			"metadata":   "",
		}
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	prerelease := ""
	metadata := ""

	if len(matches) > 4 && matches[4] != "" {
		prerelease = matches[4]
	}
	if len(matches) > 5 && matches[5] != "" {
		metadata = matches[5]
	}

	return map[string]any{
		"major":      major,
		"minor":      minor,
		"patch":      patch,
		"prerelease": prerelease,
		"metadata":   metadata,
	}
}

func semverMajor(version string) int {
	parsed := parseSemver(version)
	return parsed["major"].(int)
}

func semverMinor(version string) int {
	parsed := parseSemver(version)
	return parsed["minor"].(int)
}

func semverPatch(version string) int {
	parsed := parseSemver(version)
	return parsed["patch"].(int)
}

func semverCompare(v1, v2 string) int {
	p1 := parseSemver(v1)
	p2 := parseSemver(v2)

	major1 := p1["major"].(int)
	major2 := p2["major"].(int)
	if major1 != major2 {
		return major1 - major2
	}

	minor1 := p1["minor"].(int)
	minor2 := p2["minor"].(int)
	if minor1 != minor2 {
		return minor1 - minor2
	}

	patch1 := p1["patch"].(int)
	patch2 := p2["patch"].(int)
	if patch1 != patch2 {
		return patch1 - patch2
	}

	pre1 := p1["prerelease"].(string)
	pre2 := p2["prerelease"].(string)

	if pre1 == "" && pre2 != "" {
		return 1
	}
	if pre1 != "" && pre2 == "" {
		return -1
	}
	if pre1 != pre2 {
		return strings.Compare(pre1, pre2)
	}

	return 0
}
