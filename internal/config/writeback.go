package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// UpdateKeepAbove sets window.keep_above to value in the YAML config
// file at path. It edits the file at the yaml.v3 node level so every
// other key, value, and comment survives the round trip unchanged;
// the window mapping and/or the keep_above key are created when
// absent. The rewritten content is written to a temp file in the
// same directory and renamed over path, so any failure (read, parse,
// encode, write) returns an error and leaves the original file
// untouched.
func UpdateKeepAbove(path string, value bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}

	top, err := documentMapping(&root)
	if err != nil {
		return fmt.Errorf("config: %s: %w", path, err)
	}

	window := mapEntry(top, "window")
	if window == nil {
		window = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		appendMapEntry(top, "window", window)
	}

	keepAbove := mapEntry(window, "keep_above")
	if keepAbove == nil {
		keepAbove = &yaml.Node{Kind: yaml.ScalarNode}
		appendMapEntry(window, "keep_above", keepAbove)
	}
	keepAbove.Kind = yaml.ScalarNode
	keepAbove.Tag = "!!bool"
	keepAbove.Value = strconv.FormatBool(value)
	keepAbove.Style = 0

	encoded, err := encodeNode(&root)
	if err != nil {
		return fmt.Errorf("config: encode %s: %w", path, err)
	}

	if err := writeFileAtomic(path, encoded); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

// documentMapping returns the top-level mapping node of root, the
// parsed document. An empty document (an empty or comment-only file)
// yields a fresh, empty mapping. It is an error for the top-level
// YAML node to exist and not be a mapping: UpdateKeepAbove has
// nowhere to place window.keep_above in that case.
func documentMapping(root *yaml.Node) (*yaml.Node, error) {
	if root.Kind == 0 {
		mapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Kind = yaml.DocumentNode
		root.Content = []*yaml.Node{mapping}
		return mapping, nil
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 {
		return nil, fmt.Errorf("top-level YAML is not a single document")
	}
	top := root.Content[0]
	if top.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("top-level YAML is not a mapping")
	}
	return top, nil
}

// mapEntry returns the value node for key in mapping m, or nil if
// key is absent. m must be a MappingNode.
func mapEntry(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// appendMapEntry appends a new "key: value" pair to mapping m.
func appendMapEntry(m *yaml.Node, key string, value *yaml.Node) {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	m.Content = append(m.Content, keyNode, value)
}

// encodeNode re-encodes root with a 2-space indent, matching this
// project's YAML style, so the rewritten file differs from the
// original only on the changed line.
func encodeNode(root *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		enc.Close()
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeFileAtomic writes data to a temp file created in the same
// directory as path, then renames it over path. A temp file in the
// same directory guarantees the rename is on the same filesystem, so
// it completes atomically: readers never observe a partially written
// path, and a failure at any step before the rename leaves path
// untouched.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".config-*.yaml.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if info, statErr := os.Stat(path); statErr == nil {
		_ = os.Chmod(tmpPath, info.Mode())
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	removeTmp = false
	return nil
}
