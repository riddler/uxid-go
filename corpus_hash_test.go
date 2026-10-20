package uxid

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The vendored corpus is the one its manifest and conformance/SOURCE.json
// record, by the reference's rule (the corpus README's "The format"): the
// corpus hash is the string sha256: followed by the lowercase hex SHA-256
// of the case files' bytes concatenated in the order of the manifest's
// files, and a file's case_count is its number of cases, one per line. A
// mismatch means the vendored copy is not the one recorded; it is never
// repaired by rewriting a hash.

// corpusSource is conformance/SOURCE.json: where the vendored corpus came
// from.
type corpusSource struct {
	Repo       string `json:"repo"`
	Tag        string `json:"tag"`
	Sha        string `json:"sha"`
	CorpusHash string `json:"corpus_hash"`
}

// corpusHash is the corpus hash of the files manifest lists, read from
// corpusDir.
func corpusHash(t *testing.T, manifest corpusManifest) string {
	t.Helper()
	digest := sha256.New()
	for _, entry := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(corpusDir, entry.File))
		if err != nil {
			t.Fatal(err)
		}
		digest.Write(data)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

// sabotage: in corpusHash, hash the manifest's files in reverse order;
// this test goes red.
func TestCorpusHashIsTheManifests(t *testing.T) {
	manifest := readManifest(t)
	if got := corpusHash(t, manifest); got != manifest.CorpusHash {
		t.Fatalf("the case files hash to %s, the manifest records %s", got, manifest.CorpusHash)
	}
}

// sabotage: in caseLines, drop the last line; every file's row goes red.
func TestCorpusCaseCountsAreTheManifests(t *testing.T) {
	for _, entry := range readManifest(t).Files {
		data, err := os.ReadFile(filepath.Join(corpusDir, entry.File))
		if err != nil {
			t.Fatal(err)
		}
		lines, err := caseLines(data)
		if err != nil {
			t.Fatalf("%s: %v", entry.File, err)
		}
		if len(lines) != entry.CaseCount {
			t.Errorf("%s holds %d cases, the manifest records %d", entry.File, len(lines), entry.CaseCount)
		}
	}
}

// SOURCE.json names the reference repository, a tag, a full lowercase
// commit SHA and the corpus hash, which is the manifest's and the one the
// vendored files hash to.
//
// sabotage: change one byte of conformance/encode.json; this test goes
// red.
func TestCorpusSourceRecordsTheHash(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(corpusDir, "SOURCE.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source corpusSource
	if err := decodeStrict(data, &source); err != nil {
		t.Fatalf("SOURCE.json: %v", err)
	}
	if source.Repo != "riddler/uxid-ex" {
		t.Errorf("repo %q", source.Repo)
	}
	if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(source.Tag) {
		t.Errorf("tag %q is not a release tag", source.Tag)
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(source.Sha) {
		t.Errorf("sha %q is not a full lowercase commit SHA", source.Sha)
	}
	manifest := readManifest(t)
	if source.CorpusHash != manifest.CorpusHash {
		t.Errorf("SOURCE.json records %s, the manifest %s", source.CorpusHash, manifest.CorpusHash)
	}
	if got := corpusHash(t, manifest); source.CorpusHash != got {
		t.Errorf("SOURCE.json records %s, the case files hash to %s", source.CorpusHash, got)
	}
}
