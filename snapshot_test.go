package main

import (
	"encoding/json"
	"testing"
)

func TestRawAXValueString(t *testing.T) {
	tests := []struct {
		name string
		val  *rawAXValue
		want string
	}{
		{"nil", nil, ""},
		{"nil value", &rawAXValue{Type: "string"}, ""},
		{"string", &rawAXValue{Type: "string", Value: json.RawMessage(`"hello"`)}, "hello"},
		{"number", &rawAXValue{Type: "integer", Value: json.RawMessage(`42`)}, "42"},
		{"bool", &rawAXValue{Type: "boolean", Value: json.RawMessage(`true`)}, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.val.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInteractiveRoles(t *testing.T) {
	interactive := []string{"button", "link", "textbox", "checkbox", "radio", "tab", "menuitem"}
	for _, r := range interactive {
		if !interactiveRoles[r] {
			t.Errorf("expected %q to be interactive", r)
		}
	}

	nonInteractive := []string{"heading", "paragraph", "image", "banner", "main", "navigation"}
	for _, r := range nonInteractive {
		if interactiveRoles[r] {
			t.Errorf("expected %q to NOT be interactive", r)
		}
	}
}

func TestBuildSnapshot(t *testing.T) {
	// Simulate raw a11y nodes
	nodes := []rawAXNode{
		{
			NodeID:           "root",
			Role:             &rawAXValue{Value: json.RawMessage(`"WebArea"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Test Page"`)},
			ChildIDs:         []string{"n1", "n2", "n3"},
			BackendDOMNodeID: 1,
		},
		{
			NodeID:           "n1",
			Role:             &rawAXValue{Value: json.RawMessage(`"button"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Submit"`)},
			BackendDOMNodeID: 10,
		},
		{
			NodeID:           "n2",
			Role:             &rawAXValue{Value: json.RawMessage(`"textbox"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Email"`)},
			BackendDOMNodeID: 20,
			Properties: []rawAXProp{
				{Name: "focused", Value: &rawAXValue{Value: json.RawMessage(`"true"`)}},
			},
		},
		{
			NodeID:  "n3",
			Ignored: true,
			Role:    &rawAXValue{Value: json.RawMessage(`"none"`)},
		},
		{
			NodeID:           "n4",
			Role:             &rawAXValue{Value: json.RawMessage(`"generic"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`""`)},
			BackendDOMNodeID: 30,
		},
	}

	flat, refs := buildSnapshot(nodes, "", -1)

	// Should have 3 entries: WebArea, button, textbox (ignored + generic + none filtered)
	if len(flat) != 3 {
		t.Fatalf("expected 3 nodes, got %d: %+v", len(flat), flat)
	}

	// Check refs map
	if refs["e0"] != 1 {
		t.Errorf("e0 should map to nodeID 1, got %d", refs["e0"])
	}
	if refs["e1"] != 10 {
		t.Errorf("e1 should map to nodeID 10, got %d", refs["e1"])
	}
	if refs["e2"] != 20 {
		t.Errorf("e2 should map to nodeID 20, got %d", refs["e2"])
	}

	// Check depth
	if flat[0].Depth != 0 {
		t.Errorf("root depth should be 0, got %d", flat[0].Depth)
	}
	if flat[1].Depth != 1 {
		t.Errorf("button depth should be 1, got %d", flat[1].Depth)
	}

	// Check focused property
	if !flat[2].Focused {
		t.Error("textbox should be focused")
	}
}

func TestBuildSnapshotInteractiveFilter(t *testing.T) {
	nodes := []rawAXNode{
		{
			NodeID:           "root",
			Role:             &rawAXValue{Value: json.RawMessage(`"WebArea"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Page"`)},
			ChildIDs:         []string{"n1", "n2"},
			BackendDOMNodeID: 1,
		},
		{
			NodeID:           "n1",
			Role:             &rawAXValue{Value: json.RawMessage(`"heading"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Title"`)},
			BackendDOMNodeID: 10,
		},
		{
			NodeID:           "n2",
			Role:             &rawAXValue{Value: json.RawMessage(`"button"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Click me"`)},
			BackendDOMNodeID: 20,
		},
	}

	flat, _ := buildSnapshot(nodes, filterInteractive, -1)

	if len(flat) != 1 {
		t.Fatalf("expected 1 interactive node, got %d: %+v", len(flat), flat)
	}
	if flat[0].Role != "button" {
		t.Errorf("expected button, got %s", flat[0].Role)
	}
}

func TestBuildSnapshotDepthFilter(t *testing.T) {
	nodes := []rawAXNode{
		{
			NodeID:           "root",
			Role:             &rawAXValue{Value: json.RawMessage(`"WebArea"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Page"`)},
			ChildIDs:         []string{"n1"},
			BackendDOMNodeID: 1,
		},
		{
			NodeID:           "n1",
			Role:             &rawAXValue{Value: json.RawMessage(`"navigation"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Nav"`)},
			ChildIDs:         []string{"n2"},
			BackendDOMNodeID: 10,
		},
		{
			NodeID:           "n2",
			Role:             &rawAXValue{Value: json.RawMessage(`"link"`)},
			Name:             &rawAXValue{Value: json.RawMessage(`"Home"`)},
			BackendDOMNodeID: 20,
		},
	}

	flat, _ := buildSnapshot(nodes, "", 1)

	if len(flat) != 2 {
		t.Fatalf("expected 2 nodes at depth<=1, got %d: %+v", len(flat), flat)
	}
}
