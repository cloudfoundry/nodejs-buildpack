package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type ContrastSecurityHook struct {
	libbuildpack.DefaultHook
	Log *libbuildpack.Logger
}

type ContrastSecurityCredentials struct {
	ApiKey      string
	ApiToken    string
	OrgUuid     string
	ServiceKey  string
	ContrastUrl string //formerly Teamserver URL
	Username    string
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)

	libbuildpack.AddHook(ContrastSecurityHook{
		Log: logger,
	})
}

// shellQuote wraps a value in single quotes and escapes any embedded single
// quotes so the result is safe to embed in a shell export statement.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// writeExport returns a shell export line for the given variable, or an empty
// string when value is empty (so the line is simply omitted from the script).
func writeExport(name, value string) string {
	if value == "" {
		return ""
	}
	return "export " + name + "=" + shellQuote(value) + "\n"
}

func (h ContrastSecurityHook) AfterCompile(stager *libbuildpack.Stager) error {
	h.Log.Debug("Contrast Security after compile hook")

	success, contrastSecurityCredentials := h.GetCredentialsFromEnvironment()

	if !success {
		h.Log.Info("Contrast Security no credentials found. Will not write environment files.")
		return nil
	}

	h.Log.Info("Contrast Security credentials found. Configuring environment for [%s].", contrastSecurityCredentials.ContrastUrl)

	url := ""
	if contrastSecurityCredentials.ContrastUrl != "" {
		url = strings.TrimRight(contrastSecurityCredentials.ContrastUrl, "/")
		if !strings.HasSuffix(url, "/Contrast") {
			url += "/Contrast"
		}
	}

	contrastSecurityScript :=
		writeExport("CONTRAST__API__URL", url) +
			writeExport("CONTRAST__API__API_KEY", contrastSecurityCredentials.ApiKey) +
			writeExport("CONTRAST__API__SERVICE_KEY", contrastSecurityCredentials.ServiceKey) +
			writeExport("CONTRAST__API__USER_NAME", contrastSecurityCredentials.Username) +
			writeExport("CONTRAST__API__TOKEN", contrastSecurityCredentials.ApiToken)

	stager.WriteProfileD("contrast_security", contrastSecurityScript)

	h.Log.Debug("Contrast Security successfully wrote to .profile.d")

	return nil
}

func getContrastCredentialString(credentials map[string]interface{}, key string) string {
	if value, exists := credentials[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// serviceMatchesContrast reports whether a single service entry (identified by
// its top-level VCAP_SERVICES category key plus its own fields) should be
// treated as a Contrast Security service. All type assertions use the comma-ok
// form so malformed entries are skipped rather than panicked on.
func serviceMatchesContrast(key string, svc map[string]interface{}, query string) bool {
	if strings.Contains(key, query) {
		return true
	}
	if name, ok := svc["name"].(string); ok && strings.Contains(name, query) {
		return true
	}
	if label, ok := svc["label"].(string); ok && strings.Contains(label, query) {
		return true
	}
	if tags, ok := svc["tags"].([]interface{}); ok {
		for _, t := range tags {
			if tag, ok := t.(string); ok && strings.Contains(tag, query) {
				return true
			}
		}
	}
	return false
}

// loadVcapServicesJSON returns the raw VCAP_SERVICES JSON from whichever
// delivery mechanism is active. VCAP_SERVICES_FILE_PATH (file-based CF mode)
// takes priority; VCAP_SERVICES env var is the fallback (standard CF mode).
func (h ContrastSecurityHook) loadVcapServicesJSON() (string, error) {
	fp := os.Getenv("VCAP_SERVICES_FILE_PATH")
	if fp != "" {
		data, err := os.ReadFile(fp)
		if err != nil {
			return "", fmt.Errorf("could not read VCAP_SERVICES_FILE_PATH %q: %w", fp, err)
		}
		if os.Getenv("VCAP_SERVICES") != "" {
			h.Log.Warning("Contrast Security: both VCAP_SERVICES and VCAP_SERVICES_FILE_PATH are set; using VCAP_SERVICES_FILE_PATH.")
		}
		return string(data), nil
	}
	return os.Getenv("VCAP_SERVICES"), nil
}

func (h ContrastSecurityHook) GetCredentialsFromEnvironment() (bool, ContrastSecurityCredentials) {

	type rawVcapServicesJSONValue map[string]interface{}

	var vcapServices rawVcapServicesJSONValue

	vcapServicesEnvironment, err := h.loadVcapServicesJSON()
	if err != nil {
		h.Log.Warning("Contrast Security: %s", err.Error())
		return false, ContrastSecurityCredentials{}
	}

	if vcapServicesEnvironment == "" {
		h.Log.Debug("Contrast Security could not find VCAP_SERVICES or VCAP_SERVICES_FILE_PATH in the environment")
		return false, ContrastSecurityCredentials{}
	}

	err = json.Unmarshal([]byte(vcapServicesEnvironment), &vcapServices)
	if err != nil {
		h.Log.Warning("Contrast Security could not parse VCAP_SERVICES")
		return false, ContrastSecurityCredentials{}
	}

	for key, rawServices := range vcapServices {
		services, ok := rawServices.([]interface{})
		if !ok {
			continue
		}
		for _, rawSvc := range services {
			svc, ok := rawSvc.(map[string]interface{})
			if !ok {
				continue
			}
			if !serviceMatchesContrast(key, svc, "contrast-security") {
				continue
			}
			credentials, ok := svc["credentials"].(map[string]interface{})
			if !ok {
				continue
			}
			h.Log.Debug("Contrast Security found credentials in VCAP_SERVICES")
			return true, ContrastSecurityCredentials{
				ApiKey:      getContrastCredentialString(credentials, "api_key"),
				ApiToken:    getContrastCredentialString(credentials, "api_token"),
				OrgUuid:     getContrastCredentialString(credentials, "org_uuid"),
				ServiceKey:  getContrastCredentialString(credentials, "service_key"),
				ContrastUrl: getContrastCredentialString(credentials, "teamserver_url"),
				Username:    getContrastCredentialString(credentials, "username"),
			}
		}
	}

	return false, ContrastSecurityCredentials{}
}
