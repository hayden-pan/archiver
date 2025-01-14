package archiver

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (s *SevenZip) extract(ctx context.Context, source, destination string) error {
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
