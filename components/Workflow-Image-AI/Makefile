all:	build/linux-amd64/ror	build/macos-amd64/ror	build/windows-amd64/ror.exe
	@echo "Done"
.PHONY : all

src/select_group.go: src/select_group.y
	cd src; go generate

build/linux-amd64/ror: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go
	env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`" -o build/linux-amd64/ror src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go
	chmod +x build/linux-amd64/ror

build/macos-amd64/ror: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go
	env GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`" -o build/macos-amd64/ror src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go
	chmod +x build/macos-amd64/ror

build/windows-amd64/ror.exe: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go
	env GOOS=windows GOARCH=amd64 go build -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`" -o build/windows-amd64/ror.exe src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go
