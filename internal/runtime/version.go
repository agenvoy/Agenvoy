package runtime

var CurrentVersion = "dev"

func IsDev() bool {
	return CurrentVersion == "dev" || CurrentVersion == "(dev)"
}
