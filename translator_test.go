package astikit

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestTranslator(t *testing.T) {
	// Setup
	tl := NewTranslator(TranslatorOptions{DefaultLanguage: "fr"})

	// Parse dir
	err := tl.ParseDir("testdata/translator")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if e := map[string]string{
		"en.1":       "1",
		"en.2.3":     "3",
		"en.d1.5":    "5",
		"en.d1.d2.6": "6",
		"fr.4":       "4",
	}; !reflect.DeepEqual(e, tl.p) {
		t.Errorf("expected %+v, got %+v", e, tl.p)
	}

	// Middleware
	var o string
	s := httptest.NewServer(ChainHTTPMiddlewares(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		o = tl.TranslateCtx(r.Context(), r.Header.Get("key"))
	}), tl.HTTPMiddleware))
	defer s.Close()

	// Translate
	for _, v := range []struct {
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
	} {
		r, err := http.NewRequest(http.MethodGet, s.URL, nil)
		if err != nil {
			t.Errorf("expected no error, got %+v", err)
		}
		r.Header.Set("key", v.key)
		if v.language != "" {
			r.Header.Set("Accept-Language", v.language)
		}
		_, err = http.DefaultClient.Do(r)
		if err != nil {
			t.Errorf("expected no error, got %+v", err)
		}
		if !reflect.DeepEqual(v.expected, o) {
			t.Errorf("expected %+v, got %+v", v.expected, o)
		}
	}
}

func TestTranslator_ParseAcceptLanguage(t *testing.T) {
	tl := NewTranslator(TranslatorOptions{ValidLanguages: []string{"en", "fr"}})
	if e, g := "", tl.parseAcceptLanguage(""); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
	if e, g := "fr", tl.parseAcceptLanguage(" fr-FR, fr ; q=0.9 ,en;q=0.7,en-US;q=0.8 "); !reflect.DeepEqual(e, g) {
		t.Errorf("expected %+v, got %+v", e, g)
	}
}
