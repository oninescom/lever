package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unicode/utf16"

	"github.com/spf13/pflag"
)

func EnsureHelpFlag(flags *pflag.FlagSet) {
	if flags.Lookup("help") != nil {
		return
	}
	if flags.ShorthandLookup("h") == nil {
		flags.BoolP("help", "h", false, "显示帮助")
	} else {
		flags.Bool("help", false, "显示帮助")
	}
}

func NoArgs(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("不接受位置参数，收到 %d 个", len(args))
	}
	return nil
}

func MinimumNArgs(min int) func([]string) error {
	return func(args []string) error {
		if len(args) < min {
			return fmt.Errorf("至少需要 %d 个位置参数，收到 %d 个", min, len(args))
		}
		return nil
	}
}

func MaximumNArgs(max int) func([]string) error {
	return func(args []string) error {
		if len(args) > max {
			return fmt.Errorf("最多接受 %d 个位置参数，收到 %d 个", max, len(args))
		}
		return nil
	}
}

func StdinIsPipe() bool {
	stat, err := os.Stdin.Stat()
	return err == nil && stat.Mode()&os.ModeCharDevice == 0
}

func ValidMultiSourceDestination(sources []string, destination string) bool {
	if len(sources) <= 1 {
		return true
	}
	info, err := os.Stat(destination)
	return err == nil && info.IsDir()
}

// EditProfileFile 暴露给 commands/system.go 实现一键无痕安装/卸载
func EditProfileFile(path string, create bool, edit func(string) (string, error)) error {
	old, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if !create {
			return nil
		}
	} else if err != nil {
		return err
	}
	existing, encode, err := ProfileEncoding(old)
	if err != nil {
		return err
	}
	updated, err := edit(existing)
	if err != nil {
		return err
	}
	if updated == existing {
		return nil
	}
	if create {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, encode(updated), 0644)
}

func ProfileEncoding(data []byte) (string, func(string) []byte, error) {
	for _, format := range []struct {
		Bom   []byte
		Order binary.ByteOrder
	}{
		{[]byte{0xff, 0xfe}, binary.LittleEndian},
		{[]byte{0xfe, 0xff}, binary.BigEndian},
	} {
		if !bytes.HasPrefix(data, format.Bom) {
			continue
		}
		body := data[len(format.Bom):]
		if len(body)%2 != 0 {
			return "", nil, fmt.Errorf("PowerShell 配置文件的 UTF-16 内容不完整")
		}
		units := make([]uint16, len(body)/2)
		for i := range units {
			units[i] = format.Order.Uint16(body[i*2:])
		}
		encode := func(value string) []byte {
			units := utf16.Encode([]rune(value))
			result := append([]byte(nil), format.Bom...)
			for _, unit := range units {
				result = append(result, 0, 0)
				format.Order.PutUint16(result[len(result)-2:], unit)
			}
			return result
		}
		return string(utf16.Decode(units)), encode, nil
	}
	return string(data), func(value string) []byte { return []byte(value) }, nil
}

func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func IsWindowsHidden(path string) bool {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		return false
	}
	return attributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}
