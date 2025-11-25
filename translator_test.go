package astikit

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestTranslator(t *testing.T) {
	// Setup
	tl := NewTranslator(TranslatorOptions{DefaultLanguage: "fr"})

	// Parse dir
	err := tl.ParseDir("testdata/translator")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if e := map[string]string{
		"en.1":       "1",
		"en.2.3":     "3",
		"en.d1.5":    "5",
		"en.d1.d2.6": "6",
		"en.f":       "f%sf",
		"fr.4":       "4",
	}; !reflect.DeepEqual(e, tl.p) {
		t.Fatalf("expected %+v, got %+v", e, tl.p)
	}

	// Middleware
	var o string
	s := httptest.NewServer(ChainHTTPMiddlewares(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var args []any
		if v := r.Header.Get("args"); v != "" {
			for _, s := range strings.Split(v, ",") {
				args = append(args, s)
			}
		}
		if len(args) > 0 {
			o = tl.TranslateCf(r.Context(), r.Header.Get("key"), args...)
		} else {
			o = tl.TranslateC(r.Context(), r.Header.Get("key"))
		}
	}), tl.HTTPMiddleware))
	defer s.Close()

	// Translate
	for _, v := range []struct {
		args     []string
		expected string
		key      string
		language string
	}{
		{
			expected: "4",
			key:      "4",
		},
		{
			expected: "fr.1",
			key:      "1",
		},
		{
			expected: "3",
			key:      "2.3",
			language: "en-US,en;q=0.8",
		},
		{
			expected: "4",
			key:      "4",
			language: "en",
		},
		{
			expected: "en.5",
			key:      "5",
			language: "en",
		},
		{
			expected: "6",
			key:      "d1.d2.6",
			language: "en",
		},
		{
			expected: "4",
			key:      "4",
			language: "it",
		},
		{
			args:     []string{"arg"},
			expected: "fargf",
			key:      "f",
			language: "en",
		},
	} {
		r, err := http.NewRequest(http.MethodGet, s.URL, nil)
		if err != nil {
			t.Fatalf("expected no error, got %+v", err)
		}
		if len(v.args) > 0 {
			r.Header.Set("args", strings.Join(v.args, ","))
		}
		r.Header.Set("key", v.key)
		if v.language != "" {
			r.Header.Set("Accept-Language", v.language)
		}
		_, err = http.DefaultClient.Do(r)
		if err != nil {
			t.Fatalf("expected no error, got %+v", err)
		}
		if !reflect.DeepEqual(v.expected, o) {
			t.Fatalf("expected %+v, got %+v", v.expected, o)
		}
	}

	// With language
	twl := tl.WithLanguage("en")
	if e, g := "1", twl.Translate("1"); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := "4", twl.Translate("4"); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := "en.a", twl.Translate("a"); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := "fvf", twl.Translatef("f", "v"); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
}

func TestTranslator_LanguageFromAcceptLanguageHeader(t *testing.T) {
	tl := NewTranslator(TranslatorOptions{
		DefaultLanguage: "en",
		ValidLanguages:  []string{"en", "fr"},
	})
	if e, g := "en", tl.LanguageFromAcceptLanguageHeader(""); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
	if e, g := "fr", tl.LanguageFromAcceptLanguageHeader(" fr-FR, fr ; q=0.9 ,en;q=0.7,en-US;q=0.8 "); !reflect.DeepEqual(e, g) {
		t.Fatalf("expected %+v, got %+v", e, g)
	}
}
