package listener

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type CallbackOptions struct {
	OnFetchError func(error)
}

type RequestOptions struct {
	Url         string
	Environment string
	Cluster     string
	Instance    string
	Sections    []string
}

func (opts *RequestOptions) Validate() error {
	if _, err := url.ParseRequestURI(opts.Url); err != nil {
		return fmt.Errorf("invalid URL %s: %w", opts.Url, err)
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

	requestURL := fmt.Sprintf("%s?%s", opts.Url, params.Encode())
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

type Options struct {
	Request    RequestOptions
	Interval   IntervalOptions
	Callback   CallbackOptions
	OutputFile string
}

func (opts *Options) Validate() error {
	if err := opts.Request.Validate(); err != nil {
		return fmt.Errorf("bad request options: %w", err)
	}

	if err := opts.Interval.Validate(); err != nil {
		return fmt.Errorf("bad request options: %w", err)
	}

	info, err := os.Stat(opts.OutputFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("output file %s doesn't exist: %w", opts.OutputFile, err)
		}
		return fmt.Errorf("failed to check if output file %s exists: %w", opts.OutputFile, err)
	}

	if info.IsDir() {
		return fmt.Errorf("output file %s is a directory", opts.OutputFile)
	}

	file, err := os.OpenFile(opts.OutputFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("output file %s can't be opened for writing: %w", opts.OutputFile, err)
	}
	file.Close()

	return nil
}
