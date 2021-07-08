all:
	env GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/rpp src/rpp.go src/classify_dicom.go && chmod +x build/linux-amd64/rpp
	env GOOS=windows GOARCH=amd64 go build -o build/windows-amd64/rpp.exe src/rpp.go src/classify_dicom.go
	env GOOS=darwin GOARCH=amd64 go build -o build/macos-amd64/rpp src/rpp.go src/classify_dicom.go && chmod +x build/linux-amd64/rpp
