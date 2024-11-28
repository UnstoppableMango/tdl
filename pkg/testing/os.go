package testing

import (
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/spf13/afero"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type MockOs struct {
	FsValue     afero.Fs
	StderrValue io.Writer
	StdoutValue io.Writer
	StdinValue  tdl.Stdin
}

// Fs implements tdl.OS.
func (m *MockOs) Fs() afero.Fs {
	if m.FsValue == nil {
		m.FsValue = afero.NewMemMapFs()
	}

	return m.FsValue
}

// Stderr implements tdl.OS.
func (m *MockOs) Stderr() io.Writer {
	return m.StderrValue
}

// Stdin implements tdl.OS.
func (m *MockOs) Stdin() tdl.Stdin {
	return m.StdinValue
}

// Stdout implements tdl.OS.
func (m *MockOs) Stdout() io.Writer {
	return m.StdoutValue
}

var _ tdl.OS = &MockOs{}

type stdin struct {
	io.Reader
	StatFn func() (fs.FileInfo, error)
}

func (m *stdin) Stat() (fs.FileInfo, error) {
	if m.StatFn == nil {
		panic("unimplemented")
	}

	return m.StatFn()
}

type MockFileInfo struct {
	IsDirFn   func() bool
	ModTimeFn func() time.Time
	ModeFn    func() fs.FileMode
	NameFn    func() string
	SizeFn    func() int64
	SysFn     func() any
}

// IsDir implements fs.FileInfo.
func (f MockFileInfo) IsDir() bool {
	if f.IsDirFn == nil {
		panic("unimplemented")
	}

	return f.IsDirFn()
}

// ModTime implements fs.FileInfo.
func (f MockFileInfo) ModTime() time.Time {
	if f.ModTimeFn == nil {
		panic("unimplemented")
	}

	return f.ModTimeFn()
}

// Mode implements fs.FileInfo.
func (f MockFileInfo) Mode() fs.FileMode {
	if f.ModeFn == nil {
		panic("unimplemented")
	}

	return f.ModeFn()
}

// Name implements fs.FileInfo.
func (f MockFileInfo) Name() string {
	if f.NameFn == nil {
		panic("unimplemented")
	}

	return f.NameFn()
}

// Size implements fs.FileInfo.
func (f MockFileInfo) Size() int64 {
	if f.SizeFn == nil {
		panic("unimplemented")
	}

	return f.SizeFn()
}

// Sys implements fs.FileInfo.
func (f MockFileInfo) Sys() any {
	if f.SysFn == nil {
		panic("unimplemented")
	}

	return f.SizeFn()
}

func MockOsStdin(reader io.Reader) tdl.Stdin {
	return &stdin{
		Reader: reader,
		StatFn: func() (fs.FileInfo, error) {
			return MockFileInfo{
				ModeFn: func() fs.FileMode {
					return os.ModeDevice
				},
			}, nil
		},
	}
}

func MockTermStdin() tdl.Stdin {
	return &stdin{
		Reader: nil,
		StatFn: func() (fs.FileInfo, error) {
			return MockFileInfo{
				ModeFn: func() fs.FileMode {
					return os.ModeCharDevice
				},
			}, nil
		},
	}
}
