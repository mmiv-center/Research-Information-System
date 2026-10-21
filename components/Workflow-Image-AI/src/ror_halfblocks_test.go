package main

import (
	"image"
	"os"
	"strings"
	"testing"
)

func makeGray16(w, h int, val func(x, y int) uint16) *image.Gray16 {
	img := image.NewGray16(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := val(x, y)
			i := img.PixOffset(x, y)
			img.Pix[i] = byte(v >> 8)
			img.Pix[i+1] = byte(v)
		}
	}
	return img
}

func TestHalfBlocksTruecolor(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	// 2 wide x 4 tall; top two rows black, bottom two rows white.
	img := makeGray16(2, 4, func(x, y int) uint16 {
		if y < 2 {
			return 0
		}
		return 65535
	})
	out := printImage2HalfBlocks(img, nil, 32768)
	if !strings.Contains(out, "▀") {
		t.Fatalf("expected half-block char, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;2;0;0;0m") {
		t.Fatalf("expected black fg escape, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;2;255;255;255m") {
		t.Fatalf("expected white fg escape, got %q", out)
	}
	// exactly two terminal rows
	if got := strings.Count(out, "\n"); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}
}

func TestHalfBlocks256color(t *testing.T) {
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	img := makeGray16(2, 4, func(x, y int) uint16 {
		if y < 2 {
			return 0
		}
		return 65535
	})
	out := printImage2HalfBlocks(img, nil, 32768)
	if !strings.Contains(out, "\x1b[38;5;232m") {
		t.Fatalf("expected 256-color black, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;5;255m") {
		t.Fatalf("expected 256-color white, got %q", out)
	}
	if strings.Contains(out, "38;2;") {
		t.Fatalf("should not emit truecolor in 256 mode: %q", out)
	}
}

func TestHalfBlocksPadding(t *testing.T) {
	t.Setenv("COLORTERM", "truecolor")
	// left column is padding (value 100), right column is bright.
	img := makeGray16(2, 2, func(x, y int) uint16 {
		if x == 0 {
			return 100
		}
		return 65535
	})
	out := printImage2HalfBlocks(img, nil, 100)
	if !strings.Contains(out, "\x1b[0m ") {
		t.Fatalf("expected a blank padding cell, got %q", out)
	}
}

func TestRunesHalfBlocks(t *testing.T) {
	// 2 wide x 4 tall; top two rows black, bottom two rows white.
	img := makeGray16(2, 4, func(x, y int) uint16 {
		if y < 2 {
			return 0
		}
		return 65535
	})
	out := printImage2RunesHalfBlocks(img, nil, 32768)
	if !strings.Contains(out, "▀") {
		t.Fatalf("expected half-block glyph, got %q", out)
	}
	if !strings.Contains(out, "[#000000:#000000]▀") {
		t.Fatalf("expected black half-block tag, got %q", out)
	}
	if !strings.Contains(out, "[#ffffff:#ffffff]▀") {
		t.Fatalf("expected white half-block tag, got %q", out)
	}
	if got := strings.Count(out, "\n"); got != 2 {
		t.Fatalf("expected 2 rows, got %d", got)
	}
}

func TestRunesHalfBlocksPadding(t *testing.T) {
	// left column padding (value 7); right column dark on top, light below —
	// verifying a single cell carries two different pixel levels.
	img := makeGray16(2, 2, func(x, y int) uint16 {
		if x == 0 {
			return 7 // padding
		}
		if y == 0 {
			return 0 // dark (top)
		}
		return 65535 // light (bottom)
	})
	out := printImage2RunesHalfBlocks(img, nil, 7)
	if !strings.Contains(out, "[-:-] ") {
		t.Fatalf("expected a blank padding cell, got %q", out)
	}
	if !strings.Contains(out, "[#000000:#ffffff]▀") {
		t.Fatalf("expected a mixed top/bottom half-block, got %q", out)
	}
}

// fitImageGrid must keep the aspect ratio derived from the cell aspect
// (4:3 visual, 8:3 cells for the classic 1:2 font) and fit inside the viewer
// in BOTH dimensions — the wide-window case is the regression the user
// reported: the image used to be stretched to the full width. The aspect is
// pinned explicitly (fitImageGridAspect) so the expectations do not depend on
// the font of the terminal the tests happen to run in.
func TestFitImageGridNarrow(t *testing.T) {
	// Narrow viewer: width is the tighter constraint, so the grid should use
	// the full width and derive the height from it.
	twidth, cells := fitImageGridAspect(120, 200, 2.0)
	if twidth != 120 {
		t.Fatalf("narrow: expected full width 120, got %d", twidth)
	}
	// 8:3 aspect -> height ~ width * 3/8 = 45.
	if want := 45; cells != want {
		t.Fatalf("narrow: expected height ~%d, got %d", want, cells)
	}
}

func TestFitImageGridWide(t *testing.T) {
	// Wide viewer: the height (25) is the tighter constraint. Previously the
	// width stayed at the full 400, stretching the image. Now the width must be
	// derived from the height (25 * 8/3 = 66.7 -> 67) and stay well under 400.
	twidth, cells := fitImageGridAspect(400, 25, 2.0)
	if cells != 25 {
		t.Fatalf("wide: expected height 25, got %d", cells)
	}
	if want := 67; twidth != want {
		t.Fatalf("wide: expected width ~%d (height-constrained), got %d", want, twidth)
	}
	// The whole point of the fix: not stretched to the full width.
	if twidth > 400 {
		t.Fatalf("wide: width %d exceeds viewer width 400", twidth)
	}
}

func TestFitImageGridAspectRatio(t *testing.T) {
	// Across a range of viewer sizes the cell grid should stay close to
	// 4/3 * cell aspect. Checked for the classic 2.0 and a typical measured
	// 1.8 font.
	for _, aspect := range []float64{2.0, 1.8} {
		for _, tc := range []struct{ vw, ph int }{
			{80, 30}, {120, 40}, {200, 60}, {60, 30}, {300, 40}, {400, 25}, {50, 10},
		} {
			twidth, cells := fitImageGridAspect(tc.vw, tc.ph, aspect)
			ratio := float64(twidth) / float64(cells)
			want := 4.0 / 3.0 * aspect // 2.667 for 2.0, 2.4 for 1.8
			if ratio < want*0.95 || ratio > want*1.05 {
				t.Errorf("aspect=%.1f vw=%d ph=%d: ratio %.3f not close to %.3f", aspect, tc.vw, tc.ph, ratio, want)
			}
		}
	}
}

func TestFitImageGridMeasuredAspect(t *testing.T) {
	// The squish regression: a font whose cell is 1.8x (not 2x) as tall as
	// wide used to be drawn with the 2.0 assumption, so a 4:3 image came out
	// visually wider than 4:3 (squished vertically). With the measured aspect
	// the width-constrained grid gains rows: 120 * 3/(4*1.8) = 50, and the
	// visual aspect is 120w : 50*1.8h = 120:90 = 4:3.
	twidth, cells := fitImageGridAspect(120, 200, 1.8)
	if twidth != 120 || cells != 50 {
		t.Fatalf("expected 120x50 grid for aspect 1.8, got %dx%d", twidth, cells)
	}
	// Same height-constrained case: 25 rows * 4/3 * 1.8 = 60 columns.
	twidth, cells = fitImageGridAspect(400, 25, 1.8)
	if twidth != 60 || cells != 25 {
		t.Fatalf("expected 60x25 grid for aspect 1.8, got %dx%d", twidth, cells)
	}
}

func TestFitImageGridMinimums(t *testing.T) {
	// Degenerate viewer sizes must still yield a valid 1x1-or-larger grid.
	twidth, cells := fitImageGridAspect(1, 1, 2.0)
	if twidth < 1 || cells < 1 {
		t.Fatalf("expected >=1 grid, got %dx%d", twidth, cells)
	}
	twidth, cells = fitImageGridAspect(0, 0, 2.0)
	if twidth < 1 || cells < 1 {
		t.Fatalf("expected >=1 grid for 0x0, got %dx%d", twidth, cells)
	}
	// An invalid aspect must fall back to the classic 2.0.
	if twidth, cells := fitImageGridAspect(80, 100, 0); twidth != 80 || cells != 30 {
		t.Fatalf("expected fallback 80x30 grid for aspect 0, got %dx%d", twidth, cells)
	}
}

func TestCellAspectFallback(t *testing.T) {
	// Whatever the test environment provides (a tty with or without pixel
	// sizes, a pipe, or Windows), cellAspect must return a usable positive
	// value — the measured font aspect or the 1:2 default.
	if a := cellAspect(); a <= 0 {
		t.Fatalf("cellAspect must be positive, got %f", a)
	}
}

func TestTerminalCellPixelSizeNonTTY(t *testing.T) {
	// A non-terminal fd must report no cell size, so the caller falls back.
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer f.Close()
	if _, _, ok := terminalCellPixelSize(f.Fd()); ok {
		t.Fatalf("expected no cell pixel size for a non-tty fd")
	}
}

func TestMapGrayLevel(t *testing.T) {
	if got := mapGrayLevel(0, 0, 65535); got != 0 {
		t.Errorf("min -> 0, got %d", got)
	}
	if got := mapGrayLevel(65535, 0, 65535); got != 255 {
		t.Errorf("max -> 255, got %d", got)
	}
	if got := mapGrayLevel(32767, 0, 65535); got != 127 {
		t.Errorf("mid -> ~127, got %d", got)
	}
	// clamping
	if got := mapGrayLevel(0, 100, 100); got != 0 {
		t.Errorf("below window -> 0, got %d", got)
	}
	if got := mapGrayLevel(1000, 0, 100); got != 255 {
		t.Errorf("above window -> 255, got %d", got)
	}
}
