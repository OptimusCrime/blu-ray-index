// Package resterr defines a typed HTTP error used to communicate status codes through the render layer.
package resterr

import "errors"

// Resterr pairs an error with an HTTP status code for use with render.JSON.
type Resterr struct {
	Err        error
	StatusCode int
}

// New creates a Resterr from a string message and HTTP status code.
func New(text string, code int) Resterr {
	return Resterr{Err: errors.New(text), StatusCode: code}
}

// FromErr creates a Resterr from an existing error and HTTP status code.
func FromErr(err error, code int) Resterr {
	return Resterr{Err: err, StatusCode: code}
}
