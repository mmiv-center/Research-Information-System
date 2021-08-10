all:
	cd src; go generate; cd ../
	env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`" -o build/linux-amd64/ror src/ror.go src/classify_dicom.go src/select_group.go && chmod +x build/linux-amd64/ror
	env GOOS=windows GOARCH=amd64 go build -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`" -o build/windows-amd64/ror.exe src/ror.go src/classify_dicom.go src/select_group.go
	env GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`" -o build/macos-amd64/ror src/ror.go src/classify_dicom.go src/select_group.go && chmod +x build/macos-amd64/ror
