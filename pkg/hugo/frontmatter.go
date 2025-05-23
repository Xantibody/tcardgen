package hugo

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gohugoio/hugo/parser/pageparser"
	"github.com/pkg/errors"
)

const (
	fmTitle      = "title"
	fmAuthor     = "author"
	fmCategories = "categories"
	fmTags       = "tags"

	fmDate        = "date"        // priority high
	fmLastmod     = "lastmod"     // priority middle
	fmPublishDate = "publishDate" // priority low
)

var timeFormats = []string{
	time.RFC3339,
	time.DateOnly,
}

type FrontMatter struct {
	Title    string
	Author   string
	Category string
	Tags     []string
	Date     time.Time
}

// ParseFrontMatter parses the frontmatter of the specified Hugo content.
func ParseFrontMatter(w io.Writer, filename string, currentTime time.Time) (*FrontMatter, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseFrontMatter(w, file, currentTime)
}

func parseFrontMatter(w io.Writer, r io.Reader, currentTime time.Time) (*FrontMatter, error) {
	cfm, err := pageparser.ParseFrontMatterAndContent(r)
	if err != nil {
		return nil, err
	}

	fm := &FrontMatter{}
	if fm.Title, err = getString(&cfm, fmTitle); err != nil {
		return nil, err
	}
	if isArray := isArray(&cfm, fmAuthor); isArray {
		if fm.Author, err = getFirstStringItem(&cfm, fmAuthor); err != nil {
			fm.Author = "Anonymous"
		}
	} else {
		if fm.Author, err = getString(&cfm, fmAuthor); err != nil {
			fm.Author = "Anonymous"
		}
	}
	if isArray := isArray(&cfm, fmCategories); isArray {
		if fm.Category, err = getFirstStringItem(&cfm, fmCategories); err != nil {
			return nil, err
		}
	} else {
		if fm.Category, err = getString(&cfm, fmCategories); err != nil {
			return nil, err
		}
	}
	if fm.Tags, err = getAllStringItems(&cfm, fmTags); err != nil {
		return nil, err
	}
	if fm.Date, err = getContentDate(&cfm, currentTime); err != nil {
		var fe *FMNotExistError
		if errors.As(err, &fe) {
			fmt.Fprintf(w, "WARN: %s\n", err.Error())
			return fm, nil
		}
		return nil, err
	}

	return fm, nil
}

func getContentDate(cfm *pageparser.ContentFrontMatter, currentTime time.Time) (time.Time, error) {
	for _, key := range []string{fmDate, fmLastmod, fmPublishDate} {
		t, err := getTime(cfm, key, currentTime)
		if err != nil {
			switch err.(type) {
			case *FMNotExistError:
				continue
			}
		}
		return t, err
	}
	return currentTime, NewFMNotExistError(
		strings.Join([]string{fmDate, fmLastmod, fmPublishDate}, ", "))
}

func getTime(cfm *pageparser.ContentFrontMatter, fmKey string, currentTIme time.Time) (t time.Time, err error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return currentTIme, NewFMNotExistError(fmKey)
	}
	switch tstr := v.(type) {
	case string:
		for _, layout := range timeFormats {
			if t, err = time.Parse(layout, tstr); err == nil {
				return t, nil
			}
		}
		return currentTIme, fmt.Errorf("failed to parse time: %s, supported formats are %s", err, strings.Join(timeFormats, ", "))
	case time.Time:
		return tstr, nil
	default:
		return currentTIme, NewFMInvalidTypeError(fmKey, "time.Time or string", t)
	}
}

func getString(cfm *pageparser.ContentFrontMatter, fmKey string) (string, error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return "", NewFMNotExistError(fmKey)
	}

	switch s := v.(type) {
	case string:
		if s == "" {
			return "", NewFMNotExistError(fmKey)
		}
		return s, nil
	default:
		return "", NewFMInvalidTypeError(fmKey, "string", s)
	}
}

func getAllStringItems(cfm *pageparser.ContentFrontMatter, fmKey string) ([]string, error) {
	v, ok := cfm.FrontMatter[fmKey]
	if !ok {
		return nil, NewFMNotExistError(fmKey)
	}

	switch arr := v.(type) {
	case []interface{}:
		var strarr []string
		for _, item := range arr {
			switch s := item.(type) {
			case string:
				if s != "" {
					strarr = append(strarr, s)
				}
			default:
				return nil, NewFMInvalidTypeError(fmKey, "string", s)
			}
		}
		if len(strarr) < 1 {
			return nil, NewFMNotExistError(fmKey)
		}
		return strarr, nil

	default:
		return nil, NewFMInvalidTypeError(fmKey, "[]interface{}", arr)
	}
}

func getFirstStringItem(cfm *pageparser.ContentFrontMatter, fmKey string) (string, error) {
	arr, err := getAllStringItems(cfm, fmKey)
	if err != nil {
		return "", err
	}
	return arr[0], nil
}

func isArray(cfm *pageparser.ContentFrontMatter, fmKey string) bool {
	switch v := cfm.FrontMatter[fmKey]; v.(type) {
	case []interface{}:
		return true
	default:
		return false
	}
}
