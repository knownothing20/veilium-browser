package localrecovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var forbiddenProfileDefinitionKeys = map[string]struct{}{
	"accesstoken":  {},
	"cookie":       {},
	"cookies":      {},
	"history":      {},
	"indexeddb":    {},
	"localstorage": {},
	"password":     {},
	"refreshtoken": {},
	"secret":       {},
	"secretvalue":  {},
	"token":        {},
}

func validateProfileDefinitionExclusions(data []byte) error {
	if len(data) == 0 || len(data) > MaxProfileDefinitionBytes {
		return fmt.Errorf("%w: Profile definition size is outside bounds", ErrInvalidManifest)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%w: Profile definition is not valid JSON: %v", ErrInvalidManifest, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%w: Profile definition contains trailing data", ErrInvalidManifest)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: Profile definition must be a JSON object", ErrInvalidManifest)
	}
	return inspectProfileDefinitionObject(object)
}

func inspectProfileDefinitionObject(object map[string]any) error {
	for key, value := range object {
		normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(key), "_", ""), "-", ""))
		if _, forbidden := forbiddenProfileDefinitionKeys[normalized]; forbidden {
			return fmt.Errorf("%w: Profile definition contains excluded field %q", ErrInvalidManifest, key)
		}
		if err := inspectProfileDefinitionValue(value); err != nil {
			return err
		}
	}
	return nil
}

func inspectProfileDefinitionValue(value any) error {
	switch typed := value.(type) {
	case map[string]any:
		return inspectProfileDefinitionObject(typed)
	case []any:
		for _, item := range typed {
			if err := inspectProfileDefinitionValue(item); err != nil {
				return err
			}
		}
	}
	return nil
}
