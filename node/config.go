// Package node runs the Matter bridge node over the go-fabric SDK. It opens the
// SDK's durable store inside a host directory, gives every admitted resource a
// bridged endpoint composed by its profile and keeps each endpoint number the
// SDK has stored for it. It builds without any build tag.
package node

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/driver"
	"github.com/Project-Helianthus/helianthus-semreg/semreg/v1/graph"
	gfnode "thingfabric.dev/go-fabric/node"

	"github.com/Project-Helianthus/helianthus-matterreg/publication"
	"github.com/Project-Helianthus/helianthus-matterreg/targets/composition"
)

const (
	// StoreFile is the SDK store file inside the store directory.
	StoreFile = "matter.sqlite"
	// DevelopmentVendorID is the vendor id of the SDK's test attestation chain.
	DevelopmentVendorID uint16 = 0xFFF1
	// ProductID is the non-zero product id the bridge node presents.
	ProductID uint16 = 0x8001
)

// Config is the host's description of one bridge node.
type Config struct {
	// StoreDir is the directory that holds StoreFile.
	StoreDir string
	// Profiles are the composition profiles by resource kind.
	Profiles []composition.Profile
	// Roster is the derived Matter roster the publication rule reads.
	Roster []publication.Row
	// ProductDescription carries VendorID DevelopmentVendorID and a non-zero ProductID.
	ProductDescription gfnode.ProductDescription
	// Commissioning is the setup passcode, discriminator and PBKDF parameters.
	Commissioning gfnode.Commissioning
	// BasicInformation is the root node's BasicInformation.
	BasicInformation gfnode.BasicInformation
	// Input is the SemanticInput of the Session given to Instance.Start. Refresh
	// reads it; Apply needs none.
	Input driver.SemanticInput
	// Lens is the lens request Refresh retrieves for every cut it reads.
	Lens graph.LensRequest
}

// Configuration errors.
var (
	ErrStoreDir = errors.New("node: store directory is empty")
	ErrProduct  = errors.New("node: product description needs the development vendor id and a non-zero product id")
	ErrProfile  = errors.New("node: invalid composition profile")
)

func (c Config) validate() error {
	if c.StoreDir == "" {
		return ErrStoreDir
	}
	if c.ProductDescription.VendorID != DevelopmentVendorID || c.ProductDescription.ProductID == 0 {
		return ErrProduct
	}
	return nil
}

func (c Config) storePath() string { return filepath.Join(c.StoreDir, StoreFile) }

func (c Config) profileIndex() (map[string]composition.Profile, error) {
	index := make(map[string]composition.Profile, len(c.Profiles))
	for _, profile := range c.Profiles {
		if profile.Kind == "" {
			return nil, fmt.Errorf("%w: empty kind", ErrProfile)
		}
		if _, ok := index[profile.Kind]; ok {
			return nil, fmt.Errorf("%w: kind %s is duplicated", ErrProfile, profile.Kind)
		}
		if err := checkRoles(profile); err != nil {
			return nil, err
		}
		index[profile.Kind] = profile
	}
	return index, nil
}

func checkRoles(profile composition.Profile) error {
	seen := map[string]struct{}{}
	var walk func([]composition.PartSpec) error
	walk = func(parts []composition.PartSpec) error {
		for _, part := range parts {
			if !token(part.Role) {
				return fmt.Errorf("%w: kind %s has a malformed part role %q", ErrProfile, profile.Kind, part.Role)
			}
			if _, ok := seen[part.Role]; ok {
				return fmt.Errorf("%w: kind %s repeats part role %s", ErrProfile, profile.Kind, part.Role)
			}
			seen[part.Role] = struct{}{}
			if err := walk(part.Children); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(profile.Parts)
}

func token(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r == 0 || r == '\r' || r == '\n' || r == '\t' {
			return false
		}
	}
	return value[0] != ' ' && value[len(value)-1] != ' '
}

func ensureDir(dir string) error { return os.MkdirAll(dir, 0o700) }
