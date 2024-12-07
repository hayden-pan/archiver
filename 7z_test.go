package archiver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

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
}

func TestUnarchiveSevenZipWithPassword(t *testing.T) {
	dst := t.TempDir()
	arc := &archiver.SevenZip{OverwriteExisting: true, Password: "password"}

	// Test with wrong password
	if err := arc.Unarchive("testdata/testarchives/7z/encrypted-normal.7z", dst); err == nil {
		t.Fatal(err)
	}

	// Test with correct password
	arc.Password = "mypassword"
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
	arc := &archiver.SevenZip{OverwriteExisting: true, Password: "password"}

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
	arc.Password = "mypassword"
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
	arc := &archiver.SevenZip{OverwriteExisting: true}

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

	if err := arc.Unarchive("testdata/testarchives/7z/copy-corrupt.7z", dst); err == nil {
		t.Fatal("expected error")
	}

	arc.SkipVerifyChecksum = true
	if err := arc.Unarchive("testdata/testarchives/7z/copy-corrupt.7z", dst); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(filepath.Join(dst, "file1.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, []byte("This is a test filf.")) {
		t.Fatalf("file1.txt content mismatch, got %s", content)
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
