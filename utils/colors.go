package utils

const (
	ColorReset     = "\033[0m" // Reset all color and style attributes.
	ColorBold      = "\033[1m" // Bold text.
	ColorUnderline = "\033[4m" // Underlined text.
)

const (
	ColorRed     = "\033[31m" // Matches and errors.
	ColorGreen   = "\033[32m" // Executable files.
	ColorYellow  = "\033[33m" // Warnings.
	ColorBlue    = "\033[34m" // Directories.
	ColorMagenta = "\033[35m" // Links and special files.
	ColorCyan    = "\033[36m" // Network addresses and protocols.
	ColorWhite   = "\033[37m" // White text.
)

const (
	ColorBoldRed   = "\033[1;31m" // Highlighted grep matches.
	ColorBoldGreen = "\033[1;32m" // Bold green text.
	ColorBoldBlue  = "\033[1;34m" // Directory names.
	ColorBoldCyan  = "\033[1;36m" // Bold cyan text.
)
