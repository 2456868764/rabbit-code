package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Validate checks known Phase-2 keys (P2.F.* subset). Returns aggregated errors.
func Validate(m map[string]interface{}) []error {
	var errs []error
	if m == nil {
		return nil
	}

	checkString := func(key string) {
		v, ok := m[key]
		if !ok || v == nil {
			return
		}
		if _, ok := v.(string); !ok {
			errs = append(errs, fmt.Errorf("%q must be a string", key))
		}
	}
	checkBool := func(key string) {
		v, ok := m[key]
		if !ok || v == nil {
			return
		}
		switch x := v.(type) {
		case bool:
		case float64:
			if x != 0 && x != 1 {
				errs = append(errs, fmt.Errorf("%q must be a boolean", key))
			}
		default:
			errs = append(errs, fmt.Errorf("%q must be a boolean", key))
		}
	}

	checkString("auto_theme")
	checkBool("disable_auto_mode")
	checkBool("yolo_classifier")
	checkBool("kairos_push_notification")
	checkBool("kairos_enabled")
	checkBool("voice_mode")
	checkBool("lodestone_enabled")
	checkString("team_mem_path")

	checkOptionalHTTPURL := func(key string) {
		v, ok := m[key]
		if !ok || v == nil {
			return
		}
		s, ok := v.(string)
		if !ok {
			errs = append(errs, fmt.Errorf("%q must be a string", key))
			return
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		u, err := url.Parse(s)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			errs = append(errs, fmt.Errorf("%q must be an http(s) URL with host", key))
		}
	}
	checkOptionalHTTPURL("download_user_settings_url")
	checkOptionalHTTPURL("upload_user_settings_url")

	if v, ok := m["templates"]; ok && v != nil {
		if _, ok := v.(map[string]interface{}); !ok {
			errs = append(errs, fmt.Errorf("templates must be an object"))
		}
	}

	if v, ok := m["managed_env"]; ok && v != nil {
		mm, ok := v.(map[string]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("managed_env must be an object"))
		} else {
			for k, val := range mm {
				if _, ok := val.(string); !ok {
					errs = append(errs, fmt.Errorf("managed_env.%s must be a string", k))
				}
			}
		}
	}

	if v, ok := m["extra_ca_paths"]; ok && v != nil {
		arr, ok := v.([]interface{})
		if !ok {
			errs = append(errs, fmt.Errorf("extra_ca_paths must be an array"))
		} else {
			for i, el := range arr {
				if _, ok := el.(string); !ok {
					errs = append(errs, fmt.Errorf("extra_ca_paths[%d] must be a string", i))
				}
			}
		}
	}

	return errs
}

type ValidationErrors []error

func (ve ValidationErrors) Error() string {
	var b strings.Builder
	for i, e := range ve {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Error())
	}
	return b.String()
}
