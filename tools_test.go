package toolkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testingTools Tools

	s := testingTools.RandomString(10)

	if len(s) != 10 {
		t.Error("Wrong Length of random string")
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{
		name:          "Allowed No Rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    false,
		errorExpected: false,
	},
	{
		name:          "Allowed Rename",
		allowedTypes:  []string{"image/jpeg", "image/png"},
		renameFile:    true,
		errorExpected: false,
	},
	{
		name:          "Not Allowed",
		allowedTypes:  []string{"image/png"},
		renameFile:    false,
		errorExpected: true,
	},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		//	set up a pipe to avoid buffering ( for file uploads )
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			//	create form data field "file"
			part, err := writer.CreateFormFile("file", "./testdata/img.jpg")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.jpg")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("Error Decoding Image")
			}

			err = jpeg.Encode(part, img, &jpeg.Options{Quality: 100})
			if err != nil {
				t.Error(err)
			}
		}()

		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			//	cleanup
			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	//	set up a pipe to avoid buffering ( for file uploads )
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		//	create form data field "file"
		part, err := writer.CreateFormFile("file", "./testdata/img.jpg")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.jpg")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("Error Decoding Image")
		}

		err = jpeg.Encode(part, img, &jpeg.Options{Quality: 100})
		if err != nil {
			t.Error(err)
		}
	}()

	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools
	testTools.AllowedFileTypes = []string{"image/jpeg", "image/png"}

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads", true)
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	//	cleanup
	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName))

}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testingTools Tools

	err := testingTools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testingTools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name          string
	s             string
	expected      string
	errorExpected bool
}{
	{
		name:          "valid string",
		s:             "now is the time",
		expected:      "now-is-the-time",
		errorExpected: false,
	},
	{
		name:          "empty string",
		s:             "",
		errorExpected: true,
	},
	{
		name:          "complex string",
		s:             "Now is the time for all GOOD men! + fish & such &^123",
		expected:      "now-is-the-time-for-all-good-men-fish-such-123",
		errorExpected: false,
	},
	{
		name:          "japanese string",
		s:             "こんにちは世界",
		errorExpected: true,
	},
	{
		name:          "japanese string and roman character",
		s:             "hello world -> こんにちは世界",
		expected:      "hello-world",
		errorExpected: false,
	},
}

func TestTools_Slugify(t *testing.T) {
	var testingTools Tools

	for _, e := range slugTests {
		slug, err := testingTools.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err)
			break
		}

		if slug != e.expected && !e.errorExpected {
			t.Errorf("%s: wrong slug generated. expected %s but got %s", e.name, e.expected, slug)
			break
		}

		if e.errorExpected && err == nil {
			t.Errorf("%s: expected error but got nothing", e.name)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTools Tools
	testTools.DownloadStaticFile(rr, req, "testdata", "pic.jpg", "rowdy-cat.jpg")

	res := rr.Result()
	defer res.Body.Close()

	const fileSize = "91585"

	if res.Header["Content-Length"][0] != fileSize {
		t.Errorf("wrong content length, expected %s but got %s", res.Header["Content-Length"][0], fileSize)
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"rowdy-cat.jpg\"" {
		t.Error("wrong content disposition")
	}

	if _, err := io.ReadAll(res.Body); err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int
	allowUnknown  bool
}{
	{
		name:          "good json",
		json:          `{"foo": "bar"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "badly formatted json",
		json:          `{"foo":}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "incorrect type",
		json:          `{"foo": 3}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "two json files",
		json:          `{"foo": "bar"}{"alpha": "beta"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "empty body",
		json:          ``,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "syntax error in JSON",
		json:          `{"foo": "1}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "unknown field in JSON",
		json:          `{"fooo": "bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  false,
	},
	{
		name:          "allow unknown fields in JSON",
		json:          `{"fooo": "bar"}`,
		errorExpected: false,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "missing field name",
		json:          `{jack: "bar"}`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  true,
	},
	{
		name:          "file too large",
		json:          `{"foo": "bar"}`,
		errorExpected: true,
		maxSize:       5,
		allowUnknown:  true,
	},
	{
		name:          "not json",
		json:          `Hello World`,
		errorExpected: true,
		maxSize:       1024,
		allowUnknown:  true,
	},
}

func TestTools_ReadJSON(t *testing.T) {
	var testTools Tools

	for _, e := range jsonTests {
		//	set the max file size and allow/disallow unknown fields
		testTools.MaxJsonSize = e.maxSize
		testTools.AllowUnknownFields = e.allowUnknown

		//	declare a variable to read the decode json into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Error(err)
		}

		rr := httptest.NewRecorder()

		err = testTools.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but received: %s", e.name, err)
		}

		if e.errorExpected {
			log.Printf("%s: %s", e.name, err)
		}

		if !e.errorExpected {
			log.Printf("%s: %s", e.name, decodedJSON)
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "Booo",
	}

	headers := make(http.Header)
	headers.Add("Foo", "Bar")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	err := testTools.ErrorJSON(rr, fmt.Errorf("some error"), http.StatusServiceUnavailable)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Error("received error while decoding json", err)
	}

	if !payload.Error {
		t.Error("error set to false while it should be true")
	}

	if rr.Code != http.StatusServiceUnavailable {
		t.Error("expected 503 error code but got ", rr.Code)
	}
}

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_PushJSONToRemote(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}
	})

	var testTools Tools

	var foo struct {
		Bar string `json:"bar"`
	}
	foo.Bar = "bar"
	_, _, err := testTools.PushJSONToRemote("some_url", foo, client)
	if err != nil {
		t.Error("failed to push to remote url: ", err)
	}
}
