// Package echo is a test backend that writes the model it is given as
// JSON and imports that JSON back, so the import protocol is exercised
// before any real reverse backend exists. It is not compiled into tdl.
package echo

import (
	"context"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// Name is what this backend is called, and tdl-gen-echo on PATH.
const Name = "echo"

// File is the one file Generate writes.
const File = "model.json"

// Backend implements [plugin.Importer].
type Backend struct{}

func (Backend) Describe() plugin.Description {
	return plugin.Description{Name: Name, Version: "0.1.0", Reverse: true}
}

// Generate writes the model as JSON.
func (Backend) Generate(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	data, err := protojson.Marshal(req.GetModel())
	if err != nil {
		return nil, err
	}
	return &plugin.Response{Files: []*plugin.File{{Path: File, Content: data}}}, nil
}

// Import reads the first file as a model in JSON. Every import warns
// with lossy.order, ignoring the request's allowed codes, so the host's
// filter has something to drop.
func (Backend) Import(_ context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	resp := &plugin.ImportResponse{}
	files := req.GetFiles()
	if len(files) == 0 {
		resp.Diagnostics = append(resp.Diagnostics, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_ERROR,
			Message:  "no files to import",
		})
		return resp, nil
	}

	pos := &ir.Position{Filename: files[0].GetPath(), Line: 1, Column: 1}
	var model ir.Model
	if err := protojson.Unmarshal(files[0].GetContent(), &model); err != nil {
		resp.Diagnostics = append(resp.Diagnostics, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_ERROR,
			Message:  err.Error(),
			Position: pos,
		})
		return resp, nil
	}
	resp.Model = &model

	resp.Diagnostics = append(resp.Diagnostics, &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  "echo keeps the order JSON lists declarations in",
		Position: pos,
		Code:     "lossy.order",
	})
	return resp, nil
}
