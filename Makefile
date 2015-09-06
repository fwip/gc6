default: gc6

gc6 : $(shell find -name '*.go' )
	go build
