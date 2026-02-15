package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// A11yNode is a flattened accessibility tree node returned by /snapshot.
// Refs (e0, e1, ...) are stable within a snapshot and cached for use by /action.
type A11yNode struct {
	Ref      string `json:"ref"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Depth    int    `json:"depth"`
	Value    string `json:"value,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
	Focused  bool   `json:"focused,omitempty"`
	NodeID   int64  `json:"nodeId,omitempty"`
}

// Raw a11y tree types — we parse CDP responses manually because the typed
// cdproto library crashes on the "uninteresting" PropertyName value.

type rawAXNode struct {
	NodeID           string      `json:"nodeId"`
	Ignored          bool        `json:"ignored"`
	Role             *rawAXValue `json:"role"`
	Name             *rawAXValue `json:"name"`
	Value            *rawAXValue `json:"value"`
	Properties       []rawAXProp `json:"properties"`
	ChildIDs         []string    `json:"childIds"`
	BackendDOMNodeID int64       `json:"backendDOMNodeId"`
}

type rawAXValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type rawAXProp struct {
	Name  string      `json:"name"`
	Value *rawAXValue `json:"value"`
}

func (v *rawAXValue) String() string {
	if v == nil || v.Value == nil {
		return ""
	}
	var s string
	if err := json.Unmarshal(v.Value, &s); err == nil {
		return s
	}
	return strings.Trim(string(v.Value), `"`)
}

// interactiveRoles is the set of ARIA roles considered interactive
// for the ?filter=interactive snapshot parameter.
var interactiveRoles = map[string]bool{
	"button": true, "link": true, "textbox": true, "searchbox": true,
	"combobox": true, "listbox": true, "option": true, "checkbox": true,
	"radio": true, "switch": true, "slider": true, "spinbutton": true,
	"menuitem": true, "menuitemcheckbox": true, "menuitemradio": true,
	"tab": true, "treeitem": true,
}

// buildSnapshot converts raw a11y nodes into a flat list of A11yNode entries
// and a ref→backendNodeID map. filter and maxDepth control output.
func buildSnapshot(nodes []rawAXNode, filter string, maxDepth int) ([]A11yNode, map[string]int64) {
	// Build parent map for depth calculation
	parentMap := make(map[string]string)
	for _, n := range nodes {
		for _, childID := range n.ChildIDs {
			parentMap[childID] = n.NodeID
		}
	}
	depthOf := func(nodeID string) int {
		d := 0
		cur := nodeID
		for {
			p, ok := parentMap[cur]
			if !ok {
				break
			}
			d++
			cur = p
		}
		return d
	}

	flat := make([]A11yNode, 0)
	refs := make(map[string]int64)
	refID := 0

	for _, n := range nodes {
		if n.Ignored {
			continue
		}

		role := n.Role.String()
		name := n.Name.String()

		if role == "none" || role == "generic" || role == "InlineTextBox" {
			continue
		}
		if name == "" && role == "StaticText" {
			continue
		}

		depth := depthOf(n.NodeID)
		if maxDepth >= 0 && depth > maxDepth {
			continue
		}
		if filter == filterInteractive && !interactiveRoles[role] {
			continue
		}

		ref := fmt.Sprintf("e%d", refID)
		entry := A11yNode{
			Ref:   ref,
			Role:  role,
			Name:  name,
			Depth: depth,
		}

		if v := n.Value.String(); v != "" {
			entry.Value = v
		}
		if n.BackendDOMNodeID != 0 {
			entry.NodeID = n.BackendDOMNodeID
			refs[ref] = n.BackendDOMNodeID
		}

		for _, prop := range n.Properties {
			if prop.Name == "disabled" && prop.Value.String() == "true" {
				entry.Disabled = true
			}
			if prop.Name == "focused" && prop.Value.String() == "true" {
				entry.Focused = true
			}
		}

		flat = append(flat, entry)
		refID++
	}

	return flat, refs
}
