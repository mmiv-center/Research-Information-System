// Code written 2021 by Hauke Bartsch.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package main

// terminalCellPixelSize has no TIOCGWINSZ ioctl equivalent on Windows, so the
// cell aspect cannot be measured; the caller falls back to the assumed 1:2
// (width:height) cell aspect.
func terminalCellPixelSize(fd uintptr) (w, h int, ok bool) {
	return 0, 0, false
}
