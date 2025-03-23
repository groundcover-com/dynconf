package listener

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

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
	RequestInterval      time.Duration
	MaximumInitialJitter time.Duration
}

func (opts *IntervalOptions) Validate() error {
	if opts.RequestInterval < 0 {
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
	OutputFile string
}

func (opts *Options) Validate() error {
	if err := opts.Request.Validate(); err != nil {
		return fmt.Errorf("bad request options: %w", err)
	}

	if err := opts.Interval.Validate(); err != nil {
		return fmt.Errorf("bad request options: %w", err)
	}

	return nil
}
