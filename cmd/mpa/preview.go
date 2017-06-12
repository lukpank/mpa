// Copyright 2017 Łukasz Pankowski <lukpank at o2 dot pl>. All rights
// reserved.  This source code is licensed under the terms of the MIT
// license. See LICENSE file for details.

package main

import (
	"database/sql"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/anthonynsimon/bild/transform"
	"github.com/nfnt/resize"
	"github.com/rwcarlsen/goexif/exif"
)

func (s *server) ServeImage(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/image/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if filename, ok := s.ensurePreview(w, r, id, ".1"); ok {
		http.ServeFile(w, r, filename)
	}
}

func (s *server) ServeAPIImage(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/api/image/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if _, ok := s.ensurePreview(w, r, id, ".1"); ok {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *server) ServePreview(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path, "/preview/")
	if err != nil {
		http.Error(w, s.tr("Page not found"), http.StatusNotFound)
		return
	}
	if filename, ok := s.ensurePreview(w, r, id, ".2"); ok {
		http.ServeFile(w, r, filename)
	}
}

func (s *server) ensurePreview(w http.ResponseWriter, r *http.Request, id int64, ext string) (string, bool) {
	var sha256sum string
	err := s.db.db.QueryRow("SELECT sha256sum FROM images where iid=?", id).Scan(&sha256sum)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, s.tr("Page not found"), http.StatusNotFound)
			return "", false
		}
		http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
		log.Println(err)
		return "", false
	}
	filename := filepath.Join(s.db.previewDir, sha256sum[:3], sha256sum[3:]+ext)
	if _, err := os.Stat(filename); err != nil {
		if !os.IsNotExist(err) {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return "", false
		}
		result := make(chan error)
		s.preview <- previewRequest{id, sha256sum, result}
		if err = <-result; err != nil {
			http.Error(w, s.tr("Internal server error"), http.StatusInternalServerError)
			log.Println(err)
			return "", false
		}
	}
	return filename, true
}

type previewRequest struct {
	id        int64
	sha256sum string
	result    chan<- error
}

type previewJob struct {
	id        int64
	sha256sum string
}

type previewResult struct {
	sha256sum string
	err       error
}

func (s *server) preparePreviews(jobs []previewJob) {
	result := make(chan error)
	for _, job := range jobs {
		s.preview <- previewRequest{job.id, job.sha256sum, result}
		if err := <-result; err != nil {
			log.Printf("preview %d (%s): %v", job.id, job.sha256sum[:7], err)
		}
	}
}

var ErrQuit = errors.New("quit")

func (s *server) previewMaster(workersCnt int) {
	m := make(map[string][]previewRequest)
	q := []previewJob{}
	requests := make(chan previewJob)
	results := make(chan previewResult)
	working := 0
	for i := 0; i < workersCnt; i++ {
		go s.previewWorker(results, requests)
	}
	addReq := func(req previewRequest) {
		s := m[req.sha256sum]
		if len(s) == 0 {
			q = append(q, previewJob{req.id, req.sha256sum})
		}
		m[req.sha256sum] = append(s, req)
		if len(q) > 0 && working < workersCnt {
			requests <- q[0]
			q = q[1:]
			working++
		}
	}
	handleResult := func(result previewResult) {
		working--
		for _, req := range m[result.sha256sum] {
			req.result <- result.err
		}
		delete(m, result.sha256sum)
	}
For:
	for {
		c := s.preview
		if len(q) > 4096 {
			c = nil
		}
		select {
		case req, ok := <-c:
			if !ok {
				break For
			}
			addReq(req)
		case r := <-results:
			handleResult(r)
			if len(q) > 0 && working < workersCnt {
				requests <- q[0]
				q = q[1:]
				working++
			}
		}
	}
	close(requests)
	for working > 0 {
		handleResult(<-results)
	}
	for _, s := range m {
		for _, req := range s {
			req.result <- ErrQuit
		}
	}
}

func (s *server) previewWorker(results chan<- previewResult, requests <-chan previewJob) {
	for req := range requests {
		log.Printf("creating preview for image %d (%s)\n", req.id, req.sha256sum[:7])
		results <- previewResult{req.sha256sum, s.createPreviews(req.sha256sum)}
	}
}

func (s *server) createPreviews(sha256sum string) error {
	dirName := filepath.Join(s.db.previewDir, sha256sum[:3])
	filename := filepath.Join(dirName, sha256sum[3:])
	filename1 := filename + ".1"
	filename2 := filename + ".2"
	exists := 0
	if _, err := os.Stat(filename1); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		exists++
	}
	if _, err := os.Stat(filename2); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		exists++
	}
	if exists == 2 {
		return nil
	}
	img, orientation, err := s.readImage(sha256sum)
	if err != nil {
		return err
	}

	if err := ensureDirExists(dirName, 0755); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	if err := s.createPreview(filename1, img, 1280, orientation); err != nil {
		return err
	}
	return s.createPreview(filename2, img, 320, orientation)
}

func (s *server) readImage(sha256sum string) (image.Image, int, error) {
	filename := filepath.Join(s.db.imagesDir, sha256sum[:3], sha256sum[3:])
	f, err := os.Open(filename)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	orientation, _ := exifOrientation(f)
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, 0, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, 0, err
	}
	return img, orientation, f.Close()
}

func (s *server) createPreview(filename string, img image.Image, maxSize uint, orientation int) error {
	var pw, ph uint
	size := img.Bounds().Size()
	if size.Y > size.X {
		pw = 0
		ph = maxSize
	} else {
		pw = maxSize
		ph = 0
	}
	img = applyOrientation(resize.Resize(pw, ph, img, resize.Lanczos3), orientation)
	f, err := ioutil.TempFile(s.db.uploadDir, "tmp")
	if err != nil {
		return err
	}
	tmpFileName := f.Name()
	defer func() {
		if tmpFileName != "" {
			_ = os.Remove(tmpFileName)
		}
	}()
	defer f.Close()
	if err := jpeg.Encode(f, img, nil); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpFileName, filename); err != nil {
		return err
	}
	tmpFileName = ""
	return nil
}

func exifOrientation(r io.Reader) (int, error) {
	x, err := exif.Decode(r)
	if err != nil {
		return 1, err
	}
	o, err := x.Get("Orientation")
	if err != nil {
		return 1, err
	}
	i, err := o.Int(0)
	if err != nil {
		return 1, err
	}
	return i, nil
}

func applyOrientation(img image.Image, orientation int) image.Image {
	o := &transform.RotationOptions{ResizeBounds: true, Pivot: &image.Point{0, 0}}
	switch orientation {
	default:
		return img
	case 1:
		return img
	case 2:
		return transform.FlipH(img)
	case 3:
		return transform.Rotate(img, 180, o)
	case 4:
		return transform.FlipV(img)
	case 5:
		return transform.FlipH(transform.Rotate(img, 90, o))
	case 6:
		return transform.Rotate(img, 90, o)
	case 7:
		return transform.FlipH(transform.Rotate(img, -90, o))
	case 8:
		return transform.Rotate(img, -90, o)
	}
}
