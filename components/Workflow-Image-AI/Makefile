all:
	env GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/ src/rpp.go
	env GOOS=windows GOARCH=amd64 go build -o build/windows-amd64/ src/rpp.go
	env GOOS=darwin GOARCH=amd64 go build -o build/macos-amd64/ src/rpp.go
