package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

const secret = "bM+X2Y$^%q_D=c^Sw!LM5=d!Y+aS_$zU"

func computeHmac1(message string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func decodeImg(contentType string, r io.Reader) (image.Image, error) {
	if contentType == "image/jpeg" {
		img, err := jpeg.Decode(r)

		if err != nil {
			return nil, err
		}

		return img, nil

	} else if contentType == "image/png" {
		img, err := png.Decode(r)

		if err != nil {
			return nil, err
		}

		return img, nil

	} else {
		return nil, errors.New("unknow content type")
	}
}

func encodeImg(contentType string, w io.Writer, m image.Image, q int) {
	if contentType == "image/jpeg" {
		jpeg.Encode(w, m, &jpeg.Options{Quality: q})

	} else if contentType == "image/png" {
		png.Encode(w, m)
	}
}

func resizeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	nwidth, _ := strconv.ParseUint(vars["nwidth"], 10, 32)
	nheight, _ := strconv.ParseUint(vars["nheight"], 10, 32)
	source := strings.Split(vars["source"], ".")

	if len(source) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	url, err := base64.StdEncoding.DecodeString(source[0])

	if err != nil || len(url) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := source[1]

	resp, err := http.Get(string(url))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if hash != computeHmac1(string(body)) {
		log.Println(computeHmac1(string(body)))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	contentType := http.DetectContentType(body)

	img, err := decodeImg(contentType, bytes.NewReader(body))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	m := resize.Resize(uint(nwidth), uint(nheight), img, resize.Lanczos3)

	encodeImg(contentType, w, m, 100)
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/{nwidth:[0-9]+}/{nheight:[0-9]+}/{source}", resizeHandler)

	log.Fatal(http.ListenAndServe(":8000", r))
}
