package commands

import (
	"bufio"
	"fmt"
	"io"
	"lever/engine"
	"lever/utils"
	"os"
	"strconv"
	"strings"
)

func NewAwkCmd() *engine.Command {
	var fieldDelimiter string

	cmd := &engine.Command{
		Use:         "awk [column_indices...]",
		Short:       "简化版的 awk 文本流列切片工具",
		Long:        `接收管道流或标准输入，将每行文本按分隔符切片。参数传入要提取的列索引（1代表第一列，-1代表最后一列）。`,
		PrepareArgs: awkFlagArgs,
		// 💡 强力复用你上传的校验器：至少需要指定一个要提取的列索引
		Args: utils.MinimumNArgs(1),
		RunE: func(c *engine.Command, args []string) error {

			// 1. 解析用户传入的列索引切片（如将 "1", "2" 转为 int 数组）
			var targetCols []int
			for _, str := range args {
				idx, err := strconv.Atoi(str)
				if err != nil {
					return fmt.Errorf("awk 错误: 无效的列索引 '%s'，必须为整数", str)
				}
				targetCols = append(targetCols, idx)
			}

			// 2. 管道数据流判定与拦截处理
			if utils.StdinIsPipe() {
				return processAwkStream(os.Stdin, os.Stdout, fieldDelimiter, targetCols)
			} else {
				return fmt.Errorf("awk 错误: 未检测到管道输入流。用法示例: ls | awk 1 5")
			}
		},
	}

	flags := cmd.Flags()
	// 💡 -F 参数让用户能自由修改列切割符（如 -F , 或 -F :）
	flags.StringVarP(&fieldDelimiter, "field-separator", "F", "", "指定每行的字段分隔符 (默认使用连续空白字符)")

	return cmd
}

func awkFlagArgs(args []string) []string {
	var flags, indices []string
	positionalOnly := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if positionalOnly {
			indices = append(indices, arg)
			continue
		}
		switch {
		case arg == "--":
			positionalOnly = true
		case arg == "-F" || arg == "--field-separator":
			flags = append(flags, arg)
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		case arg == "-h" || arg == "--help" || strings.HasPrefix(arg, "-F") || strings.HasPrefix(arg, "--field-separator="):
			flags = append(flags, arg)
		case strings.HasPrefix(arg, "-") && !isNegativeIndex(arg):
			flags = append(flags, arg)
		default:
			indices = append(indices, arg)
		}
	}
	if len(indices) == 0 {
		return flags
	}
	return append(flags, append([]string{"--"}, indices...)...)
}

func isNegativeIndex(value string) bool {
	return len(value) > 1 && value[0] == '-' && strings.Trim(value[1:], "0123456789") == ""
}

// 核心流处理函数：逐行扫描，高能切片
func processAwkStream(reader io.Reader, writer io.Writer, delimiter string, targetCols []int) error {
	input := bufio.NewReader(reader)
	for {
		line, readErr := input.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		if readErr == io.EOF && line == "" {
			return nil
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		var columns []string

		// 3. 字段切割算法分流
		if delimiter == "" {
			// 默认行为：将连续的空格、制表符等全部视作单个分隔符进行盲切 [INDEX]
			columns = strings.Fields(line)
		} else {
			// 自定义字符切割模式 [INDEX]
			columns = strings.Split(line, delimiter)
		}

		// 如果切出来的列是空的，直接跳过本行
		if len(columns) == 0 {
			if readErr == io.EOF {
				return nil
			}
			continue
		}

		// 4. 根据用户索引提取并拼接目标列 [INDEX]
		var result []string
		for _, colIdx := range targetCols {
			actualIdx := 0

			if colIdx > 0 {
				// 正数情况：1 代表 columns[0]
				actualIdx = colIdx - 1
			} else if colIdx < 0 {
				// 负数情况（极客独创扩展）：-1 代表最后一列，-2 代表倒数第二列
				actualIdx = len(columns) + colIdx
			} else {
				// 传入 0 时，按照标准 Linux awk 习惯，直接输出整行 [INDEX]
				result = append(result, line)
				continue
			}

			// 安全防护边界检查：防止用户要提取的列数超出当前行实际拥有的总列数
			if actualIdx >= 0 && actualIdx < len(columns) {
				result = append(result, columns[actualIdx])
			} else {
				result = append(result, "") // 超出范围补空字符串，防止崩溃
			}
		}

		if _, err := fmt.Fprintln(writer, strings.Join(result, " ")); err != nil {
			return err
		}
		if readErr == io.EOF {
			return nil
		}
	}
}
