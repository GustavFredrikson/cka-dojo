// Package spec holds the lazy-decoding envelope shared by fault and grader
// definitions, so a lab file can say `type: systemdStop` and have the rest of
// the mapping decoded by whichever implementation claims that type.
package spec

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Spec is a typed YAML mapping whose remaining fields are decoded later.
type Spec struct {
	Type string
	node yaml.Node
}

// UnmarshalYAML captures the whole node and reads only the discriminator.
func (s *Spec) UnmarshalYAML(n *yaml.Node) error {
	var head struct {
		Type string `yaml:"type"`
	}
	if err := n.Decode(&head); err != nil {
		return err
	}
	if head.Type == "" {
		return fmt.Errorf("line %d: entry is missing `type`", n.Line)
	}
	s.Type = head.Type
	s.node = *n
	return nil
}

// MarshalYAML re-emits the captured node so round-tripping a lab file works.
func (s Spec) MarshalYAML() (any, error) { return s.node, nil }

// Decode unpacks the captured node into a concrete type.
func (s *Spec) Decode(v any) error {
	if err := s.node.Decode(v); err != nil {
		return fmt.Errorf("%s (line %d): %w", s.Type, s.node.Line, err)
	}
	return nil
}

// Line is the source line the spec started on, for error messages.
func (s *Spec) Line() int { return s.node.Line }
