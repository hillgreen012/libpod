package capabilities

// Copyright 2013-2018 Docker, Inc.

// NOTE: this package has been copied from github.com/docker/docker but been
//       changed significantly to fit the needs of libpod.

import (
	"strings"

	"github.com/containers/libpod/pkg/util"
	"github.com/pkg/errors"
	"github.com/syndtr/gocapability/capability"
)

var (
	// Used internally and populated during init().
	capabilityList []string

	// ErrUnknownCapability is thrown when an unknown capability is processed.
	ErrUnknownCapability = errors.New("unknown capability")
)

// All is a special value used to add/drop all known capababilities.
// Useful on the CLI for `--cap-add=all` etc.
const All = "ALL"

func init() {
	last := capability.CAP_LAST_CAP
	// hack for RHEL6 which has no /proc/sys/kernel/cap_last_cap
	if last == capability.Cap(63) {
		last = capability.CAP_BLOCK_SUSPEND
	}
	for _, cap := range capability.List() {
		if cap > last {
			continue
		}
		capabilityList = append(capabilityList, "CAP_"+strings.ToUpper(cap.String()))
	}
}

// AllCapabilities returns all known capabilities.
func AllCapabilities() []string {
	return capabilityList
}

// normalizeCapabilities normalizes caps by adding a "CAP_" prefix (if not yet
// present).
func normalizeCapabilities(caps []string) ([]string, error) {
	normalized := make([]string, len(caps))
	for i, c := range caps {
		c = strings.ToUpper(c)
		if c == All {
			normalized = append(normalized, c)
			continue
		}
		if !strings.HasPrefix(c, "CAP_") {
			c = "CAP_" + c
		}
		if !util.StringInSlice(c, capabilityList) {
			return nil, errors.Wrapf(ErrUnknownCapability, "%q", c)
		}
		normalized[i] = c
	}
	return normalized, nil
}

// ValidateCapabilities validates if caps only contains valid capabilities.
func ValidateCapabilities(caps []string) error {
	for _, c := range caps {
		if !util.StringInSlice(c, capabilityList) {
			return errors.Wrapf(ErrUnknownCapability, "%q", c)
		}
	}
	return nil
}

// MergeCapabilities computes a set of capabilities by adding capapbitilities
// to or dropping them from base.
//
// Note that "ALL" will cause all known capabilities to be added/dropped but
// the ones specified to be dropped/added.
func MergeCapabilities(base, adds, drops []string) ([]string, error) {
	if len(adds) == 0 && len(drops) == 0 {
		// Nothing to tweak; we're done
		return base, nil
	}

	capDrop, err := normalizeCapabilities(drops)
	if err != nil {
		return nil, err
	}
	capAdd, err := normalizeCapabilities(adds)
	if err != nil {
		return nil, err
	}

	// Make sure that capDrop and capAdd are distinct sets.
	for _, drop := range capDrop {
		if util.StringInSlice(drop, capAdd) {
			return nil, errors.Errorf("capability %q cannot be dropped and added", drop)
		}
	}

	var caps []string

	switch {
	case util.StringInSlice(All, capAdd):
		// Add all capabilities except ones on capDrop
		for _, c := range capabilityList {
			if !util.StringInSlice(c, capDrop) {
				caps = append(caps, c)
			}
		}
	case util.StringInSlice(All, capDrop):
		// "Drop" all capabilities; use what's in capAdd instead
		caps = capAdd
	default:
		// First drop some capabilities
		for _, c := range base {
			if !util.StringInSlice(c, capDrop) {
				caps = append(caps, c)
			}
		}
		// Then add the list of capabilities from capAdd
		caps = append(caps, capAdd...)
	}
	return caps, nil
}
