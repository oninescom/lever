package utils

const (
	ColorReset     = "\033[0m" // 彻底重置所有颜色与状态（每行末尾必须加，否则会染色整屏幕）
	ColorBold      = "\033[1m" // 纯文本加粗
	ColorUnderline = "\033[4m" // 下划线模式
)

const (
	ColorRed     = "\033[31m" // 红色：通常用于 grep 关键词命中、严重错误提示
	ColorGreen   = "\033[32m" // 绿色：通常用于 ls 识别出的可执行文件 (.exe, .bat)
	ColorYellow  = "\033[33m" // 黄色：通常用于警告信息、提示说明
	ColorBlue    = "\033[34m" // 蓝色：通常用于 ls 识别出的子文件夹
	ColorMagenta = "\033[35m" // 品红/紫：通常用于软链接、特殊套接字文件
	ColorCyan    = "\033[36m" // 青色：通常用于网络端口、特定协议高亮
	ColorWhite   = "\033[37m" // 白色
)

const (
	ColorBoldRed   = "\033[1;31m" // 加粗红（grep 的默认官方高亮色）
	ColorBoldGreen = "\033[1;32m" // 加粗绿
	ColorBoldBlue  = "\033[1;34m" // 加粗蓝（Linux 官方目录的经典色）
	ColorBoldCyan  = "\033[1;36m" // 加粗青
)
