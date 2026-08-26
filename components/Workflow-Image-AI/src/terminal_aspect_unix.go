// Code written 2021 by Hauke Bartsch.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build unix

package main

import (
	"syscall"
	"unsafe"
)

// terminalCellPixelSize returns the width and height (in pixels) of a single
// monospaced cell of the terminal attached to fd. It asks the kernel for the
// window size with the low-level TIOCGWINSZ ioctl, which reports both the cell
// count (ws_row x ws_col) and the window size in pixels (ws_xpixel x
// ws_ypixel); dividing one by the other gives the pixel size of one character.
//
// ok is false when fd is not a terminal (ENOTTY, e.g. a pipe), when the ioctl
// fails, or when the terminal emulator does not report pixel sizes — plain ptys
// (e.g. under `script`) and most SSH sessions leave ws_xpixel / ws_ypixel at
// zero. In all those cases the caller falls back to the assumed 1:2 cell
// aspect.
func terminalCellPixelSize(fd uintptr) (w, h int, ok bool) {
	var ws struct {
		Row    uint16 // ws_row
		Col    uint16 // ws_col
		Xpixel uint16 // ws_xpixel
		Ypixel uint16 // ws_ypixel
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0, 0, false
	}
	if ws.Row == 0 || ws.Col == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return 0, 0, false
	}
	return int(ws.Xpixel) / int(ws.Col), int(ws.Ypixel) / int(ws.Row), true
}
