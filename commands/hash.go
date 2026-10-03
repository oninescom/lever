package commands

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"strings"
)

// NewHashCmd returns a command for hashing files or standard input.
func NewHashCmd() *engine.Command {
	var algo string

	cmd := &engine.Command{
		Use:   "hash [file...]",
		Short: "Compute file or standard input checksums",
		Long:  `Compute MD5, SHA1, SHA256, SHA512, or CRC32 checksums for files or standard input.`,
		RunE: func(c *engine.Command, args []string) error {
			flags := c.Flags()
			files := flags.Args()
			algo = strings.ToLower(algo)

			var hasher hash.Hash
			switch algo {
			case "md5":
				hasher = md5.New()
			case "sha1":
				hasher = sha1.New()
			case "sha256":
				hasher = sha256.New()
			case "sha512":
				hasher = sha512.New()
			case "crc32":
				// Using the standard IEEE polynomial matrix for reliable cyclic redundancy checksums
				hasher = crc32.NewIEEE()
			default:
				return fmt.Errorf("hash error: unsupported hashing algorithm '%s'. Choose from: md5, sha1, sha256, sha512, crc32", algo)
			}

			// Read standard input when no file was supplied.
			if len(files) == 0 && utils.StdinIsPipe() {
				checksum, err := calculateHash(os.Stdin, hasher)
				if err != nil {
					return err
				}
				fmt.Printf("%s  -\n", checksum)
				return nil
			}

			if len(files) == 0 {
				_ = c.Help()
				return fmt.Errorf("hash error: no input files or standard input")
			}

			var failures []error
			for _, filename := range files {
				file, err := os.Open(filename)
				if err != nil {
					failures = append(failures, fmt.Errorf("hash error: cannot open %s: %w", filename, err))
					continue
				}

				// Reset the hasher between files.
				hasher.Reset()
				checksum, err := calculateHash(file, hasher)
				closeErr := file.Close()

				if err != nil {
					failures = append(failures, fmt.Errorf("hash error: cannot read %s: %w", filename, err))
					continue
				}
				if closeErr != nil {
					failures = append(failures, fmt.Errorf("hash error: cannot close %s: %w", filename, closeErr))
					continue
				}

				fmt.Printf("%s  %s\n", checksum, filename)
			}

			return errors.Join(failures...)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&algo, "algorithm", "a", "sha256", "Checksum algorithm (md5, sha1, sha256, sha512, crc32)")
	return cmd
}

// calculateHash computes a checksum without loading the whole input into memory.
func calculateHash(reader io.Reader, hasher hash.Hash) (string, error) {
	bufSrc := make([]byte, 32768)

	if _, err := io.CopyBuffer(hasher, reader, bufSrc); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
