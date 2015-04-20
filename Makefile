NAME = wikigo

all:
	make format
	make test
	make build

generate:
	cd src/web/; go-bindata -pkg="web" html/

format:
	find . -name "*.go" -not -path './Godeps/*' -type f -exec goimports -w=true {} \;

test:
	go test

build:
	go build -ldflags "-X main.BuildTime `date +%s` -X main.BuildHash `git describe --always`" -o "$(NAME)"

install:
	cp "$(NAME)" /usr/local/bin

uninstall:
	rm "/usr/local/bin/$(NAME)"

clean:
	rm "$(NAME)"
	rm *.pprof
	rm *.pdf

dependencies_get:
	go get golang.org/x/tools/cmd/goimports
	go get github.com/tools/godep
	go get github.com/alecthomas/gometalinter
	gometalinter --install
	go get -u github.com/jteeuwen/go-bindata/...

dependencies_save:
	godep save ./...

dependencies_restore:
	godep restore ./...

callgraph:
	go tool pprof --pdf "$(NAME)" cpu.pprof > callgraph.pdf

memograph:
	go tool pprof --pdf "$(NAME)" mem.pprof > memograph.pdf

bench:
	mkdir -p benchmarks/`git describe --always`/
	go test -test.benchmem=true -test.bench . 2> /dev/null | tee benchmarks/`git describe --always`/`date +%s`

coverage:
	rm -f coverage.out
	go test -coverprofile=coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o=/tmp/coverage.html

lint:
	rm -rf Godeps/
	gometalinter ./...; git checkout -- Godeps/
