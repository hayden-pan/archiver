package sevenzip

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unsafe"

	_ "embed"

	"golang.org/x/sys/windows"
)

//go:embed 7z.exe
var sevenZipExe []byte

//go:embed 7z.dll
var sevenZipLib []byte

type embeddedFile struct {
	data      []byte
	name      string
	execuable bool
}

var embeddedFiles = []embeddedFile{
	{data: sevenZipLib, name: "7z.dll"},
	{data: sevenZipExe, name: "7z.exe", execuable: true},
}

type sevenZip struct {
	// Whether to skip extracting of existing files.
	SkipExistingFiles bool

	// The password to open archives (optional).
	Password string

	used atomic.Bool

	tempDir string

	binPath string
}

func newSevenZip(options SevenZipOptions) *sevenZip {
	return &sevenZip{
		SkipExistingFiles: options.SkipExistingFiles,
		Password:          options.Password,
	}
}

func (s *sevenZip) Extract(ctx context.Context, source, destination string) error {
	if err := s.prepareEmbeddedFiles(); err != nil {
		return err
	}
	defer s.cleanupTempDir()

	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create job object: %w", err)
	}
	defer windows.CloseHandle(job)
	if err := setInformation(job); err != nil {
		return err
	}

	overwriteMode := "-aoa"
	if s.SkipExistingFiles {
		overwriteMode = "-aos"
	}
	opt := []string{
		"x", "-y", overwriteMode, "-p" + s.Password, "-o" + filepath.Clean(destination), filepath.Clean(source),
	}
	cmd := exec.CommandContext(ctx, s.binPath, opt...)
	cmdStr := s.binPath + " " + strings.Join(opt, " ")

	var buf bytes.Buffer
	cmd.Stderr = &buf
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %s, err: %w", cmdStr, err)
	}
	defer func() {
		_ = cmd.Process.Kill()
	}()

	handle, err := windows.OpenProcess(windows.PROCESS_ALL_ACCESS, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("failed to open process: %w", err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.AssignProcessToJobObject(job, handle); err != nil {
		return fmt.Errorf("failed to assign process to job object: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		out := buf.Bytes()
		return fmt.Errorf("failed to extract archive cmd: %s, err: %w, stdout&stderr: %s", cmdStr, err, out)
	}

	return nil
}

func (s *sevenZip) prepareEmbeddedFiles() error {
	if s.used.Swap(true) {
		return errors.New("the instance of the 7z unarchiver has already been used, please create another instance")
	}
	dir, err := os.MkdirTemp("", "archiver-7z-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	s.tempDir = dir
	for _, file := range embeddedFiles {
		path := filepath.Join(dir, file.name)
		if err := os.WriteFile(path, file.data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.name, err)
		}
		if file.execuable {
			if s.binPath != "" {
				return errors.New("more than one executable file found")
			}
			s.binPath = path
		}
	}
	return nil
}

func (s *sevenZip) cleanupTempDir() {
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func setInformation(job windows.Handle) error {
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		return fmt.Errorf("failed to set information job object: %w", err)
	}
	return nil
}
