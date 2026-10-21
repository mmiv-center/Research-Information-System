package main

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// selectionTree builds a small tree mirroring the real "Series by job" layout:
// a non-selectable root, job/group nodes (no reference), and selectable series
// nodes (non-empty reference). It returns the root and the flat node list the
// tests use to assert order and target selection.
func selectionTree(t *testing.T) (*tview.TreeNode, []*tview.TreeNode) {
	t.Helper()
	root := tview.NewTreeNode("Selections") // no reference => not a series
	job1 := tview.NewTreeNode("job 1")      // no reference => not a series
	job2 := tview.NewTreeNode("job 2")      // no reference => not a series
	s1a := tview.NewTreeNode("s1a").SetReference("UID-1a")
	s1b := tview.NewTreeNode("s1b").SetReference("UID-1b")
	s2a := tview.NewTreeNode("s2a").SetReference("UID-2a")
	job1.AddChild(s1a).AddChild(s1b)
	job2.AddChild(s2a)
	root.AddChild(job1).AddChild(job2)
	return root, flattenSelection(root)
}

func nodeText(n *tview.TreeNode) string {
	if n == nil {
		return "<nil>"
	}
	return n.GetText()
}

func TestIsSeriesNode(t *testing.T) {
	if !isSeriesNode(tview.NewTreeNode("s").SetReference("UID")) {
		t.Errorf("a non-empty string reference should be a series node")
	}
	if isSeriesNode(tview.NewTreeNode("root")) {
		t.Errorf("a node without a reference should not be a series node")
	}
	if isSeriesNode(tview.NewTreeNode("root").SetReference("")) {
		t.Errorf("an empty-string reference should not be a series node")
	}
}

func TestFlattenSelectionOrder(t *testing.T) {
	root, nodes := selectionTree(t)
	want := []string{"Selections", "job 1", "s1a", "s1b", "job 2", "s2a"}
	if len(nodes) != len(want) {
		t.Fatalf("want %d nodes, got %d (%v)", len(want), len(nodes), nodes)
	}
	for i, n := range nodes {
		if got := nodeText(n); got != want[i] {
			t.Errorf("node %d: want %q, got %q", i, want[i], got)
		}
	}

	// Collapsing a job node hides its series children from the flat list.
	job2 := root.GetChildren()[1]
	job2.SetExpanded(false)
	nodes = flattenSelection(root)
	want = []string{"Selections", "job 1", "s1a", "s1b", "job 2"}
	if len(nodes) != len(want) {
		t.Fatalf("collapsed: want %d nodes, got %d", len(want), len(nodes))
	}
	for i, n := range nodes {
		if got := nodeText(n); got != want[i] {
			t.Errorf("collapsed node %d: want %q, got %q", i, want[i], got)
		}
	}
}

func TestSelectionScrollTarget(t *testing.T) {
	_, nodes := selectionTree(t)
	// flat order: 0 Selections, 1 job 1, 2 s1a, 3 s1b, 4 job 2, 5 s2a

	cases := []struct {
		name    string
		offset  int
		height  int
		action  tview.MouseAction
		wantRef string // "" means expect nil
	}{
		{"scroll up, top visible is s1a", 0, 3, tview.MouseScrollUp, "UID-1a"},
		{"scroll down, bottom visible is s1a", 0, 3, tview.MouseScrollDown, "UID-1a"},
		{"scroll up, skip job to first series", 1, 3, tview.MouseScrollUp, "UID-1a"},
		{"scroll down, last series in range", 1, 3, tview.MouseScrollDown, "UID-1b"},
		{"scroll down, reaches s2a", 3, 3, tview.MouseScrollDown, "UID-2a"},
		{"scroll up, reaches s1b", 3, 3, tview.MouseScrollUp, "UID-1b"},
		{"no series in a one-row range on the root", 0, 1, tview.MouseScrollUp, ""},
		{"height clamped past the end", 4, 10, tview.MouseScrollDown, "UID-2a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := selectionScrollTarget(nodes, tc.offset, tc.height, tc.action)
			gotRef := ""
			if got != nil {
				gotRef, _ = got.GetReference().(string)
			}
			if gotRef != tc.wantRef {
				t.Errorf("want %q, got %q (node %s)", tc.wantRef, gotRef, nodeText(got))
			}
		})
	}
}

// refString returns the string reference of a node, or a placeholder when the
// node (or its reference) is nil.
func refString(n *tview.TreeNode) string {
	if n == nil {
		return "<nil>"
	}
	if ref, ok := n.GetReference().(string); ok {
		return ref
	}
	return "<no ref>"
}

// TestCaptureMouseScrollMovesHighlight drives a real TreeView (populated by a
// draw) to confirm captureMouse re-points the highlight on a touch-pad scroll:
// a scroll down selects the bottom visible series, a scroll up the top one, and
// the scroll event always passes through so tview still scrolls the view. It
// also confirms that a scroll positioned outside the tree is ignored.
func TestCaptureMouseScrollMovesHighlight(t *testing.T) {
	root := tview.NewTreeNode("Selections")
	job := tview.NewTreeNode("job")
	for _, id := range []string{"UID-1", "UID-2", "UID-3", "UID-4"} {
		job.AddChild(tview.NewTreeNode(id).SetReference(id))
	}
	root.AddChild(job)

	selection := tview.NewTreeView()
	selection.SetRoot(root)
	selection.SetCurrentNode(root)
	selection.SetBorder(true)
	selection.SetRect(0, 0, 40, 6) // bordered box ~6 tall => inner height ~4 rows

	// A draw populates the tree's internal node list and fixes the scroll
	// offset, so the public accessors report a real visible window.
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("init simulation screen: %v", err)
	}
	screen.SetSize(80, 24)
	selection.Draw(screen)
	if got := selection.GetRowCount(); got != 6 {
		t.Fatalf("expected 6 populated rows, got %d", got)
	}

	statusTUI := &StatusTUI{selection: selection}
	inside := tcell.NewEventMouse(20, 3, 0, 0)   // over the tree
	outside := tcell.NewEventMouse(0, 100, 0, 0) // far below the tree

	// Scroll down: from the top (offset 0), the new visible window is rows 1..4,
	// whose bottom series is UID-3. The highlight moves there and the event
	// passes through unchanged.
	if got, act := statusTUI.captureMouse(inside, tview.MouseScrollDown); got != inside || act != tview.MouseScrollDown {
		t.Fatalf("scroll-down event must pass through, got %v %v", got, act)
	}
	if ref := refString(selection.GetCurrentNode()); ref != "UID-3" {
		t.Errorf("scroll down: want highlight UID-3 (bottom visible), got %s", ref)
	}

	// Scroll up: back to the top window (rows 0..3), whose first series is UID-1.
	if got, act := statusTUI.captureMouse(inside, tview.MouseScrollUp); got != inside || act != tview.MouseScrollUp {
		t.Fatalf("scroll-up event must pass through, got %v %v", got, act)
	}
	if ref := refString(selection.GetCurrentNode()); ref != "UID-1" {
		t.Errorf("scroll up: want highlight UID-1 (top visible), got %s", ref)
	}

	// A scroll outside the tree must leave the highlight untouched.
	selection.SetCurrentNode(root)
	statusTUI.captureMouse(outside, tview.MouseScrollDown)
	if cur := selection.GetCurrentNode(); cur != root {
		t.Errorf("scroll outside the tree must not move the highlight, got %s", nodeText(cur))
	}
}
