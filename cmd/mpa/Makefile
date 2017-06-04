GOFILES := $(wildcard *.go)
ASSETSFILES := $(wildcard static/*.css static/*.js static/*.png templates/*.html)
REV := $(shell git log -1 --format=%h)

mpa: $(GOFILES) embedded-assets.go
	go build -tags embedded -ldflags '-X main.Version=mpa-0.1-$(REV)'

embedded-assets.go: $(ASSETSFILES)
	@which esc > /dev/null || ( echo "error: no 'esc' found: you probably need to 'go get github.com/mjibson/esc'" && ! : )
	( echo '// +build embedded'; echo; esc -ignore '.*~' static templates ) > embedded-assets.go
