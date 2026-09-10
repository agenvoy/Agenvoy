package tui

func msgError(text string) string {
	return errorStyle.Render("[!] " + text)
}

func msgWarn(text string) string {
	return warnStyle.Render("[~] " + text)
}

func msgLog(text string) string {
	return hintStyle.Render("[*] " + text)
}

func msgCode(text string) string {
	return systemStyle.Render("[*] " + text)
}
