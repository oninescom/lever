package commands

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"lever/engine"
	"lever/utils"
)

type tailEncoding uint8

const (
	tailUTF8 tailEncoding = iota
	tailUTF16LE
	tailUTF16BE
)

func NewTailCmd() *engine.Command {
	var follow bool
	var lineCount int64 = 10
	cmd := &engine.Command{
		Use:         "tail [file]",
		Short:       "Show the end of a file and follow new content",
		PrepareArgs: tailFlagArgs,
		Args:        utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {
			var filename string
			for _, arg := range args {
				if count, err := strconv.ParseInt(arg, 10, 64); err == nil {
					lineCount = count
					if lineCount < 0 {
						lineCount = -lineCount
					}
				} else {
					filename = arg
				}
			}
			if filename == "" {
				return fmt.Errorf("tail error: no file specified")
			}
			return runTailEngine(filename, lineCount, follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow new content appended to the file")
	return cmd
}

func tailFlagArgs(args []string) []string {
	var flags, positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) > 1 && arg[0] == '-' && strings.Trim(arg[1:], "0123456789") == "" {
			positionals = append(positionals, arg[1:])
			continue
		}
		if (arg == "-n" || arg == "--lines") && i+1 < len(args) {
			positionals = append(positionals, args[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		} else {
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) == 0 {
		return flags
	}
	return append(flags, append([]string{"--"}, positionals...)...)
}

func runTailEngine(filename string, lines int64, follow bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot open file %s: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	encoding, prefix, err := detectTailEncoding(file)
	if err != nil {
		return err
	}
	start, err := tailStart(file, stat.Size(), lines, encoding, prefix)
	if err != nil {
		return err
	}
	consumed, err := writeTailRange(file, os.Stdout, start, stat.Size(), encoding)
	if err != nil {
		return err
	}
	if !follow {
		return nil
	}
	currentOffset := start + consumed
	for {
		time.Sleep(100 * time.Millisecond)
		newStat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !os.SameFile(stat, newStat) {
			replacement, err := os.Open(filename)
			if err != nil {
				continue
			}
			_ = file.Close()
			file = replacement
			stat = newStat
			encoding, prefix, err = detectTailEncoding(file)
			if err != nil {
				return err
			}
			currentOffset = prefix
		}
		if newStat.Size() < currentOffset {
			encoding, prefix, err = detectTailEncoding(file)
			if err != nil {
				return err
			}
			currentOffset = prefix
		}
		if newStat.Size() <= currentOffset {
			continue
		}
		consumed, err := writeTailRange(file, os.Stdout, currentOffset, newStat.Size(), encoding)
		if err != nil {
			return err
		}
		currentOffset += consumed
	}
}

func detectTailEncoding(file *os.File) (tailEncoding, int64, error) {
	var header [2]byte
	n, err := file.ReadAt(header[:], 0)
	if err != nil && err != io.EOF {
		return tailUTF8, 0, err
	}
	if n == 2 {
		switch header {
		case [2]byte{0xff, 0xfe}:
			return tailUTF16LE, 2, nil
		case [2]byte{0xfe, 0xff}:
			return tailUTF16BE, 2, nil
		}
	}
	return tailUTF8, 0, nil
}

func tailStart(file *os.File, size, lines int64, encoding tailEncoding, prefix int64) (int64, error) {
	if lines == 0 {
		return size, nil
	}
	const blockSize = 4096
	buf := make([]byte, blockSize)
	remaining := lines
	end := size
	if encoding != tailUTF8 {
		end -= (end - prefix) % 2
	}
	for end > prefix {
		start := max(end-blockSize, prefix)
		chunk := buf[:int(end-start)]
		if _, err := file.ReadAt(chunk, start); err != nil {
			return 0, err
		}
		if encoding == tailUTF8 {
			for i := len(chunk) - 1; i >= 0; i-- {
				if chunk[i] != '\n' || start+int64(i) == size-1 {
					continue
				}
				remaining--
				if remaining == 0 {
					return start + int64(i) + 1, nil
				}
			}
		} else {
			for i := len(chunk) - 2; i >= 0; i -= 2 {
				var codeUnit uint16
				if encoding == tailUTF16LE {
					codeUnit = binary.LittleEndian.Uint16(chunk[i:])
				} else {
					codeUnit = binary.BigEndian.Uint16(chunk[i:])
				}
				if codeUnit != '\n' || start+int64(i) == endOfCompleteUnits(size, prefix)-2 {
					continue
				}
				remaining--
				if remaining == 0 {
					return start + int64(i) + 2, nil
				}
			}
		}
		end = start
	}
	return prefix, nil
}

func endOfCompleteUnits(size, prefix int64) int64 {
	return size - (size-prefix)%2
}

func writeTailRange(file *os.File, output io.Writer, start, end int64, encoding tailEncoding) (int64, error) {
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return 0, err
	}
	if encoding == tailUTF8 {
		return io.Copy(output, io.LimitReader(file, end-start))
	}
	completeEnd := endOfCompleteUnits(end, start)
	reader := bufio.NewReaderSize(io.LimitReader(file, completeEnd-start), 32*1024)
	var order binary.ByteOrder = binary.LittleEndian
	if encoding == tailUTF16BE {
		order = binary.BigEndian
	}
	var pair [2]byte
	var pending uint16
	var consumed int64
	buf := make([]byte, 0, 32*1024)
	flush := func() error {
		if len(buf) == 0 {
			return nil
		}
		_, err := output.Write(buf)
		buf = buf[:0]
		return err
	}
	appendRune := func(r rune) error {
		buf = utf8.AppendRune(buf, r)
		if len(buf) >= 32*1024 {
			return flush()
		}
		return nil
	}
	for {
		_, err := io.ReadFull(reader, pair[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return consumed, err
		}
		consumed += 2
		unit := order.Uint16(pair[:])
		if pending != 0 {
			if unit >= 0xdc00 && unit <= 0xdfff {
				if err := appendRune(utf16.DecodeRune(rune(pending), rune(unit))); err != nil {
					return consumed, err
				}
				pending = 0
				continue
			}
			if err := appendRune(utf8.RuneError); err != nil {
				return consumed, err
			}
			pending = 0
		}
		switch {
		case unit >= 0xd800 && unit <= 0xdbff:
			pending = unit
		case unit >= 0xdc00 && unit <= 0xdfff:
			if err := appendRune(utf8.RuneError); err != nil {
				return consumed, err
			}
		default:
			if err := appendRune(rune(unit)); err != nil {
				return consumed, err
			}
		}
	}
	if err := flush(); err != nil {
		return consumed, err
	}
	if pending != 0 {
		consumed -= 2
	}
	return consumed, nil
}
