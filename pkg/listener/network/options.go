package network

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	ErrInvalidBaseConfigurationType = errors.New("invalid default configuration type")
)

type BaseConfigurationType uint32

const (
	BaseConfigurationTypeString BaseConfigurationType = iota
	BaseConfigurationTypeFile
)

type BaseConfigurationOptions struct {
	Type   BaseConfigurationType
	String string
	File   string
}

func (options *BaseConfigurationOptions) Unmarshal() (map[string]any, error) {
	result := make(map[string]any)
	var bytes []byte

	switch options.Type {
	case BaseConfigurationTypeString:
		bytes = []byte(options.String)
	case BaseConfigurationTypeFile:
		data, err := os.ReadFile(options.File)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", options.File, err)
		}
		bytes = data
	default:
		return nil, ErrInvalidBaseConfigurationType
	}

	if err := yaml.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base configuration: %w", err)
	}
	return result, nil
}

type CallbackOptions struct {
	OnFetchError func(error)
}

type RequestOptions struct {
	URL         string
	Environment string
	Cluster     string
	Instance    string
	Sections    []string
}

func (opts *RequestOptions) Validate() error {
	if _, err := url.ParseRequestURI(opts.URL); err != nil {
		return fmt.Errorf("invalid URL %s: %w", opts.URL, err)
	}

	if len(opts.Sections) == 0 {
		return fmt.Errorf("at least one section required")
	}

	return nil
}

func (opts *RequestOptions) Build() string {
	params := url.Values{}
	params.Set("environment", opts.Environment)
	params.Set("cluster", opts.Cluster)
	params.Set("instance", opts.Instance)
	params.Set("sections", strings.Join(opts.Sections, ","))

	requestURL := fmt.Sprintf("%s?%s", opts.URL, params.Encode())
	return requestURL
}

type IntervalOptions struct {
	RequestIntervalEnabled bool
	RequestInterval        time.Duration
	MaximumInitialJitter   time.Duration
}

func (opts *IntervalOptions) Validate() error {
	if opts.RequestIntervalEnabled && opts.RequestInterval < 0 {
		return fmt.Errorf("invalid request interval %v", opts.RequestInterval)
	}

	if opts.MaximumInitialJitter < 0 {
		return fmt.Errorf("invalid maximum initial jitter %v", opts.MaximumInitialJitter)
	}

	return nil
}

type OutputMode int

const (
	OutputModeFile OutputMode = iota
	OutputModeCallback
)

type OutputOptions[Configuration any] struct {
	Mode     OutputMode
	Callback func(Configuration) error
	File     string
}

func (opts *OutputOptions[Configuration]) Validate() error {
	switch opts.Mode {
	case OutputModeCallback:
		if opts.Callback == nil {
			return fmt.Errorf("nil output callback")
		}
		return nil
	case OutputModeFile:
		info, err := os.Stat(opts.File)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("output file %s doesn't exist: %w", opts.File, err)
			}
			return fmt.Errorf("failed to check if output file %s exists: %w", opts.File, err)
		}

		if info.IsDir() {
			return fmt.Errorf("output file %s is a directory", opts.File)
		}

		file, err := os.OpenFile(opts.File, os.O_WRONLY, 0)
		if err != nil {
			return fmt.Errorf("output file %s can't be opened for writing: %w", opts.File, err)
		}
		file.Close()

		return nil
	default:
		return fmt.Errorf("invalid output callback mode")
	}
}

type Options[Configuration any] struct {
	Request  RequestOptions
	Interval IntervalOptions
	// The base configuration is the configuration that you start with. Its options define things like where it
	// originates from (etc. a file or a string).
	BaseConfiguration BaseConfigurationOptions
	Callback          CallbackOptions
	Output            OutputOptions[Configuration]
}

func (opts *Options[Configuration]) Validate() error {
	if err := opts.Request.Validate(); err != nil {
		return fmt.Errorf("bad request options: %w", err)
	}

	if err := opts.Interval.Validate(); err != nil {
		return fmt.Errorf("bad interval options: %w", err)
	}

	if err := opts.Output.Validate(); err != nil {
		return fmt.Errorf("bad output options: %w", err)
	}

	return nil
}
