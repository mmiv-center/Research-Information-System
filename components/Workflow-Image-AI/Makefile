all:	build/linux-amd64/ror	build/macos-amd64/ror	build/windows-amd64/ror.exe	build/macos-arm64/ror
	@echo "Done"
.PHONY : all

# debug information with -gcflags="all=-N -l"
GCFLAGS = 
LDFLAGS = -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`"
LDFLAGS_ARM = -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S`"
debug: GCFLAGS += -gcflags="all=-N -l"
debug: LDFLAGS_ARM = -ldflags "-X main.compileDate=`date -u +.%Y%m%d.%H%M%S` -compressdwarf=false"
debug: all

src/select_group.go: src/select_group.y
	cd src; go generate

build/linux-amd64/ror: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go src/SELECT_GRAMMAR.md
	env GOOS=linux GOARCH=amd64 go build $(GCFLAGS) $(LDFLAGS) -o build/linux-amd64/ror src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go
	chmod +x build/linux-amd64/ror

build/macos-amd64/ror: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go src/SELECT_GRAMMAR.md
	env GOOS=darwin GOARCH=amd64 go build $(GCFLAGS) $(LDFLAGS) -o build/macos-amd64/ror src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go
	chmod +x build/macos-amd64/ror

build/windows-amd64/ror.exe: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go src/SELECT_GRAMMAR.md
	env GOOS=windows GOARCH=amd64 go build $(GCFLAGS) $(LDFLAGS) -o build/windows-amd64/ror.exe src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go

build/macos-arm64/ror: src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go src/SELECT_GRAMMAR.md
	env GOOS=darwin GOARCH=arm64 go build $(GCFLAGS) $(LDFLAGS_ARM) -o build/macos-arm64/ror src/ror.go src/classify_dicom.go src/select_group.go src/status_tui.go src/annotate_tui.go src/mcp_server.go
