package settings

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/kiteco/kiteco/kite-golib/conversion"
	"github.com/kiteco/kiteco/kite-golib/domains"
	"github.com/kiteco/kiteco/kite-golib/errors"
	"github.com/kiteco/kiteco/kite-golib/kiteserver"
)

var specs = map[string]*keySpec{
	ServerKey: {
		preventDeletion: true,
		defaultValue:    toPtr(fmt.Sprintf("https://%s/", domains.Alpha)),
		validate: func(s string) error {
			u, err := url.Parse(s)
			if err != nil {
				return errors.Errorf("unable to parse url %s: %v", s, err)
			}

			switch u.Scheme {
			case "https", "http":
			default:
				return errors.Errorf("unsupported scheme: %s", u.Scheme)
			}

			if !u.IsAbs() {
				return errors.Errorf("url is not absolute")
			}

			return nil
		},
	},
	StatusIconKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(true)),
		validate:        validateBool,
	},
	CompletionsDisabledKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate: func(s string) error {
			if _, err := strconv.ParseBool(s); err != nil {
				return errors.Errorf("unable to parse boolean %s: %v", s, err)
			}
			return nil
		},
	},
	MetricsDisabledKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	HasDoneOnboardingKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	HaveShownWelcome: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	AutosearchEnabledKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(true)),
		validate:        validateBool,
	},
	AutoInstallPluginsEnabledKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	NotifyUninstalledPluginsKey: {
		preventDeletion: true,
		defaultValue:    toPtr(strconv.FormatBool(true)),
		validate:        validateBool,
	},
	proxyModeKey: {
		preventDeletion: true,
		defaultValue:    toPtr(EnvironmentProxySentinel),
		validate: func(s string) error {
			switch s {
			case NoProxySentinel, EnvironmentProxySentinel, manualProxyModeSentinel:
			default:
				return errors.Errorf("unknown proxy mode %s", s)
			}
			return nil
		},
	},
	proxyURLKey: {
		preventDeletion: false,
		defaultValue:    toPtr(""),
		validate: func(s string) error {
			if s == "" {
				// the default value is valid
				return nil
			}

			url, err := url.Parse(s)
			if err != nil {
				return errors.Errorf("unable to parse url %s: %v", s, err)
			}

			switch url.Scheme {
			case "http", "socks5":
			default:
				return errors.Errorf("unsupported url scheme %s", url.Scheme)
			}

			if url.Port() == "" {
				return errors.Errorf("port not defined: %s", s)
			}

			if url.RawQuery != "" {
				return errors.Errorf("unsupported url query %s", url.Query())
			}

			return nil
		},
	},
	TFThreadsKey: {
		preventDeletion: true,
		defaultValue:    toPtr("1"),
		validate: func(s string) error {
			_, err := strconv.Atoi(s)
			return err
		},
	},
	MaxFileSizeKey: {
		preventDeletion: true,
		defaultValue:    toPtr("1024"),
		validate: func(s string) error {
			_, err := strconv.Atoi(s)
			return err
		},
	},
	PredictiveNavMaxFilesKey: {
		preventDeletion: true,
		defaultValue:    toPtr("100000"),
		validate: func(s string) error {
			_, err := strconv.Atoi(s)
			return err
		},
	},
	ProLaunchNotificationDismissed: {
		preventDeletion: false,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	ShowCompletionsCTA: {
		preventDeletion: false,
		defaultValue:    toPtr(strconv.FormatBool(true)),
		validate:        validateBool,
	},
	ShowCompletionsCTANotif: {
		preventDeletion: false,
		defaultValue:    toPtr(strconv.FormatBool(true)),
		validate:        validateBool,
	},
	CompletionsCTALastShown: {
		preventDeletion: false,
		defaultValue:    toPtr(time.Time{}.Format(time.RFC3339)),
		validate: func(s string) error {
			if _, err := time.Parse(time.RFC3339, s); err != nil {
				return errors.Errorf("unable to parse time %s: %v", s, err)
			}
			return nil
		},
	},
	RCDisabledCompletionsCTA: {
		preventDeletion: false,
		// Let the backend determine the default trial duration.
		defaultValue: toPtr(strconv.FormatBool(false)),
		validate:     validateBool,
	},
	RCDisabledLexicalPython: {
		preventDeletion: false,
		// Let the backend determine the default trial duration.
		defaultValue: toPtr(strconv.FormatBool(false)),
		validate:     validateBool,
	},
	InstallTimeKey: {
		preventDeletion: false,
		defaultValue:    toPtr(time.Time{}.Format(time.RFC3339)),
		validate: func(s string) error {
			if _, err := time.Parse(time.RFC3339, s); err != nil {
				return errors.Errorf("unable to parse time %s: %v", s, err)
			}
			return nil
		},
	},
	ConversionCohort: {
		preventDeletion: false,
		defaultValue:    nil,
		validate: func(s string) error {
			if _, ok := conversion.CohortSet[s]; !ok {
				return errors.Errorf("onboarding cohort \"%s\" unknown", s)
			}
			return nil
		},
	},
	TrialDuration: {
		preventDeletion: false,
		// Let the backend determine the default trial duration.
		defaultValue: nil,
		validate: func(s string) error {
			if _, err := time.ParseDuration(s); err != nil {
				return errors.Errorf("unable to parse time %s: %v", s, err)
			}
			return nil
		},
	},
	PaywallCompletionsLimit: {
		preventDeletion: false,
		defaultValue:    toPtr("3"),
		validate: func(s string) error {
			_, err := strconv.Atoi(s)
			return err
		},
	},
	PaywallCompletionsRemaining: {
		preventDeletion: false,
		defaultValue:    toPtr("3"),
		validate: func(s string) error {
			_, err := strconv.Atoi(s)
			return err
		},
	},
	PaywallLastUpdated: {
		preventDeletion: false,
		defaultValue:    toPtr(time.Time{}.Format(time.RFC3339)),
		validate: func(s string) error {
			if _, err := time.Parse(time.RFC3339, s); err != nil {
				return errors.Errorf("unable to parse time %s: %v", s, err)
			}
			return nil
		},
	},
	ShowPaywallExhaustedNotif: {
		preventDeletion: false,
		defaultValue:    toPtr(strconv.FormatBool(true)),
		validate:        validateBool,
	},
	KiteServer: {
		preventDeletion: false,
		defaultValue:    toPtr(""),
		validate: func(url string) error {
			if url == "" {
				return nil
			}
			_, _, err := kiteserver.GetHealth(url)
			return err
		},
	},
	ChooseEngineKey: {
		preventDeletion: false,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	SelectedEngineKey: {
		preventDeletion: false,
		defaultValue:    toPtr("local"),
		validate: func(s string) error {
			if s != "local" && s != "cloud" {
				return errors.Errorf("invalid engine id %s", s)
			}
			return nil
		},
	},
	AllFeaturesPro: {
		preventDeletion: false,
		defaultValue:    toPtr(strconv.FormatBool(false)),
		validate:        validateBool,
	},
	EmailRequiredKey: {
		defaultValue: nil,
		validate:     validateBool,
	},
}

func validateBool(s string) error {
	if _, err := strconv.ParseBool(s); err != nil {
		return errors.Errorf("unable to parse boolean %s: %v", s, err)
	}
	return nil
}

// --

// keySpec is a collection of specifications about the behavior of certain keys
type keySpec struct {
	validate        func(string) error
	preventDeletion bool
	defaultValue    *string
}

func toPtr(val string) *string {
	return &val
}
