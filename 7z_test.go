package archiver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hayden-pan/archiver/v3"
)

func TestUnarchiveSevenZip(t *testing.T) {
	dst := t.TempDir()
	arc := &archiver.SevenZip{}

	if err := arc.Unarchive("testdata/testarchives/7z/normal.7z", dst); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test file.")) {
		t.Fatal("file1.txt content mismatch")
	}
	content, err = os.ReadFile(filepath.Join(dst, "file2.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is the second file.")) {
		t.Fatal("file2.txt content mismatch")
	}

	if err := os.WriteFile(filepath.Join(dst, "file1.txt"), []byte("This is a test file. Modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test reuse instance
	if err := arc.Unarchive("testdata/testarchives/7z/normal.7z", dst); err == nil {
		t.Fatal("expected error when reuse instance")
	}

	content, err = os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test file. Modified")) {
		t.Fatal("file1.txt content mismatch")
	}

	// Test overwrite existing files
	arc = &archiver.SevenZip{SkipExistingFiles: true}
	if err := arc.Unarchive("testdata/testarchives/7z/normal.7z", dst); err != nil {
		t.Fatal(err)
	}

	content, err = os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test file. Modified")) {
		t.Fatal("file1.txt content mismatch")
	}
	content, err = os.ReadFile(filepath.Join(dst, "file2.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is the second file.")) {
		t.Fatal("file2.txt content mismatch")
	}

	arc = &archiver.SevenZip{SkipExistingFiles: false}
	if err := arc.Unarchive("testdata/testarchives/7z/normal.7z", dst); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test file.")) {
		t.Fatal("file1.txt content mismatch")
	}
	content, err = os.ReadFile(filepath.Join(dst, "file2.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is the second file.")) {
		t.Fatal("file2.txt content mismatch")
	}
}

func TestUnarchiveSevenZipWithPassword(t *testing.T) {
	dst := t.TempDir()
	arc := &archiver.SevenZip{Password: "password"}

	// Test with wrong password
	if err := arc.Unarchive("testdata/testarchives/7z/encrypted-normal.7z", dst); err == nil {
		t.Fatal(err)
	}

	// Test with correct password
	arc = &archiver.SevenZip{Password: "mypassword"}
	if err := arc.Unarchive("testdata/testarchives/7z/encrypted-normal.7z", dst); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test file.")) {
		t.Fatal("file1.txt content mismatch")
	}
}

func TestUnarchiveSevenZipWithPasswordAndCopyOnly(t *testing.T) {
	dst := t.TempDir()
	arc := &archiver.SevenZip{Password: "password"}

	correctContent := []byte("This is a test file.")

	// Test with wrong password
	if err := arc.Unarchive("testdata/testarchives/7z/encrypted-copy.7z", dst); err == nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(content, correctContent) {
		t.Fatal("file1.txt content should be incorrect with wrong password")
	}

	// Test with correct password
	arc = &archiver.SevenZip{Password: "mypassword"}
	if err := arc.Unarchive("testdata/testarchives/7z/encrypted-copy.7z", dst); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, correctContent) {
		t.Fatal("file1.txt content mismatch")
	}
}

func TestUnarchiveSevenZipWithCopyOnly(t *testing.T) {
	dst := t.TempDir()
	arc := &archiver.SevenZip{}

	if err := arc.Unarchive("testdata/testarchives/7z/copy.7z", dst); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test file.")) {
		t.Fatal("file1.txt content mismatch")
	}

	arc = &archiver.SevenZip{}
	if err := arc.Unarchive("testdata/testarchives/7z/copy-corrupt.7z", dst); err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchSevenZip(t *testing.T) {
	arc := &archiver.SevenZip{}

	f, err := os.Open("testdata/testarchives/7z/normal.7z")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	ok, err := arc.Match(f)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ok {
		t.Fatal("expected match")
	}

	f2, err := os.Open("testdata/sample.rar")
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	ok, err = arc.Match(f2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ok {
		t.Fatal("expected no match")
	}
}

func TestUnarchiveSevenZipPreserveModTime(t *testing.T) {
	dst := t.TempDir()
	arc := &archiver.SevenZip{}

	modTime := time.Unix(1733564187, 0)

	if err := arc.Unarchive("testdata/testarchives/7z/normal.7z", dst); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if fi.ModTime().Unix() != modTime.Unix() {
		t.Fatalf("expected modified timestamp %v, got %v", modTime.Unix(), fi.ModTime().Unix())
	}
}
